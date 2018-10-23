package rpc

import (
	ptypes "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/rpc/types"
)

type channelClient struct {
	types.ChannelClient
}

type Jrpc struct {
	cli *channelClient
}

type Grpc struct {
	*channelClient
}

func Init(name string, s types.RPCServer) {
	cli := &channelClient{}
	grpc := &Grpc{channelClient: cli}
	cli.Init(name, s, &Jrpc{cli: cli}, grpc)
	ptypes.RegisterTradeServer(s.GRPC(), grpc)
}
