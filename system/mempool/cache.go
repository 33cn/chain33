// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found In the LICENSE file.

package mempool

import (
	"github.com/33cn/chain33/types"
	"github.com/hashicorp/golang-lru"
)

type Cache interface {
	Reflect(interface{}) *Item
	Push(tx *types.Transaction) error
	Remove(val interface{})
	GetSize() int
	GetTxList(minSize, height, blockTime int64, dupMap map[string]bool) []*types.Transaction
}

type BaseCache struct {
	size       int
	TxFrontTen []*types.Transaction
	AccMap     map[string][]*types.Transaction
	TxMap      map[string]interface{}
	child      Cache
}

// Item 为Mempool中包装交易的数据结构
type Item struct {
	Value     *types.Transaction
	Priority  int64
	EnterTime int64
}

func NewBaseCache(cacheSize int64) *BaseCache {
	return &BaseCache{
		size:       int(cacheSize),
		TxFrontTen: make([]*types.Transaction, 0),
		AccMap:     make(map[string][]*types.Transaction),
		TxMap:      make(map[string]interface{}),
	}
}

func (cache *BaseCache) SetChild(c Cache) {
	cache.child = c
}

// BaseCache.TxNumOfAccount返回账户在Mempool中交易数量
func (cache *BaseCache) TxNumOfAccount(addr string) int64 {
	return int64(len(cache.AccMap[addr]))
}

// txCache.GetLatestTx返回最新十条加入到txCache的交易
func (cache *BaseCache) GetLatestTx() []*types.Transaction {
	return cache.TxFrontTen
}

// txCache.AccountTxNumDecrease根据交易哈希删除对应账户的对应交易
func (cache *BaseCache) AccountTxNumDecrease(addr string, hash []byte) {
	if value, ok := cache.AccMap[addr]; ok {
		for i, t := range value {
			if string(t.Hash()) == string(hash) {
				cache.AccMap[addr] = append(cache.AccMap[addr][:i], cache.AccMap[addr][i+1:]...)
				if len(cache.AccMap[addr]) == 0 {
					delete(cache.AccMap, addr)
				}
				return
			}
		}
	}
}

// txCache.GetAccTxs用来获取对应账户地址（列表）中的全部交易详细信息
func (cache *BaseCache) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	res := &types.TransactionDetails{}
	for _, addr := range addrs.Addrs {
		if value, ok := cache.AccMap[addr]; ok {
			for _, v := range value {
				txAmount, err := v.Amount()
				if err != nil {
					// continue
					txAmount = 0
				}
				res.Txs = append(res.Txs,
					&types.TransactionDetail{
						Tx:         v,
						Amount:     txAmount,
						Fromaddr:   addr,
						ActionName: v.ActionName(),
					})
			}
		}
	}
	return res
}

func (cache *BaseCache) Exists(hash []byte) bool {
	_, exists := cache.TxMap[string(hash)]
	return exists
}

// txCache.Remove移除txCache中给定tx
func (cache *BaseCache) Remove(hash []byte) {
	value := cache.TxMap[string(hash)]
	if value == nil {
		return
	}

	cache.child.Remove(value)
	delete(cache.TxMap, string(hash))
	item := cache.child.Reflect(value)
	tx := item.Value
	addr := tx.From()
	if cache.TxNumOfAccount(addr) > 0 {
		cache.AccountTxNumDecrease(addr, hash)
	}
}

func (cache *BaseCache) RemoveBlockedTxs(dupTxs [][]byte, addedTxs *lru.Cache) {
	for _, t := range dupTxs {
		txValue, exists := cache.TxMap[string(t)]
		if exists {
			addedTxs.Add(string(t), nil)
			item := cache.child.Reflect(txValue)
			cache.Remove(item.Value.Hash())
		}
	}
}

func (cache *BaseCache) RemoveExpiredTx(height, blockTime int64) []*types.Transaction {
	var result []*types.Transaction
	for _, v := range cache.TxMap {
		item := cache.child.Reflect(v)
		hash := item.Value.Hash()
		if types.Now().Unix()-item.EnterTime >= mempoolExpiredInterval {
			// 清理滞留mempool中超过10分钟的交易
			cache.Remove(hash)
		} else if item.Value.IsExpire(height, blockTime) {
			// 清理过期的交易
			cache.Remove(hash)
		} else {
			result = append(result, item.Value)
		}
	}
	return result
}
