package mempool

import (
	"container/list"
	"sync"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

const poolCacheSize = 300000 // mempool容量

type MClient interface {
	SetQueue(q *queue.Queue)
	SendTx(tx *types.Transaction) queue.Message
}

type channelClient struct {
	qclient queue.IClient
}

type Mempool struct {
	proxyMtx sync.Mutex
	cache    *txCache
}

type txCache struct {
	// mtx  sync.Mutex
	size int
	map_ map[string]struct{}
	list *list.List
}

// NewTxCache初始化txCache
func newTxCache(cacheSize int) *txCache {
	return &txCache{
		size: cacheSize,
		map_: make(map[string]struct{}, cacheSize),
		list: list.New(),
	}
}

// txCache.Exists判断txCache中是否存在给定tx
func (cache *txCache) Exists(tx *types.Transaction) bool {
	// cache.mtx.Lock()
	_, exists := cache.map_[string(tx.Hash())]
	// cache.mtx.Unlock()
	return exists
}

// txCache.Push把给定tx添加到txCache并返回true；如果tx已经存在txCache中则返回false
func (cache *txCache) Push(tx *types.Transaction) bool {
	// cache.mtx.Lock()
	// defer cache.mtx.Unlock()

	if _, exists := cache.map_[string(tx.Hash())]; exists {
		return false
	}

	if cache.list.Len() >= cache.size {
		popped := cache.list.Front()
		poppedTx := popped.Value.(*types.Transaction)
		delete(cache.map_, string(poppedTx.Hash()))
		cache.list.Remove(popped)
	}

	cache.map_[string(tx.Hash())] = struct{}{}
	cache.list.PushBack(tx)
	return true
}

// txCache.Size返回txCache中已存tx数目
func (cache *txCache) Size() int {
	return cache.list.Len()
}

func New() *Mempool {
	pool := &Mempool{}
	pool.cache = newTxCache(poolCacheSize)
	return pool
}

// Mempool.GetTxList从txCache中返回给定数目的tx并从txCache中删除返回的tx
func (mem *Mempool) GetTxList(txListSize int) []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	txsSize := mem.cache.Size()
	var result []*types.Transaction
	var i int

	if txsSize <= txListSize {
		for i = 0; i < txsSize; i++ {
			popped := mem.cache.list.Front()
			poppedTx := popped.Value.(*types.Transaction)
			result[i] = poppedTx
			mem.cache.list.Remove(popped)
		}
		mem.Flush()
		return result
	} else {
		for i = 0; i < txListSize; i++ {
			popped := mem.cache.list.Front()
			poppedTx := popped.Value.(*types.Transaction)
			result[i] = poppedTx
			mem.cache.list.Remove(popped)
			delete(mem.cache.map_, string(poppedTx.Hash()))
		}
		return result
	}
}

// Mempool.Size返回Mempool中txCache大小
func (mem *Mempool) Size() int {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.Size()
}

// Mempool.Flush清空Mempool中的tx
func (mem *Mempool) Flush() {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	mem.cache.map_ = make(map[string]struct{}, mem.cache.size)
	mem.cache.list.Init()
}

// Mempool.CheckTx坚持tx有效性并加入Mempool中
func (mem *Mempool) CheckTx(tx *types.Transaction) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	if mem.cache.Exists(tx) {
		return false
	}
	mem.cache.Push(tx)

	return true
}

// txCache.Remove移除txCache中给定tx
func (cache *txCache) Remove(tx *types.Transaction) {
	// cache.mtx.Lock()
	delete(cache.map_, string(tx.Hash()))
	for e := cache.list.Front(); e != nil; e = e.Next() {
		if string(e.Value.(*types.Transaction).Hash()) == string(tx.Hash()) {
			cache.list.Remove(e)
		}
	}
	// cache.mtx.Unlock()
}

//Mempool.RemoveTxsOfBlock移除Mempool中已被Blockchain打包的tx
func (mem *Mempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for tx := range block.Txs {
		exist := mem.cache.Exists(block.Txs[tx])
		if exist {
			mem.cache.Remove(block.Txs[tx])
		}
	}
	return true
}

// channelClient.SendTx向"p2p"发送消息
func (client *channelClient) SendTx(tx *types.Transaction) queue.Message {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}

	msg := client.qclient.NewMessage("p2p", types.EventTxBroadcast, tx)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)

	if err != nil {
		resp.Data = err
	}

	return resp
}

func (mem *Mempool) SetQueue(q *queue.Queue) {
	var chanClient *channelClient = new(channelClient)
	client := q.GetClient()
	client.Sub("mempool")
	go func() {
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				if mem.CheckTx(msg.GetData().(*types.Transaction)) {
					chanClient.SendTx(msg.GetData().(*types.Transaction))
					msg.Reply(client.NewMessage("rpc", types.EventReply,
						types.Reply{true, nil}))
				} else {
					msg.Reply(client.NewMessage("rpc", types.EventReply,
						types.Reply{false, []byte("transaction exists")}))
				}
			} else if msg.Ty == types.EventTxAddMempool {
				if mem.CheckTx(msg.GetData().(*types.Transaction)) {
					msg.Reply(client.NewMessage("rpc", types.EventReply,
						types.Reply{true, nil}))
				} else {
					msg.Reply(client.NewMessage("rpc", types.EventReply,
						types.Reply{false, []byte("transaction exists")}))
				}
			} else if msg.Ty == types.EventTxList {
				msg.Reply(client.NewMessage("consense", types.EventTxListReply,
					types.ReplyTxList{mem.GetTxList(10000)}))
			} else if msg.Ty == types.EventAddBlock {
				mem.RemoveTxsOfBlock(msg.GetData().(*types.Block))
				msg.Reply(client.NewMessage("blockchain", types.EventReply,
					types.Reply{true, nil}))

			}
		}
	}()
}
