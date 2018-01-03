package p2p

import (
	"fmt"
	"math/rand"
	"net"

	"code.aliyun.com/chain33/chain33/p2p/nat"
	pb "code.aliyun.com/chain33/chain33/types"

	"google.golang.org/grpc"
)

type Listener interface {
	Stop() bool
}

// Implements Listener
type DefaultListener struct {
	listener net.Listener
	server   *grpc.Server
	nodeInfo *NodeInfo
	c        chan struct{}
	n        *Node
}

func NewDefaultListener(protocol string, node *Node) Listener {
	// Create listener
	log.Debug("NewDefaultListener", "localPort", node.localPort)
	listener, err := net.Listen(protocol, fmt.Sprintf(":%v", node.localPort))
	if err != nil {
		log.Crit("Failed to listen", "Error", err.Error())
		return nil
	}

	var C = make(chan struct{})
	dl := &DefaultListener{
		listener: listener,
		nodeInfo: node.nodeInfo,
		c:        C,
		n:        node,
	}

	go dl.Start() // Started upon construction
	return dl
}

func (l *DefaultListener) Start() {
	log.Debug("defaultlistener", "localport:", l.nodeInfo.GetListenAddr().Port)
	go l.NatMapPort()
	go l.listenRoutine()
	return
}

func (l *DefaultListener) NatMapPort() {
	if l.nodeInfo.cfg.GetIsSeed() == true {

		return
	}

	for i := 0; i < TryMapPortTimes; i++ {

		if nat.Map(nat.Any(), l.c, "TCP", int(l.n.GetExterPort()), int(l.n.GetLocalPort()), "chain33 p2p") != nil {

			{

				l.nodeInfo.blacklist.Delete(l.nodeInfo.GetExternalAddr().String())
				l.n.externalPort = uint16(rand.Intn(64512) + 1023)
				l.n.flushNodeInfo()
				l.nodeInfo.blacklist.Add(l.n.nodeInfo.GetExternalAddr().String())
				log.Debug("NatMapPort", "Port", l.n.nodeInfo.GetExternalAddr())

			}

			continue
		}

	}
	l.n.externalPort = DefaultPort
	l.n.flushNodeInfo()
	log.Error("Nat Map", "Error Map Port Failed ----------------")

}
func (l *DefaultListener) Stop() bool {
	log.Debug("stop", "will close natport", l.nodeInfo.GetExternalAddr().Port, l.nodeInfo.GetListenAddr().Port)
	nat.Any().DeleteMapping(Protocol, int(l.nodeInfo.GetExternalAddr().Port), int(l.nodeInfo.GetListenAddr().Port))
	log.Debug("stop", "closed natport", "close")
	l.server.Stop()
	log.Debug("stop", "closed grpc server", "close")
	l.listener.Close()
	log.Debug("stop", "DefaultListener", "close")
	return true
}

func (l *DefaultListener) listenRoutine() {

	log.Debug("LISTENING", "Start Listening+++++++++++++++++Port", l.nodeInfo.listenAddr.Port)

	pServer := NewP2pServer()
	pServer.node = l.n
	go pServer.monitor()
	pServer.innerBroadBlock()
	l.server = grpc.NewServer()
	pb.RegisterP2PgserviceServer(l.server, pServer)
	l.server.Serve(l.listener)

}
