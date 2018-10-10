package rpc

import (
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

var jrpc = &Jrpc{}

func InitRPC(s pluginmgr.RPCServer) {
	cli := channelClient{}
	cli.Init(s.GetQueueClient())
	jrpc.cli = cli
	s.JRPC().RegisterName(evmtypes.JRPCName, jrpc)
}

func Init(s pluginmgr.RPCServer) {
	InitRPC(s)
}
