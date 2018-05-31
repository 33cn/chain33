package rpc

import (
	"testing"

	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	qmocks "gitlab.33.cn/chain33/chain33/queue/mocks"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestCheckWhitlist(t *testing.T) {
	address := "0.0.0.0"
	assert.False(t, checkWhitlist(address))

	address = "192.168.3.1"
	whitlist[address] = true
	assert.False(t, checkWhitlist("192.168.3.2"))

	whitlist["0.0.0.0"] = true
	assert.True(t, checkWhitlist(address))
	assert.True(t, checkWhitlist("192.168.3.2"))

}

func TestJSONClient_Call(t *testing.T) {
	rpcCfg = new(types.Rpc)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8101"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8200"
	rpcCfg.Whitlist = []string{"127.0.0.1", "0.0.0.0"}
	Init(rpcCfg)
	server := NewJSONRPCServer(&qmocks.Client{})
	assert.NotNil(t, server)

	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)
	assert.NotNil(t, testChain33)
	server.jrpc = *testChain33
	go server.Listen()

	ret := &types.Reply{
		IsOk: true,
		Msg:  []byte("123"),
	}
	api.On("IsSync").Return(ret, nil)
	api.On("Close").Return()

	time.Sleep(100)
	jsonClient, err := NewJSONClient("http://" + rpcCfg.JrpcBindAddr + "/root")
	assert.Nil(t, err)
	assert.NotNil(t, jsonClient)
	var result = ""
	err = jsonClient.Call("Chain33.Version", nil, &result)
	assert.NotNil(t, err)
	assert.Empty(t, result)

	jsonClient, err = NewJSONClient("http://" + rpcCfg.JrpcBindAddr)
	assert.Nil(t, err)
	assert.NotNil(t, jsonClient)

	err = jsonClient.Call("Chain33.Version", nil, &result)
	assert.Nil(t, err)
	assert.NotEmpty(t, result)

	var isSnyc bool
	err = jsonClient.Call("Chain33.IsSync", &types.ReqNil{}, &isSnyc)
	assert.Nil(t, err)
	assert.Equal(t, ret.GetIsOk(), isSnyc)

	server.Close()
	mock.AssertExpectationsForObjects(t, api)
}
