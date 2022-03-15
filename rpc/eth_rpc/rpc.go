package eth_rpc

import (
	"bytes"
	"fmt"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/rpc"
	"io/ioutil"
	"net/http"
)

const(
	EthNameSpace="eth"
	NetNameSpace="net"
	DefaultEthRpcPort=8545

)

var(
	 log = log15.New("module", "eth_rpc")
)

type RPCServer struct {
	*rpc.Server
}

func (r *RPCServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	bodyData, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Error("ServeHTTP","Read body err:", err.Error())
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

func InitEthRpc(cfg *ctypes.Chain33Config,c queue.Client,api client.QueueProtocolAPI)*rpc.Server{
	server := rpc.NewServer()
	//defer server.Stop()
	if err := server.RegisterName(EthNameSpace, NewEthApi(cfg,c,api)); err != nil {
		panic(err)
	}
	return server
}