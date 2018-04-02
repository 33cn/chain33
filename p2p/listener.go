package p2p

import (
	"fmt"
	"net"

	pb "code.aliyun.com/chain33/chain33/types"

	"google.golang.org/grpc"
)

type Listener interface {
	Close() bool
}

func (l *listener) Close() bool {
	l.p2pserver.Close()
	log.Info("stop", "listener", "close")
	return true
}

type listener struct {
	server    *grpc.Server
	nodeInfo  *NodeInfo
	p2pserver *p2pServer
	node      *Node
}

func NewListener(protocol string, node *Node) Listener {
	log.Debug("NewListener", "localPort", DefaultPort)
	l, err := net.Listen(protocol, fmt.Sprintf(":%v", DefaultPort))
	if err != nil {
		log.Crit("Failed to listen", "Error", err.Error())
		return nil
	}

	dl := &listener{
		nodeInfo: node.nodeInfo,
		node:     node,
	}
	pServer := NewP2pServer()
	pServer.node = dl.node
	pServer.Start()
	dl.server = grpc.NewServer()
	dl.p2pserver = pServer
	pb.RegisterP2PgserviceServer(dl.server, pServer)
	go dl.server.Serve(l)
	return dl
}
