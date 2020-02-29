package dht

import (
	"context"
	"time"

	kbt "github.com/libp2p/go-libp2p-kbucket"

	"github.com/33cn/chain33/common/log/log15"
	//ds "github.com/ipfs/go-datastore"
	//dsync "github.com/ipfs/go-datastore/sync"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"

	//rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
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

func (d *Discovery) InitDht(ctx context.Context, host host.Host, seeds []string) host.Host {

	//开始节点发现

	//dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	///*dstore*/

	kademliaDHT, _ := dht.New(ctx, host)
	d.KademliaDHT = kademliaDHT

	// Make the routed host
	//routedHost := rhost.Wrap(host, kademliaDHT)

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.

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
	if err := d.KademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	d.routingDiscovery = discovery.NewRoutingDiscovery(d.KademliaDHT)
	discovery.Advertise(ctx, d.routingDiscovery, RendezvousString)
	return host
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
	if d.KademliaDHT.RoutingTable() != nil {
		return d.KademliaDHT.RoutingTable().Size()
	}
	return 0
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

func (d *Discovery) UPdate(pid peer.ID) error {
	_, err := d.KademliaDHT.RoutingTable().Update(pid)
	return err
}

func (d *Discovery) FindNearestPeers(pid peer.ID, count int) []peer.ID {
	return d.KademliaDHT.RoutingTable().NearestPeers(kbt.ConvertPeerID(pid), count)
}
