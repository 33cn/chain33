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
	close(n.versionDone)
	close(n.activeDone)
	close(n.onlineDone)
	close(n.offlineDone)
	log.Debug("stop", "versionDone", "close")
	n.l.Close()
	log.Debug("stop", "listen", "close")
	log.Debug("stop", "remotelisten", "close")
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
	versionDone  chan struct{}
	activeDone   chan struct{}
	errPeerDone  chan struct{}
	offlineDone  chan struct{}
	onlineDone   chan struct{}
}

func (n *Node) setQueue(q *queue.Queue) {
	n.nodeInfo.q = q
	n.nodeInfo.qclient = q.GetClient()
}

func NewNode(cfg *types.P2P) (*Node, error) {
	os.MkdirAll(cfg.GetDbPath(), 0755)
	rand.Seed(time.Now().Unix())
	node := &Node{
		localPort:    DefaultPort, //uint16(rand.Intn(64512) + 1023),
		externalPort: uint16(rand.Intn(64512) + 1023),
		outBound:     make(map[string]*peer),
		addrBook:     NewAddrBook(cfg.GetDbPath() + "/addrbook.json"),
		versionDone:  make(chan struct{}, 1),
		activeDone:   make(chan struct{}, 1),
		errPeerDone:  make(chan struct{}, 1),
		offlineDone:  make(chan struct{}, 1),
		onlineDone:   make(chan struct{}, 1),
	}

	log.Debug("newNode", "externalPort", node.externalPort)

	if cfg.GetIsSeed() == true {
		node.SetPort(DefaultPort, DefaultPort)

	}
	node.nodeInfo = new(NodeInfo)
	node.nodeInfo.monitorChan = make(chan *peer, 1024)
	node.nodeInfo.natNoticeChain = make(chan struct{}, 1)
	node.nodeInfo.natResultChain = make(chan bool, 1)
	node.nodeInfo.p2pBroadcastChan = make(chan interface{}, 1024)
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

	if n.GetServiceTy() == 7 && IsOutSide == false { //如果能作为服务方，则Nat,进行端口映射，否则，不启动Nat

		if !n.NatOk() {
			SERVICE -= NODE_NETWORK //nat 失败，不对外提供服务
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
		pcli.SendVersion(peer, n.nodeInfo)
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
				pcli.SendVersion(peer, n.nodeInfo)
			}
		case <-n.versionDone:
			break FOR_LOOP
		}
	}
}
func (n *Node) GetServiceTy() int64 {
	//确认自己的服务范围1，2，4
	if n.nodeInfo.cfg.GetIsSeed() == true {
		return SERVICE
	}

	for i := 0; i < 10; i++ {
		exnet, err := nat.Any().ExternalIP()
		if err == nil {
			if exnet.String() != ExternalIp {
				SERVICE -= NODE_NETWORK
				return SERVICE
			}
			log.Error("GetServiceTy", "Service", SERVICE, "IsOutSide", IsOutSide, "externalport", n.externalPort)
			return SERVICE
		} else {
			//如果ExternalAddr 与LocalAddr 相同，则认为在外网中
			if ExternalIp == LocalAddr {
				n.externalPort = DefaultPort //外网使用默认端口
				IsOutSide = true
				log.Error("GetServiceTy", "Service", SERVICE, "IsOutSide", IsOutSide, "externalport", n.externalPort)
				return SERVICE
			}

		}
	}
	//如果无法通过nat获取外网，并且externalAddr!=localAddr
	SERVICE -= NODE_NETWORK
	IsOutSide = true
	log.Error("GetServiceTy", "Service", SERVICE, "IsOutSide", IsOutSide, "externalport", n.externalPort)
	return SERVICE

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

		go func(n *Node, netadr *NetAddress) {
			if n.needMore() == false {
				return
			}
			log.Debug("DialPeers", "peer", netadr.String())
			peer, err := P2pComm.DialPeer(netadr, &n.nodeInfo)
			if err != nil {
				log.Error("DialPeers", "Err", err.Error())
				return
			}
			log.Debug("Addr perr", "peer", peer)
			n.AddPeer(peer)
			n.addrBook.AddAddress(netadr)
			n.addrBook.Save()

		}(n, netAddr)
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
	}
	return
}

func (n *Node) monitor() {
	go n.deleteErrPeer()
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

func (n *Node) DetectionNodeAddr() {
	cfg := n.nodeInfo.cfg
	LocalAddr = P2pComm.GetLocalAddr()
	log.Debug("detectionNodeAddr", "addr:", LocalAddr)

	if cfg.GetIsSeed() {
		ExternalIp = LocalAddr

		goto SET_ADDR
	}

	if len(cfg.Seeds) == 0 {
		goto SET_ADDR
	}
	for _, seed := range cfg.Seeds {

		pcli := NewP2pCli(nil)
		selfexaddrs := pcli.GetExternIp(seed)
		if len(selfexaddrs) != 0 {
			ExternalIp = selfexaddrs[0]
			log.Error("detectionNodeAddr", "LocalAddr", LocalAddr, "ExternalAddr", ExternalIp)
		}

	}
SET_ADDR:
	//如果nat,getSelfExternalAddr 无法发现自己的外网地址，则把localaddr 赋值给外网地址
	if len(ExternalIp) == 0 {
		ExternalIp = LocalAddr
	}

	addr := fmt.Sprintf("%v:%v", ExternalIp, n.GetExterPort())
	if exaddr, err := NewNetAddressString(addr); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
		n.nodeInfo.blacklist.Add(addr) //把自己的外网地址加入到黑名单，以防连接自己
	}
	if listaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", LocalAddr, n.GetLocalPort())); err == nil {
		n.nodeInfo.SetListenAddr(listaddr)
	}
	n.localAddr = fmt.Sprintf("%s:%v", LocalAddr, n.GetLocalPort())
}
