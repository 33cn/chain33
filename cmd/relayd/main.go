package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/cmd/relayd/relayd"
	"gitlab.33.cn/chain33/chain33/common/limits"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"golang.org/x/net/trace"
	"google.golang.org/grpc"
)

var (
	cpuNum     = runtime.NumCPU()
	configPath = flag.String("f", "relayd.toml", "configfile")
)

func main() {
	// TODO this is daemon

	runtime.GOMAXPROCS(cpuNum)
	os.Chdir(pwd())
	d, _ := os.Getwd()
	log.Info("current dir:", "dir", d)

	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}

	flag.Parse()
	cfg := relayd.NewConfig(*configPath)
	clog.SetFileLog(&cfg.Log)

	if cfg.Watch {
		// set watching
		log.Info("watching 10s")
		t := time.Tick(10 * time.Second)
		go func() {
			for range t {
				watching()
			}
		}()
	}

	if cfg.Watch {
		// set pprof
		log.Info("pprof localhost:6068")
		go func() {
			http.ListenAndServe("localhost:6068", nil)
		}()
	}

	if cfg.Trace {
		// set trace
		log.Info("listen on localhost:50055")
		grpc.EnableTracing = true
		go startTrace()
	}

	r := relayd.NewRelayd(cfg)
	go r.Start()

	interrupt := make(chan os.Signal, 2)
	signal.Notify(interrupt, os.Interrupt, os.Kill)
	s := <-interrupt
	log.Warn("Got signal:", "signal", s)

	r.Close()
}

func startTrace() {
	trace.AuthRequest = func(req *http.Request) (any, sensitive bool) {
		return true, true
	}
	go http.ListenAndServe("localhost:50055", nil)
}

func watching() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	log.Info("watching:", "NumGoroutine:", runtime.NumGoroutine())
	log.Info("watching:", "Mem:", m.Sys/(1024*1024))
}

func pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return dir
}
