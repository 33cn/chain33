// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timeline

import (
	"container/list"

	"github.com/33cn/chain33/system/mempool"
	"github.com/33cn/chain33/types"
)

type txCache struct {
	*mempool.BaseCache
	txList *list.List
}

func newTxCache(cacheSize int64, b *mempool.BaseCache) *txCache {
	return &txCache{
		BaseCache: b,
		txList:    list.New(),
	}
}

func (cache *txCache) Reflect(val interface{}) *mempool.Item {
	return val.(*list.Element).Value.(*mempool.Item)
}

// txCache.Push把给定tx添加到txCache；如果tx已经存在txCache中或Mempool已满则返回对应error
func (cache *txCache) Push(tx *types.Transaction) error {
	hash := tx.Hash()
	if cache.Exists(hash) {
		elem := cache.TxMap[string(hash)].(*list.Element)
		addedItem := elem.Value.(*mempool.Item)
		addedTime := addedItem.EnterTime
		if types.Now().Unix()-addedTime < mempoolDupResendInterval {
			return types.ErrTxExist
		}
		// 超过2分钟之后的重发交易返回nil，再次发送给P2P，但是不再次加入mempool
		// 并修改其enterTime，以避免该交易一直在节点间被重发
		newEnterTime := types.Now().Unix()
		resendItem := &mempool.Item{Value: tx, Priority: tx.Fee, EnterTime: newEnterTime}
		newItem := cache.txList.InsertAfter(resendItem, elem)
		cache.txList.Remove(elem)
		cache.TxMap[string(hash)] = newItem
		return nil
	}

	if cache.txList.Len() >= cache.Size {
		return types.ErrMemFull
	}

	it := &mempool.Item{Value: tx, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	txElement := cache.txList.PushBack(it)
	cache.TxMap[string(hash)] = txElement

	// 账户交易数量
	accountAddr := tx.From()
	cache.AccMap[accountAddr] = append(cache.AccMap[accountAddr], tx)

	if len(cache.TxFrontTen) >= 10 {
		cache.TxFrontTen = cache.TxFrontTen[len(cache.TxFrontTen)-9:]
	}
	cache.TxFrontTen = append(cache.TxFrontTen, tx)

	return nil
}

func (cache *txCache) Remove(val interface{}) {
	elem := val.(*list.Element)
	cache.txList.Remove(elem)
}

func (cache *txCache) GetSize() int {
	return cache.txList.Len()
}

func (cache *txCache) GetTxList(minSize, height, blockTime int64, dupMap map[string]bool) []*types.Transaction {
	var result []*types.Transaction
	i := 0
	for v := cache.txList.Front(); v != nil; v = v.Next() {
		if v.Value.(*mempool.Item).Value.IsExpire(height, blockTime) {
			continue
		} else {
			tx := v.Value.(*mempool.Item).Value
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
