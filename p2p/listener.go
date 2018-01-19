package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	"code.aliyun.com/chain33/chain33/p2p/nat"
	pb "code.aliyun.com/chain33/chain33/types"

	"google.golang.org/grpc"
)

func (l *DefaultListener) Close() bool {
	log.Debug("stop", "will close natport", l.nodeInfo.GetExternalAddr().Port, l.nodeInfo.GetListenAddr().Port)
	log.Debug("stop", "closed natport", "close")
	l.listener.Close()
	log.Error("stop", "DefaultListener", "close")
	return true
}

type Listener interface {
	Close() bool
}

// Implements Listener
type DefaultListener struct {
	listener net.Listener
	server   *grpc.Server
	nodeInfo *NodeInfo
	natClose chan struct{}
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

	dl := &DefaultListener{
		listener: listener,
		nodeInfo: node.nodeInfo,
		natClose: make(chan struct{}, 1),
		n:        node,
	}
	pServer := NewP2pServer()
	pServer.node = dl.n
	pServer.innerBroadBlock()
	dl.server = grpc.NewServer()
	pb.RegisterP2PgserviceServer(dl.server, pServer)
	go dl.NatMapPort()
	go dl.listenRoutine()
	return dl
}

func (l *DefaultListener) WaitForNat() {
	<-l.nodeInfo.natNoticeChain
	return
}

func (l *DefaultListener) NatMapPort() {
	if l.nodeInfo.cfg.GetIsSeed() == true {

		return
	}
	l.WaitForNat()
	var err error
	for i := 0; i < TryMapPortTimes; i++ {

		err = nat.Any().AddMapping("TCP", int(l.n.GetExterPort()), int(l.n.GetLocalPort()), "chain33-p2p", time.Minute*20)
		if err != nil {
			log.Error("NatMapPort", "err", err.Error())
			l.n.externalPort = uint16(rand.Intn(64512) + 1023)
			l.n.FlushNodeInfo()
			log.Info("NatMapPort", "Port", l.n.nodeInfo.GetExternalAddr())
			continue
		}

		break
	}
	if err != nil {
		//映射失败
		log.Error("NatMapPort", "Nat Faild", "Sevice6")
		l.nodeInfo.natResultChain <- false
		return
	}

	l.nodeInfo.natResultChain <- true
	refresh := time.NewTimer(mapUpdateInterval)
	for {
		select {
		case <-refresh.C:
			nat.Any().AddMapping("TCP", int(l.n.GetExterPort()), int(l.n.GetLocalPort()), "chain33-p2p", time.Minute*20)
			refresh.Reset(mapUpdateInterval)

		case <-l.natClose:
			nat.Any().DeleteMapping("TCP", int(l.n.GetExterPort()), int(l.n.GetLocalPort()))
			return
		}
	}

}

func (l *DefaultListener) listenRoutine() {
	log.Debug("LISTENING", "Start Listening+++++++++++++++++Port", l.nodeInfo.listenAddr.Port)
	l.server.Serve(l.listener)
}
