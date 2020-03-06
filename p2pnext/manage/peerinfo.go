package manage

import (
	"sync"
	"time"

	"github.com/33cn/chain33/queue"

	"github.com/33cn/chain33/types"
)

// manage peer info

type PeerInfoManager struct {
	peerInfo sync.Map
	client   queue.Client
	done     chan struct{}
}

type peerStoreInfo struct {
	storeTime time.Duration
	peer      *types.Peer
}

func (p *PeerInfoManager) Add(pid string, info *types.Peer) {
	var storeInfo peerStoreInfo
	storeInfo.storeTime = time.Duration(time.Now().Unix())
	storeInfo.peer = info
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

//只获取
func (p *PeerInfoManager) GetPeerInfoInMin(key string) *types.Peer {
	v, ok := p.peerInfo.Load(key)
	if !ok {
		return nil
	}
	info := v.(*peerStoreInfo)
	if time.Duration(time.Now().Unix())-info.storeTime > 60 {
		p.peerInfo.Delete(key)
		return nil
	}
	return info.peer
}

func (p *PeerInfoManager) FetchPeerInfosInMin() []*types.Peer {

	var peers []*types.Peer
	p.peerInfo.Range(func(key interface{}, value interface{}) bool {
		info := value.(*peerStoreInfo)
		if time.Duration(time.Now().Unix())-info.storeTime > 60 {
			p.peerInfo.Delete(key)
			return true
		}

		peers = append(peers, info.peer)
		return true
	})

	return peers
}

func (p *PeerInfoManager) getPeerInfos() {

	msg := p.client.NewMessage("p2p", types.EventPeerInfo, nil)
	p.client.SendTimeout(msg, true, time.Second*5)
	resp, err := p.client.WaitTimeout(msg, time.Second*20)
	if err != nil {
		log.Error("MonitorPeerInfos", "err----------->", err)
		return
	}

	if peerlist, ok := resp.GetData().(*types.PeerList); ok {
		for _, peer := range peerlist.GetPeers() {
			p.Add(peer.GetName(), peer)
		}

	}
}
func (p *PeerInfoManager) MonitorPeerInfos() {
	for {
		select {
		case <-time.After(time.Second * 30): //每30秒主动刷新
			p.getPeerInfos()
		case <-time.After(time.Minute):
			log.Info("MonitorPeerInfos", "Num", len(p.FetchPeerInfosInMin()))

		case <-p.done:
			return

		}
	}
}

func (p *PeerInfoManager) Close() {
	defer func() {
		if recover() != nil {
			log.Error("channel reclosed")
		}
	}()

	close(p.done)
}

func NewPeerInfoManager(cli queue.Client) *PeerInfoManager {

	peerInfoManage := &PeerInfoManager{done: make(chan struct{})}
	peerInfoManage.client = cli
	go peerInfoManage.MonitorPeerInfos()
	return peerInfoManage
}
