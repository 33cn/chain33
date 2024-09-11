// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"strings"
	"sync"
	"time"

	rpctypes "github.com/33cn/chain33/rpc/types"

	rclient "github.com/33cn/chain33/rpc/client"
	"github.com/33cn/chain33/rpc/ethrpc"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/queue"
	_ "github.com/33cn/chain33/rpc/grpcclient" // register grpc multiple resolver
	"github.com/33cn/chain33/types"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip" // register gzip
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

var (
	remoteIPWhitelist           = make(map[string]bool)
	rpcCfg                      *types.RPC
	jrpcFuncWhitelist           = make(map[string]bool)
	grpcFuncWhitelist           = make(map[string]bool)
	jrpcFuncBlacklist           = make(map[string]bool)
	grpcFuncBlacklist           = make(map[string]bool)
	rpcFilterPrintFuncBlacklist = make(map[string]bool)
	grpcFuncListLock            = sync.RWMutex{}
	log                         = log15.New("module", "rpc_client")
)

// Chain33  a channel client
type Chain33 struct {
	cli rclient.ChannelClient
}

// Grpc a channelClient
type Grpc struct {
	cli       rclient.ChannelClient
	cachelock sync.Mutex
	subCache  map[string]*subInfo // topic -->subInfo
}

type subInfo struct {
	//订阅的主题名称，自定义
	topic string
	//订阅的消息类型
	subType string
	//订阅开始的时间
	since time.Time
	//同一个topic下多个订阅者分配不同的channel来接收订阅的消息
	subChan map[chan *queue.Message]string
}

// Grpcserver a object
type Grpcserver struct {
	grpc *Grpc
	s    *grpc.Server
	l    net.Listener
}

func (g *Grpc) addSubInfo(in *subInfo) {
	g.cachelock.Lock()
	defer g.cachelock.Unlock()
	g.subCache[in.topic] = in
}

func (g *Grpc) addSubChan(topic string, dch chan *queue.Message) {
	g.cachelock.Lock()
	defer g.cachelock.Unlock()
	info, ok := g.subCache[topic]
	if ok {
		info.subChan[dch] = topic
	}
}

func (g *Grpc) delSubInfo(topic string, dch chan *queue.Message) error {
	g.cachelock.Lock()
	defer g.cachelock.Unlock()
	info, ok := g.subCache[topic]
	if ok {
		if dch != nil {
			delete(info.subChan, dch)
			if len(info.subChan) == 0 {
				delete(g.subCache, topic)
			}
		} else {
			delete(g.subCache, topic)

		}
		return nil
	}
	return errors.New("no this topicID")

}
func (g *Grpc) hashTopic(topic string) *subInfo {
	g.cachelock.Lock()
	defer g.cachelock.Unlock()
	if info, ok := g.subCache[topic]; ok {
		//clone subinfo
		cinfo := *info
		return &cinfo
	}
	return nil
}

func (g *Grpc) listSubInfo() []*subInfo {
	g.cachelock.Lock()
	defer g.cachelock.Unlock()
	var infos []*subInfo
	for _, info := range g.subCache {
		infos = append(infos, info)
	}
	return infos
}

func (g *Grpc) getSubUsers(topic string) uint32 {
	g.cachelock.Lock()
	defer g.cachelock.Unlock()
	info, ok := g.subCache[topic]
	if ok {
		return uint32(len(info.subChan))
	}
	return 0
}

// NewGrpcServer new  GrpcServer object
func NewGrpcServer() *Grpcserver {
	return &Grpcserver{grpc: &Grpc{}}
}

// JSONRPCServer  a json rpcserver object
type JSONRPCServer struct {
	jrpc *Chain33
	s    *rpc.Server
	l    net.Listener
}

// Close json rpcserver close
func (s *JSONRPCServer) Close() {
	if s.l != nil {
		err := s.l.Close()
		if err != nil {
			log.Error("JSONRPCServer close", "err", err)
		}
	}
	if s.jrpc != nil {
		s.jrpc.cli.Close()
	}
}

func checkBasicAuth(r *http.Request) bool {
	if rpcCfg.JrpcUserName == "" && rpcCfg.JrpcUserPasswd == "" {
		return true
	}

	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 {
		return false
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return false
	}

	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return false
	}
	return pair[0] == rpcCfg.JrpcUserName && pair[1] == rpcCfg.JrpcUserPasswd
}

func checkIPWhitelist(addr string) bool {
	//回环网络直接允许
	ip := net.ParseIP(addr)
	if ip.IsLoopback() {
		return true
	}
	ipv4 := ip.To4()
	if ipv4 != nil {
		addr = ipv4.String()
	}
	if _, ok := remoteIPWhitelist["0.0.0.0"]; ok {
		return true
	}
	if _, ok := remoteIPWhitelist[addr]; ok {
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

func checkGrpcFuncValidity(funcName string) bool {
	grpcFuncListLock.RLock()
	defer grpcFuncListLock.RUnlock()
	if _, ok := grpcFuncBlacklist[funcName]; ok {
		return false
	}
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

// Close grpcserver close
func (j *Grpcserver) Close() {
	if j == nil {
		return
	}
	if j.l != nil {
		err := j.l.Close()
		if err != nil {
			log.Error("Grpcserver close", "err", err)
		}
	}
	if j.grpc != nil {
		j.grpc.cli.Close()
	}
}

// NewGRpcServer new grpcserver object
func NewGRpcServer(c queue.Client, api client.QueueProtocolAPI) *Grpcserver {
	s := &Grpcserver{grpc: &Grpc{}}
	s.grpc.subCache = make(map[string]*subInfo)
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
	if rpcCfg.EnableTLS {
		creds, err := credentials.NewServerTLSFromFile(rpcCfg.CertFile, rpcCfg.KeyFile)
		if err != nil {
			panic(fmt.Sprintf("err=%s, if cert.pem not found, run chain33-cli cert --host=127.0.0.1 to create", err.Error()))
		}
		credsOps := grpc.Creds(creds)
		opts = append(opts, credsOps)
	}

	kp := keepalive.EnforcementPolicy{
		MinTime:             10 * time.Second,
		PermitWithoutStream: true,
	}
	opts = append(opts, grpc.KeepaliveEnforcementPolicy(kp))

	server := grpc.NewServer(opts...)
	s.s = server
	types.RegisterChain33Server(server, s.grpc)
	reflection.Register(server)
	return s
}

// NewJSONRPCServer new json rpcserver object
func NewJSONRPCServer(c queue.Client, api client.QueueProtocolAPI) *JSONRPCServer {
	j := &JSONRPCServer{jrpc: &Chain33{}}
	j.jrpc.cli.Init(c, api)
	server := rpc.NewServer()
	j.s = server
	err := server.RegisterName("Chain33", j.jrpc)
	if err != nil {
		return nil
	}
	return j
}

// RPC a type object
type RPC struct {
	cfg       *types.RPC
	allCfg    *types.Chain33Config
	gapi      *Grpcserver
	japi      *JSONRPCServer
	eapi      ethrpc.ServerAPI
	ewsapi    ethrpc.ServerAPI
	cli       queue.Client
	api       client.QueueProtocolAPI
	ctx       context.Context
	cancelCtx context.CancelFunc
}

// InitCfg  interfaces
func InitCfg(rcfg *types.RPC) {
	rpcCfg = rcfg
	InitIPWhitelist(rcfg)
	InitJrpcFuncWhitelist(rcfg)
	InitGrpcFuncWhitelist(rcfg)
	InitJrpcFuncBlacklist(rcfg)
	InitGrpcFuncBlacklist(rcfg)
	InitFilterPrintFuncBlacklist()
}

// New produce a rpc by cfg
func New(cfg *types.Chain33Config) *RPC {
	mcfg := cfg.GetModuleConfig().RPC
	InitCfg(mcfg)
	if mcfg.EnableTrace {
		grpc.EnableTracing = true
	}
	r := &RPC{cfg: mcfg, allCfg: cfg}
	r.ctx, r.cancelCtx = context.WithCancel(context.Background())
	return r
}

// SetAPI set api of rpc
func (r *RPC) SetAPI(api client.QueueProtocolAPI) {
	r.api = api
}

// SetQueueClient set queue client
func (r *RPC) SetQueueClient(c queue.Client) {

	gapi := NewGRpcServer(c, r.api)
	japi := NewJSONRPCServer(c, r.api)
	r.eapi = ethrpc.NewHTTPServer(c, r.api)
	r.ewsapi = ethrpc.NewHTTPServer(c, r.api)
	r.gapi = gapi
	r.japi = japi
	r.cli = c
	//配置rpc,websocket
	r.eapi.EnableRPC()
	r.ewsapi.EnableWS()
	go r.handleSysEvent()

	//注册系统rpc
	pluginmgr.AddRPC(r)
	r.Listen()
}

// SetQueueClientNoListen  set queue client with  no listen
func (r *RPC) SetQueueClientNoListen(c queue.Client) {
	gapi := NewGRpcServer(c, r.api)
	japi := NewJSONRPCServer(c, r.api)
	r.eapi = ethrpc.NewHTTPServer(c, r.api)
	r.ewsapi = ethrpc.NewHTTPServer(c, r.api)
	r.eapi.EnableRPC()
	r.ewsapi.EnableWS()
	r.gapi = gapi
	r.japi = japi
	r.cli = c

}

//处理订阅的rpc事件,目前只有订阅事件
func (r *RPC) handleSysEvent() {
	r.cli.Sub("rpc")
	var cli rclient.ChannelClient
	cli.Init(r.cli, r.api)
	for msg := range r.cli.Recv() {
		switch msg.Ty {
		case types.EventGetEvmNonce:
			addr := msg.GetData().(*types.ReqEvmAccountNonce)
			exec := r.allCfg.ExecName("evm")
			execty := types.LoadExecutorType(exec)
			if execty == nil {
				msg.Reply(r.cli.NewMessage("", types.EventGetEvmNonce, &types.Reply{IsOk: false}))
				continue
			}

			var param rpctypes.Query4Jrpc
			param.FuncName = "GetNonce"
			param.Execer = exec
			param.Payload = []byte(fmt.Sprintf(`{"address":"%v"}`, addr.GetAddr()))
			queryparam, err := execty.CreateQuery(param.FuncName, param.Payload)
			if err != nil {
				msg.Reply(r.cli.NewMessage("", types.EventGetEvmNonce, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
				continue
			}
			resp, err := cli.Query(param.Execer, param.FuncName, queryparam)
			if err != nil {
				msg.Reply(r.cli.NewMessage("mempool", types.EventGetEvmNonce, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
				continue
			}

			result, err := execty.QueryToJSON(param.FuncName, resp)
			if err != nil {
				msg.Reply(r.cli.NewMessage("mempool", types.EventGetEvmNonce, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
				continue
			}

			var nonce struct {
				Nonce string `json:"nonce,omitempty"`
			}
			err = json.Unmarshal(result, &nonce)
			if err != nil {
				msg.Reply(r.cli.NewMessage("mempool", types.EventGetEvmNonce, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
				continue
			}

			currentNonce, _ := strconv.Atoi(nonce.Nonce)
			msg.Reply(r.cli.NewMessage("", types.EventGetEvmNonce, &types.EvmAccountNonce{Nonce: int64(currentNonce), Addr: addr.String()}))

		case types.EventPushEVM, types.EventPushTxReceipt, types.EventPushBlockHeader, types.EventPushBlock, types.EventPushTxResult:
			topicInfo := r.gapi.grpc.hashTopic(msg.GetData().(*types.PushData).GetName())
			if topicInfo != nil {
				var ticket = time.NewTicker(time.Second)
				for ch := range topicInfo.subChan {
					select {
					case <-ticket.C:
						ticket.Reset(time.Second)
						continue
					case ch <- msg:
						msg.Reply(r.cli.NewMessage("blockchain", msg.Ty, &types.Reply{IsOk: true}))
						ticket.Reset(time.Second)

					}
				}

			} else {
				//要求blockchain模块停止推送
				log.Error("handleSysEvent", "no subscriber,all topic", r.gapi.grpc.subCache, "no topic:", msg.GetData().(*types.PushData).GetName(), "subchan:", topicInfo)
				msg.Reply(r.cli.NewMessage("blockchain", msg.Ty, &types.Reply{IsOk: false, Msg: []byte("no subscriber")}))
			}

		default:
			log.Error("rpc.handleSysEvent no support event:", msg.Ty)
		}

	}
}

// Listen rpc listen，http port,grpc port,ethrpc port,ethrpc websocket port
func (r *RPC) Listen() (port1 int, port2 int, port3, port4 int) {
	var err error
	for i := 0; i < 10; i++ {
		port1, err = r.gapi.Listen()
		if err != nil {
			log.Error("Grpc Listen", "err", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}
	for i := 0; i < 10; i++ {
		port2, err = r.japi.Listen()
		if err != nil {
			log.Error("Jrpc Listen", "err", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	port3, err = r.eapi.Start()
	if err != nil {
		log.Error("ethrpc Listen", "err", err)
	}

	port4, err = r.ewsapi.Start()
	if err != nil {
		log.Error("wsrpc Listen", "err", err)
	}
	log.Info("rpc Listen port", "grpc", port1, "jrpc", port2, "erpc", port3, "wsport:", port4)
	//sleep for a while

	time.Sleep(time.Millisecond)
	return port1, port2, port3, port4

}

// GetQueueClient get queue client
func (r *RPC) GetQueueClient() queue.Client {
	return r.cli
}

// Context get rpc context
func (r *RPC) Context() context.Context {
	return r.ctx
}

// GRPC return grpc rpc
func (r *RPC) GRPC() *grpc.Server {
	return r.gapi.s
}

// JRPC return jrpc
func (r *RPC) JRPC() *rpc.Server {
	return r.japi.s
}

// Close rpc close
func (r *RPC) Close() {

	if r.gapi != nil {
		r.gapi.Close()
	}
	if r.japi != nil {
		r.japi.Close()
	}
	if r.eapi != nil {
		r.eapi.Close()
	}
	if r.ewsapi != nil {
		r.ewsapi.Close()
	}
	r.cli.Close()
	r.cancelCtx()
}

// InitIPWhitelist init ip whitelist
func InitIPWhitelist(cfg *types.RPC) {
	if len(cfg.Whitelist) == 0 && len(cfg.Whitlist) == 0 {
		remoteIPWhitelist["127.0.0.1"] = true
		return
	}
	if len(cfg.Whitelist) == 1 && cfg.Whitelist[0] == "*" {
		remoteIPWhitelist["0.0.0.0"] = true
		return
	}
	if len(cfg.Whitlist) == 1 && cfg.Whitlist[0] == "*" {
		remoteIPWhitelist["0.0.0.0"] = true
		return
	}
	if len(cfg.Whitelist) != 0 {
		for _, addr := range cfg.Whitelist {
			remoteIPWhitelist[addr] = true
		}
		return
	}
	if len(cfg.Whitlist) != 0 {
		for _, addr := range cfg.Whitlist {
			remoteIPWhitelist[addr] = true
		}
		return
	}

}

// InitJrpcFuncWhitelist init jrpc function whitelist
func InitJrpcFuncWhitelist(cfg *types.RPC) {
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

// InitGrpcFuncWhitelist init grpc function whitelist
func InitGrpcFuncWhitelist(cfg *types.RPC) {
	grpcFuncListLock.Lock()
	defer grpcFuncListLock.Unlock()
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

// InitJrpcFuncBlacklist init jrpc function blacklist
func InitJrpcFuncBlacklist(cfg *types.RPC) {
	if len(cfg.JrpcFuncBlacklist) == 0 {
		jrpcFuncBlacklist["CloseQueue"] = true
		return
	}
	for _, funcName := range cfg.JrpcFuncBlacklist {
		jrpcFuncBlacklist[funcName] = true
	}

}

// InitGrpcFuncBlacklist init grpc function blacklist
func InitGrpcFuncBlacklist(cfg *types.RPC) {
	grpcFuncListLock.Lock()
	defer grpcFuncListLock.Unlock()
	if len(cfg.GrpcFuncBlacklist) == 0 {
		grpcFuncBlacklist["CloseQueue"] = true
		return
	}
	for _, funcName := range cfg.GrpcFuncBlacklist {
		grpcFuncBlacklist[funcName] = true
	}
}

// InitFilterPrintFuncBlacklist rpc模块打印requet信息时需要过滤掉一些敏感接口的入参打印，比如钱包密码相关的
func InitFilterPrintFuncBlacklist() {
	rpcFilterPrintFuncBlacklist["UnLock"] = true
	rpcFilterPrintFuncBlacklist["SetPasswd"] = true
	rpcFilterPrintFuncBlacklist["GetSeed"] = true
	rpcFilterPrintFuncBlacklist["SaveSeed"] = true
	rpcFilterPrintFuncBlacklist["ImportPrivkey"] = true
}

func checkFilterPrintFuncBlacklist(funcName string) bool {
	if _, ok := rpcFilterPrintFuncBlacklist[funcName]; ok {
		return true
	}
	return false
}

// PushType ...
type PushType int32

func (pushType PushType) string() string {
	return []string{"PushBlock", "PushBlockHeader", "PushTxReceipt", "PushTxResult", "PushEVMEvent", "NotSupported"}[pushType]
}
