// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRollbackblock(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	//发送交易
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), 22)
	defer mock33.Close()
	txs := util.GenCoinsTxs(mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(1)
	txs = util.GenCoinsTxs(mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(2)
	txs = util.GenNoneTxs(mock33.GetGenesisKey(), 1)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(3)
	txs = util.GenNoneTxs(mock33.GetGenesisKey(), 2)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(4)
	time.Sleep(time.Second)

	chain.Rollbackblock()
	chain.SetRollbackBlockHeight(2)
	chain.Rollback()
	require.Equal(t, int64(2), chain.GetBlockHeight())
}

func TestSetRollbackBlockHeight(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	chain := mock33.GetBlockChain()
	chain.SetRollbackBlockHeight(1)
}
