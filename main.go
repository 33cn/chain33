package main

//说明：
//main 函数会加载各个模块，组合成区块链程序
//主循环由消息队列驱动。
//消息队列本身可插拔，可以支持各种队列
//同时共识模式也是可以插拔的。
//rpc 服务也是可以插拔的

import (
	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/consensus"
	"code.aliyun.com/chain33/chain33/execs"
	"code.aliyun.com/chain33/chain33/mempool"
	"code.aliyun.com/chain33/chain33/p2p"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/store"
	log "github.com/inconshreveable/log15"
)

const Version = "v0.0.2"

func main() {
	//channel, rabitmq 等
	log.Info("chain33 " + Version)
	log.Info("loading queue")
	q := queue.New("channel")

	log.Info("loading blockchain module")
	chain := blockchain.New()
	chain.SetQueue(q)

	log.Info("loading mempool module")
	mem := mempool.New()
	mem.SetQueue(q)

	log.Info("loading p2p module")
	network := p2p.New()
	network.SetQueue(q)

	log.Info("loading execs module")
	exec := execs.New()
	exec.SetQueue(q)

	log.Info("loading store module")
	s := store.New()
	s.SetQueue(q)

	log.Info("loading consensus module")
	consensus.New("solo", q)

	//jsonrpc, grpc, channel 三种模式
	api := rpc.NewServer("jsonrpc", ":8801")
	api.SetQueue(q)
	gapi := rpc.NewServer("grpc", ":8802")
	gapi.SetQueue(q)
	q.Start()
}
