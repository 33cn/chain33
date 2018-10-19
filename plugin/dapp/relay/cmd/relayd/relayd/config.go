package relayd

import (
	"io/ioutil"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/btcsuite/btcd/rpcclient"
	"gitlab.33.cn/chain33/chain33/types"
)

type Config struct {
	Title          string
	Watch          bool
	Tick33         int
	TickBTC        int
	BtcdOrWeb      int
	SyncSetup      uint64
	SyncSetupCount uint64
	Chain33        Chain33
	FirstBtcHeight uint64
	Btcd           Btcd
	Log            types.Log
	Auth           Auth
}

type Btcd struct {
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
	ReconnectAttempts    int
}

type Auth struct {
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
	Address    string `json:"address"`
}

func (b *Btcd) BitConnConfig() *rpcclient.ConnConfig {
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
	User                 string
	Pass                 string
	DisableAutoReconnect bool
	ReconnectAttempts    int
}

func NewConfig(path string) *Config {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		panic(err)
	}
	return &cfg
}
