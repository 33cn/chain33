package tss

import (
	"context"
	cryptocli "github.com/33cn/chain33/common/crypto/client"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"time"
)

type peerManager struct {
	protocol string
	selfID   string
	tssPeers []string
	cli      queue.Client
	ctx      context.Context
}

// NewPeerManager new pm
func NewPeerManager(peers []string, protocol string) *peerManager {

	ctx := cryptocli.GetCryptoContext()
	return &peerManager{
		protocol: protocol,
		tssPeers: peers,
		cli:      ctx.Client,
		ctx:      ctx.Ctx,
	}
}

func (p *peerManager) NumPeers() uint32 {
	return uint32(len(p.tssPeers))
}

func (p *peerManager) SelfID() string {
	return p.selfID
}

func (p *peerManager) PeerIDs() []string {
	return p.tssPeers
}

func (p *peerManager) MustSend(peerId string, message interface{}) {

	protoMsg, ok := message.(types.Message)
	if !ok {
		log.Error("peerManager MustSend, invalid proto message")
		return
	}
	wMsg := &MessageWrapper{
		PeerID:   peerId,
		Protocol: p.protocol,
		Msg:      types.Encode(protoMsg),
	}

	msg := p.cli.NewMessage("p2p", types.EventCryptoTssMsg, wMsg)
	err := p.cli.Send(msg, false)
	if err != nil {
		log.Error("peerManager MustSend", "peer", peerId,
			"protocol", p.protocol, "client.Send err:", err)
	}

}

// EnsureAllConnected connects the host to specified peer and sends the message to it.
func (p *peerManager) EnsureAllConnected() {

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		peers, err := p.getConnectedPeers()
		if err == nil && p.isAllConnected(peers) {
			p.selfID = peers[len(peers)-1].Name
			return
		}
		log.Debug("peerManager EnsureAllConnected wait peer sync")
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (p *peerManager) isAllConnected(peers []*types.Peer) bool {
	peerMap := make(map[string]struct{}, len(peers))
	for _, peer := range peers {
		peerMap[peer.String()] = struct{}{}
	}

	for _, peer := range p.tssPeers {
		if _, ok := peerMap[peer]; !ok {
			return false
		}
	}
	return true
}

func (p *peerManager) getConnectedPeers() ([]*types.Peer, error) {

	msg := p.cli.NewMessage("p2p", types.EventPeerInfo, nil)
	err := p.cli.Send(msg, true)
	if err != nil {
		log.Error("getConnectedPeers", "client.Send err:", err)
		return nil, err
	}
	resp, err := p.cli.WaitTimeout(msg, 5*time.Second)
	if err != nil {
		log.Error("getConnectedPeers", "client.Wait err:", err)
		return nil, err
	}
	return resp.GetData().(*types.PeerList).GetPeers(), nil

}
