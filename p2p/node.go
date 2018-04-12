package p2p

import (
	"fmt"
	"math/rand"
	"sync/atomic"

	"sync"
	"time"

	"gitlab.33.cn/chain33/chain33/p2p/nat"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

// 启动Node节点
//1.启动监听GRPC Server
//2.检测自身地址
//3.启动端口映射
//4.启动监控模块，进行节点管理

func (n *Node) Start() {

	n.detectNodeAddr()
	go n.doNat()
	go n.monitor()
	return
}

func (n *Node) Close() {
	atomic.StoreInt32(&n.closed, 1)
	log.Debug("stop", "versionDone", "closed")
	if n.l != nil {
		n.l.Close()
	}
	log.Debug("stop", "listen", "closed")
	n.nodeInfo.addrBook.Close()
	log.Debug("stop", "addrBook", "closed")
	n.RemoveAll()
	if Filter != nil {
		Filter.Close()
	}
	n.deleteNatMapPort()
	log.Debug("stop", "PeerRemoeAll", "closed")

}

func (n *Node) IsClose() bool {
	return atomic.LoadInt32(&n.closed) == 1
}

type Node struct {
	omtx     sync.Mutex
	nodeInfo *NodeInfo
	outBound map[string]*peer
	l        Listener
	closed   int32
}

func (n *Node) SetQueueClient(client queue.Client) {
	n.nodeInfo.client = client
}

func NewNode(cfg *types.P2P) (*Node, error) {

	node := &Node{
		outBound: make(map[string]*peer),
	}

	node.nodeInfo = NewNodeInfo(cfg)
	if cfg.GetServerStart() {
		node.l = NewListener(protocol, node)
	}

	return node, nil
}
func (n *Node) FlushNodePort(localport, export uint16) {

	if exaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", n.nodeInfo.GetExternalAddr().IP.String(), export)); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
	}
	if listenAddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", LocalAddr, localport)); err == nil {
		n.nodeInfo.SetListenAddr(listenAddr)
	}

}

func (n *Node) natOk() bool {
	n.nodeInfo.natNoticeChain <- struct{}{}
	ok := <-n.nodeInfo.natResultChain
	return ok
}

func (n *Node) doNat() {
	go n.natMapPort()
	if OutSide == false && n.nodeInfo.cfg.GetServerStart() { //如果能作为服务方，则Nat,进行端口映射，否则，不启动Nat

		if !n.natOk() {
			Service -= nodeNetwork //nat 失败，不对外提供服务
			log.Info("doNat", "NatFaild", "No Support Service")
		} else {
			log.Info("doNat", "NatOk", "Support Service")
		}

	}
	n.nodeInfo.SetNatDone()
	n.nodeInfo.addrBook.AddOurAddress(n.nodeInfo.GetExternalAddr())
	n.nodeInfo.addrBook.AddOurAddress(n.nodeInfo.GetListenAddr())
	if selefNet, err := NewNetAddressString(fmt.Sprintf("127.0.0.1:%v", n.nodeInfo.GetListenAddr().Port)); err == nil {
		n.nodeInfo.addrBook.AddOurAddress(selefNet)
	}

	return
}

func (n *Node) AddPeer(pr *peer) {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	if peer, ok := n.outBound[pr.Addr()]; ok {
		log.Info("AddPeer", "delete peer", pr.Addr())
		n.nodeInfo.addrBook.RemoveAddr(peer.Addr())
		delete(n.outBound, pr.Addr())
		peer.Close()
		peer = nil
	}
	log.Debug("AddPeer", "peer", pr.Addr())
	n.outBound[pr.Addr()] = pr
	pr.Start()
	return

}

func (n *Node) Size() int {

	return n.nodeInfo.peerInfos.PeerSize()
}

func (n *Node) Has(paddr string) bool {
	n.omtx.Lock()
	defer n.omtx.Unlock()

	if _, ok := n.outBound[paddr]; ok {
		return true
	}
	return false
}

func (n *Node) getRegisterPeer(paddr string) *peer {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	if peer, ok := n.outBound[paddr]; ok {
		return peer
	}
	return nil
}

func (n *Node) getRegisterPeers() []*peer {
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

func (n *Node) getActivePeers() (map[string]*peer, map[string]*types.Peer) {
	regPeers := n.getRegisterPeers()
	infos := n.nodeInfo.peerInfos.GetPeerInfos()

	var peers = make(map[string]*peer)
	for _, peer := range regPeers {
		if _, ok := infos[peer.Addr()]; ok {

			peers[peer.Addr()] = peer
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
	go n.getAddrFromOnline()
	go n.getAddrFromOffline()
	go n.monitorPeerInfo()
	go n.monitorDialPeers()
	go n.monitorBlackList()
	go n.monitorFilter()

}

func (n *Node) needMore() bool {
	outBoundNum := n.Size()
	if outBoundNum > maxOutBoundNum || outBoundNum > stableBoundNum {
		return false
	}
	return true
}

func (n *Node) detectNodeAddr() {

	var externalIP string
	for {
		cfg := n.nodeInfo.cfg
		LocalAddr = P2pComm.GetLocalAddr()
		log.Info("DetectNodeAddr", "addr:", LocalAddr)
		if len(LocalAddr) == 0 {
			log.Error("DetectNodeAddr", "NetWork Disable p2p Disable", "Retry until Network enable")
			time.Sleep(time.Second * 5)
			continue
		}
		if cfg.GetIsSeed() {
			log.Info("DetectNodeAddr", "ExIp", LocalAddr)
			externalIP = LocalAddr
			OutSide = true
			goto SET_ADDR
		}

		if len(cfg.Seeds) == 0 {
			goto SET_ADDR
		}
		for _, seed := range cfg.Seeds {

			pcli := NewCli(nil)
			selfexaddrs, outside := pcli.GetExternIP(seed)
			if len(selfexaddrs) != 0 {
				OutSide = outside
				externalIP = selfexaddrs
				log.Info("DetectNodeAddr", " seed Exterip", externalIP)
				break
			}

		}
	SET_ADDR:
		//如果nat,getSelfExternalAddr 无法发现自己的外网地址，则把localaddr 赋值给外网地址
		if len(externalIP) == 0 {
			externalIP = LocalAddr
			log.Info("DetectNodeAddr", " SET_ADDR Exterip", externalIP)
		}

		var externaladdr string
		var externalPort int

		if cfg.GetIsSeed() == true || OutSide == true {
			externalPort = defaultPort
		} else {
			exportBytes, _ := n.nodeInfo.addrBook.bookDb.Get([]byte(externalPortTag))
			if len(exportBytes) != 0 {
				externalPort = int(P2pComm.BytesToInt32(exportBytes))
			} else {
				externalPort = defalutNatPort
			}
		}

		externaladdr = fmt.Sprintf("%v:%v", externalIP, externalPort)

		log.Debug("DetectionNodeAddr", "AddBlackList", externaladdr)
		n.nodeInfo.blacklist.Add(externaladdr) //把自己的外网地址加入到黑名单，以防连接self
		if exaddr, err := NewNetAddressString(externaladdr); err == nil {
			n.nodeInfo.SetExternalAddr(exaddr)

		} else {
			log.Error("DetectionNodeAddr", "error", err.Error())
		}
		if listaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", LocalAddr, defaultPort)); err == nil {
			n.nodeInfo.SetListenAddr(listaddr)
		}

		log.Info("DetectionNodeAddr", " Finish ExternalIp", externalIP, "LocalAddr", LocalAddr, "IsOutSide", OutSide)
		break
	}
}

func (n *Node) natMapPort() {
	if OutSide == true || n.nodeInfo.cfg.GetServerStart() == false { //在外网或者关闭p2p server 的节点不需要映射端口
		return
	}
	n.waitForNat()
	var err error

	_, nodename := n.nodeInfo.addrBook.GetPrivPubKey()
	for i := 0; i < tryMapPortTimes; i++ {
		err = nat.Any().AddMapping("TCP", int(n.nodeInfo.GetExternalAddr().Port), int(defaultPort), nodename[:8], time.Minute*20)
		if err != nil {
			if i > tryMapPortTimes/2 { //如果连续失败次数超过最大限制次数的二分之一则切换为随机端口映射
				log.Error("NatMapPort", "err", err.Error())
				n.FlushNodePort(defaultPort, uint16(rand.Intn(64512)+1023))

			}
			log.Info("NatMapPort", "External Port", n.nodeInfo.GetExternalAddr())
			continue
		}

		break
	}

	if err != nil {
		//映射失败
		log.Warn("NatMapPort", "Nat Faild", "Sevice=6")
		n.nodeInfo.natResultChain <- false
		return
	}

	n.nodeInfo.addrBook.bookDb.Set([]byte(externalPortTag),
		P2pComm.Int32ToBytes(int32(n.nodeInfo.GetExternalAddr().Port))) //把映射成功的端口信息刷入db
	log.Info("natMapPort", "export inser into db", n.nodeInfo.GetExternalAddr().Port)
	n.nodeInfo.natResultChain <- true
	refresh := time.NewTimer(mapUpdateInterval)
	for {

		select {
		case <-refresh.C:
			nat.Any().AddMapping("TCP", int(n.nodeInfo.GetExternalAddr().Port), int(defaultPort), "chain33-p2p", time.Minute*20)
			refresh.Reset(mapUpdateInterval)
		default:
			if n.IsClose() {
				return
			}
			time.Sleep(time.Second * 10)

		}

	}
}
func (n *Node) deleteNatMapPort() {
	nat.Any().DeleteMapping("TCP", int(n.nodeInfo.GetExternalAddr().Port), int(defaultPort))
}

func (n *Node) waitForNat() {
	<-n.nodeInfo.natNoticeChain
	return
}
