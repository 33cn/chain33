package ethrpc

import (
	"errors"
	"fmt"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/utils"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/rpc/ethrpc/admin"
	"github.com/33cn/chain33/rpc/ethrpc/eth"
	rpcNet "github.com/33cn/chain33/rpc/ethrpc/net"
	"github.com/33cn/chain33/rpc/ethrpc/personal"
	"github.com/33cn/chain33/rpc/ethrpc/web3"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"net"
	"net/http"
	"strings"
	"sync"
)

const (
	EthNameSpace        = "eth"
	NetNameSpace        = "net"
	PersonalNameSpace   = "personal"
	AdminNameSpace      = "admin"
	Web3NameSpace       = "web3"
	subRpctype          = "eth"
	defaultEthRpcPort   = 8545
	defaultEthWsRpcPort = 8546
)

type initApi func(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{}
type rpcAPIs map[string]initApi
var (
	log         = log15.New("module", "eth_rpc")
	DefaultApis = map[string]initApi{
		EthNameSpace:      eth.NewEthApi,
		NetNameSpace:      rpcNet.NewNetApi,
		PersonalNameSpace: personal.NewPersonalApi,
		AdminNameSpace:    admin.NewAdminApi,
		Web3NameSpace:     web3.NewWeb3Api,
	}
)

type subConfig struct {
	//ethereum json rpc bindaddr
	Enable        bool   `json:"enable,omitempty"`
	JrpcBindAddr  string `json:"jrpcBindAddr,omitempty"`
	WebsocketAddr string `json:"websocketAddr,omitempty"`
}

type HttpServer struct {
	mu          sync.Mutex
	endpoint    string
	timeouts    rpc.HTTPTimeouts
	server      *http.Server
	listener    net.Listener // non-nil when server is running
	HttpHandler *rpcHandler  //http
	WsHander    *rpcHandler  //websocket
	cfg         *ctypes.Chain33Config
	subCfg      *subConfig
	qclient     queue.Client
	api         client.QueueProtocolAPI
}

// isWebsocket checks the header of an http request for a websocket upgrade request.
func isWebsocket(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Upgrade")) == "websocket" &&
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")
}

type rpcHandler struct {
	http.Handler
	server *rpc.Server
}

//initRpcHandler 注册eth rpc
func initRpcHandler(apis rpcAPIs, cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) *rpcHandler {
	server := rpc.NewServer()
	for namespace, newApi := range apis {
		server.RegisterName(namespace, newApi(cfg, c, api))
	}
	return &rpcHandler{
		server: server,
	}
}

//NewHttpServer eth json rpcserver object
func NewHttpServer(c queue.Client, api client.QueueProtocolAPI) *HttpServer {
	var subcfg subConfig
	ctypes.MustDecode(c.GetConfig().GetSubConfig().RPC[subRpctype], &subcfg)
	log.Debug("NewHttpServer", "subcfg", subcfg)
	return &HttpServer{
		timeouts: rpc.DefaultHTTPTimeouts,
		cfg:      c.GetConfig(),
		subCfg:   &subcfg,
		qclient:  c,
		api:      api,
	}

}

//set listen addr
func (h *HttpServer) setEndPoint(listenAddr string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.endpoint = listenAddr
}

//EnableRpc http rpc
func (h *HttpServer) EnableRpc(apis rpcAPIs) {
	rpcHandler := initRpcHandler(apis, h.cfg, h.qclient, h.api)
	rpcHandler.Handler = node.NewHTTPHandlerStack(rpcHandler.server, []string{"*"}, []string{"*"})
	h.HttpHandler = rpcHandler
	if h.subCfg.JrpcBindAddr == "" {
		h.subCfg.JrpcBindAddr = fmt.Sprintf("localhost:%d", defaultEthRpcPort)
	}
	h.setEndPoint(h.subCfg.JrpcBindAddr)
	log.Debug("EnableRpc", "httpaddr", h.endpoint)
}

//EnableWS websocket rpc
func (h *HttpServer) EnableWS(apis rpcAPIs) {
	rpcHandler := initRpcHandler(apis, h.cfg, h.qclient, h.api)
	rpcHandler.Handler = rpcHandler.server.WebsocketHandler([]string{"*"})
	h.WsHander = rpcHandler
	if h.subCfg.WebsocketAddr == "" {
		h.subCfg.WebsocketAddr = fmt.Sprintf("localhost:%d", defaultEthWsRpcPort)
	}
	h.setEndPoint(h.subCfg.WebsocketAddr)
	log.Debug("EnableWS", "websocketaddr", h.endpoint)
}

func (h *HttpServer) Start() (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.subCfg.Enable {
		return 0, errors.New("rpc disable")
	}

	h.server = &http.Server{
		Handler: h,
	}

	h.server.ReadTimeout = h.timeouts.ReadTimeout
	h.server.WriteTimeout = h.timeouts.WriteTimeout
	h.server.IdleTimeout = h.timeouts.IdleTimeout
	l, err := net.Listen("tcp", h.endpoint)
	if err != nil {
		panic(err)
	}

	h.listener = l
	if !h.cfg.GetModuleConfig().RPC.EnableTLS {
		go h.server.Serve(h.listener)
	} else {
		go h.server.ServeTLS(h.listener, h.cfg.GetModuleConfig().RPC.KeyFile, h.cfg.GetModuleConfig().RPC.CertFile)
	}

	if h.WsHander != nil {
		log.Info("WebSocket enabled", "url:", fmt.Sprintf("ws://%v", h.endpoint))
	} else {
		log.Info("http enabled", "url:", fmt.Sprintf("http://%v", h.endpoint))
	}
	return h.listener.Addr().(*net.TCPAddr).Port, nil

}

func (h *HttpServer) Close() {
	if h.WsHander != nil {
		h.WsHander.server.Stop()
	} else if h.HttpHandler != nil {
		h.HttpHandler.server.Stop()
	}
}

func (h *HttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if utils.IsPublicIP(ip) {
		log.Warn("ServeHTTP", "remote client", r.RemoteAddr)
	} else {
		log.Debug("ServeHTTP", "remote client", r.RemoteAddr)
	}
	if h.HttpHandler != nil {
		h.HttpHandler.ServeHTTP(w, r)
		return
	}
	if h.WsHander != nil && isWebsocket(r) {
		h.WsHander.ServeHTTP(w, r)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}
