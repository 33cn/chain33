package rpc

import (
	rTy "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

var jrpc = &Jrpc{}

//var grpc = &Grpc{}

func InitRPC(s pluginmgr.RPCServer) {
	cli := channelClient{}
	cli.Init(s.GetQueueClient())
	jrpc.cli = cli
	//grpc.channelClient = cli
	s.JRPC().RegisterName(rTy.JRPCName, jrpc)

}

func Init(s pluginmgr.RPCServer) {
	InitRPC(s)
}
