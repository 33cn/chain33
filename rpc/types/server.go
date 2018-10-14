package types

import (
	"net/rpc"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/queue"
	"google.golang.org/grpc"
)

type RPCServer interface {
	GetQueueClient() queue.Client
	GRPC() *grpc.Server
	JRPC() *rpc.Server
}

type ChannelClient struct {
	client.QueueProtocolAPI
	accountdb *account.DB
	grpc      interface{}
	jrpc      interface{}
}

func (c *ChannelClient) Init(name string, s RPCServer, jrpc, grpc interface{}) {
	if c.QueueProtocolAPI == nil {
		c.QueueProtocolAPI, _ = client.New(s.GetQueueClient(), nil)
	}
	if jrpc != nil {
		s.JRPC().RegisterName(name, jrpc)
	}
	c.grpc = grpc
	c.jrpc = jrpc
	c.accountdb = account.NewCoinsAccount()
}

func (c *ChannelClient) GetCoinsAccountDB() *account.DB {
	return c.accountdb
}
