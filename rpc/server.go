package rpc

import (
	"net"
	"net/rpc"
	"time"

	"gitlab.33.cn/chain33/chain33/client"
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
	s    *grpc.Server
	l    net.Listener
	//addr string
}

type JSONRPCServer struct {
	jrpc Chain33
	s    *rpc.Server
	l    net.Listener
	//addr string
}

func (s *JSONRPCServer) Close() {
	if s.l != nil {
		s.l.Close()
	}
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
	if j == nil {
		return
	}
	if j.l != nil {
		j.l.Close()
	}
	j.grpc.cli.Close()
}

func NewGRpcServer(c queue.Client, api client.QueueProtocolAPI) *Grpcserver {
	s := &Grpcserver{}
	s.grpc.cli.Init(c, api)
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

func NewJSONRPCServer(c queue.Client, api client.QueueProtocolAPI) *JSONRPCServer {
	j := &JSONRPCServer{}
	j.jrpc.cli.Init(c, api)
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
	api  client.QueueProtocolAPI
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

func (r *RPC) SetAPI(api client.QueueProtocolAPI) {
	r.api = api
}

func (r *RPC) SetQueueClient(c queue.Client) {
	gapi := NewGRpcServer(c, r.api)
	japi := NewJSONRPCServer(c, r.api)
	r.gapi = gapi
	r.japi = japi
	r.c = c
	//注册系统rpc
	pluginmgr.AddRPC(r)
	r.Listen()
}

func (r *RPC) SetQueueClientNoListen(c queue.Client) {
	gapi := NewGRpcServer(c, r.api)
	japi := NewJSONRPCServer(c, r.api)
	r.gapi = gapi
	r.japi = japi
	r.c = c
}

func (rpc *RPC) Listen() (port1 int, port2 int) {
	var err error
	for i := 0; i < 10; i++ {
		port1, err = rpc.gapi.Listen()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	for i := 0; i < 10; i++ {
		port2, err = rpc.japi.Listen()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	//sleep for a while
	time.Sleep(time.Millisecond)
	return port1, port2
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
	if rpc.gapi != nil {
		rpc.gapi.Close()
	}
	if rpc.japi != nil {
		rpc.japi.Close()
	}
}

func InitIpWhitelist(cfg *types.Rpc) {
	if len(cfg.Whitelist) == 0 && len(cfg.Whitlist) == 0 {
		remoteIpWhitelist["127.0.0.1"] = true
		return
	}
	if len(cfg.Whitelist) == 1 && cfg.Whitelist[0] == "*" {
		remoteIpWhitelist["0.0.0.0"] = true
		return
	}
	if len(cfg.Whitlist) == 1 && cfg.Whitlist[0] == "*" {
		remoteIpWhitelist["0.0.0.0"] = true
		return
	}
	if len(cfg.Whitelist) != 0 {
		for _, addr := range cfg.Whitelist {
			remoteIpWhitelist[addr] = true
		}
		return
	}
	if len(cfg.Whitlist) != 0 {
		for _, addr := range cfg.Whitlist {
			remoteIpWhitelist[addr] = true
		}
		return
	}

}

func InitJrpcFuncWhitelist(cfg *types.Rpc) {
	if len(cfg.JrpcFuncWhitelist) == 0 {
		jrpcFuncWhitelist["*"] = true
		return
	}
	if len(cfg.JrpcFuncWhitelist) == 1 && cfg.JrpcFuncWhitelist[0] == "*" {
		jrpcFuncWhitelist["*"] = true
		return
	}
	for _, funcName := range cfg.JrpcFuncWhitelist {
		jrpcFuncWhitelist[funcName] = true
	}
}

func InitGrpcFuncWhitelist(cfg *types.Rpc) {
	if len(cfg.GrpcFuncWhitelist) == 0 {
		grpcFuncWhitelist["*"] = true
		return
	}
	if len(cfg.GrpcFuncWhitelist) == 1 && cfg.GrpcFuncWhitelist[0] == "*" {
		grpcFuncWhitelist["*"] = true
		return
	}
	for _, funcName := range cfg.GrpcFuncWhitelist {
		grpcFuncWhitelist[funcName] = true
	}
}

func InitJrpcFuncBlacklist(cfg *types.Rpc) {
	if len(cfg.JrpcFuncBlacklist) == 0 {
		jrpcFuncBlacklist["CloseQueue"] = true
		return
	}
	for _, funcName := range cfg.JrpcFuncBlacklist {
		jrpcFuncBlacklist[funcName] = true
	}

}

func InitGrpcFuncBlacklist(cfg *types.Rpc) {
	if len(cfg.GrpcFuncBlacklist) == 0 {
		grpcFuncBlacklist["CloseQueue"] = true
		return
	}
	for _, funcName := range cfg.GrpcFuncBlacklist {
		grpcFuncBlacklist[funcName] = true
	}
}
