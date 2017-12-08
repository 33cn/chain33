package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"code.aliyun.com/chain33/chain33/p2p/nat"
	"code.aliyun.com/chain33/chain33/queue"
	pb "code.aliyun.com/chain33/chain33/types"

	"google.golang.org/grpc"
)

type Listener interface {
	Stop() bool
}

// Implements Listener
type DefaultListener struct {
	listener net.Listener
	nodeInfo *NodeBase
	q        *queue.Queue
	c        chan struct{}
	n        *Node
}

func NewDefaultListener(protocol string, node *Node) Listener {
	// Create listener
	listener, err := net.Listen(protocol, fmt.Sprintf(":%v", node.localPort))
	if err != nil {
		log.Crit("Failed to listen", "Error", err.Error())
		return nil
	}

	var C = make(chan struct{})
	dl := &DefaultListener{
		listener: listener,
		nodeInfo: node.nodeInfo,
		q:        node.q,
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
	for i := 0; i < tryMapPortTimes; i++ {

		if nat.Map(nat.Any(), l.c, "TCP", int(l.nodeInfo.GetExternalAddr().Port), int(l.nodeInfo.GetListenAddr().Port), "chain33 p2p") != nil {

			{
				l.n.localPort = uint16(rand.Intn(64512) + 1023)
				l.n.externalPort = uint16(rand.Intn(64512) + 1023)
				l.n.flushNodeInfo()
				log.Debug("newLocalPort", "Port", l.n.localPort)
				log.Debug("newExternalPort", "Port", l.n.externalPort)
			}

			continue
		}

	}
	//TODO
	//MAP FAILED
	log.Error("Nat Map", "Error Map Port Failed ----------------")

}
func (l *DefaultListener) Stop() bool {
	nat.Any().DeleteMapping(Protocol, int(l.nodeInfo.GetExternalAddr().Port), int(l.nodeInfo.GetListenAddr().Port))
	l.listener.Close()
	return true
}

func (l *DefaultListener) listenRoutine() {

	log.Debug("LISTENING", "Start Listening+++++++++++++++++Port", l.nodeInfo.listenAddr.Port)

	pServer := NewP2pServer()
	pServer.q = l.q
	pServer.book = l.n.addrBook
	pServer.OutBound = l.n.outBound
	pServer.nodeinfo = l.nodeInfo
	pServer.Monitor()
	server := grpc.NewServer()
	pb.RegisterP2PgserviceServer(server, pServer)
	server.Serve(l.listener)

}

func initAddr(cfg *pb.P2P) {

	LOCALADDR = localBindAddr()
	log.Debug("LOCALADDR", "addr:", LOCALADDR)
	// Determine external address...
	for i := 0; i < 30; i++ {
		if cfg.GetIsSeed() {
			EXTERNALADDR = LOCALADDR
			return
		}
		exnet, err := nat.Any().ExternalIP()
		if err == nil {
			EXTERNALADDR = exnet.String()
			log.Debug("EXTERNALADDR", "Addr", EXTERNALADDR)
			break
		} else {
			log.Error("ExternalIp", "Error", err.Error())
		}
	}
	//获取外网失败，默认本地与外网地址一致
	//确认自己的服务范围1，2，4
	go func() {

		defer log.Debug("Addrs", "LocalAddr", LOCALADDR, "ExternalAddr", EXTERNALADDR)
		time.Sleep(time.Second * 10)
		serveraddr := strings.Split(cfg.Seeds[0], ":")[0]
		var trytimes uint = 3
		for {
			selfexaddrs := getSelfExternalAddr(fmt.Sprintf("%v:%v", serveraddr, DefalutP2PRemotePort))
			if len(selfexaddrs) != 0 {
				log.Debug("GetSelfExternalAddr", "Addr", selfexaddrs[0])
				if selfexaddrs[0] != EXTERNALADDR && selfexaddrs[0] != LOCALADDR {
					SERVICE -= NODE_NETWORK
				}
				EXTERNALADDR = selfexaddrs[0]
				break
			}
			trytimes--
			if trytimes == 0 {
				break
			}
			time.Sleep(time.Second * 2)
		}
		//如果nat,getSelfExternalAddr 无法发现自己的外网地址，则把localaddr 赋值给外网地址
		if len(EXTERNALADDR) == 0 {
			EXTERNALADDR = LOCALADDR
		}
	}()

}
