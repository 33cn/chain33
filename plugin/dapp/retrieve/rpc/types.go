package rpc

import (
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var jrpc = &Jrpc{}
var grpc = &Grpc{}

func InitRPC(s pluginmgr.RPCServer) {
	cli := channelClient{}
	cli.Init(s.GetQueueClient())
	jrpc.cli = cli
	grpc.channelClient = cli
	s.JRPC().RegisterName(rt.JRPCName, jrpc)
	rt.RegisterRetrieveServer(s.GRPC(), grpc)
}

func Init(name string, s pluginmgr.RPCServer) {
	name = rt.RetrieveX
	// init executor type
	types.RegistorExecutor(name, rt.NewType())

	InitRPC(s)
}
