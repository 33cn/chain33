// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found In the LICENSE file.

package mempool

import (
	"sync"

	"github.com/33cn/chain33/types"
)

// QueueCache 排队交易处理
type QueueCache interface {
	Exist(hash string) bool
	GetItem(hash string) (*Item, error)
	Push(tx *Item) error
	Remove(hash string) error
	Size() int
	Walk(count int, cb func(tx *Item) bool)
	GetProperFee() int64
	GetCacheBytes() int64
}

// Item 为Mempool中包装交易的数据结构
type Item struct {
	Value     *types.Transaction
	Priority  int64
	EnterTime int64
}

// TxCache 管理交易cache 包括账户索引，最后的交易，排队策略缓存
type txCache struct {
	*AccountTxIndex
	*LastTxCache
	qcache   QueueCache
	totalFee int64
	*SHashTxCache
	delayCache *delayTxCache
}

// NewTxCache init accountIndex and last cache
func newCache(maxTxPerAccount int64, sizeLast int64, poolCacheSize int64) *txCache {
	return &txCache{
		AccountTxIndex: NewAccountTxIndex(int(maxTxPerAccount)),
		LastTxCache:    NewLastTxCache(int(sizeLast)),
		SHashTxCache:   NewSHashTxCache(int(poolCacheSize)),
		delayCache:     newDelayTxCache(int(poolCacheSize) / 2),
	}
}

// SetQueueCache set queue cache , 这个接口可以扩展
func (cache *txCache) SetQueueCache(qcache QueueCache) {
	cache.qcache = qcache
}

// Remove 移除txCache中给定tx
func (cache *txCache) Remove(hash string) {
	item, err := cache.qcache.GetItem(hash)
	if err != nil {
		return
	}
	tx := item.Value
	err = cache.qcache.Remove(hash)
	if err != nil {
		mlog.Error("Remove", "cache Remove err", err)
	}
	cache.AccountTxIndex.Remove(tx, hash)
	cache.LastTxCache.Remove(hash)
	cache.totalFee -= tx.Fee
	cache.SHashTxCache.Remove(hash)
}

// Exist 是否存在
func (cache *txCache) Exist(hash string) bool {
	if cache.qcache == nil {
		return false
	}
	return cache.qcache.Exist(hash)
}

// Size cache tx num
func (cache *txCache) Size() int {
	if cache.qcache == nil {
		return 0
	}
	return cache.qcache.Size()
}

// TotalFee 手续费总和
func (cache *txCache) TotalFee() int64 {
	return cache.totalFee
}

// Walk iter all txs
func (cache *txCache) Walk(count int, cb func(tx *Item) bool) {
	if cache.qcache == nil {
		return
	}
	cache.qcache.Walk(count, cb)
}

// RemoveTxs 删除一组交易
func (cache *txCache) RemoveTxs(txs []string) {
	for _, t := range txs {
		cache.Remove(t)
	}
}

// Push 存入交易到cache 中
func (cache *txCache) Push(tx *types.Transaction) error {
	if !cache.AccountTxIndex.CanPush(tx) {
		return types.ErrManyTx
	}
	item := &Item{Value: tx, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	txHash := tx.Hash()
	err := cache.qcache.Push(item)
	if err != nil {
		return err
	}
	err = cache.AccountTxIndex.Push(tx, string(txHash))
	if err != nil {
		return err
	}
	cache.LastTxCache.Push(tx, string(txHash))
	cache.totalFee += tx.Fee
	cache.SHashTxCache.Push(tx, txHash)
	return nil
}

func (cache *txCache) removeExpiredTx(cfg *types.Chain33Config, height, blocktime int64) {
	var txs []string
	cache.qcache.Walk(0, func(tx *Item) bool {
		if isExpired(cfg, tx, height, blocktime) {
			txs = append(txs, string(tx.Value.Hash()))
		}
		return true
	})
	if len(txs) > 0 {
		mlog.Info("removeExpiredTx", "height", height, "totalTxs", cache.Size(), "expiredTxs", len(txs))
		cache.RemoveTxs(txs)
	}
}

// 判断交易是否过期
func isExpired(cfg *types.Chain33Config, item *Item, height, blockTime int64) bool {
	if types.Now().Unix()-item.EnterTime >= mempoolExpiredInterval {
		return true
	}
	if item.Value.IsExpire(cfg, height, blockTime) {
		return true
	}
	return false
}

// getTxByHash 通过交易hash获取tx交易信息
func (cache *txCache) getTxByHash(hash string) *types.Transaction {
	item, err := cache.qcache.GetItem(hash)
	if err != nil {
		return nil
	}
	return item.Value
}

// delayTxCache 延时交易缓存
type delayTxCache struct {
	size      int
	txCache   map[int64][]*types.Transaction // 以延时时间作为key索引
	hashCache map[string]int64               //哈希缓存，用于查重
	lock      sync.RWMutex
}

// new txCache
func newDelayTxCache(size int) *delayTxCache {
	return &delayTxCache{
		size:      size,
		txCache:   make(map[int64][]*types.Transaction, 16),
		hashCache: make(map[string]int64, 32),
	}
}

// 目前延时交易只做缓存，到期后发到mempool，通过节点广播可以基本避免局部缓存丢失问题
// TODO: 后续可以考虑增加磁盘存储
func (c *delayTxCache) addDelayTx(tx *types.DelayTx) error {

	c.lock.Lock()
	defer c.lock.Unlock()
	if tx.GetTx() == nil {
		return types.ErrNilTransaction
	}
	if len(c.hashCache) >= c.size {
		return types.ErrCacheOverFlow
	}
	txHash := string(tx.GetTx().Hash())
	if _, ok := c.hashCache[txHash]; ok {
		return types.ErrDupTx
	}
	// 标记交易哈希，并记录对应的延时时刻，方便快速查找交易
	c.hashCache[txHash] = tx.EndDelayTime
	txList, ok := c.txCache[tx.EndDelayTime]
	if !ok {
		txList = make([]*types.Transaction, 0, 16)
	}
	txList = append(txList, tx.GetTx())
	c.txCache[tx.EndDelayTime] = txList
	return nil
}

// 删除已到期的延时交易，即可以被打包的延期交易
func (c *delayTxCache) delExpiredTxs(lastBlockTime, currBlockTime, currBlockHeight int64) []*types.Transaction {
	c.lock.Lock()
	defer c.lock.Unlock()

	if len(c.hashCache) <= 0 {
		return nil
	}
	delList := make([]*types.Transaction, 0)

	// 延时时刻为区块时间
	for t := lastBlockTime + 1; t <= currBlockTime; t++ {

		if txList, ok := c.txCache[t]; ok {
			delList = append(delList, txList...)
			delete(c.txCache, t)
		}
	}

	// 延时时刻为区块高度
	if txList, ok := c.txCache[currBlockHeight]; ok {
		delList = append(delList, txList...)
		delete(c.txCache, currBlockHeight)
	}

	// 删除哈希缓存
	for _, tx := range delList {
		delete(c.hashCache, string(tx.Hash()))
	}
	return delList
}

func (c *delayTxCache) contains(txHash []byte) (int64, bool) {

	c.lock.RLock()
	defer c.lock.RUnlock()
	delayTime, exist := c.hashCache[string(txHash)]
	return delayTime, exist
}
