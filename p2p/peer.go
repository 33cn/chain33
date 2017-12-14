package p2p

import (
	"fmt"
	"time"

	"google.golang.org/grpc"
)

type Peer interface {

	//	IsOutbound() bool
	//	IsPersistent() bool
	//NodeInfo() *NodeInfo
	//	Send(byte, interface{}) bool
}

type peer struct {
	nodeInfo   **NodeInfo
	outbound   bool
	conn       *grpc.ClientConn // source connection
	persistent bool
	key        string
	mconn      *MConnection
	peerAddr   *NetAddress
	Data       map[string]interface{}
}

func (p *peer) Start() error {
	p.mconn.key = p.key
	err := p.mconn.Start()
	return err
}

func (p *peer) Stop() {
	p.mconn.Stop()

}

// makePersistent marks the peer as persistent.
func (p *peer) makePersistent() {
	if !p.outbound {
		panic("inbound peers can't be made persistent")
	}
	p.persistent = true
}

// Addr returns peer's remote network address.
func (p *peer) Addr() string {
	return p.peerAddr.String()
}

// IsPersistent returns true if the peer is persitent, false otherwise.
func (p *peer) IsPersistent() bool {
	return p.persistent
}

func dial(addr *NetAddress) (*grpc.ClientConn, error) {
	conn, err := addr.DialTimeout(DialTimeout * time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func dialPeerWithAddress(addr *NetAddress, persistent bool, nodeinfo **NodeInfo) (*peer, error) {

	log.Debug("DialPerrWithAddress", "Dial peer address", addr.String())

	peer, err := newOutboundPeer(addr, nodeinfo)
	if err != nil {
		log.Error("Failed to dial peer", "address", addr, "err", err)
		return nil, err
	}
	log.Debug("newOutboundpeer", "peer", *peer, "persistent:", persistent)

	if persistent {
		peer.makePersistent()
	}

	return peer, nil
}

//连接server out=往其他节点连接
func newOutboundPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	conn, err := dial(addr)
	if err != nil {
		return nil, fmt.Errorf("Error creating peer")
	}

	peer, err := newPeerFromConn(conn, true, addr, nodeinfo)
	if err != nil {
		conn.Close()
		return nil, err
	}
	peer.peerAddr = addr
	return peer, nil
}

func newPeerFromConn(rawConn *grpc.ClientConn, outbound bool, remote *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	conn := rawConn

	// Key and NodeInfo are set after Handshake
	p := &peer{
		outbound: outbound,
		conn:     conn,
		Data:     make(map[string]interface{}),
		nodeInfo: nodeinfo,
	}
	p.mconn = NewMConnection(conn, remote, p)

	return p, nil
}
func DialPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {
	log.Debug("dialPeer", "peer addr", addr.String())
	var persistent bool
	for _, seed := range (*nodeinfo).cfg.Seeds { //TODO待优化
		if seed == addr.String() {
			persistent = true //种子节点要一直连接
		}
	}
	peer, err := dialPeerWithAddress(addr, persistent, nodeinfo)
	if err != nil {
		log.Error("DialPeerWithAddress", "dial peer err:", err.Error())
		return nil, err
	}
	//获取远程节点的信息 peer
	log.Debug("Dial peer Success", "Peer", peer)
	return peer, nil
}
