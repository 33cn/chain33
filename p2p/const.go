package p2p

import (
	"time"
)

var (
	UpdateState        = 2 * time.Second
	PingTimeout        = 14 * time.Second
	DefaultSendTimeout = 10 * time.Second
	DialTimeout        = 5 * time.Second
	mapUpdateInterval  = 15 * time.Minute
	StreamPingTimeout  = 20 * time.Second
)

const (
	MSG_TX          = 1
	MSG_BLOCK       = 2
	TryMapPortTimes = 20
)

var (
	LocalAddr  string
	ExternalIp string
)

const (
	DefaultPort          = 13802
	DefalutP2PRemotePort = 14802
	Protocol             = "tcp"
	MixOutBoundNum       = 5
	MaxOutBoundNum       = 25
	StableBoundNum       = 15
	MaxAddrListNum       = 256
	MaxRangeBlockNum     = 100
)

const (
	NODE_NETWORK = 1
	NODE_GETUTXO = 2
	NODE_BLOOM   = 4
)

const (
	VersionNotSupport = "Version No Support"
)

var (
	SERVICE   int64 = NODE_BLOOM + NODE_NETWORK + NODE_GETUTXO
	IsOutSide bool
)
