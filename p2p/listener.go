package p2p

import (
	"fmt"
	"net"
	"time"

	pb "gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Listener interface {
	Close()
	Start()
}

func (l *listener) Start() {
	l.p2pserver.Start()
	go l.server.Serve(l.netlistener)

}
func (l *listener) Close() {
	l.p2pserver.Close()
	log.Info("stop", "listener", "close")

}

type listener struct {
	server      *grpc.Server
	nodeInfo    *NodeInfo
	p2pserver   *P2pServer
	node        *Node
	netlistener net.Listener
}

func NewListener(protocol string, node *Node) Listener {
	log.Debug("NewListener", "localPort", defaultPort)
	l, err := net.Listen(protocol, fmt.Sprintf(":%v", defaultPort))
	if err != nil {
		log.Crit("Failed to listen", "Error", err.Error())
		return nil
	}

	dl := &listener{
		nodeInfo:    node.nodeInfo,
		node:        node,
		netlistener: l,
	}

	pServer := NewP2pServer()
	pServer.node = dl.node

	msgRecvOp := grpc.MaxMsgSize(10 * 1024 * 1024)     //设置最大接收数据大小位10M
	msgSendOp := grpc.MaxSendMsgSize(10 * 1024 * 1024) //设置最大发送数据大小为10M

	//暂时不启用解压缩进行发送接收
	//compressOp := grpc.RPCCompressor(grpc.NewGZIPCompressor())       //设置grpc 采用gzip形式进行 压缩
	//decompressOp := grpc.RPCDecompressor(grpc.NewGZIPDecompressor()) //设置 grpc gzip 解压缩
	var keepparm keepalive.ServerParameters
	keepparm.Time = 100 * time.Second
	keepparm.Timeout = 5 * time.Second
	keepOp := grpc.KeepaliveParams(keepparm)

	dl.server = grpc.NewServer(msgRecvOp, msgSendOp,
		/*compressOp, decompressOp,*/ keepOp)
	dl.p2pserver = pServer
	pb.RegisterP2PgserviceServer(dl.server, pServer)
	return dl
}
