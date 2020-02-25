package p2pnext

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p-core/peerstore"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	multiaddr "github.com/multiformats/go-multiaddr"
)

const RendezvousString = "chain33-p2p-findme"

type Discovery struct {
	KademliaDHT *dht.IpfsDHT
}

func (d *Discovery) FindPeers(ctx context.Context, host host.Host, seeds []string) (<-chan peer.AddrInfo, error) {

	//开始节点发现
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}
	d.KademliaDHT = kademliaDHT

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.

	if err = d.KademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}

	for _, seed := range seeds {
		addr, _ := multiaddr.NewMultiaddr(seed)
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		err = host.Connect(context.Background(), *peerinfo)
		if err != nil {
			logger.Error("Host Connect", "err", err)
			continue
		}
		host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, peerstore.PermanentAddrTTL)

	}

	routingDiscovery := discovery.NewRoutingDiscovery(d.KademliaDHT)
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

//根据指定的peerID ,查找指定的peer,
func (d *Discovery) FindSpecialPeer(pid peer.ID) (*peer.AddrInfo, error) {
	ctx := context.Background()
	pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	peerInfo, err := d.KademliaDHT.FindPeer(pctx, pid)
	if err != nil {
		return nil, err
	}

	return &peerInfo, nil

}

//根据pid 查找当前DHT内部的peer信息
func (d *Discovery) FindSpecailLocalPeer(pid peer.ID) peer.AddrInfo {

	return d.KademliaDHT.FindLocal(pid)

}

//获取连接指定的peerId的peers信息

func (d *Discovery) FindPeersConnectedToPeer(pid peer.ID) (<-chan *peer.AddrInfo, error) {
	ctx := context.Background()
	pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return d.KademliaDHT.FindPeersConnectedToPeer(pctx, pid)

}
