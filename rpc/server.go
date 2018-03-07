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

type server struct {
	cli channelClient
}

type Chain33 server
type Grpc server

type Grpcserver struct {
	grpc Grpc
	net.Listener
}

type JsonRpcServer struct {
	jrpc Chain33
	net.Listener
}

func (s *JsonRpcServer) Close() error {
	s.jrpc.cli.Close()
	return s.Listener.Close()
}

func (j *Grpcserver) Close() error {
	j.grpc.cli.Close()
	return j.Listener.Close()
}

func NewGRpcServer(client queue.Client) Server {
	s := &Grpcserver{}
	s.grpc.cli.Client = client
	return s
}

func NewJsonRpcServer(client queue.Client) Server {
	j := &JsonRpcServer{}
	j.jrpc.cli.Client = client
	return j
}
