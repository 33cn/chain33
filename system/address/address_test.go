// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address_test

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/33cn/chain33/common/address"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/system/address/btc"
	"github.com/33cn/chain33/system/address/eth"
	"github.com/33cn/chain33/system/crypto/secp256k1"
	ctypes "github.com/33cn/chain33/system/dapp/coins/types"
	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/require"
)

func TestMultiAddrAssetTransfer(t *testing.T) {

	cfg := testnode.GetDefaultConfig()
	cfg.GetModuleConfig().Address.EnableHeight["eth"] = 0
	cfg.GetModuleConfig().Mempool.MinTxFeeRate = 0
	cfg.SetMinFee(0)
	mock33 := testnode.NewWithConfig(cfg, nil)
	defer mock33.Close()
	ethDriver, err := address.LoadDriver(eth.ID, -1)
	require.Nil(t, err)

	btcAddr, priv := util.Genaddress()
	ethAddr := ethDriver.PubKeyToAddr(priv.PubKey().Bytes())
	ethAddrChecksumAddr := common.HexToAddress(ethAddr).String()
	height := mock33.GetLastBlock().Height
	mock33.SendTx(util.CreateCoinsTx(cfg, mock33.GetGenesisKey(), ethAddr, 500))
	require.Nil(t, mock33.WaitHeight(height+1))
	mock33.SendTx(util.CreateCoinsTx(cfg, mock33.GetGenesisKey(), ethAddrChecksumAddr, 500))
	require.Nil(t, mock33.WaitHeight(height+2))
	acc := mock33.GetAccount(mock33.GetLastBlock().StateHash, ethAddrChecksumAddr)

	require.Equal(t, ethAddr, acc.Addr)
	require.Equal(t, int64(1000), acc.GetBalance())

	tx := util.CreateCoinsTx(cfg, priv, btcAddr, 100)
	tx.Signature.Ty = types.EncodeSignID(secp256k1.ID, eth.ID)
	mock33.SendTx(tx)
	require.Nil(t, mock33.WaitHeight(height+3))
	acc = mock33.GetAccount(mock33.GetLastBlock().StateHash, btcAddr)
	require.Equal(t, btcAddr, acc.Addr)
	require.Equal(t, int64(100), acc.GetBalance())

	execAddr, err := commandtypes.GetExecAddr("none", btc.NormalAddressID)
	require.Nil(t, err)
	action := &ctypes.CoinsAction{
		Ty: ctypes.CoinsActionTransferToExec,
		Value: &ctypes.CoinsAction_TransferToExec{
			TransferToExec: &types.AssetsTransferToExec{
				ExecName: "none",
				To:       execAddr,
				Amount:   100,
			},
		},
	}

	tx = &types.Transaction{Payload: types.Encode(action), Execer: []byte("coins"), To: execAddr}
	tx.Sign(types.EncodeSignID(secp256k1.ID, eth.ID), priv)
	mock33.SendTx(tx)

	require.Nil(t, mock33.WaitHeight(height+4))
	acc = mock33.GetAccount(mock33.GetLastBlock().StateHash, ethAddr)
	require.Equal(t, ethAddr, acc.Addr)
	require.Equal(t, int64(800), acc.GetBalance())
	acc = mock33.GetExecAccount(mock33.GetLastBlock().StateHash, "none", ethAddr)
	require.Equal(t, ethAddr, acc.Addr)
	require.Equal(t, int64(100), acc.GetBalance())

	action = &ctypes.CoinsAction{
		Ty: ctypes.CoinsActionWithdraw,
		Value: &ctypes.CoinsAction_Withdraw{
			Withdraw: &types.AssetsWithdraw{
				ExecName: "none",
				To:       execAddr,
				Amount:   100,
			},
		},
	}

	tx = &types.Transaction{Payload: types.Encode(action), Execer: []byte("coins"), To: execAddr}
	tx.Sign(types.EncodeSignID(secp256k1.ID, eth.ID), priv)
	mock33.SendTx(tx)

	require.Nil(t, mock33.WaitHeight(height+5))
	acc = mock33.GetAccount(mock33.GetLastBlock().StateHash, ethAddr)
	require.Equal(t, ethAddr, acc.Addr)
	require.Equal(t, int64(900), acc.GetBalance())
	acc = mock33.GetExecAccount(mock33.GetLastBlock().StateHash, "none", ethAddr)
	require.Equal(t, ethAddr, acc.Addr)
	require.Equal(t, int64(0), acc.GetBalance())

}
