package tss

import (
	"context"
	"sort"
	"time"

	cryptocli "github.com/33cn/chain33/common/crypto/client"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type peerManager struct {
	protocol  string
	sessionID string
	selfID    string
	peerIDs   []string
	cli       queue.Client
	ctx       context.Context
}

// NewPeerManager new pm， peers是参与节点id列表
func NewPeerManager(peers []string, protocol, sessionID string) *peerManager {

	ctx := cryptocli.GetCryptoContext()
	return &peerManager{
		protocol:  protocol,
		sessionID: sessionID,
		peerIDs:   peers,
		cli:       ctx.Client,
		ctx:       ctx.Ctx,
	}
}

func (p *peerManager) NumPeers() uint32 {
	return uint32(len(p.peerIDs))
}

func (p *peerManager) SelfID() string {
	return p.selfID
}

func (p *peerManager) PeerIDs() []string {
	return p.peerIDs
}

func (p *peerManager) MustSend(peerId string, message interface{}) {

	protoMsg, ok := message.(types.Message)
	if !ok {
		log.Error("peerManager MustSend, invalid proto message")
		return
	}
	wMsg := &MessageWrapper{
		PeerID:    peerId,
		Protocol:  p.protocol,
		SessionID: p.sessionID,
		Msg:       types.Encode(protoMsg),
	}

	msg := p.cli.NewMessage("p2p", types.EventCryptoTssMsg, wMsg)
	err := p.cli.Send(msg, false)
	if err != nil {
		log.Error("peerManager MustSend", "peer", peerId,
			"protocol", p.protocol, "session", p.sessionID, "client.Send err:", err)
	}

}

// EnsurePeersReady waits for peers to sync and initializes self info.
func (p *peerManager) EnsurePeersReady() {

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		peers, err := p.fetchConnectedPeers()
		if err != nil || len(peers) == 0 {
			log.Debug("EnsurePeersReady", "session", p.sessionID, "fetchConnectedPeers err:", err)
			time.Sleep(time.Second*2)
			continue
		}
		if p.selfID == "" {
			p.selfID = peers[len(peers)-1].Name
		}
		if p.hasAllPeersConnected(peers) {
			return
		}
		log.Debug("EnsurePeersReady waiting for peers to sync", "session", p.sessionID)
	
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// EnsureSignerReady waits until connected peers satisfy threshold/rank requirements.
func (p *peerManager) EnsureSignerReady(threshold uint32, bks map[string]*BK) {
	
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		peers, err := p.fetchConnectedPeers()
		if err != nil || len(peers) == 0 {
			log.Debug("EnsureSignerReady", "session", p.sessionID, "fetchConnectedPeers err:", err)
			time.Sleep(time.Second*2)
			continue
		}
		if p.selfID == "" {
			p.selfID = peers[len(peers)-1].Name
		}

		if isValidPeerCombination(peers, threshold, bks) {
			return
		}
		log.Debug("EnsureSignerReady waiting for peers to sync", "session", p.sessionID)
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *peerManager) hasAllPeersConnected(peers []*types.Peer) bool {
	if len(peers) < len(p.peerIDs) {
		return false
	}
	peerMap := make(map[string]struct{}, len(peers))
	for _, peer := range peers {
		peerMap[peer.Name] = struct{}{}
	}

	for _, peerID := range p.peerIDs {
		if _, ok := peerMap[peerID]; !ok {
			return false
		}
	}
	return true
}

func (p *peerManager) fetchConnectedPeers() ([]*types.Peer, error) {

	msg := p.cli.NewMessage("p2p", types.EventPeerInfo, nil)
	err := p.cli.Send(msg, true)
	if err != nil {
		log.Error("fetchConnectedPeers", "client.Send err:", err)
		return nil, err
	}
	resp, err := p.cli.WaitTimeout(msg, 5*time.Second)
	if err != nil {
		log.Error("fetchConnectedPeers", "client.Wait err:", err)
		return nil, err
	}
	return resp.GetData().(*types.PeerList).GetPeers(), nil

}




func isValidPeerCombination(connectedPeers []*types.Peer, threshold uint32, bks map[string]*BK) bool {

	if uint32(len(connectedPeers)) < threshold {
		return false
	}

	ranks := make([]uint32, 0, len(bks))
	for _, peer := range connectedPeers {
		bk, ok := bks[peer.Name]
		if !ok {
			continue
		}
		ranks = append(ranks, bk.Rank)
	}
	return isValidRankCombination(ranks, threshold)
}



// isValidRankCombination 验证无序rank列表是否有效
// ranks: 参与签名的rank列表（可以无序）
// threshold: 签名阈值
func isValidRankCombination(ranks []uint32, threshold uint32) bool {
    // 1. 检查参与人数是否足够
    if uint32(len(ranks)) < threshold {
        return false
    }
    
    // 2. 升序排序
    sort.Slice(ranks, func(i, j int) bool {
        return ranks[i] < ranks[j]
    })
    
    // 3. 核心验证：rank_i <= i (i = 0, 1, ..., threshold-1)
    for i := uint32(0); i < threshold; i++ {
        if ranks[i] > i {
            return false
        }
    }
    
    return true
}