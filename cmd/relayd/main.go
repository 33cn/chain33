package main

import (
	"flag"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/limits"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

var (
	cpuNum     = runtime.NumCPU()
	configPath = flag.String("f", "relayd.toml", "configfile")
)

func main() {
	//set maxprocs
	runtime.GOMAXPROCS(cpuNum)

	d, _ := os.Getwd()
	log.Info("current dir:", "dir", d)
	os.Chdir(pwd())
	d, _ = os.Getwd()
	log.Info("current dir:", "dir", d)
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}

	flag.Parse()
	cfg := config.InitCfg(*configPath)
	clog.SetFileLog(cfg.Log)
	//f, err := createFile(cfg.P2P.GetGrpcLogFile())

	//set watching
	t := time.Tick(10 * time.Second)
	go func() {
		for range t {
			watching()
		}
	}()

	//set pprof
	go func() {
		http.ListenAndServe("localhost:6061", nil)
	}()

	//set trace
	grpc.EnableTracing = true
	go startTrace()


}

// 开启trace
func startTrace() {
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		return true, true
	}
	go http.ListenAndServe("localhost:50052", nil)
	log.Info("Trace listen on localhost:50052")
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
