package p2p

import (
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
)

func GetLocalAddr() string {

	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		log.Error(err.Error())
		return ""
	}

	defer conn.Close()
	log.Debug(strings.Split(conn.LocalAddr().String(), ":")[0])
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}

func Dial(addr *NetAddress, conf grpc.ServiceConfig) (*grpc.ClientConn, error) {
	conn, err := addr.DialTimeout(DialTimeout, conf)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func DialPeerWithAddress(addr *NetAddress, persistent bool, nodeinfo **NodeInfo) (*peer, error) {

	log.Debug("dialPeerWithAddress", "Dial peer address", addr.String())

	peer, err := NewOutboundPeer(addr, nodeinfo)
	if err != nil {
		log.Error("dialPeerWithAddress", "address", addr, "err", err)
		return nil, err
	}
	log.Debug("dialPeerWithAddress", "peer", *peer, "persistent:", persistent)

	if persistent {
		peer.makePersistent()
	}

	return peer, nil
}

//连接server out=往其他节点连接
func NewOutboundPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	conn, err := Dial(addr, (*nodeinfo).GrpcConfig())
	if err != nil {
		return nil, fmt.Errorf("Error creating peer")
	}

	peer, err := NewPeerFromConn(conn, true, addr, nodeinfo)
	if err != nil {
		conn.Close()
		return nil, err
	}
	peer.peerAddr = addr
	return peer, nil
}

func NewPeerFromConn(rawConn *grpc.ClientConn, outbound bool, remote *NetAddress, nodeinfo **NodeInfo) (*peer, error) {

	conn := rawConn
	// Key and NodeInfo are set after Handshake
	p := &peer{
		outbound:   outbound,
		conn:       conn,
		streamDone: make(chan struct{}, 1),
		nodeInfo:   nodeinfo,
	}
	p.version = new(Version)
	p.version.Set(true)
	p.setRunning(true)
	p.mconn = NewMConnection(conn, remote, p)

	return p, nil
}

func DialPeer(addr *NetAddress, nodeinfo **NodeInfo) (*peer, error) {
	log.Warn("DialPeer", "will connect", addr.String())
	var persistent bool
	for _, seed := range (*nodeinfo).cfg.Seeds { //TODO待优化
		if seed == addr.String() {
			persistent = true //种子节点要一直连接
		}
	}
	peer, err := DialPeerWithAddress(addr, persistent, nodeinfo)
	if err != nil {
		log.Error("DialPeer", "dial peer err:", err.Error())
		return nil, err
	}
	//获取远程节点的信息 peer
	log.Debug("DialPeer", "Peer info", peer)
	return peer, nil
}
