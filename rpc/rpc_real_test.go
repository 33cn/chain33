// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc_test

import (
	"fmt"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"

	_ "github.com/33cn/chain33/system"
)

func getRPCClient(t *testing.T, mocker *testnode.Chain33Mock) *jsonclient.JSONClient {
	jrpcClient := mocker.GetJSONC()
	assert.NotNil(t, jrpcClient)
	return jrpcClient
}

func TestErrLog(t *testing.T) {
	// 启动RPCmocker
	mocker := testnode.New("--free--", nil)
	defer mocker.Close()
	mocker.Listen()
	jrpcClient := getRPCClient(t, mocker)
	gen := mocker.GetGenesisKey()
	//发送交易到区块链
	addr1, key1 := util.Genaddress()
	addr2, _ := util.Genaddress()
	tx1 := util.CreateCoinsTx(gen, addr1, 1*types.Coin)
	mocker.GetAPI().SendTx(tx1)
	mocker.WaitHeight(1)

	tx11 := util.CreateCoinsTx(key1, addr2, 6*int64(1e7))
	reply, err := mocker.GetAPI().SendTx(tx11)
	assert.Nil(t, err)
	assert.Equal(t, reply.GetMsg(), tx11.Hash())
	tx12 := util.CreateCoinsTx(key1, addr2, 6*int64(1e7))
	reply, err = mocker.GetAPI().SendTx(tx12)
	assert.Nil(t, err)
	assert.Equal(t, reply.GetMsg(), tx12.Hash())
	mocker.WaitTx(reply.GetMsg())
	var testResult rpctypes.TransactionDetail
	req := rpctypes.QueryParm{
		Hash: common.ToHex(tx12.Hash()),
	}
	//query transaction
	err = jrpcClient.Call("Chain33.QueryTransaction", req, &testResult)
	assert.Nil(t, err)
	assert.Equal(t, string(testResult.Receipt.Logs[0].Log), `"ErrNoBalance"`)
}

func getTx(t *testing.T, hex string) *types.Transaction {
	data, err := common.FromHex(hex)
	assert.Nil(t, err)
	var tx types.Transaction
	err = types.Decode(data, &tx)
	assert.Nil(t, err)
	return &tx
}

func TestSendToExec(t *testing.T) {
	mocker := testnode.New("--free--", nil)
	defer mocker.Close()
	mocker.Listen()
	jrpcClient := getRPCClient(t, mocker)
	//1. 调用createrawtransaction 创建交易
	req := &rpctypes.CreateTx{
		To:          address.ExecAddress("user.f3d"),
		Amount:      10,
		Fee:         1,
		Note:        "12312",
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    "user.f3d",
	}
	var res string
	err := jrpcClient.Call("Chain33.CreateRawTransaction", req, &res)
	assert.Nil(t, err)
	gen := mocker.GetGenesisKey()
	tx := getTx(t, res)
	tx.Sign(types.SECP256K1, gen)
	reply, err := mocker.GetAPI().SendTx(tx)
	assert.Nil(t, err)
	_, err = mocker.WaitTx(reply.GetMsg())
	assert.Nil(t, err)
	block := mocker.GetLastBlock()
	balance := mocker.GetExecAccount(block.StateHash, "user.f3d", mocker.GetGenesisAddress()).Balance
	assert.Equal(t, int64(10), balance)
}

func TestGetAllExecBalance(t *testing.T) {
	mocker := testnode.New("--free--", nil)
	defer mocker.Close()
	mocker.Listen()
	jrpcClient := getRPCClient(t, mocker)

	addr := "38BRY193Wvy9MkdqMjmuaYeUHnJaFjUxMP"
	req := types.ReqAddr{Addr: addr}
	var res rpctypes.AllExecBalance
	err := jrpcClient.Call("Chain33.GetAllExecBalance", req, &res)
	assert.Nil(t, err)
	assert.Equal(t, addr, res.Addr)
	assert.Nil(t, res.ExecAccount)
	assert.Equal(t, 0, len(res.ExecAccount))
}

func TestCreateTransactionUserWrite(t *testing.T) {
	mocker := testnode.New("--free--", nil)
	defer mocker.Close()
	mocker.Listen()
	jrpcClient := getRPCClient(t, mocker)
	req := &rpctypes.CreateTxIn{
		Execer:     "user.write",
		ActionName: "write",
		Payload:    []byte(`{"key":"value"}`),
	}
	var res string
	err := jrpcClient.Call("Chain33.CreateTransaction", req, &res)
	assert.Nil(t, err)
	tx := getTx(t, res)
	assert.NotNil(t, tx)
	fmt.Println(string(tx.Payload))
	assert.Nil(t, err)
	assert.Equal(t, `{"key":"value"}`, string(tx.Payload))
}
