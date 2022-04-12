package ethrpc

import (
	"bytes"
	"fmt"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/rpc/ethrpc/admin"
	"github.com/33cn/chain33/rpc/ethrpc/eth"
	rpcNet "github.com/33cn/chain33/rpc/ethrpc/net"
	"github.com/33cn/chain33/rpc/ethrpc/personal"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/rpc"
	"io/ioutil"
	"net"
	"net/http"
)

const (
	EthNameSpace      = "eth"
	NetNameSpace      = "net"
	PersonalNameSpace = "personal"
	AdminNameSpace    = "admin"
	defaultEthRpcPort = 8545
)

var (
	log = log15.New("module", "eth_rpc")
)

type RPCServer struct {
	*rpc.Server
}

func (r *RPCServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	bodyData, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("ServeHTTP", "Read body err:", err.Error())
		return
	}

	fmt.Println("remote:", req.RemoteAddr, "req method:", req.Method, "req Body:", string(bodyData))
	//Restore  to its original state
	req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyData))

	// 设置跨域头
	w.Header().Set("Access-Control-Allow-Origin", "*")                                                            // 允许访问所有域
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")                             //允许请求方法
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type,AccessToken,X-CSRF-Token, Authorization, Token") //header的类型

	//放行所有OPTIONS方法
	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	r.Server.ServeHTTP(w, req)
}

//initEthRpc 注册eth rpc
func initEthRpc(cfg *ctypes.Chain33Config, c queue.Client, api client.QueueProtocolAPI) *rpc.Server {
	server := rpc.NewServer()
	if err := server.RegisterName(EthNameSpace, eth.NewEthApi(cfg, c, api)); err != nil {
		panic(err)
	}
	if err := server.RegisterName(PersonalNameSpace, personal.NewPersonalApi(cfg, c, api)); err != nil {
		panic(err)
	}
	if err := server.RegisterName(AdminNameSpace, admin.NewAdminApi(cfg, c, api)); err != nil {
		panic(err)
	}
	if err := server.RegisterName(NetNameSpace, rpcNet.NewNetApi(cfg, c, api)); err != nil {
		panic(err)
	}
	return server
}

type EthRpcServer struct {
	s *rpc.Server
}

func (e *EthRpcServer) Close() {
	e.s.Stop()
}

//NewEthRpcServer eth json rpcserver object
func NewEthRpcServer(c queue.Client, api client.QueueProtocolAPI) *EthRpcServer {
	server := initEthRpc(c.GetConfig(), c, api)
	srv := &RPCServer{server}
	s := &http.Server{Handler: srv}
	//默认绑定localhost
	bindAddr := fmt.Sprintf("localhost:%d", defaultEthRpcPort)
	if c.GetConfig().GetModuleConfig().RPC.ErpcBindAddr != "" {
		bindAddr = c.GetConfig().GetModuleConfig().RPC.ErpcBindAddr
	}
	s.Addr = bindAddr
	l, err := net.Listen("tcp", bindAddr)
	if err != nil {
		panic(err)
	}
	go s.Serve(l)
	return &EthRpcServer{s: server}
}
