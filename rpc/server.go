package rpc

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"

	// register gzip
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	ipWhitelist       = make(map[string]bool)
	rpcCfg            *types.Rpc
	jrpcFuncWhitelist = make(map[string]bool)
	grpcFuncWhitelist = make(map[string]bool)
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

func checkIpWhitelist(addr string) bool {

	if _, ok := ipWhitelist["0.0.0.0"]; ok {
		return true
	}

	if _, ok := ipWhitelist[addr]; ok {
		return true
	}
	return false
}
func checkJrpcFuncWritelist(funcName string) bool {

	if _, ok := jrpcFuncWhitelist["*"]; ok {
		return true
	}

	if _, ok := jrpcFuncWhitelist[funcName]; ok {
		return true
	}
	return false
}
func checkGrpcFuncWritelist(funcName string) bool {

	if _, ok := grpcFuncWhitelist["*"]; ok {
		return true
	}

	if _, ok := grpcFuncWhitelist[funcName]; ok {
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
	j := &JSONRPCServer{}
	j.jrpc.cli.Init(c)
	return j
}

func Init(cfg *types.Rpc) {
	InitIpWhitelist(cfg)
	InitJrpcFuncWhitelist(cfg)
	InitGrpcFuncWhitelist(cfg)
}
func InitIpWhitelist(cfg *types.Rpc) {
	rpcCfg = cfg
	if len(cfg.GetIpWhitelist()) == 1 && cfg.GetIpWhitelist()[0] == "*" {
		ipWhitelist["0.0.0.0"] = true
		return
	}

	for _, addr := range cfg.GetIpWhitelist() {
		ipWhitelist[addr] = true
	}
}
func InitJrpcFuncWhitelist(cfg *types.Rpc) {
	rpcCfg = cfg
	if len(cfg.GetJrpcFuncWhitelist()) == 1 && cfg.GetJrpcFuncWhitelist()[0] == "*" {
		jrpcFuncWhitelist["*"] = true
		return
	}
	for _, funcName := range cfg.GetJrpcFuncWhitelist() {
		jrpcFuncWhitelist[funcName] = true
	}
}
func InitGrpcFuncWhitelist(cfg *types.Rpc) {
	rpcCfg = cfg
	if len(cfg.GetGrpcFuncWhitelist()) == 1 && cfg.GetGrpcFuncWhitelist()[0] == "*" {
		grpcFuncWhitelist["*"] = true
		return
	}
	for _, funcName := range cfg.GetGrpcFuncWhitelist() {
		grpcFuncWhitelist[funcName] = true
	}
}
