// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/33cn/chain33/client/mocks"
	qmocks "github.com/33cn/chain33/queue/mocks"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestCheckIpWhitelist(t *testing.T) {
	address := "127.0.0.1"
	assert.True(t, checkIPWhitelist(address))

	address = "::1"
	assert.True(t, checkIPWhitelist(address))

	address = "192.168.3.1"
	remoteIPWhitelist[address] = true
	assert.False(t, checkIPWhitelist("192.168.3.2"))

	remoteIPWhitelist["0.0.0.0"] = true
	assert.True(t, checkIPWhitelist(address))
	assert.True(t, checkIPWhitelist("192.168.3.2"))

}

func TestJSONClient_Call(t *testing.T) {
	rpcCfg = new(types.RPC)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8101"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8200"
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	InitCfg(rpcCfg)
	api := new(mocks.QueueProtocolAPI)
	server := NewJSONRPCServer(&qmocks.Client{}, api)
	assert.NotNil(t, server)
	done := make(chan struct{}, 1)
	go func() {
		done <- struct{}{}
		server.Listen()
	}()
	<-done
	time.Sleep(time.Millisecond)
	ret := &types.Reply{
		IsOk: true,
		Msg:  []byte("123"),
	}
	api.On("IsSync").Return(ret, nil)
	api.On("Close").Return()

	jsonClient, err := jsonclient.NewJSONClient("http://" + rpcCfg.JrpcBindAddr + "/root")
	assert.Nil(t, err)
	assert.NotNil(t, jsonClient)

	var result = ""
	err = jsonClient.Call("Chain33.Version", nil, &result)
	assert.NotNil(t, err)
	assert.Empty(t, result)

	jsonClient, err = jsonclient.NewJSONClient("http://" + rpcCfg.JrpcBindAddr)
	assert.Nil(t, err)
	assert.NotNil(t, jsonClient)

	ver := &types.VersionInfo{Chain33: "6.0.2"}
	api.On("Version").Return(ver, nil)
	var nodeVersion types.VersionInfo
	err = jsonClient.Call("Chain33.Version", nil, &nodeVersion)
	assert.Nil(t, err)
	assert.Equal(t, "6.0.2", nodeVersion.Chain33)

	var isSnyc bool
	err = jsonClient.Call("Chain33.IsSync", &types.ReqNil{}, &isSnyc)
	assert.Nil(t, err)
	assert.Equal(t, ret.GetIsOk(), isSnyc)
	var nodeInfo rpctypes.NodeNetinfo
	api.On("GetNetInfo", mock.Anything).Return(&types.NodeNetInfo{Externaladdr: "123"}, nil)
	err = jsonClient.Call("Chain33.GetNetInfo", &types.ReqNil{}, &nodeInfo)
	assert.Nil(t, err)
	assert.Equal(t, "123", nodeInfo.Externaladdr)

	var singRet = ""
	api.On("SignRawTx", mock.Anything).Return(&types.ReplySignRawTx{TxHex: "123"}, nil)
	err = jsonClient.Call("Chain33.SignRawTx", &types.ReqSignRawTx{}, &singRet)
	assert.Nil(t, err)
	assert.Equal(t, "123", singRet)

	var fee types.TotalFee
	api.On("LocalGet", mock.Anything).Return(nil, errors.New("error value"))
	err = jsonClient.Call("Chain33.QueryTotalFee", &types.ReqSignRawTx{}, &fee)
	assert.NotNil(t, err)

	var retNtp bool
	api.On("IsNtpClockSync", mock.Anything).Return(&types.Reply{IsOk: true, Msg: []byte("yes")}, nil)
	err = jsonClient.Call("Chain33.IsNtpClockSync", &types.ReqNil{}, &retNtp)
	assert.Nil(t, err)
	assert.True(t, retNtp)
	testCreateTxCoins(t, jsonClient)
	server.Close()
	mock.AssertExpectationsForObjects(t, api)
}

func testCreateTxCoins(t *testing.T, jsonClient *jsonclient.JSONClient) {
	req := &rpctypes.CreateTx{
		To:          "184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2",
		Amount:      10,
		Fee:         1,
		Note:        "12312",
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    types.ExecName("coins"),
	}
	var res string
	err := jsonClient.Call("Chain33.CreateRawTransaction", req, &res)
	assert.Nil(t, err)
	txbytes, err := hex.DecodeString(res)
	assert.Nil(t, err)
	var tx types.Transaction
	err = types.Decode(txbytes, &tx)
	assert.Nil(t, err)
	assert.Equal(t, "184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2", tx.To)
}

func TestGrpc_Call(t *testing.T) {
	rpcCfg = new(types.RPC)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8101"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8200"
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	InitCfg(rpcCfg)
	api := new(mocks.QueueProtocolAPI)
	_ = NewGrpcServer()
	server := NewGRpcServer(&qmocks.Client{}, api)
	assert.NotNil(t, server)
	go server.Listen()
	time.Sleep(time.Second)
	ret := &types.Reply{
		IsOk: true,
		Msg:  []byte("123"),
	}
	api.On("IsSync").Return(ret, nil)
	api.On("Close").Return()

	ctx := context.Background()
	c, err := grpc.DialContext(ctx, rpcCfg.GrpcBindAddr, grpc.WithInsecure())
	assert.Nil(t, err)
	assert.NotNil(t, c)

	client := types.NewChain33Client(c)
	result, err := client.IsSync(ctx, &types.ReqNil{})

	assert.Nil(t, err)
	assert.Equal(t, ret.IsOk, result.IsOk)
	assert.Equal(t, ret.Msg, result.Msg)

	rst, err := client.GetFork(ctx, &types.ReqKey{Key: []byte("ForkBlockHash")})
	assert.Nil(t, err)
	assert.Equal(t, int64(1), rst.Data)

	api.On("GetBlockBySeq", mock.Anything).Return(&types.BlockSeq{}, nil)
	blockSeq, err := client.GetBlockBySeq(ctx, &types.Int64{Data: 1})
	assert.Nil(t, err)
	assert.Equal(t, &types.BlockSeq{}, blockSeq)

	server.Close()
	mock.AssertExpectationsForObjects(t, api)
}

func TestRPC(t *testing.T) {
	cfg := &types.RPC{
		JrpcBindAddr:      "8801",
		GrpcBindAddr:      "8802",
		Whitlist:          []string{"127.0.0.1"},
		JrpcFuncBlacklist: []string{"CloseQueue"},
		GrpcFuncBlacklist: []string{"CloseQueue"},
		EnableTrace:       true,
	}
	InitCfg(cfg)
	rpc := New(cfg)
	client := &qmocks.Client{}
	rpc.SetQueueClient(client)

	assert.Equal(t, client, rpc.GetQueueClient())
	assert.NotNil(t, rpc.GRPC())
	assert.NotNil(t, rpc.JRPC())
}

func TestCheckFuncList(t *testing.T) {
	funcName := "abc"
	jrpcFuncWhitelist = make(map[string]bool)
	assert.False(t, checkJrpcFuncWhitelist(funcName))
	jrpcFuncWhitelist["*"] = true
	assert.True(t, checkJrpcFuncWhitelist(funcName))

	delete(jrpcFuncWhitelist, "*")
	jrpcFuncWhitelist[funcName] = true
	assert.True(t, checkJrpcFuncWhitelist(funcName))

	grpcFuncWhitelist = make(map[string]bool)
	assert.False(t, checkGrpcFuncWhitelist(funcName))
	grpcFuncWhitelist["*"] = true
	assert.True(t, checkGrpcFuncWhitelist(funcName))

	delete(grpcFuncWhitelist, "*")
	grpcFuncWhitelist[funcName] = true
	assert.True(t, checkGrpcFuncWhitelist(funcName))

	jrpcFuncBlacklist = make(map[string]bool)
	assert.False(t, checkJrpcFuncBlacklist(funcName))
	jrpcFuncBlacklist[funcName] = true
	assert.True(t, checkJrpcFuncBlacklist(funcName))

	grpcFuncBlacklist = make(map[string]bool)
	assert.False(t, checkGrpcFuncBlacklist(funcName))
	grpcFuncBlacklist[funcName] = true
	assert.True(t, checkGrpcFuncBlacklist(funcName))

}
