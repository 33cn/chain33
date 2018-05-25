package rpc

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"

	// register gzip
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	whitlist = make(map[string]bool)
	rpcCfg   *types.Rpc
)

type Chain33 struct {
	cli channelClient
}

type Grpc struct {
	cli channelClient
}

type Grpcserver struct {
	grpc Grpc
	//addr string
}

type JSONRPCServer struct {
	jrpc Chain33
	//addr string
}

func (s *JSONRPCServer) Close() {
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

func NewGRpcServer(c queue.Client) *Grpcserver {
	s := &Grpcserver{}
	s.grpc.cli.Init(c)
	return s
}

func NewJSONRPCServer(c queue.Client) *JSONRPCServer {
	types.InitRpcTypeUtil()
	j := &JSONRPCServer{}
	j.jrpc.cli.Init(c)
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
