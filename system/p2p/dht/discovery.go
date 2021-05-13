package dht

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/extension"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	opts "github.com/libp2p/go-libp2p-kad-dht/opts"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

const (
	// Deprecated 老版本的协议，仅做兼容，TODO 后面升级后移除
	classicDhtProtoID = "/ipfs/kad/%s/1.0.0/%d"
	dhtProtoID        = "/%s-%d/kad/1.0.0" //title-channel/kad/1.0.0
)

// Discovery dht discovery
type Discovery struct {
	ctx              context.Context
	kademliaDHT      *dht.IpfsDHT
	RoutingDiscovery *discovery.RoutingDiscovery
	mdnsService      *extension.MDNS
	subCfg           *p2pty.P2PSubConfig
	bootstraps       []peer.AddrInfo
	host             host.Host
}

// InitDhtDiscovery init dht discovery
func InitDhtDiscovery(ctx context.Context, host host.Host, peersInfo []peer.AddrInfo, chainCfg *types.Chain33Config, subCfg *p2pty.P2PSubConfig) *Discovery {

	// Make the DHT,不同的ID进入不同的网络。
	//如果不修改DHTProto 则有可能会连入IPFS网络，dhtproto=/ipfs/kad/1.0.0
	d := new(Discovery)
	opt := opts.Protocols(protocol.ID(fmt.Sprintf(dhtProtoID, chainCfg.GetTitle(), subCfg.Channel)),
		protocol.ID(fmt.Sprintf(classicDhtProtoID, chainCfg.GetTitle(), subCfg.Channel)))
	kademliaDHT, err := dht.New(ctx, host, opt, opts.BucketSize(dht.KValue), opts.RoutingTableLatencyTolerance(time.Second*5))
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

//Start  the dht
func (d *Discovery) Start() {
	//连接内置种子，以及addrbook存储的节点
	initInnerPeers(d.host, d.bootstraps, d.subCfg)
	// Bootstrap the DHT. In the default configuration, this spawns a Background
	// thread that will refresh the peer table every five minutes.
	if err := d.kademliaDHT.Bootstrap(d.ctx); err != nil {
		log.Error("Bootstrap", "err", err.Error())
	}
	d.RoutingDiscovery = discovery.NewRoutingDiscovery(d.kademliaDHT)
}

//Close close the dht
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
	pctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return d.kademliaDHT.FindPeersConnectedToPeer(pctx, pid)
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
	_, err := d.kademliaDHT.RoutingTable().Update(pid)
	return err
}

// Remove remove peer
func (d *Discovery) Remove(pid peer.ID) {
	if d.kademliaDHT == nil {
		return
	}
	d.kademliaDHT.RoutingTable().Remove(pid)

}
