// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/rpc/ethrpc"
	"golang.org/x/net/websocket"

	"github.com/golang/protobuf/proto"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/common"
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

func TestCheckBasicAuth(t *testing.T) {
	rpcCfg = new(types.RPC)
	var r = &http.Request{Header: make(http.Header)}
	assert.True(t, checkBasicAuth(r))
	r.SetBasicAuth("1212121", "chain33-mypasswd")
	assert.True(t, checkBasicAuth(r))
	rpcCfg.JrpcUserName = "chain33-user"
	rpcCfg.JrpcUserPasswd = "chain33-mypasswd"
	r.SetBasicAuth("", "chain33-mypasswd")
	assert.False(t, checkBasicAuth(r))
	r.SetBasicAuth("", "")
	assert.False(t, checkBasicAuth(r))
	r.SetBasicAuth("chain33-user", "")
	assert.False(t, checkBasicAuth(r))
	r.SetBasicAuth("chain33", "1234")
	assert.False(t, checkBasicAuth(r))
	r.SetBasicAuth("chain33-user", "chain33-mypasswd")
	assert.True(t, checkBasicAuth(r))

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
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	api.On("GetConfig", mock.Anything).Return(cfg)
	qm := &qmocks.Client{}
	qm.On("GetConfig", mock.Anything).Return(cfg)
	server := NewJSONRPCServer(qm, api)
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
	api.On("ExecWalletFunc", "wallet", "SignRawTx", mock.Anything).Return(&types.ReplySignRawTx{TxHex: "123"}, nil)
	err = jsonClient.Call("Chain33.SignRawTx", &types.ReqSignRawTx{}, &singRet)
	assert.Nil(t, err)
	assert.Equal(t, "123", singRet)

	var fee types.TotalFee
	api.On("LocalGet", mock.Anything).Return(nil, errors.New("error value"))
	err = jsonClient.Call("Chain33.QueryTotalFee", &types.LocalDBGet{Keys: [][]byte{[]byte("test")}}, &fee)
	assert.NotNil(t, err)

	var retNtp bool
	api.On("IsNtpClockSync", mock.Anything).Return(&types.Reply{IsOk: true, Msg: []byte("yes")}, nil)
	err = jsonClient.Call("Chain33.IsNtpClockSync", &types.ReqNil{}, &retNtp)
	assert.Nil(t, err)
	assert.True(t, retNtp)
	api.On("GetProperFee", mock.Anything).Return(&types.ReplyProperFee{ProperFee: 2}, nil)
	testCreateTxCoins(t, cfg, jsonClient)
	server.Close()
	mock.AssertExpectationsForObjects(t, api)
}

func testDecodeTxHex(t *testing.T, txHex string) *types.Transaction {
	txbytes, err := common.FromHex(txHex)
	assert.Nil(t, err)
	var tx types.Transaction
	err = types.Decode(txbytes, &tx)
	assert.Nil(t, err)
	return &tx
}

func testCreateTxCoins(t *testing.T, cfg *types.Chain33Config, jsonClient *jsonclient.JSONClient) {
	req := &rpctypes.CreateTx{
		To:          "184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2",
		Amount:      10,
		Fee:         1,
		Note:        "12312",
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    cfg.ExecName("coins"),
	}
	var res string
	err := jsonClient.Call("Chain33.CreateRawTransaction", req, &res)
	assert.Nil(t, err)
	tx := testDecodeTxHex(t, res)
	assert.Equal(t, "184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2", tx.To)
	assert.Equal(t, int64(1), tx.Fee)
	req.Fee = 0
	err = jsonClient.Call("Chain33.CreateRawTransaction", req, &res)
	assert.Nil(t, err)
	tx = testDecodeTxHex(t, res)
	fee, _ := tx.GetRealFee(2)
	assert.Equal(t, fee, tx.Fee)
}

func TestEthRpc_Subscribe(t *testing.T) {

	rpcCfg := new(types.RPC)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8101"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8200"
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	InitCfg(rpcCfg)

	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	subcfg := cfg.GetSubConfig()
	sub, _ := types.ModifySubConfig(subcfg.RPC["eth"], "enable", true)
	subcfg.RPC["eth"] = sub
	cfg.GetModuleConfig().RPC = rpcCfg
	api := new(mocks.QueueProtocolAPI)
	q := queue.New("test")
	q.SetConfig(cfg)

	qcli := q.Client()
	server := NewGRpcServer(qcli, api)
	assert.NotNil(t, server)
	rpc := new(RPC)
	rpc.cfg = rpcCfg
	rpc.gapi = server
	rpc.cli = qcli
	rpc.api = api
	go rpc.handleSysEvent()
	go server.Listen()
	defer rpc.gapi.Close()

	wsServer := ethrpc.NewHTTPServer(qcli, api)
	wsServer.EnableWS()
	go wsServer.Start()

	api.On("GetConfig", mock.Anything).Return(cfg)
	api.On("AddPushSubscribe", mock.Anything).Return(&types.ReplySubscribePush{IsOk: true}, nil)
	api.On("Close", mock.Anything).Return()

	time.Sleep(time.Millisecond * 500)
	//websocket client
	ws, err := websocket.Dial("ws://localhost:8546", "", "http://localhost:8546")
	assert.Nil(t, err)
	ws.Write([]byte(`{"jsonrpc":"2.0", "id": 1, "method": "eth_subscribe", "params": ["newHeads"]}`))
	var data string
	err = websocket.Message.Receive(ws, &data)
	assert.Nil(t, err)
	var subID struct {
		Result string `json:"result,omitempty"`
	}
	err = json.Unmarshal([]byte(data), &subID)
	assert.Nil(t, err)
	require.Truef(t, len(subID.Result) > 0, data)
	t.Log("subid", subID.Result)
	time.Sleep(time.Second)
	err = q.Client().Send(qcli.NewMessage("rpc", types.EventPushBlock, &types.PushData{Name: subID.Result, Value: &types.PushData_HeaderSeqs{
		HeaderSeqs: &types.HeaderSeqs{
			Seqs: []*types.HeaderSeq{
				{
					Num: 1,
					Header: &types.Header{
						Height: 1024,
					},
				},
			},
		},
	}}), false)
	assert.Nil(t, err)
	err = websocket.Message.Receive(ws, &data)
	assert.Nil(t, err)
	t.Log("data", data)

	//test evm logs

	ws, err = websocket.Dial("ws://localhost:8546", "", "http://localhost:8546")
	assert.Nil(t, err)
	ws.Write([]byte(`{"jsonrpc":"2.0", "id": 1, "method": "eth_subscribe", "params": ["logs",{"address":"1JX6b8qpVFZ4FPqP4KT2HRTjYJrzRZGw7t"}]}`))
	err = websocket.Message.Receive(ws, &data)
	assert.Nil(t, err)

	err = json.Unmarshal([]byte(data), &subID)
	assert.Nil(t, err)
	t.Log("subid", subID.Result)
	time.Sleep(time.Second)
	topic, _ := common.FromHex("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	var topics [][]byte
	topics = append(topics, topic)
	var evmlog types.EVMLog
	evmlog.Topic = topics
	sendMsg := qcli.NewMessage("rpc", types.EventPushBlock, &types.PushData{Name: subID.Result, Value: &types.PushData_EvmLogs{
		EvmLogs: &types.EVMTxLogsInBlks{
			Logs4EVMPerBlk: []*types.EVMTxLogPerBlk{
				{
					Height: 1024,
					TxAndLogs: []*types.EVMTxAndLogs{
						{
							Tx: &types.Transaction{
								To: "1JX6b8qpVFZ4FPqP4KT2HRTjYJrzRZGw7t",
							},
							LogsPerTx: &types.EVMLogsPerTx{
								Logs: []*types.EVMLog{
									&evmlog,
								},
							},
						},
					},
				},
			},
		}},
	})
	err = q.Client().Send(sendMsg, false)
	assert.Nil(t, err)
	err = websocket.Message.Receive(ws, &data)
	assert.Nil(t, err)
	t.Log("data", data)
	wsServer.Close()
	ws.Close()
}

func TestGrpc_Call(t *testing.T) {
	rpcCfg := new(types.RPC)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8101"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8200"
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	InitCfg(rpcCfg)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	api := new(mocks.QueueProtocolAPI)
	api.On("GetConfig", mock.Anything).Return(cfg)
	_ = NewGrpcServer()
	qm := &qmocks.Client{}
	qm.On("GetConfig", mock.Anything).Return(cfg)
	server := NewGRpcServer(qm, api)
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
	assert.True(t, proto.Equal(&types.BlockSeq{}, blockSeq))

	server.Close()
	mock.AssertExpectationsForObjects(t, api)
}

func TestRPC(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	rpcCfg := cfg.GetModuleConfig().RPC
	rpcCfg.JrpcBindAddr = "localhost:8801"
	rpcCfg.GrpcBindAddr = "localhost:8802"
	rpcCfg.Whitlist = []string{"127.0.0.1"}
	rpcCfg.JrpcFuncBlacklist = []string{"CloseQueue"}
	rpcCfg.GrpcFuncBlacklist = []string{"CloseQueue"}
	rpcCfg.EnableTrace = true
	InitCfg(rpcCfg)
	rpc := New(cfg)
	client := &qmocks.Client{}
	client.On("GetConfig", mock.Anything).Return(cfg)
	client.On("Sub", mock.Anything).Return(mock.Anything)
	var ret chan *queue.Message
	client.On("Recv", mock.Anything).Return(ret)
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
	assert.False(t, checkGrpcFuncValidity(funcName))
	grpcFuncWhitelist["*"] = true
	assert.True(t, checkGrpcFuncValidity(funcName))

	delete(grpcFuncWhitelist, "*")
	grpcFuncWhitelist[funcName] = true
	assert.True(t, checkGrpcFuncValidity(funcName))

	jrpcFuncBlacklist = make(map[string]bool)
	assert.False(t, checkJrpcFuncBlacklist(funcName))
	jrpcFuncBlacklist[funcName] = true
	assert.True(t, checkJrpcFuncBlacklist(funcName))

	grpcFuncBlacklist = make(map[string]bool)
	assert.True(t, checkGrpcFuncValidity(funcName))
	grpcFuncBlacklist[funcName] = true
	assert.False(t, checkGrpcFuncValidity(funcName))

}
