package p2p

import (
	"time"
)

var (
	UpdateState                = 2 * time.Second
	PingTimeout                = 14 * time.Second
	DefaultSendTimeout         = 10 * time.Second
	DialTimeout                = 5 * time.Second
	mapUpdateInterval          = 15 * time.Minute
	StreamPingTimeout          = 20 * time.Second
	MonitorPeerInfoInterval    = 10 * time.Second
	GetAddrFromOfflineInterval = 5 * time.Second
	GetAddrFromOnlineInterval  = 5 * time.Second
	CheckActivePeersInterVal   = 5 * time.Second
	CheckBlackListInterVal     = 30 * time.Second
)

const (
	msgTx           = 1
	msgBlock        = 2
	tryMapPortTimes = 20
)

var (
	LocalAddr string
)

const (
	defaultPort     = 13802
	defalutNatPort  = 23802
	maxOutBoundNum  = 25
	stableBoundNum  = 15
	maxAddrListNum  = 256
	maxAttemps      = 5
	protocol        = "tcp"
	externalPortTag = "externalport"
)

const (
	nodeNetwork = 1
	nodeGetUTXO = 2
	nodeBloom   = 4
)

const (
	Service int32 = nodeBloom + nodeNetwork + nodeGetUTXO
)

// leveldb 中p2p privkey,addrkey
const (
	addrkeyTag = "addrs"
	privKeyTag = "privkey"
)

const (
	P2pCacheTxSize = 102400
)
