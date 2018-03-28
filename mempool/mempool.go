package mempool

import (
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/inconshreveable/log15"
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
	client    queue.Client
	header    *types.Header
	minFee    int64
	addedTxs  *lru.Cache
	sync      bool
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
	pool.minFee = cfg.MinTxFee
	pool.addedTxs, _ = lru.New(mempoolAddedTxSize)
	return pool
}

func initConfig(cfg *types.MemPool) {
	if cfg.PoolCacheSize > 0 {
		poolCacheSize = cfg.PoolCacheSize
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
	if mem.header == nil {
		return -1
	}
	return mem.header.GetHeight()
}

// Mempool.BlockTime获取区块时间
func (mem *Mempool) BlockTime() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return 0
	}
	return mem.header.BlockTime
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
func (mem *Mempool) GetTxList(hashList *types.TxHashList) []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	minSize := hashList.GetCount()
	dupMap := make(map[string]bool)
	for i := 0; i < len(hashList.GetHashes()); i++ {
		dupMap[string(hashList.GetHashes()[i])] = true
	}
	var result []*types.Transaction
	i := 0
	txlist := mem.cache.txList
	for v := txlist.Front(); v != nil; v = v.Next() {
		if v.Value.(*Item).value.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
			continue
		} else {
			tx := v.Value.(*Item).value
			if _, ok := dupMap[string(tx.Hash())]; ok {
				continue
			}
			result = append(result, tx)
			i++
			if i == int(minSize) {
				break
			}
		}
	}
	return result
}

// Mempool.RemoveExpiredAndDuplicateMempoolTxs删除过期交易然后复制并返回Mempool内交易
func (mem *Mempool) RemoveExpiredAndDuplicateMempoolTxs() []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()

	var result []*types.Transaction
	for _, v := range mem.cache.txMap {
		if time.Now().UnixNano()/1000000-v.Value.(*Item).enterTime >= mempoolExpiredInterval {
			// 清理滞留Mempool中超过10分钟的交易
			mem.cache.Remove(v.Value.(*Item).value)
		} else if v.Value.(*Item).value.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
			// 清理过期的交易
			mem.cache.Remove(v.Value.(*Item).value)
		} else {
			result = append(result, v.Value.(*Item).value)
		}
	}

	return result
}

// Mempool.RemoveTxsOfBlock移除Mempool中已被Blockchain打包的tx
func (mem *Mempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, tx := range block.Txs {
		mem.addedTxs.Add(string(tx.Hash()), nil)
		exist := mem.cache.Exists(tx)
		if exist {
			mem.cache.Remove(tx)
		}
	}
	return true
}

// Mempool.DelBlock将回退的区块内的交易重新加入mempool中
func (mem *Mempool) DelBlock(block *types.Block) {
	if len(block.Txs) <= 0 {
		return
	}

	blkTxs := block.Txs
	tx0 := blkTxs[0]
	if string(tx0.Execer) == "ticket" {
		blkTxs = blkTxs[1:]
	}

	for _, tx := range blkTxs {
		err := tx.Check(mem.minFee)
		if err != nil {
			continue
		}
		if tx.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
			continue
		}
		mem.PushTx(tx)
	}
}

// Mempool.PushTx将交易推入Mempool，成功返回true，失败返回false和失败原因
func (mem *Mempool) PushTx(tx *types.Transaction) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	err := mem.cache.Push(tx)
	return err
}

// Mempool.GetLatestTx返回最新十条加入到Mempool的交易
func (mem *Mempool) GetLatestTx() []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetLatestTx()
}

func (mem *Mempool) ReTrySend() {
	for {
		time.Sleep(time.Minute * 2)
		mem.ReTry()
	}
}

func (mem *Mempool) ReTry() {
	var result []*types.Transaction
	mem.proxyMtx.Lock()
	for _, v := range mem.cache.txMap {
		if time.Now().UnixNano()/1000000-v.Value.(*Item).enterTime >= mempoolReSendInterval {
			result = append(result, v.Value.(*Item).value)
		}
	}
	mem.proxyMtx.Unlock()
	if len(result) > 0 {
		mlog.Debug("retry send tx...")
	}
	for _, tx := range result {
		mem.SendTxToP2P(tx)
	}
}

// Mempool.RemoveBlockedTxs每隔1分钟清理一次已打包的交易
func (mem *Mempool) RemoveBlockedTxs() {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	for {
		time.Sleep(time.Minute * 1)
		txs := mem.RemoveExpiredAndDuplicateMempoolTxs()
		var checkHashList types.TxHashList

		for _, tx := range txs {
			hash := tx.Hash()
			checkHashList.Hashes = append(checkHashList.Hashes, hash)
		}

		if len(checkHashList.Hashes) == 0 {
			continue
		}

		// 发送Hash过后的交易列表给blockchain模块
		hashList := mem.client.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
		err := mem.client.Send(hashList, true)
		if err != nil {
			mlog.Error("blockchain closed", "err", err.Error())
			return
		}
		dupTxList, _ := mem.client.Wait(hashList)

		// 取出blockchain返回的重复交易列表
		dupTxs := dupTxList.GetData().(*types.TxHashList).Hashes

		if len(dupTxs) == 0 {
			continue
		}

		mem.proxyMtx.Lock()
		for _, t := range dupTxs {
			txValue, exists := mem.cache.txMap[string(t)]
			if exists {
				mem.cache.Remove(txValue.Value.(*Item).value)
			}
		}
		mem.proxyMtx.Unlock()
	}
}

func (mem *Mempool) GetHeader() *types.Header {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.header
}

// Mempool.GetLastHeader获取LastHeader的height和blockTime
func (mem *Mempool) GetLastHeader() (interface{}, error) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}

	msg := mem.client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err := mem.client.Send(msg, true)
	if err != nil {
		mlog.Error("blockchain closed", "err", err.Error())
		return nil, err
	}
	return mem.client.Wait(msg)
}

func (mem *Mempool) checkTxListRemote(txlist *types.ExecTxList) (*types.ReceiptCheckTxList, error) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("execs", types.EventCheckTx, txlist)
	err := mem.client.Send(msg, true)
	if err != nil {
		mlog.Error("execs closed", "err", err.Error())
		return nil, err
	}
	msg, err = mem.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	return msg.GetData().(*types.ReceiptCheckTxList), nil
}

// Mempool.SendTxToP2P向"p2p"发送消息
func (mem *Mempool) SendTxToP2P(tx *types.Transaction) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}

	msg := mem.client.NewMessage("p2p", types.EventTxBroadcast, tx)
	mem.client.Send(msg, false)
	mlog.Debug("tx sent to p2p", "tx.Hash", tx.Hash())
}

// Mempool.pollLastHeader在初始化后循环获取LastHeader，直到获取成功后，返回
func (mem *Mempool) pollLastHeader() {
	for {
		lastHeader, err := mem.GetLastHeader()
		if err != nil {
			mlog.Error(err.Error())
			time.Sleep(time.Second)
			continue
		}
		h := lastHeader.(queue.Message).Data.(*types.Header)
		mem.setHeader(h)
		return
	}
}

func (mem *Mempool) setHeader(h *types.Header) {
	mem.proxyMtx.Lock()
	mem.header = h
	mem.proxyMtx.Unlock()
}

// Mempool.Close关闭Mempool
func (mem *Mempool) Close() {
	mlog.Debug("mempool module closed")
}

// Mempool.CheckExpireValid检查交易过期有效性，过期返回false，未过期返回true
func (mem *Mempool) CheckExpireValid(msg queue.Message) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return false
	}
	tx := msg.GetData().(*types.Transaction)
	if tx.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
		return false
	}
	return true
}

func (mem *Mempool) setSync(status bool) {
	mem.proxyMtx.Lock()
	mem.sync = status
	mem.proxyMtx.Unlock()
}

func (mem *Mempool) isSync() bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.sync
}

func (mem *Mempool) getSync() {
	for {
		if mem.client == nil {
			panic("client not bind message queue.")
		}
		msg := mem.client.NewMessage("blockchain", types.EventIsSync, nil)
		err := mem.client.Send(msg, true)
		resp, err := mem.client.Wait(msg)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if resp.GetData().(*types.IsCaughtUp).GetIscaughtup() {
			mem.setSync(true)
			return
		} else {
			time.Sleep(time.Second)
			continue
		}
	}
}

func (mem *Mempool) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetAccTxs(addrs)
}

func (mem *Mempool) SetQueueClient(client queue.Client) {
	mem.client = client
	mem.client.Sub("mempool")

	go mem.pollLastHeader()
	go mem.getSync()
	go mem.ReTrySend()
	// 从badChan读取坏消息，并回复错误信息
	go func() {
		for m := range mem.badChan {
			m.Reply(mem.client.NewMessage("rpc", types.EventReply,
				&types.Reply{false, []byte(m.Err().Error())}))
		}
	}()

	// 从goodChan读取好消息，并回复正确信息
	go func() {
		for m := range mem.goodChan {
			mem.SendTxToP2P(m.GetData().(*types.Transaction))
			m.Reply(mem.client.NewMessage("rpc", types.EventReply, &types.Reply{true, nil}))
		}
	}()

	go mem.CheckSignList()
	go mem.CheckTxList()
	go mem.RemoveBlockedTxs()

	go func() {
		for msg := range mem.client.Recv() {
			mlog.Debug("mempool recv", "msgid", msg.Id, "msg", types.GetEventName(int(msg.Ty)))
			beg := time.Now()
			switch msg.Ty {
			case types.EventTx:
				if !mem.isSync() {
					msg.Reply(mem.client.NewMessage("rpc", types.EventReply, &types.Reply{false, []byte(types.ErrNotSync.Error())}))
					mlog.Error("wrong tx", "err", types.ErrNotSync.Error())
					continue
				}
				msg := mem.CheckTx(msg)
				if msg.Err() != nil {
					mlog.Error("wrong tx", "err", msg.Err())
					mem.badChan <- msg
				} else {
					mem.signChan <- msg
				}
			case types.EventGetMempool:
				// 消息类型EventGetMempool：获取Mempool内所有交易
				msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
					&types.ReplyTxList{mem.RemoveExpiredAndDuplicateMempoolTxs()}))
				mlog.Debug("reply EventGetMempool ok", "msg", msg)
			case types.EventTxList:
				// 消息类型EventTxList：获取Mempool中一定数量交易，并把这些交易从Mempool中删除
				hashList := msg.GetData().(*types.TxHashList)
				if hashList.Count <= 0 {
					msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, types.ErrSize))
					mlog.Error("not an valid size", "msg", msg)
				} else {
					txList := mem.GetTxList(hashList)
					msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, &types.ReplyTxList{Txs: txList}))
					mlog.Debug("reply EventTxList ok", "msg", msg)
				}
			case types.EventAddBlock:
				// 消息类型EventAddBlock：将添加到区块内的交易从Mempool中删除
				block := msg.GetData().(*types.BlockDetail).Block
				if block.Height > mem.Height() {
					header := &types.Header{}
					header.BlockTime = block.BlockTime
					header.Height = block.Height
					header.StateHash = block.StateHash
					mem.setHeader(header)
				}
				mem.RemoveTxsOfBlock(block)
				mlog.Debug("handle EventAddBlock ok", "msg", msg, "msgid", msg.Id)
			case types.EventGetMempoolSize:
				// 消息类型EventGetMempoolSize：获取Mempool大小
				memSize := int64(mem.Size())
				msg.Reply(mem.client.NewMessage("rpc", types.EventMempoolSize,
					&types.MempoolSize{Size: memSize}))
				mlog.Debug("reply EventGetMempoolSize ok", "msg", msg)
			case types.EventGetLastMempool:
				// 消息类型EventGetLastMempool：获取最新十条加入到Mempool的交易
				txList := mem.GetLatestTx()
				msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
					&types.ReplyTxList{Txs: txList}))
				mlog.Debug("reply EventGetLastMempool ok", "msg", msg)
			case types.EventDelBlock:
				block := msg.GetData().(*types.BlockDetail).Block
				if block.Height != mem.GetHeader().GetHeight() {
					continue
				}
				lastHeader, err := mem.GetLastHeader()
				if err != nil {
					mlog.Error(err.Error())
					continue
				}
				h := lastHeader.(queue.Message).Data.(*types.Header)
				mem.setHeader(h)
				mem.DelBlock(block)
			case types.EventGetAddrTxs:
				addrs := msg.GetData().(*types.ReqAddrs)
				txlist := mem.GetAccTxs(addrs)
				msg.Reply(mem.client.NewMessage("", types.EventReplyAddrTxs, txlist))
				mlog.Debug("reply EventGetAddrTxs ok", "msg", msg)
			default:
			}
			mlog.Debug("mempool", "cost", time.Now().Sub(beg), "msg", types.GetEventName(int(msg.Ty)))
		}
	}()
}
