// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found In the LICENSE file.

package mempool

import (
	"github.com/33cn/chain33/types"
)

//QueueCache 排队交易处理
type QueueCache interface {
	Exist(hash string) bool
	GetItem(hash string) (*Item, error)
	Push(tx *Item) error
	Remove(hash string) error
	Size() int
	Walk(count int, cb func(tx *Item) bool)
	GetProperFee() int64
}

// Item 为Mempool中包装交易的数据结构
type Item struct {
	Value     *types.Transaction
	Priority  int64
	EnterTime int64
}

//TxCache 管理交易cache 包括账户索引，最后的交易，排队策略缓存
type txCache struct {
	*AccountTxIndex
	*LastTxCache
	qcache QueueCache
}

//NewTxCache init accountIndex and last cache
func newCache(maxTxPerAccount int64, sizeLast int64) *txCache {
	return &txCache{
		AccountTxIndex: NewAccountTxIndex(int(maxTxPerAccount)),
		LastTxCache:    NewLastTxCache(int(sizeLast)),
	}
}

//SetQueueCache set queue cache , 这个接口可以扩展
func (cache *txCache) SetQueueCache(qcache QueueCache) {
	cache.qcache = qcache
}

//Remove 移除txCache中给定tx
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
	cache.AccountTxIndex.Remove(tx)
	cache.LastTxCache.Remove(tx)
}

//Exist 是否存在
func (cache *txCache) Exist(hash string) bool {
	if cache.qcache == nil {
		return false
	}
	return cache.qcache.Exist(hash)
}

//Size cache tx num
func (cache *txCache) Size() int {
	if cache.qcache == nil {
		return 0
	}
	return cache.qcache.Size()
}

//Walk iter all txs
func (cache *txCache) Walk(count int, cb func(tx *Item) bool) {
	if cache.qcache == nil {
		return
	}
	cache.qcache.Walk(count, cb)
}

//RemoveTxs 删除一组交易
func (cache *txCache) RemoveTxs(txs []string) {
	for _, t := range txs {
		cache.Remove(t)
	}
}

//Push 存入交易到cache 中
func (cache *txCache) Push(tx *types.Transaction) error {
	if !cache.AccountTxIndex.CanPush(tx) {
		return types.ErrManyTx
	}
	item := &Item{Value: tx, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	err := cache.qcache.Push(item)
	if err != nil {
		return err
	}
	err = cache.AccountTxIndex.Push(tx)
	if err != nil {
		return err
	}
	cache.LastTxCache.Push(tx)
	return nil
}

func (cache *txCache) removeExpiredTx(height, blocktime int64) {
	var txs []string
	cache.qcache.Walk(0, func(tx *Item) bool {
		if isExpired(tx, height, blocktime) {
			txs = append(txs, string(tx.Value.Hash()))
		}
		return true
	})
	cache.RemoveTxs(txs)
}

//判断交易是否过期
func isExpired(item *Item, height, blockTime int64) bool {
	if types.Now().Unix()-item.EnterTime >= mempoolExpiredInterval {
		return true
	}
	if item.Value.IsExpire(height, blockTime) {
		return true
	}
	return false
}
