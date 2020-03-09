// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugin

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestPlugin(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)

	var txs []*types.Transaction
	_, priv := util.Genaddress()
	tx := &types.Transaction{Fee: 1000000, Execer: []byte("none")}
	tx.Sign(types.SECP256K1, priv)
	txs = append(txs, tx)
	var stateHash [32]byte
	stateHash[0] = 30
	for _, newPlugin := range globalPlugins {
		detail := &types.BlockDetail{
			Block:    &types.Block{Txs: txs, StateHash: stateHash[:]},
			Receipts: []*types.ReceiptData{{}},
		}

		plugin := newPlugin()
		_, _, err := plugin.CheckEnable(false)
		assert.NoError(t, err)
		kvs, err := plugin.ExecLocal(detail)
		assert.NoError(t, err)
		for _, kv := range kvs {
			err = kvdb.Set(kv.Key, kv.Value)
			assert.NoError(t, err)
		}
		_, err = plugin.ExecDelLocal(detail)
		assert.NoError(t, err)
	}
}

func TestPluginFlag(t *testing.T) {
	flag := new(Flag)
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)

	base := Base{}
	base.SetEnv(0, 1000000, 11111111111111)
	base.SetLocalDB(kvdb)

	k := []byte("test")
	_, _, err := flag.CheckFlag(&base, k, true)
	assert.NoError(t, err)

	v := types.Encode(&types.Int64{})
	err = kvdb.Set(k, v)
	assert.NoError(t, err)
	_, _, err = flag.CheckFlag(&base, k, true)
	assert.NoError(t, err)
}

func TestGenNewKey(t *testing.T) {
	old := []byte("old:1223")
	x := genNewKey(old, []byte("old:"), []byte("new:"))
	assert.Equal(t, []byte("new:1223"), x)
	assert.Equal(t, []byte("old:1223"), old)
}
