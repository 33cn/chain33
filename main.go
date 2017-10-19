package main

//说明：
//main 函数会加载各个模块，组合成区块链程序
//主循环由消息队列驱动。
//消息队列本身可插拔，可以支持各种队列
//同时共识模式也是可以插拔的。
//rpc 服务也是可以插拔的

import (
	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/consense"
	"code.aliyun.com/chain33/chain33/mempool"
	"code.aliyun.com/chain33/chain33/p2p"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

func main() {
	//channel, rabitmq 等
	log.Info("loading queue")
	q := queue.New("channel")

	log.Info("loading blockchain module")
	chain := blockchain.New()
	chain.SetQueue(q)

	log.Info("loading blockchain module")
	con := consense.New("raft")
	con.SetQueue(q)

	mem := mempool.New()
	mem.SetQueue(q)

	network := p2p.New()
	network.SetQueue(q)

	//jsonrpc, grpc, channel 三种模式
    api := rpc.NewServer("jsonrpc", ":8801")
	api.SetQueue(q)
	gapi := rpc.NewServer("grpc", ":8802")
	gapi.SetQueue(q)

	go func() {
		//jsonrpc, grpc, channel 三种模式
		client := rpc.NewClient("channel", "")
		//同步接口
		client.SetQueue(q)
		//tx := &types.Transaction{}
		//client.SendTx(tx)
	}()
	q.Start()
}
