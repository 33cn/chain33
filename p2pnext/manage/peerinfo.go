package manage

import (
	"sync"

	"github.com/33cn/chain33/types"
)

// manage peer info

type PeerInfoManager struct {
	peerInfo sync.Map
}

func (p *PeerInfoManager) Store(pid string, info *types.P2PPeerInfo) {
	p.peerInfo.Store(pid, info)
}
func (p *PeerInfoManager) Copy(dest *types.Peer, source *types.P2PPeerInfo) {
	dest.Addr = source.GetAddr()
	dest.Name = source.GetName()
	dest.Header = source.GetHeader()
	dest.Self = false
	dest.MempoolSize = source.GetMempoolSize()
	dest.Port = source.GetPort()
}

func (p *PeerInfoManager) Load(key string) interface{} {
	v, ok := p.peerInfo.Load(key)
	if !ok {
		return nil
	}

	return v
}

func (p *PeerInfoManager) FetchPeers() []*types.Peer {

	var peers []*types.Peer
	p.peerInfo.Range(func(key interface{}, value interface{}) bool {
		info := value.(*types.P2PPeerInfo)
		var peer types.Peer
		p.Copy(&peer, info)
		peers = append(peers, &peer)
		return true
	})

	return peers
}

func NewPeerInfoManager() *PeerInfoManager {

	return &PeerInfoManager{}
}
