package p2p

import (
	"fmt"
	"math/rand"

	"os"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/p2p/nat"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

// 启动Node节点
//1.启动监听GRPC Server
//2.初始化AddrBook,并发起对相应地址的连接
//3.如果配置了种子节点，则连接种子节点
//4.启动监控远程节点
func (n *Node) Start() {
	n.DetectionNodeAddr()
	n.l = NewDefaultListener(Protocol, n)
	go n.monitor()
	go n.exChangeVersion()

	return
}

func (n *Node) Close() {

	close(n.loopDone)
	log.Debug("stop", "versionDone", "close")
	n.l.Close()
	log.Debug("stop", "listen", "close")
	n.addrBook.Close()
	log.Debug("stop", "addrBook", "close")
	n.RemoveAll()
	log.Debug("stop", "PeerRemoeAll", "close")

}

type Node struct {
	omtx         sync.Mutex
	portmtx      sync.Mutex
	addrBook     *AddrBook // known peers
	nodeInfo     *NodeInfo
	localAddr    string
	extaddr      string
	localPort    uint16 //listen
	externalPort uint16 //nat map
	outBound     map[string]*peer
	l            Listener
	loopDone     chan struct{}
}

func (n *Node) setQueue(q *queue.Queue) {
	n.nodeInfo.q = q
	n.nodeInfo.qclient = q.NewClient()
}

func NewNode(cfg *types.P2P) (*Node, error) {
	os.MkdirAll(cfg.GetDbPath(), 0755)
	rand.Seed(time.Now().Unix())
	node := &Node{
		localPort:    DefaultPort, //uint16(rand.Intn(64512) + 1023),
		externalPort: uint16(rand.Intn(64512) + 1023),
		outBound:     make(map[string]*peer),
		addrBook:     NewAddrBook(cfg.GetDbPath() + "/addrbook.json"),
		loopDone:     make(chan struct{}, 1),
	}

	log.Debug("newNode", "externalPort", node.externalPort)

	if cfg.GetIsSeed() == true {
		node.SetPort(DefaultPort, DefaultPort)

	}
	node.nodeInfo = new(NodeInfo)
	node.nodeInfo.monitorChan = make(chan *peer, 1024)
	node.nodeInfo.natNoticeChain = make(chan struct{}, 1)
	node.nodeInfo.natResultChain = make(chan bool, 1)
	node.nodeInfo.p2pBroadcastChan = make(chan interface{}, 4096)
	node.nodeInfo.blacklist = &BlackList{badPeers: make(map[string]bool)}
	node.nodeInfo.cfg = cfg
	node.nodeInfo.peerInfos = new(PeerInfos)
	node.nodeInfo.peerInfos.infos = make(map[string]*types.Peer)

	return node, nil
}
func (n *Node) FlushNodeInfo() {

	if exaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", ExternalIp, n.externalPort)); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
	}
	if listenAddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", LocalAddr, n.localPort)); err == nil {
		n.nodeInfo.SetListenAddr(listenAddr)
	}

}

func (n *Node) NatOk() bool {
	n.nodeInfo.natNoticeChain <- struct{}{}
	ok := <-n.nodeInfo.natResultChain
	return ok
}
func (n *Node) exChangeVersion() {

	//SERVICE = n.GetServiceTy()
	if IsOutSide == false { //如果能作为服务方，则Nat,进行端口映射，否则，不启动Nat

		if !n.NatOk() {
			SERVICE -= NODE_NETWORK //nat 失败，不对外提供服务
			log.Info("ExChangeVersion", "NatFaild", "No Support Service")
		} else {
			log.Info("ExChangeVersion", "NatOk", "Support Service")
		}
	}

	//如果 IsOutSide == true 节点在外网，则不启动nat
	selefNet, err := NewNetAddressString(fmt.Sprintf("%v:%v", ExternalIp, n.GetExterPort()))
	if err == nil {
		n.addrBook.AddOurAddress(selefNet)
	}

	pcli := NewP2pCli(nil)
	peers, _ := n.GetActivePeers()
	for _, peer := range peers {
		pcli.SendVersion(peer, n)
	}
	ticker := time.NewTicker(time.Second * 20)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-ticker.C:
			log.Debug("exChangeVersion", "sendVersion", "version")
			peers, _ := n.GetActivePeers()
			for _, peer := range peers {
				pcli.SendVersion(peer, n)
			}
		case <-n.loopDone:
			break FOR_LOOP
		}
	}
}

func (n *Node) DialPeers(addrbucket map[string]bool) error {
	if len(addrbucket) == 0 {
		return nil
	}
	var addrs []string
	for addr, _ := range addrbucket {
		addrs = append(addrs, addr)
	}
	netAddrs, err := NewNetAddressStrings(addrs)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	selfaddr, err := NewNetAddressString(n.localAddr)
	if err != nil {
		log.Error("NewNetAddressString", "err", err.Error())
		return err
	}
	for _, netAddr := range netAddrs {

		if netAddr.Equals(selfaddr) || netAddr.Equals(n.nodeInfo.GetExternalAddr()) {
			log.Debug("DialPeers filter selfaddr", "addr", netAddr.String(), "externaladdr", n.nodeInfo.GetExternalAddr())
			continue
		}
		//不对已经连接上的地址重新发起连接
		if n.Has(netAddr.String()) {
			log.Debug("DialPeers", "find hash", netAddr.String())
			continue
		}

		//go func(n *Node, netadr *NetAddress) {
		if n.needMore() == false {
			return nil
		}
		log.Debug("DialPeers", "peer", netAddr.String())
		peer, err := P2pComm.DialPeer(netAddr, &n.nodeInfo)
		if err != nil {
			log.Error("DialPeers", "Err", err.Error())
			continue
		}
		log.Debug("Addr perr", "peer", peer)
		n.AddPeer(peer)
		n.addrBook.AddAddress(netAddr)
		n.addrBook.Save()

		//}(n, netAddr)
	}
	return nil
}

func (n *Node) SetPort(localport, exterport uint) {
	n.portmtx.Lock()
	defer n.portmtx.Unlock()
	n.localPort = uint16(localport)
	n.externalPort = uint16(exterport)
}

func (n *Node) GetPort() (uint16, uint16) {
	n.portmtx.Lock()
	defer n.portmtx.Unlock()
	return n.localPort, n.externalPort
}

func (n *Node) GetLocalPort() uint16 {
	n.portmtx.Lock()
	defer n.portmtx.Unlock()
	return n.localPort
}

func (n *Node) GetExterPort() uint16 {
	n.portmtx.Lock()
	defer n.portmtx.Unlock()
	return n.externalPort
}

func (n *Node) AddPeer(pr *peer) {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	if pr.outbound == false {
		return
	}
	if peer, ok := n.outBound[pr.Addr()]; ok {
		log.Info("AddPeer", "delete peer", pr.Addr())
		n.addrBook.RemoveAddr(peer.Addr())
		n.addrBook.Save()
		delete(n.outBound, pr.Addr())
		peer.Close()
		peer = nil
	}
	log.Debug("AddPeer", "peer", pr.Addr())
	n.outBound[pr.Addr()] = pr
	pr.key = n.addrBook.key
	pr.Start()
	return

}

func (n *Node) Size() int {

	return n.nodeInfo.peerInfos.PeerSize()
}

func (n *Node) Has(paddr string) bool {
	n.omtx.Lock() //TODO
	defer n.omtx.Unlock()

	if _, ok := n.outBound[paddr]; ok {
		return true
	}
	return false
}

func (n *Node) GetRegisterPeers() []*peer {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	var peers []*peer
	if len(n.outBound) == 0 {
		return peers
	}
	for _, peer := range n.outBound {
		peers = append(peers, peer)

	}
	return peers
}

func (n *Node) GetActivePeers() ([]*peer, map[string]*types.Peer) {
	regPeers := n.GetRegisterPeers()
	infos := n.nodeInfo.peerInfos.GetPeerInfos()
	var peers []*peer
	for _, peer := range regPeers {
		if _, ok := infos[peer.Addr()]; ok {
			peers = append(peers, peer)
		}
	}
	return peers, infos
}
func (n *Node) Remove(peerAddr string) {

	n.omtx.Lock()
	defer n.omtx.Unlock()
	peer, ok := n.outBound[peerAddr]
	if ok {
		delete(n.outBound, peerAddr)
		peer.Close()
		peer = nil
		return
	}

	return
}

func (n *Node) RemoveAll() {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	for addr, peer := range n.outBound {
		delete(n.outBound, addr)
		peer.Close()
		peer = nil
	}
	return
}

func (n *Node) monitor() {
	go n.monitorErrPeer()
	go n.checkActivePeers()
	go n.getAddrFromOnline()
	go n.getAddrFromOffline()

}

func (n *Node) needMore() bool {
	outBoundNum := n.Size()
	if outBoundNum > MaxOutBoundNum || outBoundNum >= StableBoundNum {
		return false
	}
	return true
}

func (n *Node) GetServiceTy() int64 {
	//确认自己的服务范围1，2，4
	defer func() {
		log.Info("GetServiceTy", "Service", SERVICE, "IsOutSide", IsOutSide, "externalport", n.externalPort)
	}()

	if SERVICE == 0 {
		log.Debug("GetServiceTy", "Service", 0, "Describle", "NetWork Disable")
		os.Exit(0)
	}
	if n.nodeInfo.cfg.GetIsSeed() == true {
		return SERVICE
	}

	if IsOutSide == true {
		n.externalPort = DefaultPort //在外网，使用默认端口号
		return SERVICE
	} else {
		ticker := time.NewTicker(time.Second * 3)
		for i := 0; i < 5; i++ {
			select {
			case <-ticker.C:
				continue
			default:
				exnet, err := nat.Any().ExternalIP()
				if err == nil {
					log.Debug("GetServeice", "NatAddr", exnet.String())

					if exnet.String() != ExternalIp {
						if ExternalIp == LocalAddr { //如果本地地址与外网地址一致，且在内网的情况下，则认为没有正确获取外网地址，默认使用nat获取的地址
							ExternalIp = exnet.String()
							return SERVICE
						}
						SERVICE -= NODE_NETWORK
						return SERVICE
					}
					return SERVICE
				}

			}
		}

	}

	SERVICE -= NODE_NETWORK
	return SERVICE

}
func (n *Node) DetectionNodeAddr() {
	//直到网络可用的时候，才返回。
	for {
		cfg := n.nodeInfo.cfg
		LocalAddr = P2pComm.GetLocalAddr()
		log.Info("DetectionNodeAddr", "addr:", LocalAddr)
		if len(LocalAddr) == 0 {
			//SERVICE = 0
			log.Error("DetectionNodeAddr", "NetWork Disable p2p Disable", "Retry until Network enable")
			time.Sleep(time.Second * 5)
			continue
			//os.Exit(0)
		}
		if cfg.GetIsSeed() {
			ExternalIp = LocalAddr
			IsOutSide = true
			goto SET_ADDR
		}

		if len(cfg.Seeds) == 0 {
			goto SET_ADDR
		}
		for _, seed := range cfg.Seeds {

			pcli := NewP2pCli(nil)
			selfexaddrs, outside := pcli.GetExternIp(seed)
			if len(selfexaddrs) != 0 {
				IsOutSide = outside
				ExternalIp = selfexaddrs
				break
				//log.Error("DetectionNodeAddr", "LocalAddr", LocalAddr, "ExternalAddr", ExternalIp)
			}

		}
	SET_ADDR:
		//如果nat,getSelfExternalAddr 无法发现自己的外网地址，则把localaddr 赋值给外网地址
		if len(ExternalIp) == 0 {
			ExternalIp = LocalAddr
		}

		addr := fmt.Sprintf("%v:%v", ExternalIp, n.GetExterPort())
		log.Debug("DetectionNodeAddr", "AddBlackList", addr)
		n.nodeInfo.blacklist.Add(addr) //把自己的外网地址加入到黑名单，以防连接self
		if exaddr, err := NewNetAddressString(addr); err == nil {
			n.nodeInfo.SetExternalAddr(exaddr)

		} else {
			log.Error("DetectionNodeAddr", "error", err.Error())
		}
		if listaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", LocalAddr, n.GetLocalPort())); err == nil {
			n.nodeInfo.SetListenAddr(listaddr)
		}
		n.localAddr = fmt.Sprintf("%s:%v", LocalAddr, n.GetLocalPort())

		log.Info("DetectionNodeAddr", " Finish ExternalIp", ExternalIp, "LocalAddr", LocalAddr)
		break
	}
}
