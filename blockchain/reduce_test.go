// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"fmt"
	"testing"

	"github.com/33cn/chain33/blockchain"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestTryReduceLocalDB(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(cfg, nil)
	//发送交易
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), 23)
	defer mock33.Close()

	blockchain.ReduceHeight = 0
	defer func() {
		blockchain.ReduceHeight = 10000
	}()

	var flagHeight int64
	count := 10
	for i := 0; i < 2; i++ {
		txs := util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), int64(count))
		for j := 0; j < len(txs); j++ {
			reply, err := mock33.GetAPI().SendTx(txs[j])
			assert.Nil(t, err)
			assert.Equal(t, reply.IsOk, true)
			waitH := i*count + (j + 1)
			mock33.WaitHeight(int64(waitH))
		}
		flagHeight = chain.TryReduceLocalDB(flagHeight, int64(count))
		assert.Equal(t, flagHeight, int64((i+1)*count+1))
	}
}

func TestFIFO(t *testing.T) {
	fifo10 := blockchain.NewFIFO(10)

	for i := 0; i < 11; i++ {
		fifo10.Add(i, []byte(fmt.Sprintf("value-%d", i)))
	}
	// check Contains Get
	ok := fifo10.Contains(0)
	assert.Equal(t, ok, false)
	value, ok := fifo10.Get(0)
	assert.Equal(t, value, nil)
	assert.Equal(t, ok, false)
	for i := 1; i < 11; i++ {
		ok := fifo10.Contains(i)
		assert.Equal(t, ok, true)
		value, ok := fifo10.Get(i)
		assert.Equal(t, ok, true)
		assert.Equal(t, value.([]byte), []byte(fmt.Sprintf("value-%d", i)))
	}
	// check Remove
	ok = fifo10.Remove(10)
	assert.Equal(t, ok, true)
	value, ok = fifo10.Get(10)
	assert.Equal(t, value, nil)
	assert.Equal(t, ok, false)

	ok = fifo10.Remove(11)
	assert.Equal(t, ok, false)

	// test for size = 0
	fifo0 := blockchain.NewFIFO(0)

	fifo0.Add(0, []byte(fmt.Sprintf("value-%d", 0)))
	value, ok = fifo0.Get(0)
	assert.Equal(t, ok, true)
	assert.Equal(t, value.([]byte), []byte(fmt.Sprintf("value-%d", 0)))

	fifo0.Add(1, []byte(fmt.Sprintf("value-%d", 1)))
	value, ok = fifo0.Get(0)
	assert.Equal(t, ok, false)

	value, ok = fifo0.Get(1)
	assert.Equal(t, ok, true)
	assert.Equal(t, value.([]byte), []byte(fmt.Sprintf("value-%d", 1)))

	// remove 0  fasle
	ok = fifo0.Remove(0)
	assert.Equal(t, ok, false)

	// remove 1 true
	ok = fifo0.Remove(1)
	assert.Equal(t, ok, true)
	// Get 1 false
	_, ok = fifo0.Get(1)
	assert.Equal(t, ok, false)
}
