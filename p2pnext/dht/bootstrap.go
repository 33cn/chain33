package dht

import (
	"context"

	p2pty "github.com/33cn/chain33/p2pnext/types"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"

	"github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var DefaultBootstrapPeers = make(map[string]*peer.AddrInfo)

func initMainNet() {
	for _, s := range []string{

		//上线后添加种子节点

	} {
		maddr := multiaddr.StringCast(s)
		p, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Error("convertPeers", "AddrInfoFromP2pAddr", err)
			continue
		}
		DefaultBootstrapPeers[p.ID.Pretty()] = p
	}
}

func initTestNet() {
	for _, s := range []string{
		"/ip4/52.229.200.231/tcp/13803/p2p/16Uiu2HAmTdgKpRmE6sXj512HodxBPMZmjh6vHG1m4ftnXY3wLSpg",
		"/ip4/120.76.102.61/tcp/13803/p2p/16Uiu2HAmHffWU9fXzNUG3hiCCgpdj8Y9q1BwbbK7ZBsxSsnaDXk3",
		//上线后添加种子节点
	} {
		maddr := multiaddr.StringCast(s)
		p, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Error("convertPeers", "AddrInfoFromP2pAddr", err)
			continue
		}
		DefaultBootstrapPeers[p.ID.Pretty()] = p
	}
}

func initInnerPeers(host host.Host, peersInfo []peer.AddrInfo, cfg *p2pty.P2PSubConfig, isTestNet bool) {
	if isTestNet {
		initTestNet()
	} else {
		initMainNet()
	}
	for _, peer := range DefaultBootstrapPeers {
		host.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
		err := host.Connect(context.Background(), *peer)
		if err != nil {
			log.Error("Host Connect", "err", err)
			delete(DefaultBootstrapPeers, peer.ID.Pretty())
			continue
		}
	}

	for _, seed := range ConvertPeers(cfg.Seeds) {
		host.Peerstore().AddAddrs(seed.ID, seed.Addrs, peerstore.PermanentAddrTTL)
		err := host.Connect(context.Background(), *seed)
		if err != nil {
			log.Error("Host Connect", "err", err, "peer", seed.ID)
			continue
		}
	}

	for _, peerinfo := range peersInfo {
		host.Peerstore().AddAddrs(peerinfo.ID, peerinfo.Addrs, peerstore.TempAddrTTL)
		err := host.Connect(context.Background(), peerinfo)
		if err != nil {
			log.Error("Host Connect", "err", err)
			continue
		}
	}
}
func ConvertPeers(peers []string) map[string]*peer.AddrInfo {
	pinfos := make(map[string]*peer.AddrInfo, len(peers))
	for _, addr := range peers {
		maddr := multiaddr.StringCast(addr)
		p, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Error("convertPeers", "AddrInfoFromP2pAddr", err)
			continue
		}
		pinfos[p.ID.Pretty()] = p
	}
	return pinfos
}
