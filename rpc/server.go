package rpc

import (
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

var (
	whitlist = make(map[string]bool)
	rpcCfg   *types.Rpc
)

type server struct {
	cli channelClient
}

type Chain33 server
type Grpc server

type Grpcserver struct {
	grpc Grpc
	addr string
}

type JsonRpcServer struct {
	jrpc Chain33
	addr string
}

func (s *JsonRpcServer) Close() {
	s.jrpc.cli.Close()

}
func checkWhitlist(addr string) bool {

	if _, ok := whitlist["0.0.0.0"]; ok {
		return true
	}

	if _, ok := whitlist[addr]; ok {
		return true
	}
	return false
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

func Init(cfg *types.Rpc) {
	rpcCfg = cfg
	if len(cfg.Whitlist) == 1 && cfg.Whitlist[0] == "*" {
		whitlist["0.0.0.0"] = true
		return
	}

	for _, addr := range cfg.Whitlist {
		whitlist[addr] = true
	}

}
