package p2p

import (
	"fmt"
	"sync"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

type NodeInfo struct {
	mtx sync.Mutex

	pubKey         []byte
	network        string
	externalAddr   *NetAddress
	listenAddr     *NetAddress
	version        string
	monitorChan    chan *peer
	natNoticeChain chan struct{}
	natResultChain chan bool
	cfg            *types.P2P
	q              *queue.Queue
	qclient        queue.Client
	blacklist      *BlackList
	peerInfos      *PeerInfos
}

func NewNodeInfo(cfg *types.P2P) *NodeInfo {
	nodeInfo := new(NodeInfo)
	nodeInfo.monitorChan = make(chan *peer, 1024)
	nodeInfo.natNoticeChain = make(chan struct{}, 1)
	nodeInfo.natResultChain = make(chan bool, 1)
	nodeInfo.blacklist = &BlackList{badPeers: make(map[string]bool)}
	nodeInfo.cfg = cfg
	nodeInfo.peerInfos = new(PeerInfos)
	nodeInfo.peerInfos.infos = make(map[string]*types.Peer)
	nodeInfo.externalAddr = new(NetAddress)
	nodeInfo.listenAddr = new(NetAddress)
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

func (p *PeerInfos) flushPeerInfos(in []*types.Peer) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	for k, _ := range p.infos {
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
	badPeers map[string]bool
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

func (bl *BlackList) Add(addr string) {
	bl.mtx.Lock()
	defer bl.mtx.Unlock()
	bl.badPeers[addr] = false
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
