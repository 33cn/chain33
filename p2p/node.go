package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"

	"strings"
	"time"

	"code.aliyun.com/chain33/chain33/p2p/nat"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

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
	rListener    RemoteListener
	versionDone  chan struct{}
	activeDone   chan struct{}
	errPeerDone  chan struct{}
	offlineDone  chan struct{}
	onlineDone   chan struct{}
}

func localBindAddr() string {

	conn, err := net.Dial("udp", "114.114.114.114:80")
	if err != nil {
		log.Error(err.Error())
		return ""
	}
	defer conn.Close()
	log.Debug(strings.Split(conn.LocalAddr().String(), ":")[0])
	return strings.Split(conn.LocalAddr().String(), ":")[0]
}

func (n *Node) setQueue(q *queue.Queue) {
	n.nodeInfo.q = q
	n.nodeInfo.qclient = q.GetClient()
}

func newNode(cfg *types.P2P) (*Node, error) {
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
	node.nodeInfo.versionChan = make(chan struct{})
	node.nodeInfo.p2pBroadcastChan = make(chan interface{}, 1024)
	node.nodeInfo.blacklist = &BlackList{badPeers: make(map[string]bool)}
	node.nodeInfo.cfg = cfg
	node.nodeInfo.peerInfos = new(PeerInfos)
	node.nodeInfo.peerInfos.infos = make(map[string]*types.Peer)

	return node, nil
}
func (n *Node) flushNodeInfo() {

	if exaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", ExternalAddr, n.externalPort)); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
	}

	if listenAddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", LocalAddr, n.localPort)); err == nil {
		n.nodeInfo.SetListenAddr(listenAddr)
	}

}
func (n *Node) exChangeVersion() {
	<-n.nodeInfo.versionChan
	peers, _ := n.GetPeers()
	for _, peer := range peers {
		peer.mconn.sendVersion()
	}
	ticker := time.NewTicker(time.Second * 20)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-ticker.C:
			log.Debug("exChangeVersion", "sendVersion", "version")
			peers, _ := n.GetPeers()
			for _, peer := range peers {
				peer.mconn.sendVersion()
			}
		case <-n.versionDone:
			break FOR_LOOP
		}
	}
}
func (n *Node) makeService() {
	//确认自己的服务范围1，2，4
	if n.nodeInfo.cfg.GetIsSeed() == true {
		n.nodeInfo.versionChan <- struct{}{}
		return
	}
	go func() {

		for i := 0; i < 10; i++ {
			exnet, err := nat.Any().ExternalIP()
			if err == nil {
				if exnet.String() != ExternalAddr && exnet.String() != LocalAddr {
					SERVICE -= NODE_NETWORK
					n.nodeInfo.versionChan <- struct{}{}
					return
				}
				log.Debug("makeService", "SERVICE", SERVICE)
				n.nodeInfo.versionChan <- struct{}{}
				return
			} else {
				//如果ExternalAddr 与LocalAddr 相同，则认为在外网中
				if ExternalAddr == LocalAddr {
					n.nodeInfo.versionChan <- struct{}{} //通知exchangeVersion 发送版本信息
					return
				}

			}
		}
		//如果无法通过nat获取外网，并且externalAddr!=localAddr

		n.nodeInfo.versionChan <- struct{}{}
	}()

}
func (n *Node) DialPeers(addrs []string) error {
	if len(addrs) == 0 {
		return nil
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
	for i, netAddr := range netAddrs {
		//will not add self addr
		//	log.Debug("DialPeers", "peeraddr", netAddr.String())
		if netAddr.Equals(selfaddr) || netAddr.Equals(n.nodeInfo.GetExternalAddr()) {
			log.Debug("DialPeers filter selfaddr", "addr", netAddr.String(), "externaladdr", n.nodeInfo.GetExternalAddr(), "i", i)
			continue
		}
		//不对已经连接上的地址重新发起连接
		if n.Has(netAddr.String()) {
			continue
		}
		//log.Warn("NewNetAddressString", "i", i)
		peer, err := DialPeer(netAddrs[i], &n.nodeInfo)
		if err != nil {
			log.Error("DialPeers", "Err", err.Error())
			continue
		}
		log.Debug("Addr perr", "peer", peer)
		n.AddPeer(peer)
		n.addrBook.AddAddress(netAddr)
		n.addrBook.Save()
		if n.needMore() == false {
			return nil
		}
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

	n.nodeInfo.peerInfos.mtx.Lock()
	defer n.nodeInfo.peerInfos.mtx.Unlock()
	return len(n.nodeInfo.peerInfos.infos)
}

func (n *Node) Has(paddr string) bool {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	if _, ok := n.outBound[paddr]; ok {
		return true
	}
	return false
}

// Get looks up a peer by the provided peerKey.
func (n *Node) Get(peerAddr string) *peer {
	defer n.omtx.Unlock()
	n.omtx.Lock()
	peer, ok := n.outBound[peerAddr]
	if ok {

		return peer
	}
	return nil
}

func (n *Node) GetRegisterPeers() []*peer {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	var peers []*peer
	for _, peer := range n.outBound {
		//if n.nodeInfo.peerInfos.GetPeerInfo(peeraddr) != nil {
		peers = append(peers, peer)
		//}

	}
	//log.Debug("GetPeers", "node", peers)
	return peers
}

func (n *Node) GetPeers() ([]*peer, map[string]*types.Peer) {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	var peers []*peer
	infos := n.nodeInfo.peerInfos.GetPeerInfos()
	for _, peer := range n.outBound {
		if _, ok := infos[peer.Addr()]; ok {
			log.Debug("GetPeers", "peer", peer.Addr())
			peers = append(peers, peer)
		}

	}
	//log.Debug("GetPeers", "node", peers)
	return peers, infos
}
func (n *Node) Remove(peerAddr string) {

	n.omtx.Lock()
	defer n.omtx.Unlock()
	peer, ok := n.outBound[peerAddr]
	if ok {
		delete(n.outBound, peerAddr)
		peer.Stop()
		return
	}

	return
}

func (n *Node) RemoveAll() {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	for addr, peer := range n.outBound {
		delete(n.outBound, addr)
		peer.Stop()
	}
	return
}
func (n *Node) checkActivePeers() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-n.activeDone:
			break FOR_LOOP
		case <-ticker.C:
			peers := n.GetRegisterPeers()
			for _, peer := range peers {
				if peer.mconn == nil {
					n.addrBook.RemoveAddr(peer.Addr())
					n.Remove(peer.Addr())
					continue
				}

				log.Debug("checkActivePeers", "remotepeer", peer.mconn.remoteAddress.String(), "isrunning ", peer.mconn.sendMonitor.IsRunning())
				if peer.mconn.sendMonitor.IsRunning() == false && peer.IsPersistent() == false || peer.Addr() == n.nodeInfo.GetExternalAddr().String() {
					n.addrBook.RemoveAddr(peer.Addr())
					n.addrBook.Save()
					n.Remove(peer.Addr())
				}

				if peerStat := n.addrBook.getPeerStat(peer.Addr()); peerStat != nil {
					peerStat.flushPeerStatus(peer.mconn.sendMonitor.MonitorInfo())

				}

			}
		}

	}
}
func (n *Node) deleteErrPeer() {

	for {

		peer := <-n.nodeInfo.monitorChan
		log.Debug("deleteErrPeer", "REMOVE", peer.Addr())
		if peer.version.Get() == false { //如果版本不支持，则加入黑名单，下次不再发起连接
			n.nodeInfo.blacklist.Add(peer.Addr()) //加入黑名单
		}
		n.addrBook.RemoveAddr(peer.Addr())
		n.addrBook.Save()
		n.Remove(peer.Addr())
	}

}
func (n *Node) getAddrFromOnline() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-n.onlineDone:
			break FOR_LOOP
		case <-ticker.C:
			if n.needMore() {
				peers, _ := n.GetPeers()
				log.Debug("getAddrFromOnline", "peers", peers)
				for _, peer := range peers { //向其他节点发起请求，获取地址列表
					log.Debug("Getpeer", "addr", peer.Addr())
					addrlist, err := peer.mconn.getAddr()
					if err != nil {
						log.Error("monitor", "ERROR", err.Error())
						continue
					}
					log.Debug("monitor", "ADDRLIST", addrlist)
					//过滤黑名单的地址
					var whitlist []string
					for _, addr := range addrlist {
						if n.nodeInfo.blacklist.Has(addr) == false {
							whitlist = append(whitlist, addr)
						} else {
							log.Warn("Filter addr", "BlackList", addr)
						}
					}
					n.DialPeers(whitlist) //对获取的地址列表发起连接

				}
			}

		}
	}
}

func (n *Node) getAddrFromOffline() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
FOR_LOOP:
	for {
		select {
		case <-n.offlineDone:
			break FOR_LOOP
		case <-ticker.C:
			//time.Sleep(time.Second * 25)
			if n.needMore() {
				var savelist []string
				for _, seed := range n.nodeInfo.cfg.Seeds {
					if n.Has(seed) == false && n.nodeInfo.blacklist.Has(seed) == false {
						savelist = append(savelist, seed)
					}
				}

				log.Debug("OUTBOUND NUM", "NUM", n.Size(), "start getaddr from peer", n.addrBook.GetPeers())
				peeraddrs := n.addrBook.GetAddrs()
				if len(peeraddrs) != 0 {
					for _, addr := range peeraddrs {
						if n.Has(addr) == false && n.nodeInfo.blacklist.Has(addr) == false {
							savelist = append(savelist, addr)
						}
						log.Debug("SaveList", "list", savelist)
					}
				}

				if len(savelist) == 0 {

					continue
				}
				n.DialPeers(savelist)
			} else {
				log.Debug("monitor", "nodestable", n.needMore())
				for _, seed := range n.nodeInfo.cfg.Seeds {
					//如果达到稳定节点数量，则断开种子节点
					if n.Has(seed) == true {
						n.Remove(seed)
					}
				}
			}

			log.Debug("Node Monitor process", "outbound num", n.Size())
		}
	}

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

func (n *Node) loadAddrBook() bool {
	var peeraddrs []string
	if n.addrBook.Size() != 0 {
		for peeraddr, _ := range n.addrBook.addrPeer {
			peeraddrs = append(peeraddrs, peeraddr)
			if len(peeraddrs) > MaxOutBoundNum {
				break
			}

		}
		if n.DialPeers(peeraddrs) != nil {
			return false
		}
		return true
	}
	return false

}
func (n *Node) detectionNodeAddr() {
	cfg := n.nodeInfo.cfg
	LocalAddr = localBindAddr()
	log.Debug("detectionNodeAddr", "addr:", LocalAddr)
	if cfg.GetIsSeed() {
		ExternalAddr = LocalAddr

		goto SET_ADDR
	}

	if len(cfg.Seeds) == 0 {
		goto SET_ADDR
	}

	for _, addr := range cfg.Seeds {
		if strings.Contains(addr, ":") == false {
			continue
		}
		seedip := strings.Split(addr, ":")[0]

		selfexaddrs := getSelfExternalAddr(fmt.Sprintf("%v:%v", seedip, DefalutP2PRemotePort))
		if len(selfexaddrs) != 0 {
			ExternalAddr = selfexaddrs[0]
			log.Debug("detectionNodeAddr", "LocalAddr", LocalAddr, "ExternalAddr", ExternalAddr)
			continue
		}

	}
SET_ADDR:
	//如果nat,getSelfExternalAddr 无法发现自己的外网地址，则把localaddr 赋值给外网地址
	if len(ExternalAddr) == 0 {
		ExternalAddr = LocalAddr
	}

	addr := fmt.Sprintf("%v:%v", ExternalAddr, n.GetExterPort())
	if exaddr, err := NewNetAddressString(addr); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
		n.nodeInfo.blacklist.Add(addr) //把自己的外网地址加入到黑名单，以防连接自己
	}
	if listaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", LocalAddr, n.GetLocalPort())); err == nil {
		n.nodeInfo.SetListenAddr(listaddr)
	}
	n.localAddr = fmt.Sprintf("%s:%v", LocalAddr, n.GetLocalPort())
}

// 启动Node节点
//1.启动监听GRPC Server
//2.初始化AddrBook,并发起对相应地址的连接
//3.如果配置了种子节点，则连接种子节点
//4.启动监控远程节点
func (n *Node) Start() {
	n.detectionNodeAddr()

	n.rListener = NewRemotePeerAddrServer()
	n.l = NewDefaultListener(Protocol, n)

	go n.monitor()
	go n.exChangeVersion()
	go n.makeService()
	return
}

func (n *Node) Stop() {
	close(n.versionDone)
	close(n.activeDone)
	close(n.onlineDone)
	close(n.offlineDone)
	log.Debug("stop", "versionDone", "close")
	n.l.Stop()
	log.Debug("stop", "listen", "close")
	n.rListener.Stop()
	log.Debug("stop", "remotelisten", "close")
	n.addrBook.Stop()
	log.Debug("stop", "addrBook", "close")

	n.RemoveAll()

}
