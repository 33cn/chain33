package rpc

import (
	"net"

	"code.aliyun.com/chain33/chain33/queue"
)

func NewServer(name string, addr string, q *queue.Queue) IServer {
	if name == "channel" {
		return newChannelServer(q)
	} else if name == "grpc" {
		return newGrpcServer(addr, q)
	} else if name == "jsonrpc" {
		return newJsonrpcServer(addr, q)
	}
	panic("server name not support")
}

type IServer interface {
	SetQueue(q *queue.Queue)
	GetQueue() *queue.Queue
	Close()
}

//channelServer 不需要做任何的事情，grpc 和 jsonrpc 需要建立服务，监听
type channelServer struct {
	q        *queue.Queue
	c        queue.Client
	listener net.Listener
}

func newChannelServer(q *queue.Queue) *channelServer {
	return &channelServer{q: q}
}

func (server *channelServer) SetQueue(q *queue.Queue) {
	server.q = q
	server.c = q.NewClient() //创建一个Queue Client

}

func (server *channelServer) GetQueue() *queue.Queue {
	return server.q
}

func (server *channelServer) Close() {
	server.listener.Close()
}

type grpcServer struct {
	channelServer
}

type jsonrpcServer struct {
	channelServer
}

func newGrpcServer(addr string, q *queue.Queue) *grpcServer {
	server := &grpcServer{}
	server.q = q
	server.CreateServer(addr)
	return server
}

func (r *grpcServer) Close() {
	r.listener.Close()
}

func newJsonrpcServer(addr string, q *queue.Queue) *jsonrpcServer {
	server := &jsonrpcServer{}
	server.q = q
	server.CreateServer(addr)
	return server
}
