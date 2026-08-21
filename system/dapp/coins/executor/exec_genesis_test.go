// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	drivers "github.com/33cn/chain33/system/dapp"
	dtypes "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

var genesisTestCfg = types.NewChain33Config(types.GetDefaultCfgstring())

// coinUnit 1 个主币 = 1e8（DefaultCoinPrecision）
const coinUnit = int64(1e8)

func init() {
	Init(driverName, genesisTestCfg, nil)
}

func initGenesisTestCoins(t *testing.T) (string, *Coins) {
	q := queue.New("testcoinsgenesis")
	q.SetConfig(genesisTestCfg)
	api, err := client.New(q.Client(), nil)
	require.Nil(t, err)
	dbDir, stateDB, kvDB := util.CreateTestDB()
	c := newCoins()
	c.SetAPI(api)
	c.SetStateDB(stateDB)
	c.SetLocalDB(kvDB)
	return dbDir, c.(*Coins)
}

func closeGenesisTestCoins(t *testing.T, dbDir string, c *Coins) {
	util.CloseTestDB(dbDir, c.GetStateDB().(db.DB))
}

func signCoinsTx(priv crypto.PrivKey, to string, action *dtypes.CoinsAction) *types.Transaction {
	tx := &types.Transaction{
		Execer:  []byte("coins"),
		Payload: types.Encode(action),
		To:      to,
		Fee:     int64(1e6),
	}
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genesisAction(amount int64) *dtypes.CoinsAction {
	return &dtypes.CoinsAction{
		Ty:    dtypes.CoinsActionGenesis,
		Value: &dtypes.CoinsAction_Genesis{Genesis: &types.AssetsGenesis{Amount: amount}},
	}
}

// TestGenesisNegativeAmount 负数创世交易必须在执行器入口被拒绝：
// 修复前 Exec_Genesis 只校验 height==0，account.GenesisInit 未做金额校验，
// 负数金额可以执行成功并生成负数余额。
func TestGenesisNegativeAmount(t *testing.T) {
	dbDir, c := initGenesisTestCoins(t)
	defer closeGenesisTestCoins(t, dbDir, c)

	victim, _ := util.Genaddress()
	_, genesisPriv := util.Genaddress()
	c.SetEnv(0, 1539918074, 1) // height 0

	tx := signCoinsTx(genesisPriv, victim, genesisAction(-100*coinUnit))
	_, err := c.Exec(tx, 0)
	require.Equal(t, types.ErrAmount, err)

	acc := c.GetCoinsAccount().LoadAccount(victim)
	require.Equal(t, int64(0), acc.Balance)
}

// TestGenesisZeroAmount 零金额创世交易同样被拒绝
func TestGenesisZeroAmount(t *testing.T) {
	dbDir, c := initGenesisTestCoins(t)
	defer closeGenesisTestCoins(t, dbDir, c)

	victim, _ := util.Genaddress()
	_, genesisPriv := util.Genaddress()
	c.SetEnv(0, 1539918074, 1)

	tx := signCoinsTx(genesisPriv, victim, genesisAction(0))
	_, err := c.Exec(tx, 0)
	require.Equal(t, types.ErrAmount, err)
}

// TestGenesisNegativeToExec 负数创世到执行器地址也必须在入口被拒绝（ErrAmount），
// 而不是在 GenesisInitExec 内部 panic 后被 recover 成 ErrActionNotSupport。
func TestGenesisNegativeToExec(t *testing.T) {
	// 注册一个名为 ticket 的驱动，使其执行器地址被 IsDriverAddress 识别
	drivers.Register(genesisTestCfg, "ticket", newCoins, 0)

	dbDir, c := initGenesisTestCoins(t)
	defer closeGenesisTestCoins(t, dbDir, c)

	retAddr, _ := util.Genaddress()
	_, genesisPriv := util.Genaddress()
	execAddr := address.ExecAddress("ticket")
	c.SetEnv(0, 1539918074, 1)

	action := &dtypes.CoinsAction{
		Ty: dtypes.CoinsActionGenesis,
		Value: &dtypes.CoinsAction_Genesis{Genesis: &types.AssetsGenesis{
			Amount:        -100 * coinUnit,
			ReturnAddress: retAddr,
		}},
	}
	tx := signCoinsTx(genesisPriv, execAddr, action)
	_, err := c.Exec(tx, 0)
	require.Equal(t, types.ErrAmount, err)
}

// TestGenesisNormal 正常金额创世交易在 height 0 执行成功
func TestGenesisNormal(t *testing.T) {
	dbDir, c := initGenesisTestCoins(t)
	defer closeGenesisTestCoins(t, dbDir, c)

	addr, _ := util.Genaddress()
	_, priv := util.Genaddress()
	c.SetEnv(0, 1539918074, 1)

	tx := signCoinsTx(priv, addr, genesisAction(100*coinUnit))
	receipt, err := c.Exec(tx, 0)
	require.Nil(t, err)
	require.Equal(t, int32(types.ExecOk), receipt.Ty)
	for _, kv := range receipt.KV {
		require.Nil(t, c.GetStateDB().Set(kv.Key, kv.Value))
	}
	acc := c.GetCoinsAccount().LoadAccount(addr)
	require.Equal(t, 100*coinUnit, acc.Balance)
}

// TestGenesisRerun 非 0 高度重复执行创世交易被拒绝
func TestGenesisRerun(t *testing.T) {
	dbDir, c := initGenesisTestCoins(t)
	defer closeGenesisTestCoins(t, dbDir, c)

	addr, _ := util.Genaddress()
	_, priv := util.Genaddress()
	c.SetEnv(500000, 1539918074, 1)

	tx := signCoinsTx(priv, addr, genesisAction(100*coinUnit))
	_, err := c.Exec(tx, 0)
	require.Equal(t, types.ErrReRunGenesis, err)
}
