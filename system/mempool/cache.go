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
}

// Item 为Mempool中包装交易的数据结构
type Item struct {
	Value     *types.Transaction
	Priority  int64
	EnterTime int64
}

//TxCache 管理交易cache 包括账户索引，最后的交易，排队策略缓存
type TxCache struct {
	accountIndex *AccountTxIndex
	last         *LastTxCache
	qcache       QueueCache
}

//NewTxCache init accountIndex and last cache
func NewTxCache(maxTxPerAccount int64, sizeLast int64) *TxCache {
	return &TxCache{
		accountIndex: NewAccountTxIndex(int(maxTxPerAccount)),
		last:         NewLastTxCache(int(sizeLast)),
	}
}

//Remove 移除txCache中给定tx
func (cache *TxCache) Remove(hash string) {
	item, err := cache.qcache.GetItem(hash)
	if err != nil {
		return
	}
	tx := item.Value
	cache.qcache.Remove(hash)
	cache.accountIndex.Remove(tx)
	cache.last.Remove(tx)
}

//RemoveTxs 删除一组交易
func (cache *TxCache) RemoveTxs(txs []string) {
	for _, t := range txs {
		cache.Remove(t)
	}
}

//Push 存入交易到cache 中
func (cache *TxCache) Push(tx *types.Transaction) error {
	if !cache.accountIndex.CanPush(tx) {
		return types.ErrManyTx
	}
	item := &Item{Value: tx, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	err := cache.qcache.Push(item)
	if err != nil {
		return err
	}
	cache.accountIndex.Push(tx)
	cache.last.Push(tx)
	return nil
}

func (cache *TxCache) removeExpiredTx(height, blocktime int64) {
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
