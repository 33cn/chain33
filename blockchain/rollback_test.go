// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	"fmt"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	types.SetDappFork("local", "store-kvmvccmavl", "ForkKvmvccmavl", 20*10000)
}

func TestRollbackblock(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	cfg.BlockChain.RollbackBlock = 0
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	chain := mock33.GetBlockChain()
	chain.Rollbackblock()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), 22)
	defer mock33.Close()

	//发送交易
	testMockSendTx(t, mock33)

	chain.Rollbackblock()
}

func TestNeedRollback(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	chain := mock33.GetBlockChain()

	curHeight := int64(5)
	rollHeight := int64(5)
	ok := chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, false, ok)

	curHeight = int64(5)
	rollHeight = int64(6)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, false, ok)

	curHeight = int64(10 * 10000)
	rollHeight = int64(5 * 10000)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, true, ok)

	curHeight = int64(20*10000 + 1)
	rollHeight = int64(20 * 10000)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, true, ok)

	curHeight = int64(22 * 10000)
	rollHeight = int64(20 * 10000)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, false, ok)

	curHeight = int64(22 * 10000)
	rollHeight = int64(20*10000 + 1)
	ok = chain.NeedRollback(curHeight, rollHeight)
	require.Equal(t, true, ok)

}

func TestRollback(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	cfg.BlockChain.RollbackBlock = 2
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), 22)
	defer mock33.Close()

	//发送交易
	testMockSendTx(t, mock33)

	chain.Rollback()
	require.Equal(t, int64(2), chain.GetBlockHeight())
}

func TestRollbackSave(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	cfg.BlockChain.RollbackBlock = 2
	cfg.BlockChain.RollbackSave = true
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), 22)
	defer mock33.Close()

	//发送交易
	testMockSendTx(t, mock33)

	height := chain.GetBlockHeight()
	chain.Rollback()

	// check
	require.Equal(t, int64(2), chain.GetBlockHeight())
	for i := height; i > 2; i-- {
		key := []byte(fmt.Sprintf("TB:%012d", i))
		_, err := chain.GetDB().Get(key)
		assert.NoError(t, err)
	}
	value, err := chain.GetDB().Get([]byte("LTB:"))
	assert.NoError(t, err)
	assert.NotNil(t, value)
	h := &types.Int64{}
	err = types.Decode(value, h)
	assert.NoError(t, err)
	assert.Equal(t, height, h.Data)
}

func TestRollbackPara(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	cfg.BlockChain.RollbackBlock = 2
	cfg.BlockChain.IsParaChain = true
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	chain := mock33.GetBlockChain()
	defer mock33.Close()

	//发送交易
	testMockSendTx(t, mock33)

	chain.Rollback()
	require.Equal(t, int64(2), chain.GetBlockHeight())
}

func testMockSendTx(t *testing.T, mock33 *testnode.Chain33Mock) {
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
}
