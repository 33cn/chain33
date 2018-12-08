// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/33cn/chain33/common/listmap"
	"github.com/33cn/chain33/types"
)

//SimpleQueue 简单队列模式(默认提供一个队列，便于测试)
type SimpleQueue struct {
	txList  *listmap.ListMap
	maxsize int
}

//NewSimpleQueue 创建队列
func NewSimpleQueue(cacheSize int) *SimpleQueue {
	return &SimpleQueue{
		txList:  listmap.New(),
		maxsize: cacheSize,
	}
}

//Exist 是否存在
func (cache *SimpleQueue) Exist(hash string) bool {
	return cache.txList.Exist(hash)
}

//GetItem 获取数据通过 key
func (cache *SimpleQueue) GetItem(hash string) (*Item, error) {
	item, err := cache.txList.GetItem(hash)
	if err != nil {
		return nil, err
	}
	return item.(*Item), nil
}

// Push 把给定tx添加到SimpleQueue；如果tx已经存在SimpleQueue中或Mempool已满则返回对应error
func (cache *SimpleQueue) Push(tx *Item) error {
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

// Remove 删除数据
func (cache *SimpleQueue) Remove(hash string) error {
	cache.txList.Remove(hash)
	return nil
}

// Size 数据总数
func (cache *SimpleQueue) Size() int {
	return cache.txList.Size()
}

// Walk 遍历整个队列
func (cache *SimpleQueue) Walk(count int, cb func(value *Item) bool) {
	i := 0
	cache.txList.Walk(func(item interface{}) bool {
		if !cb(item.(*Item)) {
			return false
		}
		i++
		return i != count
	})
}
