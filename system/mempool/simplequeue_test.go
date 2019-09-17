// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	subConfig := SubConfig{1, 100000}
	cache := NewSimpleQueue(subConfig)
	tx := &types.Transaction{Payload: []byte("123")}
	hash := string(tx.Hash())
	assert.Equal(t, false, cache.Exist(hash))
	item1 := &Item{Value: tx, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	err := cache.Push(item1)
	assert.Nil(t, err)
	assert.Equal(t, true, cache.Exist(hash))
	it, err := cache.GetItem(hash)
	assert.Nil(t, err)
	assert.Equal(t, item1, it)

	_, err = cache.GetItem(hash + ":")
	assert.Equal(t, types.ErrNotFound, err)

	err = cache.Push(item1)
	assert.Equal(t, types.ErrTxExist, err)

	tx2 := &types.Transaction{Payload: []byte("1234")}
	item2 := &Item{Value: tx2, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	err = cache.Push(item2)
	assert.Equal(t, types.ErrMemFull, err)

	cache.Remove(hash)
	assert.Equal(t, 0, cache.Size())
	//push to item
	subConfig = SubConfig{2, 100000}
	cache = NewSimpleQueue(subConfig)
	cache.Push(item1)
	cache.Push(item2)
	assert.Equal(t, 2, cache.Size())
	var data [2]*Item
	i := 0
	cache.Walk(1, func(value *Item) bool {
		data[i] = value
		i++
		return true
	})
	assert.Equal(t, 1, i)
	assert.Equal(t, data[0], item1)

	i = 0
	cache.Walk(2, func(value *Item) bool {
		data[i] = value
		i++
		return true
	})
	assert.Equal(t, 2, i)
	assert.Equal(t, data[0], item1)
	assert.Equal(t, data[1], item2)

	i = 0
	cache.Walk(2, func(value *Item) bool {
		data[i] = value
		i++
		return false
	})
	assert.Equal(t, 1, i)

	//test timeline GetProperFee
	assert.Equal(t, int64(100000), cache.GetProperFee())
}
