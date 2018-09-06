package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"os/signal"
	"syscall"

	"github.com/tendermint/tendermint/libs/log"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/tools/tm-monitor/monitor"
)

var logger = log.NewNopLogger()

func main() {
	var listenAddr string

	flag.StringVar(&listenAddr, "listen-addr", "tcp://127.0.0.1:26670", "HTTP and Websocket server listen address")

	flag.Usage = func() {
		fmt.Println(`Tendermint monitor watches over one or more Tendermint core
applications, collecting and providing various statistics to the user.

Usage:
	tm-monitor [-listen-addr="tcp://0.0.0.0:26670"] [endpoints]

Examples:
	# monitor single instance
	tm-monitor localhost:8801

	# monitor a few instances by providing comma-separated list of RPC endpoints
	tm-monitor host1:8801,host2:8801

	# monitor a few instances by providing comma-separated list of RPC endpoints and want to open a listern-adr
	tm-monitor -listen-addr="tcp://host:26670" host1:8801,host2:8801`)
		//fmt.Println("Flags:")
		//flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	f, err := os.OpenFile("monitor.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		logger = log.NewTMLogger(log.NewSyncWriter(os.Stdout))
	} else {
		logger = log.NewTMLogger(f)
	}

	m := startMonitor(flag.Arg(0))
	startRPC(listenAddr, m, logger)

	var ton *Ton
	ton = NewTon(m)
	ton.Start()

	TrapSignal(func() {
		ton.Stop()
		m.Stop()
	})
}

func startMonitor(endpoints string) *monitor.Monitor {
	m := monitor.NewMonitor()
	m.SetLogger(logger)

	for _, e := range strings.Split(endpoints, ",") {
		n := monitor.NewNode(e)
		n.SetLogger(logger)
		if err := m.Monitor(n); err != nil {
			panic(err)
		}
	}

	//if err := m.Start(); err != nil {
	//	panic(err)
	//}

	return m
}

func TrapSignal(cb func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for sig := range c {
			fmt.Printf("captured %v, exiting...\n", sig)
			if cb != nil {
				cb()
			}
			os.Exit(1)
		}
	}()
	select {}
}
