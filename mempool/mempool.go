package mempool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru"
	"gitlab.33.cn/chain33/chain33/common"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	log "gitlab.33.cn/chain33/chain33/common/log/log15"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	mlog.SetHandler(log.DiscardHandler())
}

//--------------------------------------------------------------------------------
// Module Mempool

type Mempool struct {
	proxyMtx          sync.Mutex
	cache             *txCache
	in                chan queue.Message
	out               <-chan queue.Message
	client            queue.Client
	header            *types.Header
	minFee            int64
	addedTxs          *lru.Cache
	sync              bool
	cfg               *types.MemPool
	poolHeader        chan struct{}
	isclose           int32
	wg                sync.WaitGroup
	done              chan struct{}
	removeBlockTicket *time.Ticker
}

func New(cfg *types.MemPool) *Mempool {
	pool := &Mempool{}
	initConfig(cfg)
	pool.cache = newTxCache(poolCacheSize)
	pool.in = make(chan queue.Message)
	pool.out = make(<-chan queue.Message)
	pool.done = make(chan struct{})
	pool.minFee = cfg.MinTxFee
	pool.addedTxs, _ = lru.New(mempoolAddedTxSize)
	pool.cfg = cfg
	pool.poolHeader = make(chan struct{}, 2)
	pool.removeBlockTicket = time.NewTicker(time.Minute)
	return pool
}

func initConfig(cfg *types.MemPool) {
	if cfg.PoolCacheSize > 0 {
		poolCacheSize = cfg.PoolCacheSize
	}
	if cfg.MaxTxNumPerAccount > 0 {
		maxTxNumPerAccount = cfg.MaxTxNumPerAccount
	}
}

// Mempool.Resize设置Mempool容量
func (mem *Mempool) Resize(size int) {
	mem.proxyMtx.Lock()
	mem.cache.SetSize(size)
	mem.proxyMtx.Unlock()
}

func (mem *Mempool) isClose() bool {
	return atomic.LoadInt32(&mem.isclose) == 1
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

// Mempool.GetTxList从txCache中返回给定数目的tx
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
		item := v.Value.(*Item)
		hash := item.value.Hash()
		if types.Now().Unix()-item.enterTime >= mempoolExpiredInterval {
			// 清理滞留Mempool中超过10分钟的交易
			mem.cache.Remove(hash)
		} else if item.value.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
			// 清理过期的交易
			mem.cache.Remove(hash)
		} else {
			result = append(result, item.value)
		}
	}
	return result
}

// Mempool.RemoveTxsOfBlock移除Mempool中已被Blockchain打包的tx
func (mem *Mempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, tx := range block.Txs {
		hash := tx.Hash()
		mem.addedTxs.Add(string(hash), nil)
		exist := mem.cache.Exists(hash)
		if exist {
			mem.cache.Remove(hash)
		}
	}
	return true
}

// Mempool.RemoveTxs从Mempool中删除给定Hash的txs
func (mem *Mempool) RemoveTxs(hashList *types.TxHashList) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, hash := range hashList.Hashes {
		exist := mem.cache.Exists(hash)
		if exist {
			mem.cache.Remove(hash)
		}
	}
	return nil
}

// Mempool.DelBlock将回退的区块内的交易重新加入mempool中
func (mem *Mempool) DelBlock(block *types.Block) {
	if len(block.Txs) <= 0 {
		return
	}
	blkTxs := block.Txs
	header := mem.GetHeader()
	for i := 0; i < len(blkTxs); i++ {
		tx := blkTxs[i]
		//当前包括ticket和平行链的第一笔挖矿交易，统一actionName为miner
		if i == 0 && tx.ActionName() == types.MinerAction {
			continue
		}
		groupCount := int(tx.GetGroupCount())
		if groupCount > 1 && i+groupCount <= len(blkTxs) {
			group := types.Transactions{Txs: blkTxs[i : i+groupCount]}
			tx = group.Tx()
			i = i + groupCount - 1
		}
		err := tx.Check(header.GetHeight(), mem.minFee)
		if err != nil {
			continue
		}
		if !mem.checkExpireValid(tx) {
			continue
		}

		mem.addedTxs.Remove(string(tx.Hash()))
		mem.PushTx(tx)
	}
}

// Mempool.PushTx将交易推入Mempool，并返回结果（error）
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

// Mempool.ReTrySend每隔两分钟运行一次ReTry
func (mem *Mempool) ReTrySend() {
	for {
		time.Sleep(time.Minute * 2)
		mem.ReTry()
	}
}

// Mempool.ReTry检查Mempool，将未过期的交易重发送给P2P
func (mem *Mempool) ReTry() {
	var result []*types.Transaction
	mem.proxyMtx.Lock()
	for _, v := range mem.cache.txMap {
		if types.Now().Unix()-v.Value.(*Item).enterTime >= mempoolReSendInterval {
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
	defer mem.wg.Done()
	defer mlog.Info("RemoveBlockedTxs quit")
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	for {
		select {
		case <-mem.removeBlockTicket.C:
			if mem.isClose() {
				return
			}
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
			dupTxList, err := mem.client.Wait(hashList)
			if err != nil {
				mlog.Error("blockchain get txhashlist err", "err", err)
				continue
			}

			// 取出blockchain返回的重复交易列表
			dupTxs := dupTxList.GetData().(*types.TxHashList).Hashes

			if len(dupTxs) == 0 {
				continue
			}

			mem.proxyMtx.Lock()
			for _, t := range dupTxs {
				txValue, exists := mem.cache.txMap[string(t)]
				if exists {
					mem.addedTxs.Add(string(t), nil)
					mem.cache.Remove(txValue.Value.(*Item).value.Hash())
				}
			}
			mem.proxyMtx.Unlock()
		case <-mem.done:
			return
		}
	}
}

// Mempool.GetHeader获取Mempool.header
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

// Mempool.GetAccTxs用来获取对应账户地址（列表）中的全部交易详细信息
func (mem *Mempool) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetAccTxs(addrs)
}

// Mempool.SendTxToP2P向"p2p"发送消息
func (mem *Mempool) SendTxToP2P(tx *types.Transaction) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("p2p", types.EventTxBroadcast, tx)
	mem.client.Send(msg, false)
	mlog.Debug("tx sent to p2p", "tx.Hash", common.ToHex(tx.Hash()))
}

// Mempool.CheckExpireValid检查交易过期有效性，过期返回false，未过期返回true
func (mem *Mempool) CheckExpireValid(msg queue.Message) (bool, error) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return false, types.ErrHeaderNotSet
	}
	tx := msg.GetData().(types.TxGroup).Tx()
	ok := mem.checkExpireValid(tx)
	if !ok {
		return ok, types.ErrTxExpire
	}
	return ok, nil
}

func (mem *Mempool) checkExpireValid(tx *types.Transaction) bool {
	if tx.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
		return false
	}
	if tx.Expire > 1000000000 && tx.Expire < types.Now().Unix()+int64(time.Minute/time.Second) {
		return false
	}
	return true
}

// Mempool.Close关闭Mempool
func (mem *Mempool) Close() {
	atomic.StoreInt32(&mem.isclose, 1)
	close(mem.in)
	close(mem.done)
	mem.client.Close()
	mem.removeBlockTicket.Stop()
	mlog.Info("mempool module closing")
	mem.wg.Wait()
	mlog.Info("mempool module closed")
}

// Mempool.checkTxListRemote发送消息给执行模块检查交易
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

// Mempool.pollLastHeader在初始化后循环获取LastHeader，直到获取成功后，返回
func (mem *Mempool) pollLastHeader() {
	defer mem.wg.Done()
	defer func() {
		mlog.Info("pollLastHeader quit")
		mem.poolHeader <- struct{}{}
	}()
	for {
		if mem.isClose() {
			return
		}
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

// Mempool.setHeader设置Mempool.header
func (mem *Mempool) setHeader(h *types.Header) {
	mem.proxyMtx.Lock()
	mem.header = h
	mem.proxyMtx.Unlock()
}

func (mem *Mempool) WaitPollLastHeader() {
	<-mem.poolHeader
	//wait sync
	<-mem.poolHeader
}

// Mempool.setSync设置Mempool同步状态
func (mem *Mempool) setSync(status bool) {
	mem.proxyMtx.Lock()
	mem.sync = status
	mem.proxyMtx.Unlock()
}

func (mem *Mempool) SetSync(status bool) {
	mem.setSync(status)
}

// Mempool.isSync检查Mempool是否同步完成
func (mem *Mempool) isSync() bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.sync
}

// Mempool.getSync获取Mempool同步状态
func (mem *Mempool) getSync() {
	defer func() {
		mlog.Info("getsync quit")
		mem.poolHeader <- struct{}{}
	}()
	defer mem.wg.Done()
	if mem.isSync() {
		return
	}
	if mem.cfg.ForceAccept {
		mem.setSync(true)
	}
	for {
		if mem.isClose() {
			return
		}
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

func (mem *Mempool) checkSign(data queue.Message) queue.Message {
	tx, ok := data.GetData().(types.TxGroup)
	if ok && tx.CheckSign() {
		return data
	} else {
		mlog.Error("wrong tx", "err", types.ErrSign)
		data.Data = types.ErrSign
		return data
	}
}

func (mem *Mempool) pipeLine() <-chan queue.Message {
	//check sign
	step1 := func(data queue.Message) queue.Message {
		if data.Err() != nil {
			return data
		}
		return mem.checkSign(data)
	}
	chs := make([]<-chan queue.Message, processNum)
	for i := 0; i < processNum; i++ {
		chs[i] = step(mem.done, mem.in, step1)
	}
	out1 := merge(mem.done, chs)

	//checktx remote
	step2 := func(data queue.Message) queue.Message {
		if data.Err() != nil {
			return data
		}
		return mem.checkTxRemote(data)
	}
	chs2 := make([]<-chan queue.Message, processNum)
	for i := 0; i < processNum; i++ {
		chs2[i] = step(mem.done, out1, step2)
	}
	return merge(mem.done, chs2)
}

func (mem *Mempool) SetQueueClient(client queue.Client) {
	mem.client = client
	mem.client.Sub("mempool")
	mem.wg.Add(1)
	go mem.pollLastHeader()
	mem.wg.Add(1)
	go mem.getSync()
	//	go mem.ReTrySend()
	// 从badChan读取坏消息，并回复错误信息
	mem.out = mem.pipeLine()
	mlog.Info("mempool piple line start")
	mem.wg.Add(1)
	go func() {
		defer mlog.Info("piple line quit")
		defer mem.wg.Done()
		for m := range mem.out {
			if m.Err() != nil {
				m.Reply(mem.client.NewMessage("rpc", types.EventReply,
					&types.Reply{false, []byte(m.Err().Error())}))
			} else {
				mem.SendTxToP2P(m.GetData().(types.TxGroup).Tx())
				m.Reply(mem.client.NewMessage("rpc", types.EventReply, &types.Reply{true, nil}))
			}
		}
	}()
	mem.wg.Add(1)
	go mem.RemoveBlockedTxs()
	mem.wg.Add(1)
	go func() {
		defer mlog.Info("mempool message recv quit")
		defer mem.wg.Done()
		for msg := range mem.client.Recv() {
			mlog.Debug("mempool recv", "msgid", msg.Id, "msg", types.GetEventName(int(msg.Ty)))
			beg := types.Now()
			switch msg.Ty {
			case types.EventTx:
				if !mem.isSync() {
					msg.Reply(mem.client.NewMessage("", types.EventReply, &types.Reply{false, []byte(types.ErrNotSync.Error())}))
					mlog.Error("wrong tx", "err", types.ErrNotSync.Error())
				} else {
					checkedMsg := mem.CheckTxs(msg)
					mem.in <- checkedMsg
				}
			case types.EventGetMempool:
				// 消息类型EventGetMempool：获取Mempool内所有交易
				msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
					&types.ReplyTxList{mem.RemoveExpiredAndDuplicateMempoolTxs()}))
			case types.EventTxList:
				// 消息类型EventTxList：获取Mempool中一定数量交易
				hashList := msg.GetData().(*types.TxHashList)
				if hashList.Count <= 0 {
					msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, types.ErrSize))
					mlog.Error("not an valid size", "msg", msg)
				} else {
					txList := mem.GetTxList(hashList)
					msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, &types.ReplyTxList{Txs: txList}))
				}
			case types.EventDelTxList:
				// 消息类型EventDelTxList：获取Mempool中一定数量交易，并把这些交易从Mempool中删除
				hashList := msg.GetData().(*types.TxHashList)
				if len(hashList.GetHashes()) == 0 {
					msg.ReplyErr("EventDelTxList", types.ErrSize)
				} else {
					err := mem.RemoveTxs(hashList)
					msg.ReplyErr("EventDelTxList", err)
				}
			case types.EventAddBlock:
				// 消息类型EventAddBlock：将添加到区块内的交易从Mempool中删除
				block := msg.GetData().(*types.BlockDetail).Block
				if block.Height > mem.Height() || (block.Height == 0 && mem.Height() == 0) {
					header := &types.Header{}
					header.BlockTime = block.BlockTime
					header.Height = block.Height
					header.StateHash = block.StateHash
					mem.setHeader(header)
				}
				mem.RemoveTxsOfBlock(block)
			case types.EventGetMempoolSize:
				// 消息类型EventGetMempoolSize：获取Mempool大小
				memSize := int64(mem.Size())
				msg.Reply(mem.client.NewMessage("rpc", types.EventMempoolSize,
					&types.MempoolSize{Size: memSize}))
			case types.EventGetLastMempool:
				// 消息类型EventGetLastMempool：获取最新十条加入到Mempool的交易
				txList := mem.GetLatestTx()
				msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
					&types.ReplyTxList{Txs: txList}))
			case types.EventDelBlock:
				// 回滚区块，把该区块内交易重新加回Mempool
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
				// 获取Mempool中对应账户（组）所有交易
				addrs := msg.GetData().(*types.ReqAddrs)
				txlist := mem.GetAccTxs(addrs)
				msg.Reply(mem.client.NewMessage("", types.EventReplyAddrTxs, txlist))
			default:
			}
			mlog.Debug("mempool", "cost", types.Since(beg), "msg", types.GetEventName(int(msg.Ty)))
		}
	}()
}
