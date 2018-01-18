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
	"os"
	"runtime"

	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/config"
	"code.aliyun.com/chain33/chain33/common/limits"
	"code.aliyun.com/chain33/chain33/consensus"
	"code.aliyun.com/chain33/chain33/execs"
	"code.aliyun.com/chain33/chain33/mempool"
	"code.aliyun.com/chain33/chain33/p2p"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/store"
	"code.aliyun.com/chain33/chain33/wallet"
	log "github.com/inconshreveable/log15"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	CPUNUM     = runtime.NumCPU()
	configpath = flag.String("f", "chain33.toml", "configfile")
)

const Version = "v0.1.0"

func main() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}

	runtime.GOMAXPROCS(CPUNUM)
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	grpc.EnableTracing = true
	go startTrace()
	flag.Parse()

	cfg := config.InitCfg(*configpath)
	common.SetFileLog(cfg.LogFile, cfg.Loglevel, cfg.LogConsoleLevel)

	f, err := CreateFile(cfg.P2P.GetGrpcLogFile())
	if err != nil {
		glogv2 := grpclog.NewLoggerV2(os.Stdin, os.Stdin, os.Stderr)
		grpclog.SetLoggerV2(glogv2)
	} else {
		glogv2 := grpclog.NewLoggerV2(f, f, f)
		grpclog.SetLoggerV2(glogv2)
	}
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

// 开启trace
func startTrace() {
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		return true, true
	}
	go http.ListenAndServe(":50051", nil)
	log.Error("Trace listen on 50051")
}

func CreateFile(filename string) (*os.File, error) {

	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}
