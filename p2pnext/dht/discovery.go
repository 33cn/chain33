package dht

import (
	"context"
	"time"

	"github.com/33cn/chain33/common/log/log15"

	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"

	multiaddr "github.com/multiformats/go-multiaddr"
)

var (
	log = log15.New("module", "p2p.manage")
)

const RendezvousString = "chain33-p2p-findme"

type Discovery struct {
	KademliaDHT      *dht.IpfsDHT
	routingDiscovery *discovery.RoutingDiscovery
}

func (d *Discovery) InitDht(ctx context.Context, host host.Host, seeds []string) {

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
		host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, peerstore.PermanentAddrTTL)
		err = host.Connect(context.Background(), *peerinfo)
		if err != nil {
			log.Error("Host Connect", "err", err)
			continue
		}

	}

	d.routingDiscovery = discovery.NewRoutingDiscovery(d.KademliaDHT)
	discovery.Advertise(ctx, d.routingDiscovery, RendezvousString)
}

//
func (d *Discovery) FindPeers() (<-chan peer.AddrInfo, error) {

	peerChan, err := d.routingDiscovery.FindPeers(context.Background(), RendezvousString)
	if err != nil {
		panic(err)
	}

	return peerChan, nil
}

//routingTable 路由表的节点信息
func (d *Discovery) RoutingTale() []peer.ID {
	return d.KademliaDHT.RoutingTable().ListPeers()
}

//routingTable size
func (d *Discovery) RoutingTableSize() int {
	return d.KademliaDHT.RoutingTable().Size()
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

//获取连接指定的peerId的peers信息,查找连接PID=A的所有节点

func (d *Discovery) FindPeersConnectedToPeer(pid peer.ID) (<-chan *peer.AddrInfo, error) {
	ctx := context.Background()
	pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return d.KademliaDHT.FindPeersConnectedToPeer(pctx, pid)

}
