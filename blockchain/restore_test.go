// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestNeedReExec(t *testing.T) {
	defer func() {
		r := recover()
		if r == "not support upgrade store to greater than 2.0.0" {
			assert.Equal(t, r, "not support upgrade store to greater than 2.0.0")
		} else if r == "not support degrade the program" {
			assert.Equal(t, r, "not support degrade the program")
		}
	}()
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	mock33 := testnode.NewWithConfig(cfg, nil)

	//发送交易
	chain := mock33.GetBlockChain()
	// 相当于数据库中的版本
	testcase := []*types.UpgradeMeta{
		{Starting: true},
		{Starting: false, Version: "1.0.0"},
		{Starting: false, Version: "1.0.0"},
		{Starting: false, Version: "2.0.0"},
		{Starting: false, Version: "2.0.0"},
	}
	// 程序中的版本
	vsCase := []string{
		"1.0.0",
		"1.0.0",
		"2.0.0",
		"3.0.0",
		"1.0.0",
	}
	result := []bool{
		true,
		false,
		true,
		false,
		false,
	}
	for i, c := range testcase {
		version.SetStoreDBVersion(vsCase[i])
		res := chain.NeedReExec(c)
		assert.Equal(t, res, result[i])
	}
}

func TestUpgradeStore(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.GetModuleConfig().BlockChain.EnableReExecLocal = true
	mock33 := testnode.NewWithConfig(cfg, nil)
	//发送交易
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), 22)
	defer mock33.Close()
	txs := util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(1)
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(2)
	txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 1)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(3)
	txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 2)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(4)
	time.Sleep(time.Second)
	addr := address.PubKeyToAddress(mock33.GetGenesisKey().PubKey().Bytes()).String()
	count1, err := GetAddrTxsCount(db, addr)
	assert.NoError(t, err)
	version.SetStoreDBVersion("2.0.0")
	chain.UpgradeStore()
	count2, err := GetAddrTxsCount(db, addr)
	assert.NoError(t, err)
	assert.Equal(t, count1, count2)
}

func GetAddrTxsCount(db dbm.DB, addr string) (int64, error) {
	count := types.Int64{}
	TxsCount, err := db.Get(types.CalcAddrTxsCountKey(addr))
	if err != nil && err != types.ErrNotFound {
		return 0, err
	}
	if len(TxsCount) == 0 {
		return 0, nil
	}
	err = types.Decode(TxsCount, &count)
	if err != nil {
		return 0, err
	}
	return count.Data, nil
}
