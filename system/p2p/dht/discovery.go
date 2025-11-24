package dht

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/extension"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kbt "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	rout "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

const (
	dhtProtoID = "/%s-%d/kad/1.0.0" //title-channel/kad/1.0.0
)

// Discovery dht rout
type Discovery struct {
	ctx              context.Context
	kademliaDHT      *dht.IpfsDHT
	RoutingDiscovery *rout.RoutingDiscovery
	mdnsService      *extension.MDNS
	subCfg           *p2pty.P2PSubConfig
	bootstraps       []peer.AddrInfo
	host             host.Host
}

// InitDhtDiscovery init dht rout
func InitDhtDiscovery(ctx context.Context, host host.Host, peersInfo []peer.AddrInfo, chainCfg *types.Chain33Config, subCfg *p2pty.P2PSubConfig) *Discovery {

	// Make the DHT,不同的ID进入不同的网络。
	//如果不修改DHTProto 则有可能会连入IPFS网络，dhtproto=/ipfs/kad/1.0.0
	d := new(Discovery)

	kademliaDHT, err := dht.New(ctx, host, dht.V1ProtocolOverride(protocol.ID(fmt.Sprintf(dhtProtoID, chainCfg.GetTitle(), subCfg.Channel))), dht.BootstrapPeers(peersInfo...), dht.RoutingTableRefreshPeriod(time.Minute), dht.Mode(dht.ModeServer))
	if err != nil {
		panic(err)
	}
	d.kademliaDHT = kademliaDHT
	d.ctx = ctx
	d.bootstraps = peersInfo
	d.subCfg = subCfg
	d.host = host
	return d

}

// Start  the dht
func (d *Discovery) Start() {
	//连接内置种子，以及addrbook存储的节点
	go initInnerPeers(d.ctx, d.host, d.bootstraps, d.subCfg)
	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	if err := d.kademliaDHT.Bootstrap(d.ctx); err != nil {
		log.Error("Bootstrap", "err", err.Error())
	}
	d.RoutingDiscovery = rout.NewRoutingDiscovery(d.kademliaDHT)
}

// Close close the dht
func (d *Discovery) Close() error {
	if d.kademliaDHT != nil {
		return d.kademliaDHT.Close()
	}
	d.CloseFindLANPeers()
	return nil
}

// FindLANPeers 查找局域网内的其他节点
func (d *Discovery) FindLANPeers(host host.Host, serviceTag string) (<-chan peer.AddrInfo, error) {
	mdns, err := extension.NewMDNS(d.ctx, host, serviceTag)
	if err != nil {
		return nil, err
	}
	d.mdnsService = mdns
	return d.mdnsService.PeerChan(), nil
}

// CloseFindLANPeers close peers
func (d *Discovery) CloseFindLANPeers() {
	if d.mdnsService != nil {
		d.mdnsService.Service.Close()
	}
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
	return d.kademliaDHT.FindLocal(d.ctx, pid)
}

// FindLocalPeers find local peers
func (d *Discovery) FindLocalPeers(pids []peer.ID) []peer.AddrInfo {
	var addrinfos []peer.AddrInfo
	for _, pid := range pids {
		addrinfos = append(addrinfos, d.FindLocalPeer(pid))
	}
	return addrinfos
}

// FindNearestPeers find nearest peers
func (d *Discovery) FindNearestPeers(pid peer.ID, count int) []peer.ID {
	if d.kademliaDHT == nil {
		return nil
	}

	return d.kademliaDHT.RoutingTable().NearestPeers(kbt.ConvertPeerID(pid), count)
}

// RoutingTable get routing table
func (d *Discovery) RoutingTable() *kbt.RoutingTable {
	return d.kademliaDHT.RoutingTable()
}

// ListPeers routingTable 路由表的节点信息
func (d *Discovery) ListPeers() []peer.ID {
	if d.kademliaDHT == nil {
		return nil
	}
	return d.kademliaDHT.RoutingTable().ListPeers()
}

// Update update peer
func (d *Discovery) Update(pid peer.ID) error {
	_, err := d.kademliaDHT.RoutingTable().TryAddPeer(pid, true, true)
	return err
}

// Remove remove peer
func (d *Discovery) Remove(pid peer.ID) {
	if d.kademliaDHT == nil {
		return
	}
	d.kademliaDHT.RoutingTable().RemovePeer(pid)

}
