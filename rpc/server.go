package rpc

import (
	"queue"
)

func NewServer(name string, addr string) IServer {
	if name == "channel" {
		return newChannelServer()
	} else if name == "grpc" {
		return newGrpcServer(addr)
	} else if name == "jsonrpc" {
		return newJsonrpcServer(addr)
	}
	panic("server name not support")
}

type IServer interface {
	SetQueue(q *queue.Queue)
	GetQueue() *queue.Queue
}

//channelServer 不需要做任何的事情，grpc 和 jsonrpc 需要建立服务，监听
type channelServer struct {
	q *queue.Queue
	c queue.IClient
}

func newChannelServer() *channelServer {
	return &channelServer{}
}

func (server *channelServer) SetQueue(q *queue.Queue) {
	server.q = q
	server.c = q.GetClient() //创建一个Queue Client
	server.c.Sub("channel")  //订阅channal 的replay 消息

}
func (server *channelServer) GetQueue() *queue.Queue {

	return server.q
}

type grpcServer struct {
	channelServer
}

type jsonrpcServer struct {
	channelServer
}

func newGrpcServer(addr string) *grpcServer {
	return &grpcServer{}
}

func newJsonrpcServer(addr string) *jsonrpcServer {
	server := &jsonrpcServer{}
	server.CreateServer(addr)

	return server
}
