// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/hashicorp/golang-lru"
)

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	mlog.SetHandler(log.DiscardHandler())
}

var mlog = log.New("module", "BaseMempool")

type BaseMempool struct {
	ProxyMtx          sync.Mutex
	in                chan queue.Message
	out               <-chan queue.Message
	client            queue.Client
	header            *types.Header
	sync              bool
	cfg               *types.MemPool
	poolHeader        chan struct{}
	isclose           int32
	wg                sync.WaitGroup
	done              chan struct{}
	removeBlockTicket *time.Ticker
	addedTxs          *lru.Cache
	baseCache         *BaseCache
}

func (mem *BaseMempool) GetSync() bool {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.sync
}

func (mem *BaseMempool) GetBaseCache() *BaseCache {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.baseCache
}

func (mem *BaseMempool) SetBaseCache(b *BaseCache) {
	mem.ProxyMtx.Lock()
	mem.baseCache=b
	mem.ProxyMtx.Unlock()
}

func NewMempool(cfg *types.MemPool) *BaseMempool {
	pool := &BaseMempool{}
	initConfig(cfg)
	pool.in = make(chan queue.Message)
	pool.out = make(<-chan queue.Message)
	pool.done = make(chan struct{})
	pool.addedTxs, _ = lru.New(mempoolAddedTxSize)
	pool.cfg = cfg
	pool.poolHeader = make(chan struct{}, 2)
	pool.removeBlockTicket = time.NewTicker(time.Minute)
	pool.baseCache = NewBaseCache(poolCacheSize)
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

func (mem *BaseMempool) Close() {
	if mem.IsClose() {
		return
	}
	atomic.StoreInt32(&mem.isclose, 1)
	close(mem.in)
	close(mem.done)
	mem.client.Close()
	mem.removeBlockTicket.Stop()
	mlog.Info("mempool module closing")
	mem.wg.Wait()
	mlog.Info("mempool module closed")
}

func (mem *BaseMempool) SetQueueClient(client queue.Client) {
	mem.client = client
	mem.client.Sub("mempool")
	mem.wg.Add(1)
	go mem.pollLastHeader()
	mem.wg.Add(1)
	go mem.checkSync()
	//	go mem.ReTrySend()
	// 从badChan读取坏消息，并回复错误信息
	mem.out = mem.pipeLine()
	mlog.Info("mempool piple line start")
	mem.wg.Add(1)
	go mem.p2pServer()
	mem.wg.Add(1)
	go mem.RemoveBlockedTxs()
	mem.wg.Add(1)
	go mem.EventProcess()
}

func (mem *BaseMempool) p2pServer() {
	defer mlog.Info("piple line quit")
	defer mem.wg.Done()
	for m := range mem.out {
		if m.Err() != nil {
			m.Reply(mem.client.NewMessage("rpc", types.EventReply,
				&types.Reply{IsOk: false, Msg: []byte(m.Err().Error())}))
		} else {
			mem.sendTxToP2P(m.GetData().(types.TxGroup).Tx())
			m.Reply(mem.client.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: true, Msg: nil}))
		}
	}
}

// Size 返回mempool中txCache大小
func (mem *BaseMempool) Size() int {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.baseCache.child.GetSize()
}

//// Mempool.SetMinFee设置最小交易费用
//func (mem *BaseMempool) SetMinFee(fee int64) {
//	mem.ProxyMtx.Lock()
//	mem.cfg.MinTxFee = fee
//	mem.ProxyMtx.Unlock()
//}
//
//// Mempool.GetMinFee获取最小交易费用
//func (mem *BaseMempool) GetMinFee() int64 {
//	mem.ProxyMtx.Lock()
//	defer mem.ProxyMtx.Unlock()
//	return mem.cfg.MinTxFee
//}

// GetTxList 从txCache中返回给定数目的tx
func (mem *BaseMempool) GetTxList(filterList *types.TxHashList) []*types.Transaction {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	minSize := filterList.GetCount()
	dupMap := make(map[string]bool)
	for i := 0; i < len(filterList.GetHashes()); i++ {
		dupMap[string(filterList.GetHashes()[i])] = true
	}
	return mem.baseCache.child.GetTxList(minSize, mem.header.GetHeight(), mem.header.GetBlockTime(), dupMap)
}

// RemoveTxs 从mempool中删除给定Hash的txs
func (mem *BaseMempool) RemoveTxs(hashList *types.TxHashList) error {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	for _, hash := range hashList.Hashes {
		exist := mem.baseCache.Exists(hash)
		if exist {
			mem.baseCache.Remove(hash)
		}
	}
	return nil
}

// PushTx 将交易推入mempool，并返回结果（error）
func (mem *BaseMempool) PushTx(tx *types.Transaction) error {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	err := mem.baseCache.child.Push(tx)
	return err
}

//  setHeader设置mempool.header
func (mem *BaseMempool) setHeader(h *types.Header) {
	mem.ProxyMtx.Lock()
	mem.header = h
	mem.ProxyMtx.Unlock()
}

// Mempool.GetHeader获取Mempool.header
func (mem *BaseMempool) GetHeader() *types.Header {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.header
}

func (mem *BaseMempool) IsClose() bool {
	return atomic.LoadInt32(&mem.isclose) == 1
}

func (mem *BaseMempool) pipeLine() <-chan queue.Message {
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
		return mem.CheckTxRemote(data)
	}
	chs2 := make([]<-chan queue.Message, processNum)
	for i := 0; i < processNum; i++ {
		chs2[i] = step(mem.done, out1, step2)
	}
	return merge(mem.done, chs2)
}

func (mem *BaseMempool) checkSign(data queue.Message) queue.Message {
	tx, ok := data.GetData().(types.TxGroup)
	if ok && tx.CheckSign() {
		return data
	}
	mlog.Error("wrong tx", "err", types.ErrSign)
	data.Data = types.ErrSign
	return data
}

// Mempool.GetLastHeader获取LastHeader的height和blockTime
func (mem *BaseMempool) GetLastHeader() (interface{}, error) {
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

// GetAccTxs 用来获取对应账户地址（列表）中的全部交易详细信息
func (mem *BaseMempool) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.baseCache.GetAccTxs(addrs)
}

// GetLatestTx 返回最新十条加入到mempool的交易
func (mem *BaseMempool) GetLatestTx() []*types.Transaction {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.baseCache.GetLatestTx()
}

// TxNumOfAccount 返回账户在mempool中交易数量
func (mem *BaseMempool) TxNumOfAccount(addr string) int64 {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.baseCache.TxNumOfAccount(addr)
}

// pollLastHeader在初始化后循环获取LastHeader，直到获取成功后，返回
func (mem *BaseMempool) pollLastHeader() {
	defer mem.wg.Done()
	defer func() {
		mlog.Info("pollLastHeader quit")
		mem.poolHeader <- struct{}{}
	}()
	for {
		if mem.IsClose() {
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

// checkTxListRemote 发送消息给执行模块检查交易
func (mem *BaseMempool) CheckTxListRemote(txlist *types.ExecTxList) (*types.ReceiptCheckTxList, error) {
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

func (mem *BaseMempool) RemoveExpiredAndDuplicateMempoolTxs() []*types.Transaction {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	return mem.baseCache.RemoveExpiredTx(mem.header.GetHeight(), mem.header.GetBlockTime())
}

// RemoveBlockedTxs 每隔1分钟清理一次已打包的交易
func (mem *BaseMempool) RemoveBlockedTxs() {
	defer mem.wg.Done()
	defer mlog.Info("RemoveBlockedTxs quit")
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	for {
		select {
		case <-mem.removeBlockTicket.C:
			if mem.IsClose() {
				return
			}
			txs := mem.RemoveExpiredAndDuplicateMempoolTxs()
			if len(txs) == 0 {
				continue
			}
			var checkHashList types.TxHashList
			for _, tx := range txs {
				hash := tx.Hash()
				checkHashList.Hashes = append(checkHashList.Hashes, hash)
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

			mem.ProxyMtx.Lock()
			mem.baseCache.RemoveBlockedTxs(dupTxs, mem.addedTxs)
			mem.ProxyMtx.Unlock()
		case <-mem.done:
			return
		}
	}
}

// RemoveTxsOfBlock 移除mempool中已被Blockchain打包的tx
func (mem *BaseMempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	for _, tx := range block.Txs {
		hash := tx.Hash()
		mem.addedTxs.Add(string(hash), nil)
		exist := mem.baseCache.Exists(hash)
		if exist {
			mem.baseCache.Remove(hash)
		}
	}
	return true
}

// Mempool.DelBlock将回退的区块内的交易重新加入mempool中
func (mem *BaseMempool) delBlock(block *types.Block) {
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
		err := tx.Check(header.GetHeight(), mem.cfg.MinTxFee)
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

// CheckExpireValid 检查交易过期有效性，过期返回false，未过期返回true
func (mem *BaseMempool) CheckExpireValid(msg queue.Message) (bool, error) {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
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

func (mem *BaseMempool) checkExpireValid(tx *types.Transaction) bool {
	if tx.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
		return false
	}
	if tx.Expire > 1000000000 && tx.Expire < types.Now().Unix()+int64(time.Minute/time.Second) {
		return false
	}
	return true
}

// Mempool.Height获取区块高度
func (mem *BaseMempool) Height() int64 {
	mem.ProxyMtx.Lock()
	defer mem.ProxyMtx.Unlock()
	if mem.header == nil {
		return -1
	}
	return mem.header.GetHeight()
}

func (mem *BaseMempool) WaitPollLastHeader() {
	<-mem.poolHeader
	//wait sync
	<-mem.poolHeader
}

// SendTxToP2P 向"p2p"发送消息
func (mem *BaseMempool) sendTxToP2P(tx *types.Transaction) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("p2p", types.EventTxBroadcast, tx)
	mem.client.Send(msg, false)
	mlog.Debug("tx sent to p2p", "tx.Hash", common.ToHex(tx.Hash()))
}

// BaseMempool.checkSync检查并获取mempool同步状态
func (mem *BaseMempool) checkSync() {
	defer func() {
		mlog.Info("getsync quit")
		mem.poolHeader <- struct{}{}
	}()
	defer mem.wg.Done()
	if mem.GetSync() {
		return
	}
	if mem.cfg.ForceAccept {
		mem.setSync(true)
	}
	for {
		if mem.IsClose() {
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
		}
		time.Sleep(time.Second)
		continue
	}
}

func (mem *BaseMempool) setSync(status bool) {
	mem.ProxyMtx.Lock()
	mem.sync = status
	mem.ProxyMtx.Unlock()
}

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *BaseMempool) CheckTx(msg queue.Message) queue.Message {
	tx := msg.GetData().(types.TxGroup).Tx()
	// 检查接收地址是否合法
	if err := address.CheckAddress(tx.To); err != nil {
		msg.Data = types.ErrInvalidAddress
		return msg
	}
	// 检查交易是否为重复交易
	if mem.addedTxs.Contains(string(tx.Hash())) {
		msg.Data = types.ErrDupTx
		return msg
	}

	// 检查交易账户在mempool中是否存在过多交易
	from := tx.From()
	if mem.TxNumOfAccount(from) >= maxTxNumPerAccount {
		msg.Data = types.ErrManyTx
		return msg
	}
	// 检查交易是否过期
	valid, err := mem.CheckExpireValid(msg)
	if !valid {
		msg.Data = err
		return msg
	}
	return msg
}

// CheckTxs 初步检查并筛选交易消息
func (mem *BaseMempool) CheckTxs(msg queue.Message) queue.Message {
	// 判断消息是否含有nil交易
	if msg.GetData() == nil {
		msg.Data = types.ErrEmptyTx
		return msg
	}
	header := mem.GetHeader()
	txmsg := msg.GetData().(*types.Transaction)
	//普通的交易
	tx := types.NewTransactionCache(txmsg)
	err := tx.Check(header.GetHeight(), mem.cfg.MinTxFee)
	if err != nil {
		msg.Data = err
		return msg
	}
	//检查txgroup 中的每个交易
	txs, err := tx.GetTxGroup()
	if err != nil {
		msg.Data = err
		return msg
	}
	msg.Data = tx
	//普通交易
	if txs == nil {
		return mem.CheckTx(msg)
	}
	//txgroup 的交易
	for i := 0; i < len(txs.Txs); i++ {
		msgitem := mem.CheckTx(queue.Message{Data: txs.Txs[i]})
		if msgitem.Err() != nil {
			msg.Data = msgitem.Err()
			return msg
		}
	}
	return msg
}

// Mempool.checkTxList检查账户余额是否足够，并加入到Mempool，成功则传入goodChan，若加入Mempool失败则传入badChan
func (mem *BaseMempool) CheckTxRemote(msg queue.Message) queue.Message {
	tx := msg.GetData().(types.TxGroup)
	txlist := &types.ExecTxList{}
	txlist.Txs = append(txlist.Txs, tx.Tx())

	lastheader := mem.GetHeader()
	txlist.BlockTime = lastheader.BlockTime
	txlist.Height = lastheader.Height
	txlist.StateHash = lastheader.StateHash
	// 增加这个属性，在执行器中会使用到
	txlist.Difficulty = uint64(lastheader.Difficulty)
	txlist.IsMempool = true
	result, err := mem.CheckTxListRemote(txlist)
	if err != nil {
		msg.Data = err
		return msg
	}
	errstr := result.Errs[0]
	if errstr == "" {
		err1 := mem.PushTx(txlist.Txs[0])
		if err1 != nil {
			mlog.Error("wrong tx", "err", err1)
			msg.Data = err1
		}
		return msg
	}
	mlog.Error("wrong tx", "err", errstr)
	msg.Data = errors.New(errstr)
	return msg
}

// 处理其他模块的消息
func (mem *BaseMempool) EventProcess() {
	defer mem.wg.Done()
	for msg := range mem.client.Recv() {
		mlog.Debug("mempool recv", "msgid", msg.ID, "msg", types.GetEventName(int(msg.Ty)))
		beg := types.Now()
		switch msg.Ty {
		case types.EventTx:
			mem.EventTx(msg)
		case types.EventGetMempool:
			// 消息类型EventGetMempool：获取mempool内所有交易
			mem.EventGetMempool(msg)
		case types.EventTxList:
			// 消息类型EventTxList：获取mempool中一定数量交易
			mem.EventTxList(msg)
		case types.EventDelTxList:
			// 消息类型EventDelTxList：获取mempool中一定数量交易，并把这些交易从mempool中删除
			mem.EventDelTxList(msg)
		case types.EventAddBlock:
			// 消息类型EventAddBlock：将添加到区块内的交易从mempool中删除
			mem.EventAddBlock(msg)
		case types.EventGetMempoolSize:
			// 消息类型EventGetMempoolSize：获取mempool大小
			mem.EventGetMempoolSize(msg)
		case types.EventGetLastMempool:
			// 消息类型EventGetLastMempool：获取最新十条加入到mempool的交易
			mem.EventGetLastMempool(msg)
		case types.EventDelBlock:
			// 回滚区块，把该区块内交易重新加回mempool
			mem.EventDelBlock(msg)
		case types.EventGetAddrTxs:
			// 获取mempool中对应账户（组）所有交易
			mem.EventGetAddrTxs(msg)
		default:
		}
		mlog.Debug("mempool", "cost", types.Since(beg), "msg", types.GetEventName(int(msg.Ty)))
	}
}

// 消息类型 EventTx：初步筛选后存入mempool
func (mem *BaseMempool) EventTx(msg queue.Message) {
	if !mem.GetSync() {
		msg.Reply(mem.client.NewMessage("", types.EventReply, &types.Reply{Msg: []byte(types.ErrNotSync.Error())}))
		mlog.Error("wrong tx", "err", types.ErrNotSync.Error())
	} else {
		checkedMsg := mem.CheckTxs(msg)
		mem.in <- checkedMsg
	}
}

// 消息类型 EventGetMempool：获取Mempool内所有交易
func (mem *BaseMempool) EventGetMempool(msg queue.Message) {
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
		&types.ReplyTxList{Txs: mem.RemoveExpiredAndDuplicateMempoolTxs()}))
}

// 消息类型EventDelTxList：获取Mempool中一定数量交易，并把这些交易从Mempool中删除
func (mem *BaseMempool) EventDelTxList(msg queue.Message) {
	hashList := msg.GetData().(*types.TxHashList)
	if len(hashList.GetHashes()) == 0 {
		msg.ReplyErr("EventDelTxList", types.ErrSize)
	} else {
		err := mem.RemoveTxs(hashList)
		msg.ReplyErr("EventDelTxList", err)
	}
}

// 消息类型 EventTxList：获取mempool中一定数量交易
func (mem *BaseMempool) EventTxList(msg queue.Message) {
	hashList := msg.GetData().(*types.TxHashList)
	if hashList.Count <= 0 {
		msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, types.ErrSize))
		mlog.Error("not an valid size", "msg", msg)
	} else {
		txList := mem.GetTxList(hashList)
		msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, &types.ReplyTxList{Txs: txList}))
	}
}

// 消息类型 EventAddBlock：将添加到区块内的交易从mempool中删除
func (mem *BaseMempool) EventAddBlock(msg queue.Message) {
	block := msg.GetData().(*types.BlockDetail).Block
	if block.Height > mem.Height() || (block.Height == 0 && mem.Height() == 0) {
		header := &types.Header{}
		header.BlockTime = block.BlockTime
		header.Height = block.Height
		header.StateHash = block.StateHash
		mem.setHeader(header)
	}
	mem.RemoveTxsOfBlock(block)
}

// 消息类型 EventGetMempoolSize：获取mempool大小
func (mem *BaseMempool) EventGetMempoolSize(msg queue.Message) {
	memSize := int64(mem.Size())
	msg.Reply(mem.client.NewMessage("rpc", types.EventMempoolSize,
		&types.MempoolSize{Size: memSize}))
}

// 消息类型 EventGetLastMempool：获取最新十条加入到mempool的交易
func (mem *BaseMempool) EventGetLastMempool(msg queue.Message) {
	txList := mem.GetLatestTx()
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
		&types.ReplyTxList{Txs: txList}))
}

// 消息类型 EventDelBlock：回滚区块，把该区块内交易重新加回mempool
func (mem *BaseMempool) EventDelBlock(msg queue.Message) {
	block := msg.GetData().(*types.BlockDetail).Block
	if block.Height != mem.GetHeader().GetHeight() {
		return
	}
	lastHeader, err := mem.GetLastHeader()
	if err != nil {
		mlog.Error(err.Error())
		return
	}
	h := lastHeader.(queue.Message).Data.(*types.Header)
	mem.setHeader(h)
	mem.delBlock(block)
}

// 消息类型 EventGetAddrTxs：获取mempool中对应账户（组）所有交易
func (mem *BaseMempool) EventGetAddrTxs(msg queue.Message) {
	addrs := msg.GetData().(*types.ReqAddrs)
	txlist := mem.GetAccTxs(addrs)
	msg.Reply(mem.client.NewMessage("", types.EventReplyAddrTxs, txlist))
}
