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

	log.Info("peerManager MustSend start", "peerID", peerId, "protocol", p.protocol, "session", p.sessionID)
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

func (p *peerManager) removeSelf() {
	ids := p.peerIDs[:0]
	for _, id := range p.peerIDs {
		if id != p.selfID {
			ids = append(ids, id)
		}
	}
	p.peerIDs = ids
}

// EnsurePeersReady waits for peers to sync and initializes self info.
func (p *peerManager) EnsurePeersReady() {

	timeout := 3 * time.Second
	ticker := time.NewTicker(timeout)
	defer ticker.Stop()
	for {
		peers, err := FetchConnectedPeers(p.cli, timeout)
		if err != nil || len(peers) == 0 {
			log.Debug("EnsurePeersReady", "session", p.sessionID, "fetchConnectedPeers err:", err, "peers", len(peers))
			time.Sleep(timeout)
			continue
		}
		if p.selfID == "" {
			p.selfID = peers[len(peers)-1].Name
			p.removeSelf()
		}
		if p.hasAllPeersConnected(peers) {
			return
		}
		log.Debug("EnsurePeersReady waiting for peers to sync", "session", p.sessionID, "fetchPeers", len(peers))

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

// FetchConnectedPeers 获取已连接的节点
func FetchConnectedPeers(cli queue.Client, timeout time.Duration) ([]*types.Peer, error) {

	msg := cli.NewMessage("p2p", types.EventPeerInfo, nil)
	err := cli.Send(msg, true)
	if err != nil {
		log.Error("FetchConnectedPeers", "client.Send err:", err)
		return nil, err
	}
	resp, err := cli.WaitTimeout(msg, timeout)
	if err != nil {
		log.Error("FetchConnectedPeers", "client.Wait err:", err)
		return nil, err
	}
	return resp.GetData().(*types.PeerList).GetPeers(), nil

}

// GetValidPeerCombination 获取满足签名阈值要求的节点列表
func GetValidPeerCombination(cli queue.Client, threshold uint32, bks map[string]*BK) []string {

	connectedPeers, err := FetchConnectedPeers(cli, 3*time.Second)
	if err != nil || len(connectedPeers) < int(threshold) {
		log.Warn("GetValidPeerCombination", "threshold", threshold, 
		"connectedPeers", len(connectedPeers), "fetchConnectedPeers err:", err)
		return nil
	}

	ranks := make([]uint32, 0, len(bks))
	validPeers := make([]string, 0, len(connectedPeers))
	for _, peer := range connectedPeers {
		if bk, ok := bks[peer.Name]; ok {
			ranks = append(ranks, bk.Rank)
			validPeers = append(validPeers, peer.Name)
		}
	}
	if isValidRankCombination(ranks, threshold) {
		return validPeers
	}
	return nil
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
