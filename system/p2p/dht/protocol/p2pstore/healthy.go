package p2pstore

import (
	"context"
	"time"

	protocol2 "github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (p *Protocol) updateHealthyRoutingTable() {
	for _, pid := range p.RoutingTable.RoutingTable().ListPeers() {
		if err := p.checkPeerHealth(pid); err != nil {
			log.Error("checkPeerHealth", "error", err, "pid", pid)
		}
	}
}

func (p *Protocol) checkPeerHealth(id peer.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, id, protocol2.IsHealthy)
	if err != nil {
		return err
	}
	defer protocol2.CloseStream(stream)
	err = protocol2.WriteStream(&types.P2PRequest{
		Request: &types.P2PRequest_HealthyHeight{
			HealthyHeight: 50,
		},
	}, stream)
	if err != nil {
		return err
	}
	var res types.P2PResponse
	err = protocol2.ReadStream(&res, stream)
	if err != nil {
		return err
	}
	if reply, ok := res.Response.(*types.P2PResponse_Reply); ok && reply.Reply.IsOk {
		if _, err = p.healthyRoutingTable.Update(id); err != nil {
			return err
		}
	}
	return nil
}
