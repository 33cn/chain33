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
	"path/filepath"
	"runtime"
	"time"

	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/config"
	"code.aliyun.com/chain33/chain33/common/limits"
	"code.aliyun.com/chain33/chain33/consensus"
	"code.aliyun.com/chain33/chain33/executor"
	"code.aliyun.com/chain33/chain33/mempool"
	"code.aliyun.com/chain33/chain33/p2p"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/rpc"
	"code.aliyun.com/chain33/chain33/store"
	"code.aliyun.com/chain33/chain33/wallet"
	log "github.com/inconshreveable/log15"
	"github.com/stackimpact/stackimpact-go"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	cpuNum     = runtime.NumCPU()
	configPath = flag.String("f", "chain33.toml", "configfile")
)

const Version = "v0.1.0"

func main() {
	d, _ := os.Getwd()
	log.Info("current dir:", "dir", d)
	//os.Chdir(pwd())
	//d, _ = os.Getwd()
	//log.Info("current dir:", "dir", d)
	//set file limit
	agent := stackimpact.Start(stackimpact.Options{
		AgentKey: "eb4c12dfe2d4b23b22634e7fed4d65899d5ca925",
		AppName:  "MyGoApp",
	})
	span := agent.Profile()
	defer span.Stop()
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	//set watching
	t := time.Tick(10 * time.Second)
	go func() {
		for range t {
			watching()
		}
	}()
	//set pprof
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	//set trace
	grpc.EnableTracing = true
	go startTrace()
	//set maxprocs
	runtime.GOMAXPROCS(cpuNum)

	flag.Parse()
	//set config
	cfg := config.InitCfg(*configPath)

	//set file log
	common.SetFileLog(cfg.LogFile, cfg.Loglevel, cfg.LogConsoleLevel)
	//set grpc log
	f, err := createFile(cfg.P2P.GetGrpcLogFile())
	if err != nil {
		glogv2 := grpclog.NewLoggerV2(os.Stdin, os.Stdin, os.Stderr)
		grpclog.SetLoggerV2(glogv2)
	} else {
		glogv2 := grpclog.NewLoggerV2(f, f, f)
		grpclog.SetLoggerV2(glogv2)
	}
	//开始区块链模块加载
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

	log.Info("loading execs module")
	exec := executor.New()
	exec.SetQueue(q)

	log.Info("loading store module")
	s := store.New(cfg.Store)
	s.SetQueue(q)

	log.Info("loading consensus module")
	cs := consensus.New(cfg.Consensus)
	cs.SetQueue(q)

	var network *p2p.P2p
	if cfg.P2P.Enable {
		log.Info("loading p2p module")
		network = p2p.New(cfg.P2P)
		network.SetQueue(q)
	}
	//jsonrpc, grpc, channel 三种模式
	gapi := rpc.NewGRpcServer(q.NewClient())
	go gapi.Listen(":8802")
	api := rpc.NewJsonRpcServer(q.NewClient())
	go api.Listen(":8801")

	log.Info("loading wallet module")
	walletm := wallet.New(cfg.Wallet)
	walletm.SetQueue(q)

	defer func() {
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
	}()
	q.Start()
}

// 开启trace
func startTrace() {
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		return true, true
	}
	go http.ListenAndServe(":50051", nil)
	log.Info("Trace listen on 50051")
}

func createFile(filename string) (*os.File, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func watching() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Info("info:", "NumGoroutine:", runtime.NumGoroutine())
	log.Info("info:", "Mem:", m.Sys/(1024*1024))
}

func pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return dir
}
