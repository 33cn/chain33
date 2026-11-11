package tss

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	cryptocli "github.com/33cn/chain33/common/crypto/client"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	alicetypes "github.com/getamis/alice/types"
)

type peerManager struct {
	protocol  string
	sessionID string
	selfID    string
	peerIDs   []string
	cli       queue.Client
	ctx       context.Context
	// ready reports whether every listed peer was connected and the local peer id was resolved
	// before returning. Only NewReadyPeerManagerWithTimeout callers look at it.
	ready bool
}

// NewReadyPeerManager new peer manager with peers ready. It waits for the peers without a bound
// (until the process context is cancelled); callers that have a phase deadline should use
// NewReadyPeerManagerWithTimeout instead.
func NewReadyPeerManager(peers []string, protocol, sessionID string) alicetypes.PeerManager {
	return newPeerManager(peers, protocol, sessionID, 0)
}

// NewReadyPeerManagerWithTimeout behaves like NewReadyPeerManager but gives up after timeout, so a
// peer that never connects cannot block the calling phase for ever. A non-positive timeout keeps
// the unbounded behaviour. It returns errPeersNotReady instead of a half-initialised manager: the
// alice cores index their parameter maps by SelfID, so handing them a manager with an unresolved
// self id would panic.
func NewReadyPeerManagerWithTimeout(peers []string, protocol, sessionID string, timeout time.Duration) (alicetypes.PeerManager, error) {
	pm := newPeerManager(peers, protocol, sessionID, timeout)
	if !pm.ready {
		return nil, fmt.Errorf("%w: session %s, peers %v", errPeersNotReady, sessionID, peers)
	}
	return pm, nil
}

func newPeerManager(peers []string, protocol, sessionID string, timeout time.Duration) *peerManager {
	ctx := cryptocli.GetCryptoContext()
	pids := make([]string, len(peers))
	copy(pids, peers)
	pm := &peerManager{
		protocol:  protocol,
		sessionID: sessionID,
		peerIDs:   pids,
		cli:       ctx.Client,
		ctx:       ctx.Ctx,
	}
	pm.waitPeersReady(timeout)
	return pm
}

var errPeersNotReady = errors.New("tss: peers not connected before the phase timeout")

func (p *peerManager) NumPeers() uint32 {
	return uint32(len(p.peerIDs))
}

func (p *peerManager) SelfID() string {
	return p.selfID
}

func (p *peerManager) PeerIDs() []string {
	return p.peerIDs
}

func (p *peerManager) MustSend(peerID string, message interface{}) {

	protoMsg, ok := message.(types.Message)
	if !ok {
		log.Error("peerManager MustSend, invalid proto message")
		return
	}
	wMsg := &MessageWrapper{
		PeerID:    peerID,
		Protocol:  p.protocol,
		SessionID: p.sessionID,
		Msg:       types.Encode(protoMsg),
	}
	msg := p.cli.NewMessage("p2p", types.EventCryptoTssMsg, wMsg)
	err := p.cli.Send(msg, false)
	if err != nil {
		log.Error("peerManager MustSend", "peer", peerID,
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

// waitPeersReady waits for peers to sync and initializes self info. A non-positive timeout waits
// until the process context is cancelled; otherwise it gives up once the timeout elapses and
// leaves ready false.
func (p *peerManager) waitPeersReady(timeout time.Duration) {
	interval := 3 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	// A nil channel blocks for ever in select, which is exactly the unbounded case.
	var deadline <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		deadline = timer.C
	}
	for {
		peers, err := FetchConnectedPeers(p.cli, interval)
		if err == nil && len(peers) > 0 {
			if p.selfID == "" && peers[len(peers)-1].Self {
				p.selfID = peers[len(peers)-1].Name
				p.removeSelf()
			}
			if p.selfID != "" && p.hasAllPeersConnected(peers) {
				p.ready = true
				return
			}
			log.Debug("waitPeersReady waiting for peers to sync",
				"session", p.sessionID, "fetchPeers", len(peers))
		} else {
			log.Debug("waitPeersReady",
				"session", p.sessionID, "fetchConnectedPeers err:", err, "peers", len(peers))
		}
		select {
		case <-p.ctx.Done():
			return
		case <-deadline:
			log.Warn("waitPeersReady timed out", "session", p.sessionID,
				"peers", p.peerIDs, "selfID", p.selfID)
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

// GetValidPeerCombination 获取满足签名阈值要求的节点列表。
//
// 返回值恰好 threshold 个节点、按 rank 升序、且一定包含本节点：ProcessSign 要求
// len(peers) == threshold，而 alice 的签名核心按 SelfID 索引 bks / partial public key /
// Pedersen 参数，本地节点不在列表里会解引用 nil。之前的实现返回"所有已连接且有 bk 的
// peer"（可能多于 threshold，顺序是连接顺序），调用方按返回顺序截前 t 个可能漏掉本节点
// 或漏掉必需的 rank-0 节点，表现为签名阶段报错。
func GetValidPeerCombination(cli queue.Client, threshold uint32, bks map[string]*BK) []string {

	connectedPeers, err := FetchConnectedPeers(cli, 3*time.Second)
	if err != nil {
		log.Warn("GetValidPeerCombination", "threshold", threshold,
			"connectedPeers", len(connectedPeers), "fetchConnectedPeers err:", err)
		return nil
	}
	if threshold == 0 {
		return nil
	}

	candidates := make([]*types.Peer, 0, len(connectedPeers))
	for _, peer := range connectedPeers {
		if _, ok := bks[peer.Name]; ok {
			candidates = append(candidates, peer)
		}
	}
	return selectSignerSubset(candidates, threshold, bks)
}

// selectSignerSubset picks exactly threshold peers out of candidates: the lowest-rank ones, always
// including the local node, ordered by rank. It returns nil when no such subset satisfies the
// Birkhoff rule (rank_i <= i for i = 0..threshold-1).
func selectSignerSubset(candidates []*types.Peer, threshold uint32, bks map[string]*BK) []string {
	if threshold == 0 || uint32(len(candidates)) < threshold {
		log.Warn("selectSignerSubset", "threshold", threshold, "candidates", len(candidates))
		return nil
	}
	// Sort a copy: callers keep their own ordering, and the selection must not depend on the order
	// the transport happened to report peers in.
	sorted := make([]*types.Peer, len(candidates))
	copy(sorted, candidates)
	sort.SliceStable(sorted, func(i, j int) bool {
		if bks[sorted[i].Name].Rank != bks[sorted[j].Name].Rank {
			return bks[sorted[i].Name].Rank < bks[sorted[j].Name].Rank
		}
		return sorted[i].Name < sorted[j].Name
	})

	selected := make([]*types.Peer, threshold)
	copy(selected, sorted[:threshold])
	if !containsSelf(selected) {
		// The local node must be in the subset (alice indexes by SelfID), so swap it in for the
		// highest-rank pick and re-sort before the rank check.
		var self *types.Peer
		for _, peer := range sorted[threshold:] {
			if peer.Self {
				self = peer
				break
			}
		}
		if self == nil {
			log.Warn("selectSignerSubset self not among candidates", "threshold", threshold,
				"candidates", len(candidates))
			return nil
		}
		selected[threshold-1] = self
		sort.SliceStable(selected, func(i, j int) bool {
			if bks[selected[i].Name].Rank != bks[selected[j].Name].Rank {
				return bks[selected[i].Name].Rank < bks[selected[j].Name].Rank
			}
			return selected[i].Name < selected[j].Name
		})
	}

	ranks := make([]uint32, 0, len(selected))
	validPeers := make([]string, 0, len(selected))
	for _, peer := range selected {
		ranks = append(ranks, bks[peer.Name].Rank)
		validPeers = append(validPeers, peer.Name)
	}
	if !isValidRankCombination(ranks, threshold) {
		log.Warn("selectSignerSubset invalid rank combination", "threshold", threshold, "ranks", ranks)
		return nil
	}
	return validPeers
}

func containsSelf(peers []*types.Peer) bool {
	for _, p := range peers {
		if p.Self {
			return true
		}
	}
	return false
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
