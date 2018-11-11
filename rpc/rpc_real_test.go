// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"

	_ "github.com/33cn/chain33/system"
)

func TestErrLog(t *testing.T) {
	// 启动RPCmocker
	mocker := testnode.New("--free--", nil)
	defer mocker.Close()
	mocker.Listen()

	rpcCfg := mocker.GetCfg().Rpc
	jrpcClient, err := jsonclient.NewJSONClient(fmt.Sprintf("http://%s/", rpcCfg.JrpcBindAddr))
	assert.NoError(t, err)
	assert.NotNil(t, jrpcClient)
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
