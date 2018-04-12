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
	checkSlowPeerInterVal      = 30 * time.Second
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
	defaultPort      = 13802
	defalutNatPort   = 23802
	mixOutBoundNum   = 5
	maxOutBoundNum   = 25
	stableBoundNum   = 15
	maxAddrListNum   = 256
	maxRangeBlockNum = 100
	maxAttemps       = 5
	protocol         = "tcp"
	externalPortTag  = "externalport"
)

const (
	nodeNetwork = 1
	nodeGetUTXO = 2
	nodeBloom   = 4
)

var (
	Service int64 = nodeBloom + nodeNetwork + nodeGetUTXO
	OutSide bool
)

// leveldb 中p2p privkey,addrkey
const (
	addrkeyTag = "addrs"
	privKeyTag = "privkey"
)
