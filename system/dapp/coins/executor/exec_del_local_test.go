// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

var testCfg = types.NewChain33Config(types.GetDefaultCfgstring())

const testCoinUnit = int64(1e8)

// The coins driver is registered once by the init() in exec_genesis_test.go.

func initTestCoins(t *testing.T) (string, *Coins) {
	q := queue.New("testcoinsdellocal")
	q.SetConfig(testCfg)
	api, err := client.New(q.Client(), nil)
	require.Nil(t, err)
	dbDir, stateDB, kvDB := util.CreateTestDB()
	c := newCoins()
	c.SetAPI(api)
	c.SetStateDB(stateDB)
	c.SetLocalDB(kvDB)
	return dbDir, c.(*Coins)
}

func closeTestCoins(t *testing.T, dbDir string, c *Coins) {
	util.CloseTestDB(dbDir, c.GetStateDB().(db.DB))
}

// signCoinsTx and genesisAction are shared helpers defined in exec_genesis_test.go.

// TestExecDelLocalGenesis checks that ExecDelLocal of a genesis tx reverts the
// address receiver stat (LODB-coins-Addr:*) written by ExecLocal, so rolling
// back a block at height 0 leaves no stale local data.
func TestExecDelLocalGenesis(t *testing.T) {
	dbDir, c := initTestCoins(t)
	defer closeTestCoins(t, dbDir, c)

	addr, _ := util.Genaddress()
	_, priv := util.Genaddress()
	c.SetEnv(0, 1539918074, 1)

	tx := signCoinsTx(priv, addr, genesisAction(100*testCoinUnit))
	receiptData := &types.ReceiptData{Ty: types.ExecOk}

	// ExecLocal writes the address receiver stat
	lset, err := c.ExecLocal(tx, receiptData, 0)
	require.Nil(t, err)
	require.Equal(t, 1, len(lset.KV))
	require.Nil(t, c.GetLocalDB().Set(lset.KV[0].Key, lset.KV[0].Value))
	recv, err := getAddrReciver(c.GetLocalDB(), addr)
	require.Nil(t, err)
	require.Equal(t, 100*testCoinUnit, recv)

	// ExecDelLocal must produce the rollback KV, and applying it reverts the stat
	delSet, err := c.ExecDelLocal(tx, receiptData, 0)
	require.Nil(t, err)
	require.Equal(t, 1, len(delSet.KV))
	require.Nil(t, c.GetLocalDB().Set(delSet.KV[0].Key, delSet.KV[0].Value))
	recv, err = getAddrReciver(c.GetLocalDB(), addr)
	require.Nil(t, err)
	require.Equal(t, int64(0), recv)
}
