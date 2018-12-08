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
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
)

var mlog = log.New("module", "BaseMempool")

//BaseMempool mempool 基础类
type BaseMempool struct {
	proxyMtx          sync.Mutex
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
	cache             *TxCache
}

//GetSync 判断是否mempool 同步
func (mem *BaseMempool) GetSync() bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.sync
}

//NewMempool 新建mempool 实例
func NewMempool(cfg *types.MemPool) *BaseMempool {
	pool := &BaseMempool{}
	if cfg.MaxTxNumPerAccount == 0 {
		cfg.MaxTxNumPerAccount = maxTxNumPerAccount
	}
	if cfg.MaxTxLast == 0 {
		cfg.MaxTxLast = maxTxLast
	}
	pool.in = make(chan queue.Message)
	pool.out = make(<-chan queue.Message)
	pool.done = make(chan struct{})
	pool.cfg = cfg
	pool.poolHeader = make(chan struct{}, 2)
	pool.removeBlockTicket = time.NewTicker(time.Minute)
	return pool
}

//Close 关闭mempool
func (mem *BaseMempool) Close() {
	if mem.isClose() {
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

//SetQueueClient 初始化mempool模块
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
	go mem.removeBlockedTxs()
	mem.wg.Add(1)
	go mem.eventProcess()
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
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.qcache.Size()
}

// SetMinFee 设置最小交易费用
func (mem *BaseMempool) SetMinFee(fee int64) {
	mem.proxyMtx.Lock()
	mem.cfg.MinTxFee = fee
	mem.proxyMtx.Unlock()
}

// GetMinFee 获取最小交易费用
func (mem *BaseMempool) GetMinFee() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cfg.MinTxFee
}

//SetQueueCache 设置排队策略
func (mem *BaseMempool) SetQueueCache(qcache QueueCache) {
	mem.cache.qcache = qcache
}

// GetTxList 从txCache中返回给定数目的tx
func (mem *BaseMempool) getTxList(filterList *types.TxHashList) (txs []*types.Transaction) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	count := filterList.GetCount()
	dupMap := make(map[string]bool)
	for i := 0; i < len(filterList.GetHashes()); i++ {
		dupMap[string(filterList.GetHashes()[i])] = true
	}
	return mem.filterTxList(count, dupMap)
}

func (mem *BaseMempool) filterTxList(count int64, dupMap map[string]bool) (txs []*types.Transaction) {
	height := mem.header.GetHeight()
	blocktime := mem.header.GetBlockTime()
	mem.cache.qcache.Walk(int(count), func(tx *Item) bool {
		if dupMap == nil {
			return true
		}
		if _, ok := dupMap[string(tx.Value.Hash())]; ok {
			return true
		}
		if isExpired(tx, height, blocktime) {
			return true
		}
		txs = append(txs, tx.Value)
		return true
	})
	return txs
}

// RemoveTxs 从mempool中删除给定Hash的txs
func (mem *BaseMempool) RemoveTxs(hashList *types.TxHashList) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, hash := range hashList.Hashes {
		exist := mem.cache.qcache.Exist(string(hash))
		if exist {
			mem.cache.Remove(string(hash))
		}
	}
	return nil
}

// PushTx 将交易推入mempool，并返回结果（error）
func (mem *BaseMempool) PushTx(tx *types.Transaction) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	err := mem.cache.Push(tx)
	return err
}

//  setHeader设置mempool.header
func (mem *BaseMempool) setHeader(h *types.Header) {
	mem.proxyMtx.Lock()
	mem.header = h
	mem.proxyMtx.Unlock()
}

// GetHeader 获取Mempool.header
func (mem *BaseMempool) GetHeader() *types.Header {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.header
}

//IsClose 判断是否mempool 关闭
func (mem *BaseMempool) isClose() bool {
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
		return mem.checkTxRemote(data)
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

// GetLastHeader 获取LastHeader的height和blockTime
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
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.accountIndex.GetAccTxs(addrs)
}

// TxNumOfAccount 返回账户在mempool中交易数量
func (mem *BaseMempool) TxNumOfAccount(addr string) int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.accountIndex.Size(addr)
}

// GetLatestTx 返回最新十条加入到mempool的交易
func (mem *BaseMempool) GetLatestTx() []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.last.GetLatestTx()
}

// pollLastHeader在初始化后循环获取LastHeader，直到获取成功后，返回
func (mem *BaseMempool) pollLastHeader() {
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

// checkTxListRemote 发送消息给执行模块检查交易
func (mem *BaseMempool) checkTxListRemote(txlist *types.ExecTxList) (*types.ReceiptCheckTxList, error) {
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

func (mem *BaseMempool) removeExpired() {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	mem.cache.removeExpiredTx(mem.header.GetHeight(), mem.header.GetBlockTime())
}

// removeBlockedTxs 每隔1分钟清理一次已打包的交易
func (mem *BaseMempool) removeBlockedTxs() {
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
			mem.removeExpired()
		case <-mem.done:
			return
		}
	}
}

// RemoveTxsOfBlock 移除mempool中已被Blockchain打包的tx
func (mem *BaseMempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, tx := range block.Txs {
		hash := tx.Hash()
		exist := mem.cache.qcache.Exist(string(hash))
		if exist {
			mem.cache.Remove(string(hash))
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
		mem.PushTx(tx)
	}
}

// CheckExpireValid 检查交易过期有效性，过期返回false，未过期返回true
func (mem *BaseMempool) CheckExpireValid(msg queue.Message) (bool, error) {
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

func (mem *BaseMempool) checkExpireValid(tx *types.Transaction) bool {
	if tx.IsExpire(mem.header.GetHeight(), mem.header.GetBlockTime()) {
		return false
	}
	if tx.Expire > 1000000000 && tx.Expire < types.Now().Unix()+int64(time.Minute/time.Second) {
		return false
	}
	return true
}

// Height 获取区块高度
func (mem *BaseMempool) Height() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return -1
	}
	return mem.header.GetHeight()
}

// WaitPollLastHeader wait mempool ready
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
		}
		time.Sleep(time.Second)
		continue
	}
}

func (mem *BaseMempool) setSync(status bool) {
	mem.proxyMtx.Lock()
	mem.sync = status
	mem.proxyMtx.Unlock()
}

// CheckTx 初步检查并筛选交易消息
func (mem *BaseMempool) CheckTx(msg queue.Message) queue.Message {
	tx := msg.GetData().(types.TxGroup).Tx()
	// 检查接收地址是否合法
	if err := address.CheckAddress(tx.To); err != nil {
		msg.Data = types.ErrInvalidAddress
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
func (mem *BaseMempool) checkTxs(msg queue.Message) queue.Message {
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

//checkTxList 检查账户余额是否足够，并加入到Mempool，成功则传入goodChan，若加入Mempool失败则传入badChan
func (mem *BaseMempool) checkTxRemote(msg queue.Message) queue.Message {
	tx := msg.GetData().(types.TxGroup)
	txlist := &types.ExecTxList{}
	txlist.Txs = append(txlist.Txs, tx.Tx())
	//检查是否重复
	lastheader := mem.GetHeader()
	txlist.BlockTime = lastheader.BlockTime
	txlist.Height = lastheader.Height
	txlist.StateHash = lastheader.StateHash
	// 增加这个属性，在执行器中会使用到
	txlist.Difficulty = uint64(lastheader.Difficulty)
	txlist.IsMempool = true
	//add check dup tx
	newtxs, err := util.CheckDupTx(mem.client, txlist.Txs, txlist.Height)
	if err != nil {
		msg.Data = err
		return msg
	}
	if len(newtxs) != len(txlist.Txs) {
		msg.Data = types.ErrDupTx
		return msg
	}
	result, err := mem.checkTxListRemote(txlist)
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
