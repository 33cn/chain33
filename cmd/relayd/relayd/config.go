package relayd

import (
	"io/ioutil"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/btcsuite/btcd/rpcclient"
	"gitlab.33.cn/chain33/chain33/types"
)

type Config struct {
	Title             string
	Watch             bool
	Pprof             bool
	Trace             bool
	Heartbeat33       int
	ReconnectAttempts int
	HeartbeatBTC      int
	Chain33           Chain33
	BitCoin           BitCoin
	Log               types.Log
}

type BitCoin struct {
	Id                   string
	Host                 string
	Endpoint             string
	User                 string
	Pass                 string
	DisableTLS           bool
	CertPath             string
	Proxy                string
	ProxyUser            string
	ProxyPass            string
	DisableAutoReconnect bool
	DisableConnectOnNew  bool
	HTTPPostMode         bool
	EnableBCInfoHacks    bool
}

func (b *BitCoin) BitConnConfig() *rpcclient.ConnConfig {
	conn := &rpcclient.ConnConfig{}
	conn.Host = b.Host
	conn.Endpoint = b.Endpoint
	conn.User = b.User
	conn.Pass = b.Pass
	conn.DisableTLS = b.DisableTLS
	conn.Proxy = b.Proxy
	conn.ProxyUser = b.ProxyUser
	conn.ProxyPass = b.ProxyPass
	conn.DisableAutoReconnect = b.DisableAutoReconnect
	conn.DisableConnectOnNew = b.DisableConnectOnNew
	conn.HTTPPostMode = b.HTTPPostMode
	conn.EnableBCInfoHacks = b.EnableBCInfoHacks
	certs, err := ioutil.ReadFile(filepath.Join(b.CertPath, "rpc.cert"))
	if err != nil {
		panic(err)
	}
	conn.Certificates = certs
	return conn
}

type Chain33 struct {
	Id                   string
	Host                 string
	Endpoint             string
	User                 string
	Pass                 string
	DisableAutoReconnect bool
}

func NewConfig(path string) *Config {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		panic(err)
	}
	return &cfg
}
