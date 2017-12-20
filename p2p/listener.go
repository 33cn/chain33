package p2p

import (
	"fmt"
	"math/rand"
	"net"

	"strings"
	"time"

	"code.aliyun.com/chain33/chain33/p2p/nat"
	//"code.aliyun.com/chain33/chain33/queue"
	pb "code.aliyun.com/chain33/chain33/types"

	"google.golang.org/grpc"
)

type Listener interface {
	Stop() bool
}

// Implements Listener
type DefaultListener struct {
	listener net.Listener
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

		if nat.Map(nat.Any(), l.c, "TCP", int(l.nodeInfo.GetExternalAddr().Port), int(l.nodeInfo.GetListenAddr().Port), "chain33 p2p") != nil {

			{
				l.n.localPort = uint16(rand.Intn(64512) + 1023)
				l.n.externalPort = uint16(rand.Intn(64512) + 1023)
				l.n.flushNodeInfo()
				log.Debug("NatMapPort", "Port", l.n.localPort)
				log.Debug("NatMapPort", "Port", l.n.externalPort)
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
	pServer.node = l.n
	pServer.monitor()
	pServer.innerBroadBlock()
	server := grpc.NewServer()
	pb.RegisterP2PgserviceServer(server, pServer)
	server.Serve(l.listener)

}

func DetectionNodeAddr(cfg *pb.P2P) {

	LOCALADDR = localBindAddr()
	log.Debug("LOCALADDR", "addr:", LOCALADDR)
	if cfg.GetIsSeed() {
		EXTERNALADDR = LOCALADDR
	}

	if len(cfg.Seeds) == 0 {
		return
	}
	//TODO 增加校验IP格式的方法,目前临时方法，不完善
	if strings.Contains(cfg.Seeds[0], ":") == false {
		return
	}
	serveraddr := strings.Split(cfg.Seeds[0], ":")[0]
	var trytimes uint = 3
	for {
		selfexaddrs := getSelfExternalAddr(fmt.Sprintf("%v:%v", serveraddr, DefalutP2PRemotePort))
		if len(selfexaddrs) != 0 {
			log.Debug("getSelfExternalAddr", "Addr", selfexaddrs[0])
			EXTERNALADDR = selfexaddrs[0]
			log.Debug("DetectionNodeAddr", "LocalAddr", LOCALADDR, "ExternalAddr", EXTERNALADDR)
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
}
