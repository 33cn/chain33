package p2p

import (
	"time"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
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
	pbase        *NodeBase
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

type NodeBase struct {
	pubKey       []byte      `json:"pub_key"`
	network      string      `json:"network"`
	externalAddr *NetAddress `json:"remote_addr"`
	listenAddr   *NetAddress `json:"listen_addr"`
	version      string      `json:"version"` // major.minor.revision
	monitorChan  chan *peer
	cfg          *types.P2P
	q            *queue.Queue
	other        []string `json:"other"` // other application specific data
}

func (b *NodeBase) Set(n *NodeBase) {
	b = n
}

func (b *NodeBase) Get() *NodeBase {
	return b
}
func (b *NodeBase) SetExternalAddr(addr *NetAddress) {
	b.externalAddr = addr
}

func (b *NodeBase) GetExternalAddr() *NetAddress {
	return b.externalAddr
}

func (b *NodeBase) SetListenAddr(addr *NetAddress) {
	b.listenAddr = addr
}

func (b *NodeBase) GetListenAddr() *NetAddress {
	return b.listenAddr
}
