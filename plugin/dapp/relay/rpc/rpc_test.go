package rpc

import (
	"gitlab.33.cn/chain33/chain33/client/mocks"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
)

func newTestChannelClient() *channelClient {
	api := &mocks.QueueProtocolAPI{}
	return &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
	}
}

func newTestJrpcClient() *Jrpc {
	return &Jrpc{cli: newTestChannelClient()}
}
