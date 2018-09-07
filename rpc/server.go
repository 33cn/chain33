package rpc

import (
	"net"
	"net/rpc"

	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/net/context"

	// register gzip
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	remoteIpWhitelist = make(map[string]bool)
	rpcCfg            *types.Rpc
	jrpcFuncWhitelist = make(map[string]bool)
	grpcFuncWhitelist = make(map[string]bool)
	jrpcFuncBlacklist = make(map[string]bool)
	grpcFuncBlacklist = make(map[string]bool)
)

type Chain33 struct {
	cli channelClient
}

type Grpc struct {
	cli channelClient
}

type Grpcserver struct {
	grpc Grpc
	l    net.Listener
	s    *grpc.Server
	//addr string
}

type JSONRPCServer struct {
	jrpc Chain33
	s    *rpc.Server
	//addr string
}

func (s *JSONRPCServer) Close() {
	s.jrpc.cli.Close()
}

func checkIpWhitelist(addr string) bool {
	//回环网络直接允许
	ip := net.ParseIP(addr)
	if ip.IsLoopback() {
		return true
	}
	ipv4 := ip.To4()
	if ipv4 != nil {
		addr = ipv4.String()
	}
	if _, ok := remoteIpWhitelist["0.0.0.0"]; ok {
		return true
	}
	if _, ok := remoteIpWhitelist[addr]; ok {
		return true
	}
	return false
}

func checkJrpcFuncWhitelist(funcName string) bool {

	if _, ok := jrpcFuncWhitelist["*"]; ok {
		return true
	}

	if _, ok := jrpcFuncWhitelist[funcName]; ok {
		return true
	}
	return false
}
func checkGrpcFuncWhitelist(funcName string) bool {

	if _, ok := grpcFuncWhitelist["*"]; ok {
		return true
	}

	if _, ok := grpcFuncWhitelist[funcName]; ok {
		return true
	}
	return false
}
func checkJrpcFuncBlacklist(funcName string) bool {
	if _, ok := jrpcFuncBlacklist[funcName]; ok {
		return true
	}
	return false
}
func checkGrpcFuncBlacklist(funcName string) bool {
	if _, ok := grpcFuncBlacklist[funcName]; ok {
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
	var opts []grpc.ServerOption
	//register interceptor
	//var interceptor grpc.UnaryServerInterceptor
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if err := auth(ctx, info); err != nil {
			return nil, err
		}
		// Continue processing the request
		return handler(ctx, req)
	}
	opts = append(opts, grpc.UnaryInterceptor(interceptor))
	server := grpc.NewServer(opts...)
	s.s = server
	types.RegisterChain33Server(server, &s.grpc)
	return s
}

func NewJSONRPCServer(c queue.Client) *JSONRPCServer {
	j := &JSONRPCServer{}
	j.jrpc.cli.Init(c)
	server := rpc.NewServer()
	j.s = server
	server.RegisterName("Chain33", &j.jrpc)
	return j
}

type RPC struct {
	cfg  *types.Rpc
	gapi *Grpcserver
	japi *JSONRPCServer
	c    queue.Client
}

func InitCfg(cfg *types.Rpc) {
	rpcCfg = cfg
	InitIpWhitelist(cfg)
	InitJrpcFuncWhitelist(cfg)
	InitGrpcFuncWhitelist(cfg)
	InitJrpcFuncBlacklist(cfg)
	InitGrpcFuncBlacklist(cfg)
}

func New(cfg *types.Rpc) *RPC {
	InitCfg(cfg)
	return &RPC{cfg: cfg}
}

func (r *RPC) SetQueueClient(c queue.Client) {
	gapi := NewGRpcServer(c)
	japi := NewJSONRPCServer(c)
	r.gapi = gapi
	r.japi = japi
	r.c = c
	//注册系统rpc
	pluginmgr.AddRPC(r)
	go gapi.Listen()
	go japi.Listen()
}

func (rpc *RPC) GetQueueClient() queue.Client {
	return rpc.c
}

func (rpc *RPC) GRPC() *grpc.Server {
	return rpc.gapi.s
}

func (rpc *RPC) JRPC() *rpc.Server {
	return rpc.japi.s
}

func (rpc *RPC) Close() {
	rpc.gapi.Close()
	rpc.japi.Close()
}

func InitIpWhitelist(cfg *types.Rpc) {
	if len(cfg.GetWhitelist()) == 0 && len(cfg.GetWhitlist()) == 0 {
		remoteIpWhitelist["127.0.0.1"] = true
		return
	}
	if len(cfg.GetWhitelist()) == 1 && cfg.GetWhitelist()[0] == "*" {
		remoteIpWhitelist["0.0.0.0"] = true
		return
	}
	if len(cfg.GetWhitlist()) == 1 && cfg.GetWhitlist()[0] == "*" {
		remoteIpWhitelist["0.0.0.0"] = true
		return
	}
	if len(cfg.GetWhitelist()) != 0 {
		for _, addr := range cfg.GetWhitelist() {
			remoteIpWhitelist[addr] = true
		}
		return
	}
	if len(cfg.GetWhitlist()) != 0 {
		for _, addr := range cfg.GetWhitlist() {
			remoteIpWhitelist[addr] = true
		}
		return
	}

}
func InitJrpcFuncWhitelist(cfg *types.Rpc) {
	if len(cfg.GetJrpcFuncWhitelist()) == 0 {
		jrpcFuncWhitelist["*"] = true
		return
	}
	if len(cfg.GetJrpcFuncWhitelist()) == 1 && cfg.GetJrpcFuncWhitelist()[0] == "*" {
		jrpcFuncWhitelist["*"] = true
		return
	}
	for _, funcName := range cfg.GetJrpcFuncWhitelist() {
		jrpcFuncWhitelist[funcName] = true
	}
}
func InitGrpcFuncWhitelist(cfg *types.Rpc) {
	if len(cfg.GetGrpcFuncWhitelist()) == 0 {
		grpcFuncWhitelist["*"] = true
		return
	}
	if len(cfg.GetGrpcFuncWhitelist()) == 1 && cfg.GetGrpcFuncWhitelist()[0] == "*" {
		grpcFuncWhitelist["*"] = true
		return
	}
	for _, funcName := range cfg.GetGrpcFuncWhitelist() {
		grpcFuncWhitelist[funcName] = true
	}
}
func InitJrpcFuncBlacklist(cfg *types.Rpc) {
	if len(cfg.GetJrpcFuncBlacklist()) == 0 {
		jrpcFuncBlacklist["CloseQueue"] = true
		return
	}
	for _, funcName := range cfg.GetJrpcFuncBlacklist() {
		jrpcFuncBlacklist[funcName] = true
	}

}
func InitGrpcFuncBlacklist(cfg *types.Rpc) {
	if len(cfg.GetGrpcFuncBlacklist()) == 0 {
		grpcFuncBlacklist["CloseQueue"] = true
		return
	}
	for _, funcName := range cfg.GetGrpcFuncBlacklist() {
		grpcFuncBlacklist[funcName] = true
	}
}
