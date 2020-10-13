package manage

import (
	"context"
	"sync"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
)

// manage peer info
const diffheightValue = 512

// PeerInfoManager peer info manager
type PeerInfoManager struct {
	ctx       context.Context
	peerInfo  sync.Map
	client    queue.Client
	host      host.Host
	blacklist *TimeCache
}

type peerStoreInfo struct {
	storeTime time.Time
	peer      *types.Peer
}

// NewPeerInfoManager new peer info manager
func NewPeerInfoManager(ctx context.Context, host host.Host, cli queue.Client) *PeerInfoManager {
	peerInfoManage := &PeerInfoManager{
		ctx:       ctx,
		client:    cli,
		host:      host,
		blacklist: NewTimeCache(ctx, time.Minute*5),
	}
	go peerInfoManage.start()
	return peerInfoManage
}

func (p *PeerInfoManager) Refresh(peer *types.Peer) {
	storeInfo := peerStoreInfo{
		storeTime: time.Now(),
		peer:      peer,
	}
	p.peerInfo.Store(peer.Name, &storeInfo)
}

func (p *PeerInfoManager) FetchAll() []*types.Peer {
	var peers []*types.Peer
	p.peerInfo.Range(func(key, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Now().Sub(info.storeTime) > time.Minute {
			p.peerInfo.Delete(key)
			return true
		}
		peers = append(peers, value.(*types.Peer))
		return true
	})
	return peers
}

func (p *PeerInfoManager) PeerHeight(pid peer.ID) int64 {
	v, ok := p.peerInfo.Load(pid.Pretty())
	if !ok {
		return -1
	}
	peerInfo, ok := v.(*types.Peer)
	if !ok {
		return -1
	}
	return peerInfo.Header.Height

}

func (p *PeerInfoManager) start() {
	for {
		select {
		case <-p.ctx.Done():
			return
		case <-time.After(time.Second * 30):
			//获取当前高度，过滤掉高度较低的节点
			msg := p.client.NewMessage("blockchain", types.EventGetLastHeader, nil)
			err := p.client.Send(msg, true)
			if err != nil {
				continue
			}
			resp, err := p.client.WaitTimeout(msg, time.Second*10)
			if err != nil {
				continue
			}
			header, ok := resp.GetData().(*types.Header)
			if !ok {
				continue
			}
			p.prune(header.GetHeight())
		}
	}
}
func (p *PeerInfoManager) prune(height int64) {
	p.peerInfo.Range(func(key interface{}, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Now().Sub(info.storeTime) > time.Minute {
			p.peerInfo.Delete(key)
			return true
		}
		//check blockheight,删除落后512高度的节点
		if info.peer.Header.GetHeight()+diffheightValue < height {
			id, _ := peer.Decode(key.(string))
			for _, conn := range p.host.Network().ConnsToPeer(id) {
				//判断是Inbound 还是Outbound
				if conn.Stat().Direction == network.DirOutbound {
					//断开连接
					_ = conn.Close()
				}
			}
		}
		return true
	})
}
