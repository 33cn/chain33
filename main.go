package main

//说明：
//main 函数会加载各个模块，组合成区块链程序
//主循环由消息队列驱动。
//消息队列本身可插拔，可以支持各种队列
//同时共识模式也是可以插拔的。
//rpc 服务也是可以插拔的

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"runtime"

	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/config"
	"code.aliyun.com/chain33/chain33/consensus"
	"code.aliyun.com/chain33/chain33/execs"
	"code.aliyun.com/chain33/chain33/mempool"
	"code.aliyun.com/chain33/chain33/p2p"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/store"
	"code.aliyun.com/chain33/chain33/wallet"
	log "github.com/inconshreveable/log15"
)

var (
	CPUNUM     = runtime.NumCPU()
	configpath = flag.String("f", "chain33.toml", "configfile")
)

const Version = "v0.1.0"

func main() {

	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	flag.Parse()

	cfg := config.InitCfg(*configpath)
	common.SetFileLog(cfg.LogFile, cfg.Loglevel)
	//channel, rabitmq 等
	log.Info("chain33 " + Version)
	log.Info("loading queue")
	q := queue.New("channel")

	log.Info("loading blockchain module")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueue(q)
	log.Info("loading mempool module")
	mem := mempool.New(cfg.MemPool)
	mem.SetQueue(q)

	var network *p2p.P2p
	if cfg.P2P.Enable {
		log.Info("loading p2p module")
		network = p2p.New(cfg.P2P)
		network.SetQueue(q)
	}

	log.Info("loading execs module")
	exec := execs.New()
	exec.SetQueue(q)

	log.Info("loading store module")
	s := store.New(cfg.Store)
	s.SetQueue(q)

	log.Info("loading consensus module")
	cs := consensus.New(cfg.Consensus)
	cs.SetQueue(q)

	log.Info("loading wallet module")
	walletm := wallet.New(cfg.Wallet)
	walletm.SetQueue(q)

	//jsonrpc, grpc, channel 三种模式
	api := rpc.NewServer("jsonrpc", ":8801", q)
	//api.SetQueue(q)
	gapi := rpc.NewServer("grpc", ":8802", q)
	//gapi.SetQueue(q)
	q.Start()

	//close all module,clean some resource
	log.Info("begin close blockchain module")
	chain.Close()
	log.Info("begin close mempool module")
	mem.Close()
	if cfg.P2P.Enable {
		log.Info("begin close P2P module")
		network.Close()
	}
	log.Info("begin close execs module")
	exec.Close()
	log.Info("begin close store module")
	s.Close()
	log.Info("begin close consensus module")
	cs.Close()
	log.Info("begin close jsonrpc module")
	api.Close()
	log.Info("begin close grpc module")
	gapi.Close()
	log.Info("begin close queue module")
	q.Close()
	log.Info("begin close wallet module")
	walletm.Close()
}
