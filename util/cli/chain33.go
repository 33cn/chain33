// +build go1.8

package cli

//说明：
//main 函数会加载各个模块，组合成区块链程序
//主循环由消息队列驱动。
//消息队列本身可插拔，可以支持各种队列
//同时共识模式也是可以插拔的。
//rpc 服务也是可以插拔的

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"time"

	"github.com/33cn/chain33/blockchain"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/limits"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/consensus"
	"github.com/33cn/chain33/executor"
	"github.com/33cn/chain33/mempool"
	"github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/rpc"
	"github.com/33cn/chain33/store"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/wallet"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
)

var (
	cpuNum     = runtime.NumCPU()
	configPath = flag.String("f", "", "configfile")
	datadir    = flag.String("datadir", "", "data dir of chain33, include logs and datas")
	versionCmd = flag.Bool("v", false, "version")
	fixtime    = flag.Bool("fixtime", false, "fix time")
)

func RunChain33(name string) {
	flag.Parse()
	if *versionCmd {
		fmt.Println(version.GetVersion())
		return
	}
	if *configPath == "" {
		if name == "" {
			*configPath = "chain33.toml"
		} else {
			*configPath = name + ".toml"
		}
	}
	d, _ := os.Getwd()
	log.Info("current dir:", "dir", d)
	os.Chdir(pwd())
	d, _ = os.Getwd()
	log.Info("current dir:", "dir", d)
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	//set config: bityuan 用 bityuan.toml 这个配置文件
	cfg, sub := types.InitCfg(*configPath)
	if *datadir != "" {
		resetDatadir(cfg, *datadir)
	}
	if *fixtime {
		cfg.FixTime = *fixtime
	}
	//set test net flag
	types.Init(cfg.Title, cfg)
	if cfg.FixTime {
		go fixtimeRoutine()
	}
	//compare minFee in wallet, mempool, exec
	//set file log
	clog.SetFileLog(cfg.Log)
	//set grpc log
	f, err := createFile(cfg.P2P.GrpcLogFile)
	if err != nil {
		glogv2 := grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
		grpclog.SetLoggerV2(glogv2)
	} else {
		glogv2 := grpclog.NewLoggerV2WithVerbosity(f, f, f, 10)
		grpclog.SetLoggerV2(glogv2)
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
		if cfg.Pprof != nil {
			http.ListenAndServe(cfg.Pprof.ListenAddr, nil)
		} else {
			http.ListenAndServe("localhost:6060", nil)
		}
	}()
	//set trace
	grpc.EnableTracing = true
	go startTrace()
	//set maxprocs
	runtime.GOMAXPROCS(cpuNum)
	//check mvcc switch，if use kvmvcc then cfg.Exec.EnableMVCC should be always false.
	/*todo
	if cfg.Store.Name == "kvmvcc" {
		if cfg.Exec.EnableMVCC {
			log.Error("store type is kvmvcc but enableMVCC is configured true.")
			panic("store type is kvmvcc, configure item enableMVCC should be false.please check it.")
		}
	}
	*/
	//开始区块链模块加载
	//channel, rabitmq 等
	log.Info(cfg.Title + " " + version.GetVersion())
	log.Info("loading queue")
	q := queue.New("channel")

	log.Info("loading mempool module")
	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())

	log.Info("loading execs module")
	exec := executor.New(cfg.Exec, sub.Exec)
	exec.SetQueueClient(q.Client())

	log.Info("loading store module")
	s := store.New(cfg.Store, sub.Store)
	s.SetQueueClient(q.Client())

	log.Info("loading blockchain module")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	chain.UpgradeChain()

	log.Info("loading consensus module")
	cs := consensus.New(cfg.Consensus, sub.Consensus)
	cs.SetQueueClient(q.Client())

	var network *p2p.P2p
	if cfg.P2P.Enable {
		log.Info("loading p2p module")
		network = p2p.New(cfg.P2P)
		network.SetQueueClient(q.Client())
	}
	//jsonrpc, grpc, channel 三种模式
	rpcapi := rpc.New(cfg.Rpc)
	rpcapi.SetQueueClient(q.Client())

	log.Info("loading wallet module")
	walletm := wallet.New(cfg.Wallet, sub.Wallet)
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
		log.Info("begin close rpc module")
		rpcapi.Close()
		log.Info("begin close wallet module")
		walletm.Close()
		log.Info("begin close queue module")
		q.Close()

	}()
	q.Start()
}

func resetDatadir(cfg *types.Config, datadir string) {
	// Check in case of paths like "/something/~/something/"
	if datadir[:2] == "~/" {
		usr, _ := user.Current()
		dir := usr.HomeDir
		datadir = filepath.Join(dir, datadir[2:])
	}
	log.Info("current user data dir is ", "dir", datadir)
	cfg.Log.LogFile = filepath.Join(datadir, cfg.Log.LogFile)
	cfg.BlockChain.DbPath = filepath.Join(datadir, cfg.BlockChain.DbPath)
	cfg.P2P.DbPath = filepath.Join(datadir, cfg.P2P.DbPath)
	cfg.Wallet.DbPath = filepath.Join(datadir, cfg.Wallet.DbPath)
	cfg.Store.DbPath = filepath.Join(datadir, cfg.Store.DbPath)
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
	log.Info("info:", "HeapAlloc:", m.HeapAlloc/(1024*1024))
}

func pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return dir
}

func fixtimeRoutine() {
	hosts := types.NtpHosts
	for i := 0; i < len(hosts); i++ {
		t, err := common.GetNtpTime(hosts[i])
		if err == nil {
			log.Info("time", "host", hosts[i], "now", t)
		} else {
			log.Error("time", "err", err)
		}
	}
	t := common.GetRealTimeRetry(hosts, 10)
	if !t.IsZero() {
		//update
		types.SetTimeDelta(int64(time.Until(t)))
		log.Info("change time", "delta", time.Until(t), "real.now", types.Now())
	}
	//时间请求频繁一点:
	ticket := time.NewTicker(time.Minute * 1)
	for range ticket.C {
		t = common.GetRealTimeRetry(hosts, 10)
		if !t.IsZero() {
			//update
			log.Info("change time", "delta", time.Until(t))
			types.SetTimeDelta(int64(time.Until(t)))
		}
	}
}
