// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

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
}

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
	return nodeInfo
}

type PeerInfos struct {
	mtx   sync.Mutex
	infos map[string]*types.Peer
}

func (p *PeerInfos) PeerSize() int {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	return len(p.infos)
}

func (p *PeerInfos) FlushPeerInfos(in []*types.Peer) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for k := range p.infos {
		delete(p.infos, k)
	}

	for _, peer := range in {
		p.infos[fmt.Sprintf("%v:%v", peer.GetAddr(), peer.GetPort())] = peer
	}
}

func (p *PeerInfos) GetPeerInfos() map[string]*types.Peer {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	var pinfos = make(map[string]*types.Peer)
	for k, v := range p.infos {
		pinfos[k] = v
	}
	return pinfos
}

func (p *PeerInfos) SetPeerInfo(peer *types.Peer) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	key := fmt.Sprintf("%v:%v", peer.GetAddr(), peer.GetPort())
	p.infos[key] = peer
}

func (p *PeerInfos) GetPeerInfo(key string) *types.Peer {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	if _, ok := p.infos[key]; ok {
		return p.infos[key]
	}
	return nil
}

type BlackList struct {
	mtx      sync.Mutex
	badPeers map[string]int64
}

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

		if peer.Addr() == n.nodeInfo.GetExternalAddr().String() { //fmt.Sprintf("%v:%v", ExternalIp, m.network.node.GetExterPort())
			continue
		}
		peerinfo, err := peer.GetPeerInfo(nf.cfg.Version)
		if err != nil {
			if err == types.ErrVersion {
				peer.version.SetSupport(false)
				P2pComm.CollectPeerStat(err, peer)
				log.Error("latestPeerInfo", "Err", err.Error(), "peer", peer.Addr())

			}
			continue
		}
		P2pComm.CollectPeerStat(err, peer)
		var pr types.Peer
		pr.Addr = peerinfo.GetAddr()
		pr.Port = peerinfo.GetPort()
		pr.Name = peerinfo.GetName()
		pr.MempoolSize = peerinfo.GetMempoolSize()
		pr.Header = peerinfo.GetHeader()
		peerlist[fmt.Sprintf("%v:%v", peerinfo.Addr, peerinfo.Port)] = &pr
	}
	return peerlist
}
func (nf *NodeInfo) Set(n *NodeInfo) {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	nf = n
}

func (nf *NodeInfo) Get() *NodeInfo {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	return nf
}
func (nf *NodeInfo) SetExternalAddr(addr *NetAddress) {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	nf.externalAddr = addr
}

func (nf *NodeInfo) GetExternalAddr() *NetAddress {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	return nf.externalAddr
}

func (nf *NodeInfo) SetListenAddr(addr *NetAddress) {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	nf.listenAddr = addr
}

func (nf *NodeInfo) GetListenAddr() *NetAddress {
	nf.mtx.Lock()
	defer nf.mtx.Unlock()
	return nf.listenAddr
}

func (nf *NodeInfo) SetNatDone() {
	atomic.StoreInt32(&nf.natDone, 1)
}

func (nf *NodeInfo) IsNatDone() bool {
	return atomic.LoadInt32(&nf.natDone) == 1
}

func (nf *NodeInfo) IsOutService() bool {

	if !nf.cfg.ServerStart {
		return false
	}

	if nf.OutSide() || nf.ServiceTy() == Service {
		return true
	}
	return false
}

func (nf *NodeInfo) SetServiceTy(ty int32) {
	atomic.StoreInt32(&nf.ServiceType, ty)
}

func (nf *NodeInfo) ServiceTy() int32 {
	return atomic.LoadInt32(&nf.ServiceType)
}

func (nf *NodeInfo) SetNetSide(ok bool) {
	var isoutside int32 = 0
	if ok {
		isoutside = 1
	}
	atomic.StoreInt32(&nf.outSide, isoutside)

}

func (nf *NodeInfo) OutSide() bool {
	return atomic.LoadInt32(&nf.outSide) == 1
}

func (bl *BlackList) Add(addr string, deadline int64) {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()
	bl.badPeers[addr] = types.Now().Unix() + deadline

}

func (bl *BlackList) Delete(addr string) {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()
	delete(bl.badPeers, addr)
}

func (bl *BlackList) Has(addr string) bool {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	if _, ok := bl.badPeers[addr]; ok {
		return true
	}
	return false
}

func (bl *BlackList) GetBadPeers() map[string]int64 {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()

	var copyData = make(map[string]int64)

	for k, v := range bl.badPeers {
		copyData[k] = v
	}
	return copyData

}
