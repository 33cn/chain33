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
		localPort:    uint16(rand.Intn(64512) + 1023),
		externalPort: uint16(rand.Intn(64512) + 1023),
		outBound:     make(map[string]*peer),
		addrBook:     NewAddrBook(cfg.GetDbPath() + "/addrbook.json"),
		versionDone:  make(chan struct{}, 1),
	}
	log.Debug("newNode", "localport", node.localPort)
	log.Debug("newNode", "externalPort", node.externalPort)

	if cfg.GetIsSeed() == true {
		if cfg.GetSeedPort() != 0 {
			node.SetPort(uint(cfg.GetSeedPort()), uint(cfg.GetSeedPort()))

		} else {
			node.SetPort(DefaultPort, DefaultPort)
		}

	}
	node.nodeInfo = new(NodeInfo)
	node.nodeInfo.monitorChan = make(chan *peer, 1024)
	node.nodeInfo.versionChan = make(chan struct{})
	node.nodeInfo.p2pBroadcastChan = make(chan interface{}, 1024)
	node.nodeInfo.blacklist = &BlackList{badPeers: make(map[string]bool)}
	node.nodeInfo.cfg = cfg

	return node, nil
}
func (n *Node) flushNodeInfo() {

	if exaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", ExternalAddr, n.externalPort)); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
	}

	if listenAddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", ExternalAddr, n.localPort)); err == nil {
		n.nodeInfo.SetListenAddr(listenAddr)
	}

}
func (n *Node) exChangeVersion() {
	<-n.nodeInfo.versionChan
	peers := n.GetPeers()
	for _, peer := range peers {
		peer.mconn.sendVersion()
	}
FOR_LOOP:
	for {
		ticker := time.NewTicker(time.Second * 20)
		select {
		case <-ticker.C:
			log.Debug("exChangeVersion", "sendVersion", "version")
			peers := n.GetPeers()
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
	go func() {
		for i := 0; i < 10; i++ {
			exnet, err := nat.Any().ExternalIP()
			if err == nil {
				if exnet.String() != ExternalAddr && exnet.String() != LocalAddr {
					SERVICE -= NODE_NETWORK
					break
				}
				log.Debug("makeService", "SERVICE", SERVICE)
				n.nodeInfo.versionChan <- struct{}{}
				log.Debug("makeService", "Sig", "Ok")
				break
			} else {
				log.Error("ExternalIp", "Error", err.Error())
			}
		}
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
		if netAddr.Equals(selfaddr) || netAddr.Equals(n.nodeInfo.GetExternalAddr()) {
			continue
		}
		//不对已经连接上的地址重新发起连接
		if n.Has(netAddr.String()) {
			continue
		}

		peer, err := DialPeer(netAddrs[i], &n.nodeInfo)
		if err != nil {
			log.Error("DialPeers", "XXXXXXXXXX", err.Error())
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
func (n *Node) dialSeeds(addrs []string) error {
	netAddrs, err := NewNetAddressStrings(addrs)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	if n.Size() < MaxOutBoundNum {
		//没有存储其他任何远程机节点的地址信息
		selfaddr, err := NewNetAddressString(n.localAddr)
		if err != nil {
			log.Error("NewNetAddressString", "err", err.Error())
			return err
		}

		for i, netAddr := range netAddrs {
			log.Debug("SHOW", "netaddr", netAddr.String(), "selfAddr", selfaddr.String())

			//will not add self addr
			if netAddr.Equals(selfaddr) || netAddr.Equals(n.nodeInfo.GetExternalAddr()) {

				continue
			}
			//不对已经连接上的地址重新发起连接

			if n.Has(netAddr.String()) {
				continue
			}

			peer, err := DialPeer(netAddrs[i], &n.nodeInfo)
			if err != nil {
				log.Error(err.Error())
				return err
			}

			n.AddPeer(peer)
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

	n.omtx.Lock()
	defer n.omtx.Unlock()
	return len(n.outBound)
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
func (n *Node) GetPeers() []*peer {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	var peers []*peer
	for _, peer := range n.outBound {
		peers = append(peers, peer)
	}
	log.Debug("GetPeers", "node", peers)
	return peers
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

func (n *Node) checkActivePeers() {

	for {
		ticker := time.NewTicker(time.Second * 5)
		select {
		case <-ticker.C:
			peers := n.GetPeers()
			for _, peer := range peers {
				if peer.mconn == nil {
					n.addrBook.RemoveAddr(peer.Addr())
					n.Remove(peer.Addr())
					continue
				}

				log.Debug("checkActivePeers", "remotepeer", peer.mconn.remoteAddress.String(), "isrunning ", peer.mconn.sendMonitor.IsRunning())
				if peer.mconn.sendMonitor.IsRunning() == false && peer.IsPersistent() == false {
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
		log.Warn("deleteErrPeer", "REMOVE", peer.Addr())
		n.nodeInfo.blacklist.Add(peer.Addr()) //加入黑名单
		n.addrBook.RemoveAddr(peer.Addr())
		n.addrBook.Save()
		n.Remove(peer.Addr())
	}
}
func (n *Node) getAddrFromOnline() {
	for {
		time.Sleep(time.Second * 5)
		if n.needMore() {
			peers := n.GetPeers()
			log.Debug("getAddrFromOnline", "peers", peers)
			for _, peer := range peers { //向其他节点发起请求，获取地址列表
				log.Debug("Getpeer", "addr", peer.Addr())
				addrlist, err := peer.mconn.getAddr()
				if err != nil {
					log.Error("monitor", "ERROR", err.Error())
					continue
				}
				log.Debug("monitor", "ADDRLIST", addrlist)
				n.DialPeers(addrlist) //对获取的地址列表发起连接

			}
		}
	}
}

func (n *Node) getAddrFromOffline() {
	for {

		time.Sleep(time.Second * 25)
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
	}

	if len(cfg.Seeds) == 0 {
		return
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

	//如果nat,getSelfExternalAddr 无法发现自己的外网地址，则把localaddr 赋值给外网地址
	if len(ExternalAddr) == 0 {
		ExternalAddr = LocalAddr
	}
	exaddr := fmt.Sprintf("%v:%v", ExternalAddr, n.GetExterPort())
	if exaddr, err := NewNetAddressString(exaddr); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
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
	n.makeService()
	n.rListener = NewRemotePeerAddrServer()
	n.l = NewDefaultListener(Protocol, n)

	//连接种子节点，或者导入远程节点文件信息
	log.Debug("Load", "Load addrbook")
	go n.loadAddrBook()
	go n.monitor()
	go n.exChangeVersion()
	return
}

func (n *Node) Stop() {
	close(n.versionDone)
	n.l.Stop()
	n.rListener.Stop()
	n.addrBook.Stop()
	peers := n.GetPeers()
	for _, peer := range peers {
		peer.Stop()
	}

}
