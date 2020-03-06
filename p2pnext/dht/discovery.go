package dht

import (
	"context"
	"fmt"
	"time"

	protocol "github.com/libp2p/go-libp2p-core/protocol"

	p2pty "github.com/33cn/chain33/p2pnext/types"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	kbt "github.com/libp2p/go-libp2p-kbucket"

	"github.com/33cn/chain33/common/log/log15"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

var (
	log = log15.New("module", "p2p.dht")
)

const RendezvousString = "chain33-let's play!"
const DhtProtoID = "/ipfs/kad/chain33/1.0.0"

type Discovery struct {
	KademliaDHT      *dht.IpfsDHT
	routingDiscovery *discovery.RoutingDiscovery
	mndsService      *mdns
}

func (d *Discovery) InitDht(host host.Host, peersInfo []peer.AddrInfo, cfg *p2pty.P2PSubConfig, isTestNet bool) {

	initInnerPeers(host, peersInfo, cfg, isTestNet)
	// Make the DHT,不同的ID进入不同的网络
	opt := opts.Protocols(protocol.ID(DhtProtoID + "/" + fmt.Sprintf("%d", cfg.Channel)))
	kademliaDHT, _ := dht.New(context.Background(), host, opt)
	d.KademliaDHT = kademliaDHT

	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	if err := d.KademliaDHT.Bootstrap(context.Background()); err != nil {
		panic(err)
	}

	return
}

func (d *Discovery) FindPeers() (<-chan peer.AddrInfo, error) {
	d.routingDiscovery = discovery.NewRoutingDiscovery(d.KademliaDHT)
	discovery.Advertise(context.Background(), d.routingDiscovery, RendezvousString)
	peerChan, err := d.routingDiscovery.FindPeers(context.Background(), RendezvousString)
	if err != nil {
		panic(err)
	}

	return peerChan, nil
}

//查找局域网内的其他节点
func (d *Discovery) FindLANPeers(host host.Host, serviceTag string) (<-chan peer.AddrInfo, error) {
	mdns, err := initMDNS(context.Background(), host, serviceTag)
	if err != nil {
		return nil, err
	}
	d.mndsService = mdns
	return d.mndsService.PeerChan(), nil
}

func (d *Discovery) CloseFindLANPeers() {
	if d.mndsService != nil {
		d.mndsService.Service.UnregisterNotifee(d.mndsService.notifee)
	}
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
func (d *Discovery) FindLocalPeer(pid peer.ID) peer.AddrInfo {

	return d.KademliaDHT.FindLocal(pid)
}

func (d *Discovery) FindLocalPeers(pids []peer.ID) []peer.AddrInfo {
	var addrinfos []peer.AddrInfo
	for _, pid := range pids {
		addrinfos = append(addrinfos, d.FindLocalPeer(pid))
	}
	return addrinfos
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

func (d *Discovery) Remove(pid peer.ID) {
	d.KademliaDHT.RoutingTable().Remove(pid)

}
