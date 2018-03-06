package rpc

import (
	"io"
	"net"

	"code.aliyun.com/chain33/chain33/queue"
)

type Server interface {
	Listen(addr string)
	io.Closer
}

type rpcServer struct {
	cli channelClient
	net.Listener
}

func (s *rpcServer) Close() error {
	return s.Listener.Close()
}

func NewGRpcServer(q *queue.Queue) Server {
	s := rpcServer{}
	s.cli.Client = q.NewClient()
	return (*Grpc)(&s)
}

func NewJsonRpcServer(q *queue.Queue) Server {
	s := rpcServer{}
	s.cli.Client = q.NewClient()
	return (*Chain33)(&s)
}
