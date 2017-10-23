package mempool

import (
	"container/list"
	"sync"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/queue"
)

const cacheSize = 300000

type MClient interface {
	SetQueue(q *queue.Queue)
	SendTx(tx *types.Transaction) queue.Message
}

type ChannelClient struct {
	qclient queue.IClient
}

type Mempool struct {
	proxyMtx sync.Mutex
	cache    *TxCache
}

type TxCache struct {
	mtx  sync.Mutex
	size int
	map_ map[*types.Transaction]struct{}
	list *list.List // to remove oldest tx when cache gets too big
}

// NewTxCache returns a new TxCache.
func NewTxCache(cacheSize int) *TxCache {
	return &TxCache{
		size: cacheSize,
		map_: make(map[*types.Transaction]struct{}, cacheSize),
		list: list.New(),
	}
}

// Reset resets the TxCache to empty.
func (cache *TxCache) Reset() {
	cache.mtx.Lock()
	cache.map_ = make(map[*types.Transaction]struct{}, cacheSize)
	cache.list.Init()
	cache.mtx.Unlock()
}

// Exists returns true if the given tx is cached.
func (cache *TxCache) Exists(tx *types.Transaction) bool {
	cache.mtx.Lock()
	_, exists := cache.map_[tx]
	cache.mtx.Unlock()
	return exists
}

// Push adds the given tx to the TxCache. It returns false if tx is already in the cache.
func (cache *TxCache) Push(tx *types.Transaction) bool {
	cache.mtx.Lock()
	defer cache.mtx.Unlock()

	if _, exists := cache.map_[tx]; exists {
		return false
	}

	if cache.list.Len() >= cache.size {
		popped := cache.list.Front()
		poppedTx := popped.Value.(*types.Transaction)
		// NOTE: the tx may have already been removed from the map
		// but deleting a non-existent element is fine
		delete(cache.map_, poppedTx)
		cache.list.Remove(popped)
	}

	cache.map_[tx] = struct{}{}
	cache.list.PushBack(tx)
	return true
}

// Remove removes the given tx from the cache.
func (cache *TxCache) Remove(tx *types.Transaction) {
	cache.mtx.Lock()
	delete(cache.map_, tx)
	cache.mtx.Unlock()
}

func (mem *Mempool) Lock() {
	mem.proxyMtx.Lock()
}

func (mem *Mempool) Unlock() {
	mem.proxyMtx.Unlock()
}

func (mem *Mempool) Size() int {
	return mem.cache.list.Len()
}

func (mem *Mempool) Flush() {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	mem.cache.Reset()

	var next *list.Element
    for e := mem.cache.list.Front(); e != nil; e = next {
        next = e.Next()
        mem.cache.list.Remove(e)
    }
}

func (mem *Mempool) CheckTx(tx *types.Transaction) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	// CACHE
	if mem.cache.Exists(tx) {
		return false // TODO: return an error (?)
	}
	mem.cache.Push(tx)
	// END CACHE

	return true
}

func (client *ChannelClient) SendTx(tx *types.Transaction) queue.Message {
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
	var chanClient *ChannelClient = new(ChannelClient)
	client := q.GetClient()
	client.Sub("mempool")
	go func () {
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				if mem.CheckTx(msg.GetData().(*types.Transaction)) {
					chanClient.SendTx(msg.GetData().(*types.Transaction))
					msg.Reply(client.NewMessage("rpc", types.EventReply, types.Reply{true, []byte("OK")}))
				} else {
					msg.Reply(client.NewMessage("rpc", types.EventReply, types.Reply{true, []byte("error")}))
				}
			} else if msg.Ty == types.EventTxAddMempool {
				if mem.CheckTx(msg.GetData().(*types.Transaction)) {
					msg.Reply(client.NewMessage("rpc", types.EventReply, types.Reply{true, []byte("OK")}))
				} else {
					msg.Reply(client.NewMessage("rpc", types.EventReply, types.Reply{true, []byte("error")}))
				}
			} else if msg.Ty == types.EventTxList {
				msg.Reply(client.NewMessage("consense", types.EventTxListReply, types.ReplyTxList, mem.cache.list))
				mem.Flush()
			}
		}
	}()
}