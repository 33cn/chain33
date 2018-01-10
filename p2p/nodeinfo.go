package p2p

import (
	"fmt"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

type NodeInfo struct {
	mtx sync.Mutex

	pubKey           []byte      `json:"pub_key"`
	network          string      `json:"network"`
	externalAddr     *NetAddress `json:"remote_addr"`
	listenAddr       *NetAddress `json:"listen_addr"`
	version          string      `json:"version"`
	monitorChan      chan *peer
	versionChan      chan struct{}
	p2pBroadcastChan chan interface{}
	cfg              *types.P2P
	q                *queue.Queue
	qclient          queue.IClient
	blacklist        *BlackList
	peerInfos        *PeerInfos
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
	//首先清空之前的数据
	for k, _ := range p.infos {
		delete(p.infos, k)
	}
	//重新插入新数据
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
func (nf *NodeInfo) GrpcConfig() grpc.ServiceConfig {

	var ready = true
	var defaultRespSize = 1024 * 1024 * 60
	var defaultReqSize = 1024 * 1024 * 10
	var defaulttimeout = 30 * time.Second
	var pingtimeout = 10 * time.Second
	var MethodConf = map[string]grpc.MethodConfig{
		"/types/p2pgservice/Ping":           grpc.MethodConfig{WaitForReady: &ready, Timeout: &pingtimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types/p2pgservice/Version2":       grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types/p2pgservice/BroadCastTx":    grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types/p2pgservice/GetMemPool":     grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types/p2pgservice/GetData":        grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types/p2pgservice/GetBlocks":      grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types/p2pgservice/GetPeerInfo":    grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
		"/types/p2pgservice/BroadCastBlock": grpc.MethodConfig{WaitForReady: &ready, Timeout: &defaulttimeout, MaxRespSize: &defaultRespSize, MaxReqSize: &defaultReqSize},
	}

	return grpc.ServiceConfig{Methods: MethodConf}

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
