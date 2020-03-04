package manage

import (
	"sync"
	"time"

	"github.com/33cn/chain33/types"
)

// manage peer info

type PeerInfoManager struct {
	peerInfo sync.Map
}

type peerStoreInfo struct {
	storeTime time.Duration
	peerInfo  *types.P2PPeerInfo
}

func (p *PeerInfoManager) Store(pid string, info *types.P2PPeerInfo) {
	var storeInfo peerStoreInfo
	storeInfo.storeTime = time.Duration(time.Now().Unix())
	storeInfo.peerInfo = info
	p.peerInfo.Store(pid, &storeInfo)
}
func (p *PeerInfoManager) Copy(dest *types.Peer, source *types.P2PPeerInfo) {
	dest.Addr = source.GetAddr()
	dest.Name = source.GetName()
	dest.Header = source.GetHeader()
	dest.Self = false
	dest.MempoolSize = source.GetMempoolSize()
	dest.Port = source.GetPort()
}

func (p *PeerInfoManager) Load(key string) *types.P2PPeerInfo {
	v, ok := p.peerInfo.Load(key)
	if !ok {
		return nil
	}
	info := v.(*peerStoreInfo)
	if time.Duration(time.Now().Unix())-info.storeTime > 50 {
		p.peerInfo.Delete(key)
		return nil
	}
	return info.peerInfo
}

func (p *PeerInfoManager) FetchPeers() []*types.Peer {

	var peers []*types.Peer
	p.peerInfo.Range(func(key interface{}, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Duration(time.Now().Unix())-info.storeTime > 50 {
			p.peerInfo.Delete(key)
			return true
		}
		var peer types.Peer
		p.Copy(&peer, info.peerInfo)
		peers = append(peers, &peer)
		return true
	})

	return peers
}

func NewPeerInfoManager() *PeerInfoManager {

	return &PeerInfoManager{}
}
