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
	server channelClient
	net.Listener
}

func (s *rpcServer) Close() error {
	s.server.Close()
	return s.Listener.Close()
}

func NewGrpcServer(c queue.Client) *Grpc {
	s := rpcServer{}
	s.server.Client = c
	return (*Grpc)(&s)
}

func NewJsonRpcServer(c queue.Client) *Chain33 {
	s := rpcServer{}
	s.server.Client = c
	return (*Chain33)(&s)
}
