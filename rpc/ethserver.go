package rpc

import (
	"fmt"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	eth "github.com/33cn/chain33/rpc/ethrpc"
	"github.com/ethereum/go-ethereum/rpc"
	"net"
	"net/http"
)

type EthRpcServer struct {
	s    *rpc.Server

}

func (e*EthRpcServer)Close(){
	e.s.Stop()
}

//NewEthJsonRpcServer eth json rpcserver object
func NewEthJsonRpcServer(c queue.Client, api client.QueueProtocolAPI)*EthRpcServer{
	server:= eth.InitEthRpc(c.GetConfig(),c,api)
	srv := &eth.RPCServer{server}
	s := &http.Server{Handler: srv}
	//默认绑定localhost
	bindAddr:=fmt.Sprintf("localhost:%d", eth.DefaultEthRpcPort)
	if c.GetConfig().GetModuleConfig().RPC.ErpcBindAddr!=""{
		bindAddr=c.GetConfig().GetModuleConfig().RPC.ErpcBindAddr
	}
	s.Addr=bindAddr
	l,err:= net.Listen("tcp",bindAddr)
	if err!=nil{
		panic(err)
	}
	go s.Serve(l)
	return   &EthRpcServer{s:server}
}

