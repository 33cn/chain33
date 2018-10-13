package pluginmgr

import (
	"net/rpc"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/queue"
	"google.golang.org/grpc"

	"github.com/spf13/cobra"
)

type RPCServer interface {
	GetQueueClient() queue.Client
	GRPC() *grpc.Server
	JRPC() *rpc.Server
}

type ChannelClient struct {
	client.QueueProtocolAPI
	grpc interface{}
	jrpc interface{}
}

func (c *ChannelClient) Init(name string, s RPCServer, jrpc, grpc interface{}) {
	c.QueueProtocolAPI, _ = client.New(s.GetQueueClient(), nil)
	c.grpc = grpc
	c.jrpc = jrpc
	if jrpc != nil {
		println(name, jrpc)
		s.JRPC().RegisterName(name, jrpc)
	}
}

//
type Plugin interface {
	// 获取整个插件的包名，用以计算唯一值、做前缀等
	GetName() string
	// 获取插件中执行器名
	GetExecutorName() string
	// 初始化执行器时会调用该接口
	InitExec()
	AddCmd(rootCmd *cobra.Command)
	AddRPC(s RPCServer)
}
