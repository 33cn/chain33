package rpc

import (
	"code.aliyun.com/chain33/chain33/queue"
)

type server struct {
	cli channelClient
}

type Chain33 server
type Grpc server

type Grpcserver struct {
	grpc Grpc
}

type JsonRpcServer struct {
	jrpc Chain33
}

func (s *JsonRpcServer) Close() {
	s.jrpc.cli.Close()

}

func (j *Grpcserver) Close() {
	j.grpc.cli.Close()

}

func NewGRpcServer(client queue.Client) *Grpcserver {
	s := &Grpcserver{}
	s.grpc.cli.Client = client
	return s
}

func NewJsonRpcServer(client queue.Client) *JsonRpcServer {
	j := &JsonRpcServer{}
	j.jrpc.cli.Client = client
	return j
}
