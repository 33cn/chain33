// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timeline

import (
	"container/list"

	"github.com/33cn/chain33/common/listmap"
	"github.com/33cn/chain33/system/mempool"
	"github.com/33cn/chain33/types"
)

type txCache struct {
	txList  *listmap.ListMap
	maxsize int
}

func newTxCache(cacheSize int) *txCache {
	return &txCache{
		txList:  listmap.New(),
		maxsize: cacheSize,
	}
}

func (cache *txCache) Exist(hash string) bool {
	return cache.txList.Exist(hash)
}

func (cache *txCache) GetItem(hash string) (*mempool.Item, error) {
	item, err := cache.txList.GetItem(hash)
	if err != nil {
		return nil, err
	}
	return item.(*list.Element).Value.(*mempool.Item), nil
}

// Push 把给定tx添加到txCache；如果tx已经存在txCache中或Mempool已满则返回对应error
func (cache *txCache) Push(tx *mempool.Item) error {
	hash := tx.Value.Hash()
	if cache.Exist(string(hash)) {
		return types.ErrTxExist
	}
	if cache.txList.Size() >= cache.maxsize {
		return types.ErrMemFull
	}
	cache.txList.Push(string(hash), tx)
	return nil
}

func (cache *txCache) Remove(hash string) error {
	cache.txList.Remove(hash)
	return nil
}

func (cache *txCache) Size() int {
	return cache.txList.Size()
}

func (cache *txCache) Walk(count int, cb func(value *mempool.Item) bool) {
	i := 0
	cache.txList.Walk(func(item interface{}) bool {
		if !cb(item.(*mempool.Item)) {
			return false
		}
		i++
		if i == count {
			return false
		}
		return true
	})
}
