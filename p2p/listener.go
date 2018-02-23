package p2p

import (
	"fmt"
	"net"

	pb "code.aliyun.com/chain33/chain33/types"

	"google.golang.org/grpc"
)

func (l *DefaultListener) Close() bool {

	l.listener.Close()
	l.server.Stop()
	log.Info("stop", "DefaultListener", "close")
	close(l.nodeInfo.p2pBroadcastChan) //close p2pserver manageStream
	close(l.p2pserver.loopdone)
	return true
}

type Listener interface {
	Close() bool
}

// Implements Listener
type DefaultListener struct {
	listener  net.Listener
	server    *grpc.Server
	nodeInfo  *NodeInfo
	p2pserver *p2pServer
	n         *Node
}

func NewDefaultListener(protocol string, node *Node) Listener {

	log.Debug("NewDefaultListener", "localPort", DefaultPort)
	listener, err := net.Listen(protocol, fmt.Sprintf(":%v", DefaultPort))
	if err != nil {
		log.Crit("Failed to listen", "Error", err.Error())
		return nil
	}

	dl := &DefaultListener{
		listener: listener,
		nodeInfo: node.nodeInfo,
		n:        node,
	}
	pServer := NewP2pServer()
	pServer.node = dl.n
	pServer.ManageStream()
	dl.server = grpc.NewServer()
	dl.p2pserver = pServer
	pb.RegisterP2PgserviceServer(dl.server, pServer)
	go dl.listenRoutine()
	return dl
}

func (l *DefaultListener) listenRoutine() {
	l.server.Serve(l.listener)
}
