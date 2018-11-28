// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package main 挖矿监控
package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"net/rpc/jsonrpc"
	"os"
	"path/filepath"
	"strings"

	l "github.com/33cn/chain33/common/log/log15"
	tml "github.com/BurntSushi/toml"
	"github.com/rs/cors"

	"github.com/33cn/chain33/cmd/miner_accounts/accounts"
)

var (
	log        = l.New("module", "main")
	configPath = flag.String("f", "miner_accounts.toml", "configfile")
)

//HTTPConn http连接
type HTTPConn struct {
	in  io.Reader
	out io.Writer
}

func (c *HTTPConn) Read(p []byte) (n int, err error)  { return c.in.Read(p) }
func (c *HTTPConn) Write(d []byte) (n int, err error) { return c.out.Write(d) }

//Close 关闭连接
func (c *HTTPConn) Close() error { return nil }

func main() {
	d, _ := os.Getwd()
	log.Debug("current dir:", "dir", d)
	os.Chdir(pwd())
	d, _ = os.Getwd()
	log.Debug("current dir:", "dir", d)
	flag.Parse()
	cfg := InitCfg(*configPath)
	log.Debug("load config", "cfgPath", *configPath, "wl", cfg.Whitelist, "addr", cfg.JrpcBindAddr, "miners", cfg.MinerAddr)
	whitelist := InitWhiteList(cfg)

	listen, err := net.Listen("tcp", cfg.JrpcBindAddr)
	if err != nil {
		panic(err)
	}

	go accounts.SyncBlock(cfg.Chain33Host)

	shower := accounts.ShowMinerAccount{DataDir: cfg.DataDir, Addrs: cfg.MinerAddr}
	server := rpc.NewServer()
	server.Register(&shower)

	var handler http.Handler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			//fmt.Println(r.URL, r.Header, r.Body)

			if !checkWhitlist(strings.Split(r.RemoteAddr, ":")[0], whitelist) {
				log.Error("HandlerFunc", "peer not whitelist", r.RemoteAddr)
				w.Write([]byte(`{"errcode":"-1","result":null,"msg":"reject"}`))
				return
			}

			if r.URL.Path == "/" {
				serverCodec := jsonrpc.NewServerCodec(&HTTPConn{in: r.Body, out: w})
				w.Header().Set("Content-type", "application/json")
				w.WriteHeader(200)

				err := server.ServeRequest(serverCodec)
				if err != nil {
					log.Debug("Error while serving JSON request: %v", err)
					return
				}
			}
		})

	//co := cors.New(cors.Options{
	//    AllowedOrigins: []string{"http://foo.com"},
	//    Debug: true,
	//})
	co := cors.New(cors.Options{})
	handler = co.Handler(handler)

	http.Serve(listen, handler)

	fmt.Println(handler)

}

//InitCfg 初始化cfg
func InitCfg(path string) *accounts.Config {
	var cfg accounts.Config
	if _, err := tml.DecodeFile(path, &cfg); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	fmt.Println(cfg)
	return &cfg
}

//InitWhiteList 初始化白名单
func InitWhiteList(cfg *accounts.Config) map[string]bool {
	whitelist := map[string]bool{}
	if len(cfg.Whitelist) == 1 && cfg.Whitelist[0] == "*" {
		whitelist["0.0.0.0"] = true
		return whitelist
	}

	for _, addr := range cfg.Whitelist {
		log.Debug("initWhitelist", "addr", addr)
		whitelist[addr] = true
	}
	return whitelist
}

func pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return dir
}

func checkWhitlist(addr string, whitlist map[string]bool) bool {
	if _, ok := whitlist["0.0.0.0"]; ok {
		return true
	}

	if _, ok := whitlist[addr]; ok {
		return true
	}
	return false
}
