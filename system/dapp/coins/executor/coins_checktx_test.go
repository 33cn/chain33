// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/queue"
	dtypes "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

var checkTxCfg = types.NewChain33Config(types.GetDefaultCfgstring())

func init() {
	Init(driverName, checkTxCfg, nil)
}

func initCheckTxCoins(t *testing.T) *Coins {
	q := queue.New("testcoinschecktx")
	q.SetConfig(checkTxCfg)
	api, err := client.New(q.Client(), nil)
	require.Nil(t, err)
	c := newCoins()
	c.SetAPI(api)
	return c.(*Coins)
}

func signCheckTx(priv crypto.PrivKey, to string, action *dtypes.CoinsAction) *types.Transaction {
	tx := &types.Transaction{
		Execer:  []byte("coins"),
		Payload: types.Encode(action),
		To:      to,
		Fee:     int64(1e6),
	}
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func transferCheckTxAction(amount int64) *dtypes.CoinsAction {
	return &dtypes.CoinsAction{
		Ty:    dtypes.CoinsActionTransfer,
		Value: &dtypes.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amount}},
	}
}

// TestCheckTxZeroAmountTransfer 零金额转账必须在 CheckTx 层即被拒绝
func TestCheckTxZeroAmountTransfer(t *testing.T) {
	c := initCheckTxCoins(t)
	_, priv := util.Genaddress()
	to, _ := util.Genaddress()

	tx := signCheckTx(priv, to, transferCheckTxAction(0))
	err := c.CheckTx(tx, 0)
	require.Equal(t, types.ErrAmount, err)
}

// TestCheckTxNegativeAmountTransfer 负金额转账仍被 CheckTx 拒绝
func TestCheckTxNegativeAmountTransfer(t *testing.T) {
	c := initCheckTxCoins(t)
	_, priv := util.Genaddress()
	to, _ := util.Genaddress()

	tx := signCheckTx(priv, to, transferCheckTxAction(-1))
	err := c.CheckTx(tx, 0)
	require.Equal(t, types.ErrAmount, err)
}

// TestCheckTxNormalActionUnaffected 正常金额的 transfer/withdraw 不受 CheckTx 影响
func TestCheckTxNormalActionUnaffected(t *testing.T) {
	c := initCheckTxCoins(t)
	_, priv := util.Genaddress()
	to, _ := util.Genaddress()

	transferTx := signCheckTx(priv, to, transferCheckTxAction(1))
	require.Nil(t, c.CheckTx(transferTx, 0))

	withdrawAction := &dtypes.CoinsAction{
		Ty: dtypes.CoinsActionWithdraw,
		Value: &dtypes.CoinsAction_Withdraw{
			Withdraw: &types.AssetsWithdraw{ExecName: "ticket", Amount: 1},
		},
	}
	withdrawTx := signCheckTx(priv, to, withdrawAction)
	require.Nil(t, c.CheckTx(withdrawTx, 0))
}
