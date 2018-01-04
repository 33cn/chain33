package mempool

import (
	//	"container/heap"
	"errors"
	"runtime"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/inconshreveable/log15"
	"github.com/petar/GoLLRB/llrb"
)

var poolCacheSize int64 = 10240           // mempool容量
var channelSize int64 = 1024              // channel缓存大小
var minTxFee int64 = 10000000             // 最低交易费
var maxMsgByte int64 = 100000             // 交易消息最大字节数
var mempoolExpiredInterval int64 = 600000 // mempool内交易过期时间，10分钟
var mempoolAddedTxSize int = 102400       // 已添加过的交易缓存大小
var maxTxNumPerAccount int64 = 10000

// error codes
var (
	//	e00 = errors.New("success")
	e01 = errors.New("transaction exists")
	e02 = errors.New("low transaction fee")
	e03 = errors.New("you have too many transactions")
	e04 = errors.New("wrong signature")
	e05 = errors.New("low balance")
	e06 = errors.New("message too big")
	e07 = errors.New("message expired")
	e08 = errors.New("loadacconts error")
	e09 = errors.New("empty transaction")
	e10 = errors.New("duplicated transaction")
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
	qclient   queue.IClient
	height    int64
	blockTime int64
	minFee    int64
	addedTxs  *lru.Cache
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
	pool.addedTxs, _ = lru.New(mempoolAddedTxSize)
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
		poppedTx := mem.cache.txLlrb.DeleteMax().(*Item).value
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
	for _, v := range mem.cache.txMap {
		if time.Now().UnixNano()/1000000-v.enterTime >= mempoolExpiredInterval {
			// 清理滞留Mempool中超过10分钟的交易
			mem.cache.Remove(v.value)
		} else {
			result = append(result, v.value)
		}
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
func (mem *Mempool) PushTx(tx *types.Transaction) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	if mem.addedTxs.Contains(string(tx.Hash())) {
		return e10
	}

	err := mem.cache.Push(tx)
	if err == nil {
		mem.addedTxs.Add(string(tx.Hash()), nil)
	}

	return err
}

// Mempool.GetLatestTx返回最新十条加入到Mempool的交易
func (mem *Mempool) GetLatestTx() []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetLatestTx()
}

// Mempool.RemoveLeftOverTxs每隔10分钟清理一次滞留时间过长的交易
func (mem *Mempool) RemoveLeftOverTxs() {
	for {
		time.Sleep(time.Minute * 10)
		mem.DuplicateMempoolTxs()
	}
}

// Mempool.RemoveBlockedTxs每隔1分钟清理一次已打包的交易
func (mem *Mempool) RemoveBlockedTxs() {
	if mem.qclient == nil {
		panic("client not bind message queue.")
	}
	for {
		time.Sleep(time.Minute * 1)
		txs := mem.DuplicateMempoolTxs()
		var checkHashList types.TxHashList

		for _, tx := range txs {
			hash := tx.Hash()
			checkHashList.Hashes = append(checkHashList.Hashes, hash)
		}

		if len(checkHashList.Hashes) == 0 {
			continue
		}

		// 发送Hash过后的交易列表给blockchain模块
		hashList := mem.qclient.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
		mem.qclient.Send(hashList, true)
		dupTxList, _ := mem.qclient.Wait(hashList)

		// 取出blockchain返回的重复交易列表
		dupTxs := dupTxList.GetData().(*types.TxHashList).Hashes

		if len(dupTxs) == 0 {
			continue
		}

		mem.proxyMtx.Lock()
		for _, t := range dupTxs {
			txValue, exists := mem.cache.txMap[string(t)]
			if exists {
				mem.cache.Remove(txValue.value)
			}
		}
		mem.proxyMtx.Unlock()
	}
}

// Mempool.SendTxToP2P向"p2p"发送消息
func (mem *Mempool) SendTxToP2P(tx *types.Transaction) {
	if mem.qclient == nil {
		panic("client not bind message queue.")
	}

	msg := mem.qclient.NewMessage("p2p", types.EventTxBroadcast, tx)
	mem.qclient.Send(msg, false)
}

// Mempool.GetLastHeader获取LastHeader的height和blockTime
func (mem *Mempool) GetLastHeader() (interface{}, error) {
	if mem.qclient == nil {
		panic("client not bind message queue.")
	}

	msg := mem.qclient.NewMessage("blockchain", types.EventGetLastHeader, nil)
	mem.qclient.Send(msg, true)
	return mem.qclient.Wait(msg)
}

// Mempool.Close关闭Mempool
func (mem *Mempool) Close() {
	mlog.Info("mempool module closed")
}

//--------------------------------------------------------------------------------
// Module txCache

type txCache struct {
	size       int
	txMap      map[string]*Item
	txLlrb     *llrb.LLRB
	txFrontTen []*types.Transaction
	accountMap map[string]int64
}

// NewTxCache初始化txCache
func newTxCache(cacheSize int64) *txCache {
	return &txCache{
		size:       int(cacheSize),
		txMap:      make(map[string]*Item, cacheSize),
		txLlrb:     llrb.New(),
		txFrontTen: make([]*types.Transaction, 0),
		accountMap: make(map[string]int64),
	}
}

// txCache.LowestFee返回txHeap第一个元素的交易Fee
func (cache *txCache) LowestFee() int64 {
	return cache.txLlrb.Min().(*Item).priority
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
func (cache *txCache) Push(tx *types.Transaction) error {
	if cache.Exists(tx) {
		return e01
	}

	if cache.txLlrb.Len() >= cache.size {
		expired := 0
		for _, v := range cache.txMap {
			if time.Now().UnixNano()/1000000-v.enterTime >= mempoolExpiredInterval {
				cache.txLlrb.Delete(v)
				delete(cache.txMap, string(v.value.Hash()))
				mlog.Warn("Delete expired unpacked tx", "tx", v.value)
				expired++
			}
		}
		if tx.Fee <= cache.LowestFee() {
			return e02
		}
		if expired == 0 {
			poppedTx := cache.txLlrb.DeleteMin().(*Item).value
			delete(cache.txMap, string(poppedTx.Hash()))
			mlog.Warn("Delete lowest-fee unpacked tx", "tx", poppedTx)
		}
	}

	it := &Item{value: tx, priority: tx.Fee, enterTime: time.Now().UnixNano() / 1000000}
	cache.txLlrb.InsertNoReplace(it)
	cache.txMap[string(tx.Hash())] = it

	// 账户交易数量
	accountAddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if _, ok := cache.accountMap[accountAddr]; ok {
		cache.accountMap[accountAddr]++
	} else {
		cache.accountMap[accountAddr] = 1
	}

	if len(cache.txFrontTen) >= 10 {
		cache.txFrontTen = cache.txFrontTen[len(cache.txFrontTen)-10:]
	}
	cache.txFrontTen = append(cache.txFrontTen, tx)

	return nil
}

// txCache.GetLatestTx返回最新十条加入到txCache的交易
func (cache *txCache) GetLatestTx() []*types.Transaction {
	return cache.txFrontTen
}

// txCache.Remove移除txCache中给定tx
func (cache *txCache) Remove(tx *types.Transaction) {
	removed := cache.txMap[string(tx.Hash())]
	cache.txLlrb.Delete(removed)
	delete(cache.txMap, string(tx.Hash()))
	// 账户交易数量减1
	cache.AccountTxNumDecrease(account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String())
}

// txCache.Size返回txCache中已存tx数目
func (cache *txCache) Size() int {
	return cache.txLlrb.Len()
}

// txCache.SetSize用来设置Mempool容量
func (cache *txCache) SetSize(newSize int) {
	if cache.txLlrb.Len() > 0 {
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
// Module LLRB

type Item struct {
	value     *types.Transaction
	priority  int64
	enterTime int64
}

func (i Item) Less(it llrb.Item) bool {
	return i.priority < it.(*Item).priority
}

//--------------------------------------------------------------------------------

func (mem *Mempool) SetQueue(q *queue.Queue) {
	mem.memQueue = q
	mem.qclient = q.GetClient()
	mem.qclient.Sub("mempool")

	go func() {
		for {
			lastHeader, err := mem.GetLastHeader()
			if err != nil {
				mlog.Error(err.Error())
				time.Sleep(time.Second)
				continue
			}

			mem.proxyMtx.Lock()
			mem.height = lastHeader.(queue.Message).Data.(*types.Header).GetHeight()
			mem.blockTime = lastHeader.(queue.Message).Data.(*types.Header).GetBlockTime()
			mem.proxyMtx.Unlock()
			return
		}
	}()

	go mem.CheckTxList()
	go mem.CheckSignList()
	go mem.CheckBalanList()

	// 从badChan读取坏消息，并回复错误信息
	go func() {
		for m := range mem.badChan {
			m.Reply(mem.qclient.NewMessage("rpc", types.EventReply,
				&types.Reply{false, []byte(m.Err().Error())}))
			mlog.Warn("reply EventTx ok", "msg", m)
		}
	}()

	// 从goodChan读取好消息，并回复正确信息
	go func() {
		for m := range mem.goodChan {
			mem.SendTxToP2P(m.GetData().(*types.Transaction))
			mlog.Warn("send to p2p ok", "msg", m)
			m.Reply(mem.qclient.NewMessage("rpc", types.EventReply, &types.Reply{true, nil}))
			mlog.Warn("reply EventTx ok", "msg", m)
		}
	}()

	go mem.RemoveLeftOverTxs()
	go mem.RemoveBlockedTxs()

	go func() {
		for msg := range mem.qclient.Recv() {
			mlog.Warn("mempool recv", "msg", msg)
			switch msg.Ty {
			case types.EventTx:
				// 消息类型EventTx：申请添加交易到Mempool
				if msg.GetData() == nil { // 判断消息是否含有nil交易
					mlog.Error("wrong tx", "err", e09)
					msg.Data = e09
					mem.badChan <- msg
					continue
				}
				valid := mem.CheckExpireValid(msg) // 检查交易是否过期
				if valid {
					// 未过期，交易消息传入txChan，待检查
					mlog.Warn("check tx", "txChan", msg)
					mem.txChan <- msg
				} else {
					mlog.Error("wrong tx", "err", e07)
					msg.Data = e07
					mem.badChan <- msg
					continue
				}
			case types.EventGetMempool:
				// 消息类型EventGetMempool：获取Mempool内所有交易
				msg.Reply(mem.qclient.NewMessage("rpc", types.EventReplyTxList,
					&types.ReplyTxList{mem.DuplicateMempoolTxs()}))
				mlog.Warn("reply EventGetMempool ok", "msg", msg)
			case types.EventTxList:
				// 消息类型EventTxList：获取Mempool中一定数量交易，并把这些交易从Mempool中删除
				txListSize := msg.GetData().(int)
				if txListSize <= 0 {
					msg.Reply(mem.qclient.NewMessage("consensus", types.EventReplyTxList,
						errors.New("not an valid size")))
					mlog.Warn("not an valid size", "msg", msg)
					continue
				}
				txList := mem.GetTxList(txListSize)
				msg.Reply(mem.qclient.NewMessage("consensus", types.EventReplyTxList,
					&types.ReplyTxList{Txs: txList}))
				mlog.Warn("reply EventTxList ok", "msg", msg)
			case types.EventAddBlock:
				// 消息类型EventAddBlock：将添加到区块内的交易从Mempool中删除
				mem.RemoveTxsOfBlock(msg.GetData().(*types.BlockDetail).Block)
			case types.EventGetMempoolSize:
				// 消息类型EventGetMempoolSize：获取Mempool大小
				memSize := int64(mem.Size())
				msg.Reply(mem.qclient.NewMessage("rpc", types.EventMempoolSize,
					&types.MempoolSize{Size: memSize}))
				mlog.Warn("reply EventGetMempoolSize ok", "msg", msg)
			case types.EventGetLastMempool:
				// 消息类型EventGetLastMempool：获取最新十条加入到Mempool的交易
				txList := mem.GetLatestTx()
				msg.Reply(mem.qclient.NewMessage("rpc", types.EventReplyTxList,
					&types.ReplyTxList{Txs: txList}))
				mlog.Warn("reply EventGetLastMempool ok", "msg", msg)
			default:
				continue
			}
		}
	}()
}
