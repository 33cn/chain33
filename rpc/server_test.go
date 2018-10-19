package rpc

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	qmocks "gitlab.33.cn/chain33/chain33/queue/mocks"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestCheckIpWhitelist(t *testing.T) {
	address := "127.0.0.1"
	assert.True(t, checkIpWhitelist(address))

	address = "::1"
	assert.True(t, checkIpWhitelist(address))

	address = "192.168.3.1"
	remoteIpWhitelist[address] = true
	assert.False(t, checkIpWhitelist("192.168.3.2"))

	remoteIpWhitelist["0.0.0.0"] = true
	assert.True(t, checkIpWhitelist(address))
	assert.True(t, checkIpWhitelist("192.168.3.2"))

}

func TestJSONClient_Call(t *testing.T) {
	rpcCfg = new(types.Rpc)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8101"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8200"
	rpcCfg.MainnetJrpcAddr = rpcCfg.JrpcBindAddr
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	InitCfg(rpcCfg)
	server := NewJSONRPCServer(&qmocks.Client{}, nil)
	assert.NotNil(t, server)

	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)
	assert.NotNil(t, testChain33)
	server.jrpc = *testChain33
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

	err = jsonClient.Call("Chain33.Version", nil, &result)
	assert.Nil(t, err)
	assert.NotEmpty(t, result)

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

	var ticket []types.TicketMinerInfo
	api.On("LocalList", mock.Anything).Return(nil, errors.New("error value"))
	err = jsonClient.Call("Chain33.QueryTicketInfoList", &types.ReqSignRawTx{}, &ticket)
	assert.NotNil(t, err)

	var retNtp bool
	api.On("IsNtpClockSync", mock.Anything).Return(&types.Reply{IsOk: true, Msg: []byte("yes")}, nil)
	err = jsonClient.Call("Chain33.IsNtpClockSync", &types.ReqNil{}, &retNtp)
	assert.Nil(t, err)
	assert.True(t, retNtp)

	server.Close()
	mock.AssertExpectationsForObjects(t, api)
}
func TestGrpc_Call(t *testing.T) {
	rpcCfg = new(types.Rpc)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8101"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8200"
	rpcCfg.MainnetJrpcAddr = rpcCfg.JrpcBindAddr
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	InitCfg(rpcCfg)
	server := NewGRpcServer(&qmocks.Client{}, nil)
	assert.NotNil(t, server)

	api := new(mocks.QueueProtocolAPI)
	server.grpc.cli.QueueProtocolAPI = api
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
	assert.Equal(t, ret, result)

	server.Close()
	mock.AssertExpectationsForObjects(t, api)
}
