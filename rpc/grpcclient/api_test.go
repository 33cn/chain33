package grpcclient_test

import (
	"context"
	"testing"
	"time"

	"github.com/33cn/chain33/client/mocks"
	qmocks "github.com/33cn/chain33/queue/mocks"
	"github.com/33cn/chain33/rpc"
	"github.com/33cn/chain33/rpc/grpcclient"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

func TestMultipleGRPC(t *testing.T) {
	qapi := new(mocks.QueueProtocolAPI)
	qapi.On("Query", "ticket", "RandNumHash", mock.Anything).Return(&types.ReplyHash{Hash: []byte("hello")}, nil)
	//testnode setup
	rpcCfg := new(types.RPC)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8003"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8004"
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	rpc.InitCfg(rpcCfg)
	server := rpc.NewGRpcServer(&qmocks.Client{}, qapi)
	assert.NotNil(t, server)
	go server.Listen()
	time.Sleep(time.Second)
	//一个IP 有效，一个IP 无效
	paraRemoteGrpcClient := "127.0.0.1:8004,127.0.0.1:8003,127.0.0.1"
	conn, err := grpc.Dial(grpcclient.NewMultipleURL(paraRemoteGrpcClient), grpc.WithInsecure())
	assert.Nil(t, err)
	grpcClient := types.NewChain33Client(conn)
	param := &types.ReqRandHash{
		ExecName: "ticket",
		BlockNum: 5,
		Hash:     []byte("hello"),
	}
	reply, err := grpcClient.QueryRandNum(context.Background(), param)
	assert.Nil(t, err)
	assert.Equal(t, reply.Hash, []byte("hello"))
}

func TestMultipleGRPCLocalhost(t *testing.T) {
	qapi := new(mocks.QueueProtocolAPI)
	qapi.On("Query", "ticket", "RandNumHash", mock.Anything).Return(&types.ReplyHash{Hash: []byte("hello")}, nil)
	//testnode setup
	rpcCfg := new(types.RPC)
	rpcCfg.GrpcBindAddr = "localhost:8003"
	rpcCfg.JrpcBindAddr = "localhost:8004"
	rpcCfg.Whitelist = []string{"localhost", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	rpc.InitCfg(rpcCfg)
	server := rpc.NewGRpcServer(&qmocks.Client{}, qapi)
	assert.NotNil(t, server)
	go server.Listen()
	time.Sleep(time.Second)
	//一个IP 有效，一个IP 无效
	paraRemoteGrpcClient := "localhost:8004,localhost:8003,localhost"
	conn, err := grpc.Dial(grpcclient.NewMultipleURL(paraRemoteGrpcClient), grpc.WithInsecure())
	assert.Nil(t, err)
	grpcClient := types.NewChain33Client(conn)
	param := &types.ReqRandHash{
		ExecName: "ticket",
		BlockNum: 5,
		Hash:     []byte("hello"),
	}
	reply, err := grpcClient.QueryRandNum(context.Background(), param)
	assert.Nil(t, err)
	assert.Equal(t, reply.Hash, []byte("hello"))
}

func TestNewParaClient(t *testing.T) {
	qapi := new(mocks.QueueProtocolAPI)
	qapi.On("Query", "ticket", "RandNumHash", mock.Anything).Return(&types.ReplyHash{Hash: []byte("hello")}, nil)
	//testnode setup
	rpcCfg := new(types.RPC)
	rpcCfg.GrpcBindAddr = "127.0.0.1:8003"
	rpcCfg.JrpcBindAddr = "127.0.0.1:8004"
	rpcCfg.Whitelist = []string{"127.0.0.1", "0.0.0.0"}
	rpcCfg.JrpcFuncWhitelist = []string{"*"}
	rpcCfg.GrpcFuncWhitelist = []string{"*"}
	rpc.InitCfg(rpcCfg)
	server := rpc.NewGRpcServer(&qmocks.Client{}, qapi)
	assert.NotNil(t, server)
	go server.Listen()
	time.Sleep(time.Second)
	//一个IP 有效，一个IP 无效
	paraRemoteGrpcClient := "127.0.0.1:8004,127.0.0.1:8003,127.0.0.1"
	grpcClient, err := grpcclient.NewMainChainClient(paraRemoteGrpcClient)
	assert.Nil(t, err)
	param := &types.ReqRandHash{
		ExecName: "ticket",
		BlockNum: 5,
		Hash:     []byte("hello"),
	}
	reply, err := grpcClient.QueryRandNum(context.Background(), param)
	assert.Nil(t, err)
	assert.Equal(t, reply.Hash, []byte("hello"))
}

func TestNewMainChainClient(t *testing.T) {
	grpcClient1, err := grpcclient.NewMainChainClient("")
	assert.Nil(t, err)
	grpcClient2, err := grpcclient.NewMainChainClient("")
	assert.Nil(t, err)
	if grpcClient1 != grpcClient2 {
		t.Error("grpc client is the same")
	}

	grpcClient3, err := grpcclient.NewMainChainClient("127.0.0.1")
	assert.Nil(t, err)
	grpcClient4, err := grpcclient.NewMainChainClient("127.0.0.1")
	assert.Nil(t, err)
	if grpcClient3 == grpcClient4 {
		t.Error("grpc client is not the same")
	}
}
