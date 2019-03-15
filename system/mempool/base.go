// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var mlog = log.New("module", "mempool.base")

//Mempool mempool 基础类
type Mempool struct {
	proxyMtx          sync.Mutex
	in                chan *queue.Message
	out               <-chan *queue.Message
	client            queue.Client
	header            *types.Header
	sync              bool
	cfg               *types.Mempool
	poolHeader        chan struct{}
	isclose           int32
	wg                sync.WaitGroup
	done              chan struct{}
	removeBlockTicket *time.Ticker
	cache             *txCache
}

//GetSync 判断是否mempool 同步
func (mem *Mempool) getSync() bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.sync
}

//NewMempool 新建mempool 实例
func NewMempool(cfg *types.Mempool) *Mempool {
	pool := &Mempool{}
	if cfg.MaxTxNumPerAccount == 0 {
		cfg.MaxTxNumPerAccount = maxTxNumPerAccount
	}
	if cfg.MaxTxLast == 0 {
		cfg.MaxTxLast = maxTxLast
	}
	if cfg.PoolCacheSize == 0 {
		cfg.PoolCacheSize = poolCacheSize
	}
	pool.in = make(chan *queue.Message)
	pool.out = make(<-chan *queue.Message)
	pool.done = make(chan struct{})
	pool.cfg = cfg
	pool.poolHeader = make(chan struct{}, 2)
	pool.removeBlockTicket = time.NewTicker(time.Minute)
	pool.cache = newCache(cfg.MaxTxNumPerAccount, cfg.MaxTxLast)
	return pool
}

//Close 关闭mempool
func (mem *Mempool) Close() {
	if mem.isClose() {
		return
	}
	atomic.StoreInt32(&mem.isclose, 1)
	close(mem.done)
	if mem.client != nil {
		mem.client.Close()
	}
	mem.removeBlockTicket.Stop()
	mlog.Info("mempool module closing")
	mem.wg.Wait()
	mlog.Info("mempool module closed")
}

//SetQueueClient 初始化mempool模块
func (mem *Mempool) SetQueueClient(client queue.Client) {
	mem.client = client
	mem.client.Sub("mempool")
	mem.wg.Add(1)
	go mem.pollLastHeader()
	mem.wg.Add(1)
	go mem.checkSync()
	mem.wg.Add(1)
	go mem.removeBlockedTxs()

	mem.wg.Add(1)
	go mem.eventProcess()
}

// Size 返回mempool中txCache大小
func (mem *Mempool) Size() int {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.Size()
}

// SetMinFee 设置最小交易费用
func (mem *Mempool) SetMinFee(fee int64) {
	mem.proxyMtx.Lock()
	mem.cfg.MinTxFee = fee
	mem.proxyMtx.Unlock()
}

//SetQueueCache 设置排队策略
func (mem *Mempool) SetQueueCache(qcache QueueCache) {
	mem.cache.SetQueueCache(qcache)
}

// GetTxList 从txCache中返回给定数目的tx
func (mem *Mempool) getTxList(filterList *types.TxHashList) (txs []*types.Transaction) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	count := filterList.GetCount()
	dupMap := make(map[string]bool)
	for i := 0; i < len(filterList.GetHashes()); i++ {
		dupMap[string(filterList.GetHashes()[i])] = true
	}
	return mem.filterTxList(count, dupMap)
}

func (mem *Mempool) filterTxList(count int64, dupMap map[string]bool) (txs []*types.Transaction) {
	height := mem.header.GetHeight()
	blocktime := mem.header.GetBlockTime()
	mem.cache.Walk(int(count), func(tx *Item) bool {
		if dupMap != nil {
			if _, ok := dupMap[string(tx.Value.Hash())]; ok {
				return true
			}
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
func (mem *Mempool) RemoveTxs(hashList *types.TxHashList) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, hash := range hashList.Hashes {
		exist := mem.cache.Exist(string(hash))
		if exist {
			mem.cache.Remove(string(hash))
		}
	}
	return nil
}

// PushTx 将交易推入mempool，并返回结果（error）
func (mem *Mempool) PushTx(tx *types.Transaction) error {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	err := mem.cache.Push(tx)
	return err
}

//  setHeader设置mempool.header
func (mem *Mempool) setHeader(h *types.Header) {
	mem.proxyMtx.Lock()
	mem.header = h
	mem.proxyMtx.Unlock()
}

// GetHeader 获取Mempool.header
func (mem *Mempool) GetHeader() *types.Header {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.header
}

//IsClose 判断是否mempool 关闭
func (mem *Mempool) isClose() bool {
	return atomic.LoadInt32(&mem.isclose) == 1
}

// GetLastHeader 获取LastHeader的height和blockTime
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

// GetAccTxs 用来获取对应账户地址（列表）中的全部交易详细信息
func (mem *Mempool) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetAccTxs(addrs)
}

// TxNumOfAccount 返回账户在mempool中交易数量
func (mem *Mempool) TxNumOfAccount(addr string) int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return int64(mem.cache.TxNumOfAccount(addr))
}

// GetLatestTx 返回最新十条加入到mempool的交易
func (mem *Mempool) GetLatestTx() []*types.Transaction {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	return mem.cache.GetLatestTx()
}

// pollLastHeader在初始化后循环获取LastHeader，直到获取成功后，返回
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
		h := lastHeader.(*queue.Message).Data.(*types.Header)
		mem.setHeader(h)
		return
	}
}

func (mem *Mempool) removeExpired() {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	mem.cache.removeExpiredTx(mem.header.GetHeight(), mem.header.GetBlockTime())
}

// removeBlockedTxs 每隔1分钟清理一次已打包的交易
func (mem *Mempool) removeBlockedTxs() {
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
func (mem *Mempool) RemoveTxsOfBlock(block *types.Block) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	for _, tx := range block.Txs {
		hash := tx.Hash()
		exist := mem.cache.Exist(string(hash))
		if exist {
			mem.cache.Remove(string(hash))
		}
	}
	return true
}

// Mempool.DelBlock将回退的区块内的交易重新加入mempool中
func (mem *Mempool) delBlock(block *types.Block) {
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
		err := tx.Check(header.GetHeight(), mem.cfg.MinTxFee, mem.cfg.MaxTxFee)
		if err != nil {
			continue
		}
		if !mem.checkExpireValid(tx) {
			continue
		}
		err = mem.PushTx(tx)
		if err != nil {
			mlog.Error("mem", "push tx err", err)
		}
	}
}

// Height 获取区块高度
func (mem *Mempool) Height() int64 {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return -1
	}
	return mem.header.GetHeight()
}

// Wait wait mempool ready
func (mem *Mempool) Wait() {
	<-mem.poolHeader
	//wait sync
	<-mem.poolHeader
}

// SendTxToP2P 向"p2p"发送消息
func (mem *Mempool) sendTxToP2P(tx *types.Transaction) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("p2p", types.EventTxBroadcast, tx)
	err := mem.client.Send(msg, false)
	if err != nil {
		mlog.Error("tx sent to p2p", "tx.Hash", common.ToHex(tx.Hash()))
		return
	}
	mlog.Debug("tx sent to p2p", "tx.Hash", common.ToHex(tx.Hash()))
}

// Mempool.checkSync检查并获取mempool同步状态
func (mem *Mempool) checkSync() {
	defer func() {
		mlog.Info("getsync quit")
		mem.poolHeader <- struct{}{}
	}()
	defer mem.wg.Done()
	if mem.getSync() {
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
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
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

func (mem *Mempool) setSync(status bool) {
	mem.proxyMtx.Lock()
	mem.sync = status
	mem.proxyMtx.Unlock()
}
