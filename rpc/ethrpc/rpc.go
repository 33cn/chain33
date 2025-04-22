package ethrpc

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

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
)

const (
	ethNameSpace        = "eth"
	netNameSpace        = "net"
	personalNameSpace   = "personal"
	adminNameSpace      = "admin"
	web3NameSpace       = "web3"
	subRpctype          = "eth"
	defaultEthRPCPort   = 8545
	defaultEthWsRPCPort = 8546
)

type initAPI func(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) interface{}
type rpcAPIs map[string]initAPI

var (
	log = log15.New("module", "eth_rpc")
	// default apis
	defaultApis = map[string]initAPI{
		ethNameSpace:      eth.NewEthAPI,
		netNameSpace:      rpcNet.NewNetAPI,
		personalNameSpace: personal.NewPersonalAPI,
		adminNameSpace:    admin.NewAdminAPI,
		web3NameSpace:     web3.NewWeb3API,
	}
)

type subConfig struct {
	//ethereum json rpc bindaddr
	Enable          bool     `json:"enable,omitempty"`
	EnableRlpTxHash bool     `json:"enableRlpTxHash,omitempty"`
	HTTPAddr        string   `json:"httpAddr,omitempty"`
	HTTPAPI         []string `json:"httpApi,omitempty"` //eth,admin,net,web3/personal
	WsAddr          string   `json:"wsAddr,omitempty"`
	WsAPI           []string `json:"wsApi,omitempty"` //eth,admin,net,web3/personal
	Web3CliVer      string   `json:"web3CliVer,omitempty"`
}

// ServerAPI ...
type ServerAPI interface {
	EnableRPC()
	EnableWS()
	Start() (int, error)
	Close()
}

type httpServer struct {
	mu          sync.Mutex
	endpoint    string
	timeouts    rpc.HTTPTimeouts
	server      *http.Server
	listener    net.Listener // non-nil when server is running
	httpHandler *rpcHandler  //http
	wsHander    *rpcHandler  //websocket
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

// initRpcHandler 注册eth rpc
func initRPCHandler(apis rpcAPIs, cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) *rpcHandler {
	server := rpc.NewServer()
	if len(apis) == 0 {
		apis = defaultApis
	}
	for namespace, newAPI := range apis {
		server.RegisterName(namespace, newAPI(cfg, c, api))
	}
	return &rpcHandler{
		server: server,
	}
}

// NewHTTPServer eth json rpcserver object
func NewHTTPServer(c queue.Client, api client.QueueProtocolAPI) ServerAPI {
	var subcfg subConfig
	ctypes.MustDecode(c.GetConfig().GetSubConfig().RPC[subRpctype], &subcfg)

	log.Debug("NewHttpServer", "subcfg", subcfg)
	return &httpServer{
		timeouts: rpc.DefaultHTTPTimeouts,
		cfg:      c.GetConfig(),
		subCfg:   &subcfg,
		qclient:  c,
		api:      api,
	}

}

// set listen addr
func (h *httpServer) setEndPoint(listenAddr string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.endpoint = listenAddr
}

// EnableRpc  register http rpc
func (h *httpServer) EnableRPC() {
	var apis = make(rpcAPIs)
	for _, namespace := range h.subCfg.HTTPAPI {
		if api, ok := defaultApis[namespace]; ok {
			apis[namespace] = api
		}
	}

	rpcHandler := initRPCHandler(apis, h.cfg, h.qclient, h.api)
	rpcHandler.Handler = node.NewHTTPHandlerStack(rpcHandler.server, []string{"*"}, []string{"*"}, nil)
	h.httpHandler = rpcHandler
	if h.subCfg.HTTPAddr == "" {
		h.subCfg.HTTPAddr = fmt.Sprintf("localhost:%d", defaultEthRPCPort)
	}
	h.setEndPoint(h.subCfg.HTTPAddr)
	log.Debug("EnableRpc", "httpaddr", h.endpoint)
}

// EnableWS register websocket rpc
func (h *httpServer) EnableWS() {
	var apis = make(rpcAPIs)
	for _, namespace := range h.subCfg.WsAPI {
		if api, ok := defaultApis[namespace]; ok {
			apis[namespace] = api
		}
	}

	rpcHandler := initRPCHandler(apis, h.cfg, h.qclient, h.api)
	rpcHandler.Handler = rpcHandler.server.WebsocketHandler([]string{"*"})
	h.wsHander = rpcHandler
	if h.subCfg.WsAddr == "" {
		h.subCfg.WsAddr = fmt.Sprintf("localhost:%d", defaultEthWsRPCPort)
	}
	h.setEndPoint(h.subCfg.WsAddr)
	log.Debug("EnableWS", "websocketaddr", h.endpoint)
}

// Start server start
func (h *httpServer) Start() (int, error) {
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
		go h.server.ServeTLS(h.listener, h.cfg.GetModuleConfig().RPC.CertFile, h.cfg.GetModuleConfig().RPC.KeyFile)
	}

	if h.wsHander != nil {
		log.Info("WebSocket enabled", "url:", fmt.Sprintf("ws://%v", h.endpoint))
	} else {
		log.Info("http enabled", "url:", fmt.Sprintf("http://%v", h.endpoint))
	}
	return h.listener.Addr().(*net.TCPAddr).Port, nil

}

// Close close service
func (h *httpServer) Close() {
	if h.wsHander != nil {
		h.wsHander.server.Stop()
	} else if h.wsHander != nil {
		h.wsHander.server.Stop()
	}
}

// ServeHTTP rewrite ServeHTTP
func (h *httpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTION" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	if utils.IsPublicIP(ip) {
		log.Debug("ServeHTTP", "remote client", r.RemoteAddr)
	} else {
		log.Debug("ServeHTTP", "remote client", r.RemoteAddr)
	}
	if h.httpHandler != nil {
		h.httpHandler.ServeHTTP(w, r)
		return
	}
	if h.wsHander != nil && isWebsocket(r) {
		h.wsHander.ServeHTTP(w, r)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}
