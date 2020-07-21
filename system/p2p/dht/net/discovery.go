package net

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/33cn/chain33/types"

	protocol "github.com/libp2p/go-libp2p-core/protocol"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	kbt "github.com/libp2p/go-libp2p-kbucket"

	"github.com/33cn/chain33/common/log/log15"
	coredis "github.com/libp2p/go-libp2p-core/discovery"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

var (
	log = log15.New("module", "p2p.dht")
)

const dhtProtoID = "/ipfs/kad/%s/1.0.0/%d"

// Discovery dht discovery
type Discovery struct {
	kademliaDHT      *dht.IpfsDHT
	RoutingDiscovery *discovery.RoutingDiscovery
	mdnsService      *mdns
	ctx              context.Context
}

// InitDhtDiscovery init dht discovery
func InitDhtDiscovery(ctx context.Context, host host.Host, peersInfo []peer.AddrInfo, chainCfg *types.Chain33Config, subCfg *p2pty.P2PSubConfig) *Discovery {

	// Make the DHT,不同的ID进入不同的网络。
	//如果不修改DHTProto 则有可能会连入IPFS网络，dhtproto=/ipfs/kad/1.0.0
	d := new(Discovery)
	opt := opts.Protocols(protocol.ID(fmt.Sprintf(dhtProtoID, chainCfg.GetTitle(), subCfg.Channel)))
	kademliaDHT, err := dht.New(ctx, host, opt)
	if err != nil {
		panic(err)
	}
	d.kademliaDHT = kademliaDHT
	d.ctx = ctx
	//连接内置种子，以及addrbook存储的节点
	initInnerPeers(host, peersInfo, subCfg)
	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	if err = d.kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	d.RoutingDiscovery = discovery.NewRoutingDiscovery(d.kademliaDHT)
	return d
}

// FindPeers find peers
func (d *Discovery) FindPeers(RendezvousString string, gossip bool) ([]peer.AddrInfo, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if gossip {
		discovery.Advertise(ctx, d.RoutingDiscovery, RendezvousString)
	}

	addrinfos, err := discovery.FindPeers(ctx, d.RoutingDiscovery, RendezvousString, coredis.Limit(100))
	//peerChan, err := d.routingDiscovery.FindPeers(context.Background(), RendezvousString)
	if err != nil {
		//panic(err)
		log.Error("FindPeers", "err", err.Error())
		return nil, err
	}

	return addrinfos, nil
}

// FindLANPeers 查找局域网内的其他节点
func (d *Discovery) FindLANPeers(host host.Host, serviceTag string) (<-chan peer.AddrInfo, error) {
	mdns, err := initMDNS(d.ctx, host, serviceTag)
	if err != nil {
		return nil, err
	}
	d.mdnsService = mdns
	return d.mdnsService.PeerChan(), nil
}

// CloseFindLANPeers close peers
func (d *Discovery) CloseFindLANPeers() {
	if d.mdnsService != nil {
		d.mdnsService.Service.UnregisterNotifee(d.mdnsService.notifee)
	}
}

// ListPeers routingTable 路由表的节点信息
func (d *Discovery) ListPeers() []peer.ID {
	if d.kademliaDHT == nil {
		return nil
	}
	return d.kademliaDHT.RoutingTable().ListPeers()
}

// RoutingTableSize routingTable size
func (d *Discovery) RoutingTableSize() int {
	if d.kademliaDHT == nil {
		return 0
	}
	return d.kademliaDHT.RoutingTable().Size()
}

// FindSpecialPeer 根据指定的peerID ,查找指定的peer,
func (d *Discovery) FindSpecialPeer(pid peer.ID) (*peer.AddrInfo, error) {
	if d.kademliaDHT == nil {
		return nil, errors.New("empty ptr")
	}
	ctx := context.Background()
	pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	peerInfo, err := d.kademliaDHT.FindPeer(pctx, pid)
	if err != nil {
		return nil, err
	}

	return &peerInfo, nil

}

// FindLocalPeer 根据pid 查找当前DHT内部的peer信息
func (d *Discovery) FindLocalPeer(pid peer.ID) peer.AddrInfo {
	if d.kademliaDHT == nil {
		return peer.AddrInfo{}
	}
	return d.kademliaDHT.FindLocal(pid)
}

// FindLocalPeers find local peers
func (d *Discovery) FindLocalPeers(pids []peer.ID) []peer.AddrInfo {
	var addrinfos []peer.AddrInfo
	for _, pid := range pids {
		addrinfos = append(addrinfos, d.FindLocalPeer(pid))
	}
	return addrinfos
}

// FindPeersConnectedToPeer 获取连接指定的peerId的peers信息,查找连接PID=A的所有节点
func (d *Discovery) FindPeersConnectedToPeer(pid peer.ID) (<-chan *peer.AddrInfo, error) {
	if d.kademliaDHT == nil {
		return nil, errors.New("empty ptr")

	}

	ctx := context.Background()
	pctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return d.kademliaDHT.FindPeersConnectedToPeer(pctx, pid)

}

// Update update peer
func (d *Discovery) Update(pid peer.ID) error {
	_, err := d.kademliaDHT.RoutingTable().Update(pid)
	return err
}

// FindNearestPeers find nearest peers
func (d *Discovery) FindNearestPeers(pid peer.ID, count int) []peer.ID {
	if d.kademliaDHT == nil {
		return nil
	}

	return d.kademliaDHT.RoutingTable().NearestPeers(kbt.ConvertPeerID(pid), count)
}

// Remove remove peer
func (d *Discovery) Remove(pid peer.ID) {
	if d.kademliaDHT == nil {
		return
	}
	d.kademliaDHT.RoutingTable().Remove(pid)

}

// RoutingTable get routing table
func (d *Discovery) RoutingTable() *kbt.RoutingTable {
	return d.kademliaDHT.RoutingTable()
}
