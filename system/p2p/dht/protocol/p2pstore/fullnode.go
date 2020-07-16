package p2pstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"

	"github.com/libp2p/go-libp2p-core/peer"
)

func (p *Protocol) broadcastFullNodes() {
	var fullNodes []peer.AddrInfo
	if p.SubConfig.IsFullNode {
		addrInfo := peer.AddrInfo{
			ID:    p.Host.ID(),
			Addrs: p.Host.Peerstore().Addrs(p.Host.ID()),
		}
		fullNodes = append(fullNodes, addrInfo)
	}
	p.fullNodes.Range(func(k, v interface{}) bool {
		//广播之前确认节点是否可达
		_, err := p.checkPeerHealth(k.(peer.ID))
		if err != nil {
			p.fullNodes.Delete(k)
			return true
		}
		fullNodes = append(fullNodes, v.(peer.AddrInfo))
		return true
	})
	if len(fullNodes) == 0 {
		return
	}
	data, err := json.Marshal(fullNodes)
	if err != nil {
		log.Error("broadcastFullNodes", "marshal error", err)
		return
	}
	for _, pid := range p.RoutingTable.RoutingTable().ListPeers() {
		if err := p.notifyFullNodes(pid, data); err != nil {
			log.Error("broadcastFullNodes", "pid", pid, "error", err)
		}
	}
}

func (p *Protocol) notifyFullNodes(pid peer.ID, addrInfo []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, protocol.BroadcastFullNode)
	if err != nil {
		return err
	}
	defer protocol.CloseStream(stream)

	return protocol.WriteStream(&types.P2PRequest{
		Request: &types.P2PRequest_AddrInfo{
			AddrInfo: addrInfo,
		},
	}, stream)
}
