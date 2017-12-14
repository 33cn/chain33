package p2p

import (
	"time"
)

var (
	updateState        = 2 * time.Second
	pingTimeout        = 14 * time.Second
	defaultSendTimeout = 10 * time.Second
	//p2pconf            types.P2P
)

const (
	MSG_TX                 = 1
	MSG_BLOCK              = 2
	numBufferedConnections = 25
	tryListenSeconds       = 5
	tryMapPortTimes        = 30
)

var (
	DefaultPort  = 13802
	LOCALADDR    string
	EXTERNALADDR string
)

const (
	DefalutP2PRemotePort = 14802
	Protocol             = "tcp"
	MixOutBoundNum       = 5
	MaxOutBoundNum       = 25
	StableBoundNum       = 5
	MaxAddrListNum       = 256
)

const (
	DialTimeout      = 15
	HandshakeTimeout = 20
)

const (
	NODE_NETWORK = 1
	NODE_GETUTXO = 2
	NODE_BLOOM   = 4
)

var (
	SERVICE int64 = NODE_BLOOM + NODE_NETWORK + NODE_GETUTXO
)
