// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"bytes"
	"testing"
	"time"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/require"
)

var (
	//区块0产生的kv对数量
	// 不需要保存hash->height的索引，所以少一个kv对
	kvCount = 24
)

func TestReindex(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(cfg, nil)
	//发送交易
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	require.Equal(t, len(kvs), kvCount)
	defer mock33.Close()
	txs := util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(1)
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(2)
	txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 1)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(3)
	txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 2)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(4)
	time.Sleep(time.Second)
	kvs1 := getAllKeys(db)
	version.SetLocalDBVersion("10000.0.0")
	chain.UpgradeChain()
	kvs2 := getAllKeys(db)
	require.Equal(t, kvs1, kvs2)
}

func getAllKeys(db dbm.IteratorDB) (kvs []*types.KeyValue) {
	it := db.Iterator(nil, types.EmptyValue, false)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		key := copyBytes(it.Key())
		val := it.ValueCopy()
		//meta 信息是唯一不同的地方
		if string(key) == "LocalDBMeta" {
			continue
		}
		if bytes.HasPrefix(key, []byte("TotalFee")) {
			//println("--", string(key)[0:4], common.ToHex(key))
			totalFee := &types.TotalFee{}
			types.Decode(val, totalFee)
			//println("val", totalFee.String())
		}
		kvs = append(kvs, &types.KeyValue{Key: key, Value: val})
	}
	return kvs
}

func copyBytes(keys []byte) []byte {
	data := make([]byte, len(keys))
	copy(data, keys)
	return data
}

func TestUpgradeChain(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	cfg.GetModuleConfig().Store.LocalDBVersion = "0.0.0"
	mock33 := testnode.NewWithConfig(cfg, nil)
	//发送交易
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	require.Equal(t, len(kvs), kvCount)
	defer mock33.Close()
	txs := util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(1)
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(2)
	txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 1)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(3)
	txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 2)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		require.Nil(t, err)
		require.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(4)
	time.Sleep(time.Second)

	version.SetLocalDBVersion("1.0.0")
	chain.UpgradeChain()
	ver, err := getUpgradeMeta(db)
	require.Nil(t, err)
	require.Equal(t, ver.GetVersion(), "1.0.0")

	version.SetLocalDBVersion("2.0.0")
	chain.UpgradeChain()

	ver, err = getUpgradeMeta(db)
	require.Nil(t, err)
	require.Equal(t, ver.GetVersion(), "2.0.0")
}

// GetUpgradeMeta 获取blockchain的数据库版本号
func getUpgradeMeta(db dbm.DB) (*types.UpgradeMeta, error) {
	ver := types.UpgradeMeta{}
	version, err := db.Get(version.LocalDBMeta)
	if err != nil && err != dbm.ErrNotFoundInDb {
		return nil, err
	}
	if len(version) == 0 {
		return &types.UpgradeMeta{Version: "0.0.0"}, nil
	}
	err = types.Decode(version, &ver)
	if err != nil {
		return nil, err
	}
	return &ver, nil
}
