package manage

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// PeerInfoManager peer info manager
type PeerInfoManager struct {
	maxHeight int64
	ctx       context.Context
	peerInfo  sync.Map
	client    queue.Client
	host      host.Host
}

type peerStoreInfo struct {
	storeTime time.Time
	peer      *types.Peer
}

// NewPeerInfoManager new peer info manager
func NewPeerInfoManager(ctx context.Context, host host.Host, cli queue.Client) *PeerInfoManager {
	peerInfoManage := &PeerInfoManager{
		ctx:    ctx,
		client: cli,
		host:   host,
	}
	go peerInfoManage.start()
	return peerInfoManage
}

// Refresh refreshes peer info
func (p *PeerInfoManager) Refresh(peer *types.Peer) {
	if peer == nil {
		return
	}
	storeInfo := peerStoreInfo{
		storeTime: time.Now(),
		peer:      peer,
	}
	p.peerInfo.Store(peer.Name, &storeInfo)
	if peer.GetHeader().GetHeight() > atomic.LoadInt64(&p.maxHeight) {
		atomic.StoreInt64(&p.maxHeight, peer.GetHeader().GetHeight())
	}
}

// Fetch returns info of given peer
func (p *PeerInfoManager) Fetch(pid peer.ID) *types.Peer {
	key := pid.Pretty()
	v, ok := p.peerInfo.Load(key)
	if !ok {
		return nil
	}
	if info, ok := v.(*peerStoreInfo); ok {
		if time.Since(info.storeTime) > time.Minute*30 {
			p.peerInfo.Delete(key)
			return nil
		}
		return info.peer
	}
	return nil
}

// FetchAll returns all peers info
func (p *PeerInfoManager) FetchAll() []*types.Peer {
	var peers []*types.Peer
	var self *types.Peer
	p.peerInfo.Range(func(key, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Since(info.storeTime) > time.Minute {
			p.peerInfo.Delete(key)
			return true
		}
		if key.(string) == p.host.ID().Pretty() {
			self = info.peer
			return true
		}
		peers = append(peers, info.peer)
		return true
	})
	if self != nil {
		peers = append(peers, self)
	}
	return peers
}

// PeerHeight returns block height of given peer
func (p *PeerInfoManager) PeerHeight(pid peer.ID) int64 {
	v, ok := p.peerInfo.Load(pid.Pretty())
	if !ok {
		return -1
	}
	info, ok := v.(*peerStoreInfo)
	if !ok {
		return -1
	}
	if info.peer.GetHeader() == nil {
		return -1
	}
	return info.peer.GetHeader().Height
}

// PeerMaxHeight returns max block height of all connected peers.
func (p *PeerInfoManager) PeerMaxHeight() int64 {
	return atomic.LoadInt64(&p.maxHeight)
}

func (p *PeerInfoManager) start() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-time.After(time.Second * 30):
			p.prune()
		}
	}
}
func (p *PeerInfoManager) prune() {
	p.peerInfo.Range(func(key interface{}, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Since(info.storeTime) > time.Minute*30 {
			p.peerInfo.Delete(key)
			return true
		}
		return true
	})
}
