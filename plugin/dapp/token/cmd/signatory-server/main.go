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

	tml "github.com/BurntSushi/toml"
	l "github.com/inconshreveable/log15"
	"github.com/rs/cors"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/token/cmd/signatory-server/signatory"
)

var (
	log        = l.New("module", "signatory")
	configPath = flag.String("f", "signatory.toml", "configfile")
)

// 独立的服务， 提供两个功能
//   1. 帮忙做审核token的交易签名
//       1. a帐号有审核的权限， 客服核完需要用他的私钥对审核交易进行签名
//       1. 输入是生成好的， 选择输入 owner, symbol
//       1. 输出是签过名的交易
//   1. 给指定帐号打手续费    1bty
//       1. 指定帐号
//       1. 输出是签过名的交易
//  实现基于http 的 json rpc
//    app-proto
//      |
//      V
//     rpc
//      |
//      V
//     http     serve listener --> conn, io -> type HandlerFunc func(ResponseWriter, *Request)
//      |
//      V
//     tcp
type HTTPConn struct {
	in  io.Reader
	out io.Writer
}

func (c *HTTPConn) Read(p []byte) (n int, err error)  { return c.in.Read(p) }
func (c *HTTPConn) Write(d []byte) (n int, err error) { return c.out.Write(d) }
func (c *HTTPConn) Close() error                      { return nil }

func main() {
	d, _ := os.Getwd()
	log.Debug("current dir:", "dir", d)
	os.Chdir(pwd())
	d, _ = os.Getwd()
	log.Debug("current dir:", "dir", d)
	flag.Parse()
	cfg := InitCfg(*configPath)
	log.Debug("load config", "cfgPath", *configPath, "wl", cfg.Whitelist, "addr", cfg.JrpcBindAddr, "key", cfg.Privkey)
	whitelist := InitWhiteList(cfg)

	listen, err := net.Listen("tcp", cfg.JrpcBindAddr)
	if err != nil {
		panic(err)
	}

	approver := signatory.Signatory{cfg.Privkey}
	server := rpc.NewServer()
	server.Register(&approver)

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

func InitCfg(path string) *signatory.Config {
	var cfg signatory.Config
	if _, err := tml.DecodeFile(path, &cfg); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	fmt.Println(cfg)
	return &cfg
}

func InitWhiteList(cfg *signatory.Config) map[string]bool {
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
