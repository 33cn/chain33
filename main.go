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

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/limits"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/wallet"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	_ "google.golang.org/grpc/encoding/gzip"
)

var (
	cpuNum     = runtime.NumCPU()
	configPath = flag.String("f", "chain33.toml", "configfile")
)

func main() {
	d, _ := os.Getwd()
	log.Info("current dir:", "dir", d)
	os.Chdir(pwd())
	d, _ = os.Getwd()
	log.Info("current dir:", "dir", d)
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
	//compare minFee in wallet, mempool, exec
	if cfg.Exec.MinExecFee > cfg.MemPool.MinTxFee || cfg.MemPool.MinTxFee > cfg.Wallet.MinFee {
		panic("config must meet: wallet.minFee >= mempool.minTxFee >= exec.minExecFee")
	}

	//set file log
	clog.SetFileLog(cfg.Log)
	//set grpc log
	f, err := createFile(cfg.P2P.GetGrpcLogFile())
	if err != nil {
		glogv2 := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
		grpclog.SetLoggerV2(glogv2)
	} else {
		glogv2 := grpclog.NewLoggerV2WithVerbosity(f, f, f, 10)
		grpclog.SetLoggerV2(glogv2)
	}
	//开始区块链模块加载
	//channel, rabitmq 等
	log.Info("chain33 " + version.GetVersion())
	log.Info("loading queue")
	q := queue.New("channel")

	log.Info("loading blockchain module")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	log.Info("loading mempool module")
	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())

	log.Info("loading execs module")
	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())

	log.Info("loading store module")
	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	log.Info("loading consensus module")
	cs := consensus.New(cfg.Consensus)
	cs.SetQueueClient(q.Client())

	var network *p2p.P2p
	if cfg.P2P.Enable {
		log.Info("loading p2p module")
		network = p2p.New(cfg.P2P)
		network.SetQueueClient(q.Client())
	}
	//jsonrpc, grpc, channel 三种模式
	rpc.Init(cfg.Rpc)
	gapi := rpc.NewGRpcServer(q.Client())
	go gapi.Listen()
	japi := rpc.NewJsonRpcServer(q.Client())
	go japi.Listen()

	log.Info("loading wallet module")
	walletm := wallet.New(cfg.Wallet)
	walletm.SetQueueClient(q.Client())

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
		japi.Close()
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
	go http.ListenAndServe("localhost:50051", nil)
	log.Info("Trace listen on localhost:50051")
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
