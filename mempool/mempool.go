package mempool

import (
	"container/heap"
	"errors"
	"runtime"
	"sync"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var poolCacheSize int64 = 10240    // mempool容量
var channelSize int64 = 1024       // channel缓存大小
var minTxFee int64 = 10000000      // 最低交易费
var maxMsgByte int64 = 100000      // 交易消息最大字节数
var expireBound int64 = 1000000000 // 交易过期分界线，小于expireBound比较height，大于expireBound比较blockTime

// error codes
const (
	e00 = "success"
	e01 = "transaction exists"
	e02 = "low transaction fee"
	e03 = "you have too many transactions"
	e04 = "wrong signature"
	e05 = "low balance"
	e06 = "message too big"
	e07 = "message expired"
)

var (
	mlog       = log.New("module", "mempool")
	processNum = runtime.NumCPU()
)

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	mlog.SetHandler(log.DiscardHandler())
}

type MClient interface {
	SetQueue(q *queue.Queue)
	SendTx(tx *types.Transaction) queue.Message
}

//--------------------------------------------------------------------------------
// Module channelClient

type channelClient struct {
	qclient queue.IClient
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

// channelClient.GetLastHeader获取LastHeader的height和blockTime
func (client *channelClient) GetLastHeader() (bool, interface{}) {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}

	msg := client.qclient.NewMessage("blockchain", types.EventGetLastHeader, nil)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)

	if err != nil {
		return false, nil
	}

	return true, resp
}

//--------------------------------------------------------------------------------
// Module Mempool

type Mempool struct {
	proxyMtx  sync.Mutex
	cache     *txCache
	txChan    chan queue.Message
	signChan  chan queue.Message
	badChan   chan queue.Message
	balanChan chan queue.Message
	goodChan  chan queue.Message
	memQueue  *queue.Queue
	height    int64
	blockTime int64
	minFee    int64
}

func New(cfg *types.MemPool) *Mempool {
	pool := &Mempool{}
	initConfig(cfg)
	pool.cache = newTxCache(poolCacheSize)
	pool.txChan = make(chan queue.Message, channelSize)
	pool.signChan = make(chan queue.Message, channelSize)
	pool.badChan = make(chan queue.Message, channelSize)
	pool.balanChan = make(chan queue.Message, channelSize)
	pool.goodChan = make(chan queue.Message, channelSize)
	pool.minFee = minTxFee
	return pool
}

func initConfig(cfg *types.MemPool) {
	if cfg.PoolCacheSize > 0 {
		poolCacheSize = cfg.PoolCacheSize
	}
	if cfg.MinTxFee > 0 {
		minTxFee = cfg.MinTxFee
	}
}

// Mempool.Resize设置Mempool容量
func (mem *Mempool) Resize(size int) {
	mem.proxyMtx.Lock()
	mem.cache.SetSize(size)
	mem.proxyMtx.Unlock()
}

// Mempool.SetMinFee设置最小交易费用
func (mem *Mempool) SetMinFee(fee int64) {
	mem.proxyMtx.Lock()
	mem.minFee = fee
	mem.proxyMtx.Unlock()
}

// Mempool.GetMinFee获取最小交易费用
func (mem *Mempool) GetMinFee() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.minFee
}

// Mempool.Height获取区块高度
func (mem *Mempool) Height() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.height
}

// Mempool.BlockTime获取区块时间
func (mem *Mempool) BlockTime() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.blockTime
}

// Mempool.LowestFee获取当前Mempool中最低的交易Fee
func (mem *Mempool) LowestFee() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.LowestFee()
}

// Mempool.Size返回Mempool中txCache大小
func (mem *Mempool) Size() int {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.Size()
}

// Mempool.TxNumOfAccount返回账户在Mempool中交易数量
func (mem *Mempool) TxNumOfAccount(addr string) int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.TxNumOfAccount(addr)
}

// Mempool.GetTxList从txCache中返回给定数目的tx并从txCache中删除返回的tx
func (mem *Mempool) GetTxList(txListSize int) []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	txsSize := mem.cache.Size()
	var result []*types.Transaction
	var i int
	var minSize int

	if txsSize <= txListSize {
		minSize = txsSize
	} else {
		minSize = txListSize
	}

	for i = 0; i < minSize; i++ {
		popped := heap.Pop(mem.cache.txHeap)
		poppedTx := popped.(*Item).value
		result = append(result, poppedTx)
		delete(mem.cache.txMap, string(poppedTx.Hash()))
		// 账户交易数量减1
		mem.cache.AccountTxNumDecrease(account.PubKeyToAddress(poppedTx.GetSignature().GetPubkey()).String())
	}

	return result
}

// Mempool.DuplicateMempoolTxs复制并返回Mempool内交易
func (mem *Mempool) DuplicateMempoolTxs() []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	var result []*types.Transaction
	for _, v := range *mem.cache.txHeap {
		result = append(result, v.value)
	}

	return result
}

// Mempool.RemoveTxsOfBlock移除Mempool中已被Blockchain打包的tx
func (mem *Mempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	mem.height = block.GetHeight()
	mem.blockTime = block.GetBlockTime()

	for tx := range block.Txs {
		exist := mem.cache.Exists(block.Txs[tx])
		if exist {
			mem.cache.Remove(block.Txs[tx])
		}
	}
	return true
}

// Mempool.PushTx将交易推入Mempool，成功返回true，失败返回false和失败原因
func (mem *Mempool) PushTx(tx *types.Transaction) (bool, string) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	ok, err := mem.cache.Push(tx)
	return ok, err
}

//--------------------------------------------------------------------------------
// Module txCache

type txCache struct {
	size       int
	txMap      map[string]*Item
	txHeap     *PriorityQueue
	accountMap map[string]int64
}

// NewTxCache初始化txCache
func newTxCache(cacheSize int64) *txCache {
	return &txCache{
		size:       int(cacheSize),
		txMap:      make(map[string]*Item, cacheSize),
		txHeap:     new(PriorityQueue),
		accountMap: make(map[string]int64),
	}
}

// txCache.LowestFee返回txHeap第一个元素的交易Fee
func (cache *txCache) LowestFee() int64 {
	return cache.txHeap.Top().(*Item).priority
}

// txCache.TxNumOfAccount返回账户在Mempool中交易数量
func (cache *txCache) TxNumOfAccount(addr string) int64 {
	return cache.accountMap[addr]
}

// txCache.Exists判断txCache中是否存在给定tx
func (cache *txCache) Exists(tx *types.Transaction) bool {
	_, exists := cache.txMap[string(tx.Hash())]
	return exists
}

// txCache.Push把给定tx添加到txCache并返回true；如果tx已经存在txCache中则返回false
func (cache *txCache) Push(tx *types.Transaction) (bool, string) {
	if _, exists := cache.txMap[string(tx.Hash())]; exists {
		return false, e01
	}

	if cache.txHeap.Len() >= cache.size {
		if tx.Fee <= cache.LowestFee() {
			return false, e02
		}
		popped := heap.Pop(cache.txHeap)
		poppedTx := popped.(*Item).value
		delete(cache.txMap, string(poppedTx.Hash()))
	}

	it := &Item{value: tx, priority: tx.Fee, index: cache.Size()}
	cache.txHeap.Push(it)
	cache.txMap[string(tx.Hash())] = it

	// 账户交易数量
	accountAddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if _, ok := cache.accountMap[accountAddr]; ok {
		cache.accountMap[accountAddr]++
	} else {
		cache.accountMap[accountAddr] = 1
	}

	return true, e00
}

// txCache.Remove移除txCache中给定tx
func (cache *txCache) Remove(tx *types.Transaction) {
	removed := cache.txMap[string(tx.Hash())].index
	cache.txHeap.Remove(removed)
	delete(cache.txMap, string(tx.Hash()))
	// 账户交易数量减1
	cache.AccountTxNumDecrease(account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String())
}

// txCache.Size返回txCache中已存tx数目
func (cache *txCache) Size() int {
	return cache.txHeap.Len()
}

// txCache.SetSize用来设置Mempool容量
func (cache *txCache) SetSize(newSize int) {
	if cache.txHeap.Len() > 0 {
		panic("only can set a empty size")
	}
	cache.size = newSize
}

// txCache.AccountTxNumDecrease使得账户的交易计数减一
func (cache *txCache) AccountTxNumDecrease(addr string) {
	if value, ok := cache.accountMap[addr]; ok {
		if value > 1 {
			cache.accountMap[addr]--
		} else {
			delete(cache.accountMap, addr)
		}
	}
}

//--------------------------------------------------------------------------------

func (mem *Mempool) SetQueue(q *queue.Queue) {
	mem.memQueue = q
	var chanClient *channelClient = new(channelClient)
	chanClient.qclient = q.GetClient()
	client := q.GetClient()
	client.Sub("mempool")

	go func() {
		for {
			flag, lastHeader := chanClient.GetLastHeader()
			if flag {
				mem.proxyMtx.Lock()
				mem.height = lastHeader.(queue.Message).Data.(*types.Header).GetHeight()
				mem.blockTime = lastHeader.(queue.Message).Data.(*types.Header).GetBlockTime()
				mem.proxyMtx.Unlock()
				return
			}
		}
	}()

	go mem.CheckTxList()
	go mem.CheckSignList()
	go mem.CheckBalanList()

	// 从badChan读取坏消息，并回复错误信息
	go func() {
		for m := range mem.badChan {
			m.Reply(client.NewMessage("rpc", types.EventReply,
				&types.Reply{false, []byte(m.Err().Error())}))
		}
	}()

	// 从goodChan读取好消息，并回复正确信息
	go func() {
		for m := range mem.goodChan {
			// chanClient.SendTx(m.GetData().(*types.Transaction))
			m.Reply(client.NewMessage("rpc", types.EventReply, &types.Reply{true, nil}))
		}
	}()

	go func() {
		for msg := range client.Recv() {
			mlog.Info("mempool recv", "msg", msg)
			if msg.Ty == types.EventTx {
				// 消息类型EventTx：申请添加交易到Mempool
				valid := mem.CheckExpire(msg) // 检查交易是否过期
				if valid {
					// 未过期，交易消息传入txChan，待检查
					mem.txChan <- msg
				} else {
					msg.Data = errors.New(e07)
					mem.badChan <- msg
				}
			} else if msg.Ty == types.EventGetMempool {
				// 消息类型EventGetMempool：获取Mempool内所有交易
				msg.Reply(client.NewMessage("rpc", types.EventReplyTxList,
					&types.ReplyTxList{mem.DuplicateMempoolTxs()}))
			} else if msg.Ty == types.EventTxList {
				// 消息类型EventTxList：获取Mempool中一定数量交易，并把这些交易从Mempool中删除
				msg.Reply(client.NewMessage("consensus", types.EventReplyTxList,
					&types.ReplyTxList{mem.GetTxList(10000)}))
			} else if msg.Ty == types.EventAddBlock {
				// 消息类型EventAddBlock：将添加到区块内的交易从Mempool中删除
				mem.RemoveTxsOfBlock(msg.GetData().(*types.BlockDetail).Block)
			} else if msg.Ty == types.EventGetMempoolSize {
				// 消息类型EventGetMempoolSize：获取Mempool大小
				msg.Reply(client.NewMessage("rpc", types.EventMempoolSize,
					&types.MempoolSize{int64(mem.Size())}))
			} else {
				continue
			}
		}
	}()
}
