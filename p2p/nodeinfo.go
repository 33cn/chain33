// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"sync"
	"sync/atomic"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// NodeInfo is interface object of the node
type NodeInfo struct {
	mtx            sync.Mutex
	externalAddr   *NetAddress
	listenAddr     *NetAddress
	monitorChan    chan *Peer
	natNoticeChain chan struct{}
	natResultChain chan bool
	cfg            *types.P2P
	client         queue.Client
	blacklist      *BlackList
	peerInfos      *PeerInfos
	addrBook       *AddrBook // known peers
	natDone        int32
	outSide        int32
	ServiceType    int32
	channelVersion int32
}

// NewNodeInfo new a node object
func NewNodeInfo(cfg *types.P2P) *NodeInfo {
	nodeInfo := new(NodeInfo)
	nodeInfo.monitorChan = make(chan *Peer, 1024)
	nodeInfo.natNoticeChain = make(chan struct{}, 1)
	nodeInfo.natResultChain = make(chan bool, 1)
	nodeInfo.blacklist = &BlackList{badPeers: make(map[string]int64)}
	nodeInfo.cfg = cfg
	nodeInfo.peerInfos = new(PeerInfos)
	nodeInfo.peerInfos.infos = make(map[string]*types.Peer)
	nodeInfo.externalAddr = new(NetAddress)
	nodeInfo.listenAddr = new(NetAddress)
	nodeInfo.addrBook = NewAddrBook(cfg)
	nodeInfo.channelVersion = calcChannelVersion(cfg.Channel)
	return nodeInfo
}

// PeerInfos encapsulation peer information
type PeerInfos struct {
	mtx sync.Mutex
	//key:peerName
	infos map[string]*types.Peer
}

// PeerSize return a size of peer information
func (p *PeerInfos) PeerSize() int {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return len(p.infos)
}

// FlushPeerInfos flush peer information
func (p *PeerInfos) FlushPeerInfos(in []*types.Peer) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for k := range p.infos {
		delete(p.infos, k)
	}

	for _, peer := range in {
		p.infos[peer.GetName()] = peer
	}
}

// GetPeerInfos return a map for peerinfos
func (p *PeerInfos) GetPeerInfos() map[string]*types.Peer {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	var pinfos = make(map[string]*types.Peer)
	for k, v := range p.infos {
		pinfos[k] = v
	}
	return pinfos
}

// SetPeerInfo modify peer infos
func (p *PeerInfos) SetPeerInfo(peer *types.Peer) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if peer.GetName() == "" {
		return
	}
	p.infos[peer.GetName()] = peer
}

// GetPeerInfo return a infos by key
func (p *PeerInfos) GetPeerInfo(peerName string) *types.Peer {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if peer, ok := p.infos[peerName]; ok {
		return peer
	}
	return nil
}

// BlackList badpeers list
type BlackList struct {
	mtx      sync.Mutex
	badPeers map[string]int64
}

// FetchPeerInfo get peerinfo by node
func (nf *NodeInfo) FetchPeerInfo(n *Node) {
	var peerlist []*types.Peer
	peerInfos := nf.latestPeerInfo(n)
	for _, peerinfo := range peerInfos {
		peerlist = append(peerlist, peerinfo)
	}
	nf.flushPeerInfos(peerlist)
}

func (nf *NodeInfo) flushPeerInfos(in []*types.Peer) {
	nf.peerInfos.FlushPeerInfos(in)
}
func (nf *NodeInfo) latestPeerInfo(n *Node) map[string]*types.Peer {
	var peerlist = make(map[string]*types.Peer)
	peers := n.GetRegisterPeers()
	log.Debug("latestPeerInfo", "register peer num", len(peers))
	for _, peer := range peers {

		if !peer.GetRunning() || peer.Addr() == n.nodeInfo.GetExternalAddr().String() {
			n.remove(peer.Addr())
			continue
		}

		peerinfo, err := peer.GetPeerInfo()
		if err != nil || peerinfo.GetName() == "" {
			P2pComm.CollectPeerStat(err, peer)
			log.Error("latestPeerInfo", "Err", err, "peer", peer.Addr())
			continue
		}
		var pr types.Peer
		pr.Addr = peerinfo.GetAddr()
		pr.Port = peerinfo.GetPort()
		pr.Name = peerinfo.GetName()
		pr.MempoolSize = peerinfo.GetMempoolSize()
		pr.Header = peerinfo.GetHeader()
		peerlist[pr.Name] = &pr
	}
	return peerlist
}

// Set modidy nodeinfo by nodeinfo
func (nf *NodeInfo) Set(n *NodeInfo) {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	nf = n
}

// Get return nodeinfo
func (nf *NodeInfo) Get() *NodeInfo {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	return nf
}

// SetExternalAddr modidy address of the nodeinfo
func (nf *NodeInfo) SetExternalAddr(addr *NetAddress) {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	nf.externalAddr = addr
}

// GetExternalAddr return external address
func (nf *NodeInfo) GetExternalAddr() *NetAddress {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	return nf.externalAddr
}

// SetListenAddr modify listen address
func (nf *NodeInfo) SetListenAddr(addr *NetAddress) {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	nf.listenAddr = addr
}

// GetListenAddr return listen address
func (nf *NodeInfo) GetListenAddr() *NetAddress {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	return nf.listenAddr
}

// SetNatDone modify natdone
func (nf *NodeInfo) SetNatDone() {
	atomic.StoreInt32(&nf.natDone, 1)
}

// IsNatDone return ture and false
func (nf *NodeInfo) IsNatDone() bool {
	return atomic.LoadInt32(&nf.natDone) == 1
}

// IsOutService return true and false for out service
func (nf *NodeInfo) IsOutService() bool {

	if !nf.cfg.ServerStart {
		return false
	}

	if nf.OutSide() || nf.ServiceTy() == Service {
		return true
	}
	return false
}

// SetServiceTy set service type
func (nf *NodeInfo) SetServiceTy(ty int32) {
	atomic.StoreInt32(&nf.ServiceType, ty)
}

// ServiceTy return serveice type
func (nf *NodeInfo) ServiceTy() int32 {
	return atomic.LoadInt32(&nf.ServiceType)
}

// SetNetSide set net side
func (nf *NodeInfo) SetNetSide(ok bool) {
	var isoutside int32
	if ok {
		isoutside = 1
	}
	atomic.StoreInt32(&nf.outSide, isoutside)

}

// OutSide return true and false for outside
func (nf *NodeInfo) OutSide() bool {
	return atomic.LoadInt32(&nf.outSide) == 1
}

// Add add badpeer
func (bl *BlackList) Add(addr string, deadline int64) {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()
	bl.badPeers[addr] = types.Now().Unix() + deadline

}

// Delete delete badpeer
func (bl *BlackList) Delete(addr string) {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()
	delete(bl.badPeers, addr)
}

// Has the badpeer true and false
func (bl *BlackList) Has(addr string) bool {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	if _, ok := bl.badPeers[addr]; ok {
		return true
	}
	return false
}

// GetBadPeers reurn black list peers
func (bl *BlackList) GetBadPeers() map[string]int64 {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	var copyData = make(map[string]int64)

	for k, v := range bl.badPeers {
		copyData[k] = v
	}
	return copyData

}
