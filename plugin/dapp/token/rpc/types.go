package rpc

import (
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

var log = log15.New("module", "token.rpc")

type Jrpc struct {
	cli *channelClient
}

type Grpc struct {
	*channelClient
}

type channelClient struct {
	pluginmgr.ChannelClient
}

func Init(name string, s pluginmgr.RPCServer) {
	cli := &channelClient{}
	grpc := &Grpc{channelClient: cli}
	cli.Init(name, s, &Jrpc{cli: cli}, grpc)
}
