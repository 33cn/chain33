package net

import (
	"context"
	"time"

	circuit "github.com/libp2p/go-libp2p-circuit"
	coredis "github.com/libp2p/go-libp2p-core/discovery"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	swarm "github.com/libp2p/go-libp2p-swarm"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	relay "github.com/libp2p/go-libp2p/p2p/host/relay"
)

type Relay struct {
	advertise *discovery.RoutingDiscovery
}

func NewRelayDiscovery(adv *discovery.RoutingDiscovery) *Relay {
	r := new(Relay)
	r.advertise = adv
	return r
}

//Advertise 如果自己支持relay模式，愿意充当relay中继器，则需要调用此函数
func (r *Relay) Advertise(ctx context.Context) {
	discovery.Advertise(ctx, r.advertise, relay.RelayRendezvous)
}

// FindOpPeers 如果需要用到中继相关功能，则需要先调用FindOpPeers查找到relay中继节点
func (r *Relay) FindOpPeers() ([]peer.AddrInfo, error) {
	dctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	return discovery.FindPeers(dctx, r.advertise, relay.RelayRendezvous, coredis.Limit(100))
}

//DialDestPeer 通过hop中继节点连接dst节点
func (r *Relay) DialDestPeer(host host.Host, hop, dst peer.AddrInfo) error {
	rhost, err := circuit.NewRelay(context.Background(), host, swarmt.GenUpgrader(host.Network().(*swarm.Swarm)), circuit.OptDiscovery)
	if err != nil {
		return err
	}
	rctx, rcancel := context.WithTimeout(context.Background(), time.Second)
	defer rcancel()
	_, err = rhost.DialPeer(rctx, hop, dst)
	return err

}

// CheckHOp 检查请求的节点是否支持relay中继
func (r *Relay) CheckHOp(host host.Host, isop peer.ID) (bool, error) {
	rhost, err := circuit.NewRelay(context.Background(), host, swarmt.GenUpgrader(host.Network().(*swarm.Swarm)), circuit.OptDiscovery)
	if err != nil {
		return false, err
	}
	rctx, rcancel := context.WithTimeout(context.Background(), time.Second)
	defer rcancel()
	canhop, err := rhost.CanHop(rctx, isop)
	if err != nil {
		return false, err
	}

	if !canhop {
		return false, nil
	}
	return true, nil
}
