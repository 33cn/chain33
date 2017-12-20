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
	addrBook     *AddrBook // known peers
	nodeInfo     *NodeInfo
	localAddr    string
	extaddr      string
	localPort    uint16 //listen
	externalPort uint16 //nat map
	outBound     map[string]*peer
	l            Listener
	rListener    RemoteListener
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
	DetectionNodeAddr(cfg)
	os.MkdirAll(cfg.GetDbPath(), 0755)
	rand.Seed(time.Now().Unix())
	node := &Node{

		outBound:     make(map[string]*peer),
		addrBook:     NewAddrBook(cfg.GetDbPath() + "/addrbook.json"),
		localPort:    uint16(rand.Intn(64512) + 1023),
		externalPort: uint16(rand.Intn(64512) + 1023),
	}
	log.Debug("newNode", "localport", node.localPort)
	log.Debug("newNode", "externalPort", node.externalPort)

	if cfg.GetIsSeed() == true {
		if cfg.GetSeedPort() != 0 {
			node.localPort = uint16(cfg.GetSeedPort())
			node.externalPort = uint16(cfg.GetSeedPort())
		} else {
			node.localPort = uint16(DefaultPort)
			node.externalPort = uint16(DefaultPort)
		}

	}
	node.nodeInfo = new(NodeInfo)
	exaddr := fmt.Sprintf("%v:%v", EXTERNALADDR, node.externalPort)
	node.nodeInfo.externalAddr, _ = NewNetAddressString(exaddr)
	node.nodeInfo.listenAddr, _ = NewNetAddressString(fmt.Sprintf("%v:%v", LOCALADDR, node.localPort))
	node.nodeInfo.monitorChan = make(chan *peer, 1024)
	node.nodeInfo.versionChan = make(chan struct{})
	node.nodeInfo.p2pBlockChan = make(chan *types.P2PBlock, 1024)
	node.nodeInfo.cfg = cfg
	node.localAddr = fmt.Sprintf("%s:%v", LOCALADDR, node.localPort)

	return node, nil
}
func (n *Node) flushNodeInfo() {
	n.nodeInfo.externalAddr, _ = NewNetAddressString(fmt.Sprintf("%v:%v", EXTERNALADDR, n.externalPort))
	n.nodeInfo.listenAddr, _ = NewNetAddressString(fmt.Sprintf("%v:%v", LOCALADDR, n.localPort))
}
func (n *Node) makeService() {
	//确认自己的服务范围1，2，4
	go func() {
		for i := 0; i < 10; i++ {
			exnet, err := nat.Any().ExternalIP()
			if err == nil {
				if exnet.String() != EXTERNALADDR && exnet.String() != LOCALADDR {
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
			log.Debug("DialPeers", "Check Equal", "Find Local Addr", netAddr.String())
			continue
		}
		//不对已经连接上的地址重新发起连接
		if _, ok := n.outBound[netAddr.String()]; ok {
			continue
		}

		peer, err := DialPeer(netAddrs[i], &n.nodeInfo)
		if err != nil {
			log.Error(err.Error())
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

	if len(n.outBound) < MaxOutBoundNum {
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
				log.Debug("DialSeeds", "Check Equal", "Find Local Addr", netAddr.String())
				time.Sleep(time.Second * 10)
				continue
			}
			//不对已经连接上的地址重新发起连接
			if _, ok := n.outBound[netAddr.String()]; ok {
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
func (n *Node) AddPeer(pr *peer) {
	if pr.outbound == true {
		defer n.omtx.Unlock()
		n.omtx.Lock()
		n.outBound[pr.Addr()] = pr
		pr.key = n.addrBook.key
		pr.Start()
		return
	}

}

func (n *Node) Size() int {
	defer n.omtx.Unlock()
	n.omtx.Lock()
	return len(n.outBound)
}

func (n *Node) Has(paddr string) bool {
	defer n.omtx.Unlock()
	n.omtx.Lock()
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
	defer n.omtx.Unlock()
	n.omtx.Lock()
	var peers = make([]*peer, 0)
	for _, peer := range n.outBound {
		peers = append(peers, peer)
	}

	return peers
}
func (n *Node) Remove(peerAddr string) {
	defer n.omtx.Unlock()
	n.omtx.Lock()
	peer, ok := n.outBound[peerAddr]
	if ok {
		peer.Stop()
		delete(n.outBound, peerAddr)
		return
	}

	return

}
func (n *Node) checkActivePeers() {
	for {
		peers := n.GetPeers()
		for _, peer := range peers {
			if peer.mconn == nil {
				n.addrBook.RemoveAddr(peer.Addr())
				n.Remove(peer.Addr())
				continue
			}

			log.Info("ISRUNNING", "remotepeer", peer.mconn.remoteAddress.String(), "isrunning ", peer.mconn.sendMonitor.isrunning)
			if peer.mconn.sendMonitor.isrunning == false && peer.IsPersistent() == false {
				n.addrBook.RemoveAddr(peer.Addr())
				n.addrBook.Save()
				n.Remove(peer.Addr())
			}
			if _, ok := n.addrBook.addrPeer[peer.Addr()]; ok {
				n.addrBook.addrPeer[peer.Addr()].Attempts = peer.mconn.sendMonitor.count
				n.addrBook.addrPeer[peer.Addr()].LastAttempt = time.Unix(int64(peer.mconn.sendMonitor.lastop), 0)
				n.addrBook.addrPeer[peer.Addr()].LastSuccess = time.Unix(int64(peer.mconn.sendMonitor.lastok), 0)

			}

		}
		time.Sleep(time.Second * 5)
	}
}
func (n *Node) deletePingErr() {
	for {
		peer := <-n.nodeInfo.monitorChan
		log.Warn("RemoveBadPeer", "REMOVE", peer.Addr())
		n.addrBook.RemoveAddr(peer.Addr())
		n.addrBook.Save()
		n.Remove(peer.Addr())
	}
}
func (n *Node) monitor() {
	go n.deletePingErr()
	go n.checkActivePeers()

	for {
		time.Sleep(time.Second * 5)
		if n.needMore() {
			var savelist = make([]string, 0)
			for _, seed := range n.nodeInfo.cfg.Seeds {
				if n.Has(seed) == false {
					savelist = append(savelist, seed)
				}
			}
			log.Debug("OUTBOUND NUM", "NUM", len(n.outBound), "start getaddr from peer", n.addrBook.addrPeer)
			log.Debug("OutBound", "peers", n.outBound)
			peeraddrs := n.addrBook.GetAddrs()
			for _, peeraddr := range peeraddrs {
				if n.Has(peeraddr) == false {
					savelist = append(savelist, peeraddr)
				}
				//对存储的地址列表进行重新捡漏
				log.Debug("SaveList", "list", savelist)
			}
			n.DialPeers(savelist)
			peers := n.GetPeers()
			for _, peer := range peers { //向其他节点发起请求，获取地址列表
				addrlist, err := peer.mconn.GetAddr()
				if err != nil {
					log.Error("GetAddr", "ERROR", err.Error())
					continue
				}
				log.Warn("ADDRLIST", "LIST", addrlist)
				n.DialPeers(addrlist) //对获取的地址列表发起连接
				if !n.needMore() {
					break
				}
			}
		} else {
			//close seed conn
			for _, seed := range n.nodeInfo.cfg.Seeds {
				if _, ok := n.outBound[seed]; ok {
					n.Remove(seed)
				}
			}
		}

		log.Debug("Node Monitor process", "outbound num", len(n.outBound))
	}
}

func (n *Node) needMore() bool {
	if len(n.outBound) > MaxOutBoundNum || len(n.outBound) >= StableBoundNum {
		return false
	}
	return true
}

func (n *Node) loadAddrBook() bool {
	var peeraddrs = make([]string, 0)
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

// 启动Node节点
//1.启动监听GRPC Server
//2.初始化AddrBook,并发起对相应地址的连接
//3.如果配置了种子节点，则连接种子节点
//4.启动监控远程节点
func (n *Node) Start() {
	n.makeService()
	n.rListener = NewRemotePeerAddrServer()
	n.l = NewDefaultListener(Protocol, n)

	//连接种子节点，或者导入远程节点文件信息
	log.Debug("Load", "Load addrbook")
	n.loadAddrBook()
	go n.monitor()

	return
}

func (n *Node) Stop() {
	n.l.Stop()
	n.rListener.Stop()
	n.addrBook.Stop()
	for _, peer := range n.outBound {
		peer.Stop()
	}

}
