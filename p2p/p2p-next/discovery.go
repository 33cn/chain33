package p2p_next

import (
	"context"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-discovery"
	host "github.com/libp2p/go-libp2p-core/host"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

const RendezvousString = "chain33-p2p-findme"

type Discovery struct {
}

func (d *Discovery) FindPeers(ctx context.Context, host host.Host) (<-chan peer.AddrInfo, error) {

	//开始节点发现
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	discovery.Advertise(ctx, routingDiscovery, RendezvousString)

	// Now, look for others who have announced
	// This is like your friend telling you the location to meet you.
	logger.Debug("Searching for other peers...")
	peerChan, err := routingDiscovery.FindPeers(ctx, RendezvousString)
	if err != nil {
		panic(err)
	}

	return peerChan, nil
}
