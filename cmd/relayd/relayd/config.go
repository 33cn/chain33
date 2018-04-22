package relayd

import (
	"github.com/BurntSushi/toml"
	"github.com/btcsuite/btcd/rpcclient"
	"io/ioutil"
	"path/filepath"
)

type Config struct {
	Title   string
	Chain33 []Chain33
	BitCoin []BitCoin
}

type BitCoin struct {
	id                   string
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
	id                   string
	Host                 string
	Endpoint             string
	User                 string
	Pass                 string
	DisableAutoReconnect bool
}

func config(path string) *Config {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		panic(err)
	}
	return &cfg
}
