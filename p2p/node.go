// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"fmt"
	"math/rand"

	"github.com/33cn/chain33/p2p/manage"

	//"strings"
	"sync/atomic"

	"sync"
	"time"

	"github.com/33cn/chain33/common/pubsub"
	"github.com/33cn/chain33/p2p/nat"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// 启动Node节点
// 1.启动监听GRPC Server
// 2.检测自身地址
// 3.启动端口映射
// 4.启动监控模块，进行节点管理

// Start Node server
func (n *Node) Start() {
	if n.server != nil {
		n.server.Start()
	}
	n.detectNodeAddr()
	n.monitor()
	atomic.StoreInt32(&n.closed, 0)
	go n.doNat()

}

// Close node server
func (n *Node) Close() {
	//避免重复
	if !atomic.CompareAndSwapInt32(&n.closed, 0, 1) {
		return
	}
	if n.server != nil {
		n.server.Close()
	}
	log.Debug("stop", "listen", "closed")
	n.nodeInfo.addrBook.Close()
	n.nodeInfo.monitorChan <- nil
	log.Debug("stop", "addrBook", "closed")
	n.removeAll()
	if peerAddrFilter != nil {
		peerAddrFilter.Close()
	}
	n.deleteNatMapPort()
	n.pubsub.Shutdown()
	log.Info("stop", "PeerRemoeAll", "closed")

}

func (n *Node) isClose() bool {
	return atomic.LoadInt32(&n.closed) == 1
}

// Node attribute
type Node struct {
	omtx       sync.Mutex
	nodeInfo   *NodeInfo
	cmtx       sync.Mutex
	cacheBound map[string]*Peer
	outBound   map[string]*Peer
	server     *listener
	listenPort int
	innerSeeds sync.Map
	cfgSeeds   sync.Map
	closed     int32
	pubsub     *pubsub.PubSub
	chainCfg   *types.Chain33Config
	p2pMgr     *manage.P2PMgr
}

// SetQueueClient return client for nodeinfo
func (n *Node) SetQueueClient(client queue.Client) {
	n.nodeInfo.client = client
}

// NewNode produce a node object
func NewNode(mgr *manage.P2PMgr, mcfg *subConfig) (*Node, error) {

	cfg := mgr.ChainCfg
	node := &Node{
		outBound:   make(map[string]*Peer),
		cacheBound: make(map[string]*Peer),
		pubsub:     pubsub.NewPubSub(10200),
		p2pMgr:     mgr,
	}
	node.listenPort = 13802
	if mcfg.Port != 0 && mcfg.Port <= 65535 && mcfg.Port > 1024 {
		node.listenPort = int(mcfg.Port)

	}

	if mcfg.InnerSeedEnable {
		seeds := MainNetSeeds
		if cfg.IsTestNet() {
			seeds = TestNetSeeds
		}

		for _, seed := range seeds {
			node.innerSeeds.Store(seed, "inner")
		}
	}

	for _, seed := range mcfg.Seeds {
		node.cfgSeeds.Store(seed, "cfg")
	}
	node.nodeInfo = NewNodeInfo(cfg.GetModuleConfig().P2P, mcfg)
	if mcfg.ServerStart {
		node.server = newListener(protocol, node)
	}
	node.chainCfg = cfg
	return node, nil
}

func (n *Node) flushNodePort(localport, export uint16) {

	if exaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", n.nodeInfo.GetExternalAddr().IP.String(), export)); err == nil {
		n.nodeInfo.SetExternalAddr(exaddr)
		n.nodeInfo.addrBook.AddOurAddress(exaddr)
	}

	if listenAddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", n.nodeInfo.GetListenAddr().IP.String(), localport)); err == nil {
		n.nodeInfo.SetListenAddr(listenAddr)
		n.nodeInfo.addrBook.AddOurAddress(listenAddr)
	}

}

func (n *Node) natOk() bool {
	n.nodeInfo.natNoticeChain <- struct{}{}
	ok := <-n.nodeInfo.natResultChain
	return ok
}

func (n *Node) doNat() {
	//测试是否在外网，当连接节点数大于0的时候，测试13802端口
	for {
		if n.Size() > 0 {
			break
		}
		time.Sleep(time.Second)
	}
	testExaddr := fmt.Sprintf("%v:%v", n.nodeInfo.GetExternalAddr().IP.String(), n.listenPort)
	log.Info("TestNetAddr", "testExaddr", testExaddr)
	if len(P2pComm.AddrRouteble([]string{testExaddr}, n.nodeInfo.channelVersion)) != 0 {
		log.Info("node outside")
		n.nodeInfo.SetNetSide(true)
		if netexaddr, err := NewNetAddressString(testExaddr); err == nil {
			n.nodeInfo.SetExternalAddr(netexaddr)
			n.nodeInfo.addrBook.AddOurAddress(netexaddr)
		}
		return
	}
	log.Info("node inside")
	//在内网，并且非种子节点，则进行端口映射
	if !n.nodeInfo.OutSide() && !n.nodeInfo.cfg.IsSeed && n.nodeInfo.cfg.ServerStart {

		go n.natMapPort()
		if !n.natOk() {
			log.Info("doNat", "Nat", "Faild")
		} else {
			//检测映射成功后，能否对外提供服务
			for {
				if n.Size() > 0 {
					break
				}
				time.Sleep(time.Millisecond * 100)
			}

			p2pcli := NewNormalP2PCli()
			//测试映射后的端口能否连通或者外网+本地端口
			if p2pcli.CheckPeerNatOk(n.nodeInfo.GetExternalAddr().String(), n.nodeInfo) ||
				p2pcli.CheckPeerNatOk(fmt.Sprintf("%v:%v", n.nodeInfo.GetExternalAddr().IP.String(), n.listenPort), n.nodeInfo) {
				n.nodeInfo.SetServiceTy(Service)
				log.Info("doNat", "NatOk", "Support Service")
			} else {
				n.nodeInfo.SetServiceTy(Service - nodeNetwork)
				log.Info("doNat", "NatOk", "No Support Service")
			}

		}

	}

	//n.nodeInfo.SetNatDone()
	n.nodeInfo.addrBook.AddOurAddress(n.nodeInfo.GetExternalAddr())
	n.nodeInfo.addrBook.AddOurAddress(n.nodeInfo.GetListenAddr())
	if selefNet, err := NewNetAddressString(fmt.Sprintf("127.0.0.1:%v", n.nodeInfo.GetListenAddr().Port)); err == nil {
		n.nodeInfo.addrBook.AddOurAddress(selefNet)
	}
}

func (n *Node) addPeer(pr *Peer) {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	if peer, ok := n.outBound[pr.Addr()]; ok {
		log.Info("AddPeer", "delete peer", pr.Addr())
		n.nodeInfo.addrBook.RemoveAddr(peer.Addr())
		delete(n.outBound, pr.Addr())
		peer.Close()

	}
	log.Debug("AddPeer", "peer", pr.Addr())
	n.outBound[pr.Addr()] = pr
	pr.Start()
}

// AddCachePeer  add cacheBound map by addr
func (n *Node) AddCachePeer(pr *Peer) {
	n.cmtx.Lock()
	defer n.cmtx.Unlock()
	n.cacheBound[pr.Addr()] = pr
}

// RemoveCachePeer remove cacheBound by addr
func (n *Node) RemoveCachePeer(addr string) {
	n.cmtx.Lock()
	defer n.cmtx.Unlock()
	delete(n.cacheBound, addr)
}

// HasCacheBound peer whether exists according to address
func (n *Node) HasCacheBound(addr string) bool {
	n.cmtx.Lock()
	defer n.cmtx.Unlock()
	_, ok := n.cacheBound[addr]
	return ok

}

// CacheBoundsSize return node cachebount size
func (n *Node) CacheBoundsSize() int {
	n.cmtx.Lock()
	defer n.cmtx.Unlock()
	return len(n.cacheBound)
}

// GetCacheBounds get node cachebounds
func (n *Node) GetCacheBounds() []*Peer {
	n.cmtx.Lock()
	defer n.cmtx.Unlock()
	var peers []*Peer
	if len(n.cacheBound) == 0 {
		return peers
	}
	for _, peer := range n.cacheBound {
		peers = append(peers, peer)

	}
	return peers
}

// Size return size for peersize
func (n *Node) Size() int {

	return n.nodeInfo.peerInfos.PeerSize()
}

// Has peer whether exists according to address
func (n *Node) Has(paddr string) bool {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	_, ok := n.outBound[paddr]
	return ok
}

// GetRegisterPeer return one peer according to paddr
func (n *Node) GetRegisterPeer(paddr string) *Peer {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	if peer, ok := n.outBound[paddr]; ok {
		return peer
	}
	return nil
}

// GetRegisterPeers return peers
func (n *Node) GetRegisterPeers() []*Peer {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	var peers []*Peer
	if len(n.outBound) == 0 {
		return peers
	}
	for _, peer := range n.outBound {
		peers = append(peers, peer)

	}
	return peers
}

// GetActivePeers return activities of the peers and infos
func (n *Node) GetActivePeers() (map[string]*Peer, map[string]*types.Peer) {
	regPeers := n.GetRegisterPeers()
	infos := n.nodeInfo.peerInfos.GetPeerInfos()

	var peers = make(map[string]*Peer)
	for _, peer := range regPeers {
		name := peer.GetPeerName()
		if _, ok := infos[name]; ok {

			peers[name] = peer
		}
	}
	return peers, infos
}
func (n *Node) remove(peerAddr string) {

	n.omtx.Lock()
	defer n.omtx.Unlock()
	peer, ok := n.outBound[peerAddr]
	if ok {
		delete(n.outBound, peerAddr)
		peer.Close()
	}
}

func (n *Node) removeAll() {
	n.omtx.Lock()
	defer n.omtx.Unlock()
	for addr, peer := range n.outBound {
		delete(n.outBound, addr)
		peer.Close()
	}
}

func (n *Node) monitor() {
	go n.monitorErrPeer()
	//固定模式, 只连接seeds配置节点
	go n.monitorCfgSeeds()
	if !n.nodeInfo.cfg.FixedSeed {
		go n.getAddrFromOnline()
		go n.getAddrFromAddrBook()
	}
	go n.monitorPeerInfo()
	go n.monitorDialPeers()
	go n.monitorBlackList()
	go n.monitorFilter()
	go n.monitorPeers()
	go n.nodeReBalance()
}

func (n *Node) needMore() bool {
	outBoundNum := n.Size()
	return !(outBoundNum >= maxOutBoundNum)
}

func (n *Node) detectNodeAddr() {

	var externalIP string
	for {
		cfg := n.nodeInfo.cfg
		laddr := P2pComm.GetLocalAddr()
		//LocalAddr = laddr
		log.Info("DetectNodeAddr", "addr:", laddr)
		if laddr == "" {
			log.Error("DetectNodeAddr", "NetWork Disable p2p Disable", "Retry until Network enable")
			time.Sleep(time.Second * 5)
			continue
		}
		log.Info("detectNodeAddr", "LocalAddr", laddr)
		if cfg.IsSeed {
			log.Info("DetectNodeAddr", "ExIp", laddr)
			externalIP = laddr
			n.nodeInfo.SetNetSide(true)
			//goto SET_ADDR
		}

		//如果nat,getSelfExternalAddr 无法发现自己的外网地址，则把localaddr 赋值给外网地址
		if len(externalIP) == 0 {
			externalIP = laddr
		}

		var externaladdr string
		var externalPort int

		if cfg.IsSeed {
			externalPort = n.listenPort
		} else {
			exportBytes, err := n.nodeInfo.addrBook.bookDb.Get([]byte(externalPortTag))
			if len(exportBytes) != 0 {
				externalPort = int(P2pComm.BytesToInt32(exportBytes))
			} else {
				externalPort = n.listenPort
			}
			if err != nil {
				log.Error("bookDb Get", "nodePort", n.listenPort, "externalPortTag fail err:", err)
			}
		}

		externaladdr = fmt.Sprintf("%v:%v", externalIP, externalPort)
		log.Debug("DetectionNodeAddr", "AddBlackList", externaladdr)
		n.nodeInfo.blacklist.Add(externaladdr, 0) //把自己的外网地址永久加入到黑名单，以防连接self
		if exaddr, err := NewNetAddressString(externaladdr); err == nil {
			n.nodeInfo.SetExternalAddr(exaddr)
			n.nodeInfo.addrBook.AddOurAddress(exaddr)

		} else {
			log.Error("DetectionNodeAddr", "error", err.Error())
		}

		if listaddr, err := NewNetAddressString(fmt.Sprintf("%v:%v", laddr, n.listenPort)); err == nil {
			n.nodeInfo.SetListenAddr(listaddr)
			n.nodeInfo.addrBook.AddOurAddress(listaddr)
		}

		break
	}
}

func (n *Node) natMapPort() {

	n.natNotice()
	for {
		if n.Size() > 0 {
			break
		}
		time.Sleep(time.Second)
	}
	var err error
	if len(P2pComm.AddrRouteble([]string{n.nodeInfo.GetExternalAddr().String()}, n.nodeInfo.channelVersion)) != 0 { //判断能否连通要映射的端口
		log.Info("natMapPort", "addr", "routeble")
		p2pcli := NewNormalP2PCli() //检查要映射的IP地址是否已经被映射成功
		ok := p2pcli.CheckSelf(n.nodeInfo.GetExternalAddr().String(), n.nodeInfo)
		if !ok {
			log.Info("natMapPort", "port is used", n.nodeInfo.GetExternalAddr().String())
			n.flushNodePort(uint16(n.listenPort), uint16(rand.Intn(64512)+1023))
		}

	}
	_, nodename := n.nodeInfo.addrBook.GetPrivPubKey()
	log.Info("natMapPort", "netport", n.nodeInfo.GetExternalAddr().Port)
	for i := 0; i < tryMapPortTimes; i++ {
		//映射事件持续约48小时
		err = nat.Any().AddMapping("TCP", int(n.nodeInfo.GetExternalAddr().Port), n.listenPort, nodename[:8], time.Hour*48)
		if err != nil {
			if i > tryMapPortTimes/2 { //如果连续失败次数超过最大限制次数的二分之一则切换为随机端口映射
				log.Warn("TryNatMapPortFailed", "tryTimes", i, "err", err.Error())
				n.flushNodePort(uint16(n.listenPort), uint16(rand.Intn(64512)+1023))

			}
			log.Info("NatMapPort", "External Port", n.nodeInfo.GetExternalAddr().Port)
			continue
		}

		break
	}

	if err != nil {
		//映射失败
		log.Warn("NatMapPort", "Nat", "Faild")
		n.flushNodePort(uint16(n.listenPort), uint16(n.listenPort))
		n.nodeInfo.natResultChain <- false
		return
	}

	err = n.nodeInfo.addrBook.bookDb.Set([]byte(externalPortTag),
		P2pComm.Int32ToBytes(int32(n.nodeInfo.GetExternalAddr().Port))) //把映射成功的端口信息刷入db
	if err != nil {
		log.Error("NatMapPort", "dbErr", err)
		return
	}
	log.Info("natMapPort", "export insert into db", n.nodeInfo.GetExternalAddr().Port)
	n.nodeInfo.natResultChain <- true
	refresh := time.NewTimer(mapUpdateInterval)
	defer refresh.Stop()
	for {
		<-refresh.C
		log.Info("NatWorkRefresh")
		for {
			if err := nat.Any().AddMapping("TCP", int(n.nodeInfo.GetExternalAddr().Port), n.listenPort, nodename[:8], time.Hour*48); err != nil {
				log.Error("NatMapPort update", "err", err.Error())
				time.Sleep(time.Second)
				continue
			}
			break
		}
		refresh.Reset(mapUpdateInterval)

	}
}
func (n *Node) deleteNatMapPort() {

	if n.nodeInfo.OutSide() {
		return
	}

	err := nat.Any().DeleteMapping("TCP", int(n.nodeInfo.GetExternalAddr().Port), n.listenPort)
	if err != nil {
		log.Error("deleteNatMapPort", "DeleteMapping err", err.Error())
	}

}

func (n *Node) natNotice() {
	<-n.nodeInfo.natNoticeChain
}

func (n *Node) verifyP2PChannel(channel int32) bool {
	return channel == n.nodeInfo.cfg.Channel
}

//检测该节点地址是否作为客户端连入, 此时需要维护双向连接, 增加了节点间的连接冗余
func (n *Node) isInBoundPeer(peerName string) (bool, *innerpeer) {

	if n.server == nil || n.server.p2pserver == nil {
		return false, nil
	}
	//查询连入的客户端
	info := n.server.p2pserver.getInBoundPeerInfo(peerName)
	return info != nil, info
}
