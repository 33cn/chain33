// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timeline

import (
	"testing"

	"github.com/33cn/chain33/system/mempool"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestCache(t *testing.T) {
	cache := newTxCache(1)
	tx := &types.Transaction{Payload: []byte("123")}
	hash := string(tx.Hash())
	assert.Equal(t, false, cache.Exist(hash))
	item := &mempool.Item{Value: tx, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	err := cache.Push(item)
	assert.Nil(t, err)
	assert.Equal(t, true, cache.Exist(hash))
	it, err := cache.GetItem(hash)
	assert.Nil(t, err)
	assert.Equal(t, item, it)

	_, err = cache.GetItem(hash + ":")
	assert.Equal(t, types.ErrNotFound, err)

	err = cache.Push(item)
	assert.Equal(t, types.ErrTxExist, err)

	tx2 := &types.Transaction{Payload: []byte("1234")}
	item = &mempool.Item{Value: tx2, Priority: tx.Fee, EnterTime: types.Now().Unix()}
	err = cache.Push(item)
	assert.Equal(t, types.ErrMemFull, err)

	cache.Remove(hash)
	assert.Equal(t, 0, cache.Size())
}
