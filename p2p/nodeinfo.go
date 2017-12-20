package p2p

import (
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

type NodeInfo struct {
	pubKey       []byte      `json:"pub_key"`
	network      string      `json:"network"`
	externalAddr *NetAddress `json:"remote_addr"`
	listenAddr   *NetAddress `json:"listen_addr"`
	version      string      `json:"version"`
	monitorChan  chan *peer
	versionChan  chan struct{}
	p2pBlockChan chan *types.P2PBlock
	cfg          *types.P2P
	q            *queue.Queue
	qclient      queue.IClient
	other        []string `json:"other"` // other application specific data
}

func (b *NodeInfo) Set(n *NodeInfo) {
	b = n
}

func (b *NodeInfo) Get() *NodeInfo {
	return b
}
func (b *NodeInfo) SetExternalAddr(addr *NetAddress) {
	b.externalAddr = addr
}

func (b *NodeInfo) GetExternalAddr() *NetAddress {
	return b.externalAddr
}

func (b *NodeInfo) SetListenAddr(addr *NetAddress) {
	b.listenAddr = addr
}

func (b *NodeInfo) GetListenAddr() *NetAddress {
	return b.listenAddr
}
