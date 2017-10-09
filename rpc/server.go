package rpc

import "code.aliyun.com/chain33/chain33/queue"

//rpc server for user
//支持三种服务, channel, grpc, jsonrpc
//对 channel来说 实际上是直接发送模式，不需要网络发送
//grpc 获取到数据后，也会以channel 的模式发送数据，然后以grpc的格式返回
//jsonrpc 也是一样的，只是数据的格式有区别

func New(name string) IServer {
	if name == "channel" {
		return newChannelServer()
	} else if name == "grpc" {
		return newGrpcServer()
	} else if name == "jsonrpc" {
		return newJsonrpcServer()
	}
	panic("server name not support")
}

type IServer interface {
	SetQueue(q *queue.Queue)
}

type channelServer struct{}

func newChannelServer() *channelServer {
	return &channelServer{}
}

func (server *channelServer) SetQueue(q *queue.Queue) {

}

type grpcServer struct {
	channelServer
}

type jsonrpcServer struct {
	channelServer
}

func newGrpcServer() *grpcServer {
	return &grpcServer{}
}

func newJsonrpcServer() *jsonrpcServer {
	return &jsonrpcServer{}
}
