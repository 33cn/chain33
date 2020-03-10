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
	//上线后添加种子节点
	for _, s := range []string{} {
		addr, _ := multiaddr.NewMultiaddr(s)
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		DefaultBootstrapPeers[peerinfo.ID.Pretty()] = peerinfo
	}
}

func initTestNet() {

	//上线后添加种子节点
	for _, s := range []string{} {
		addr, _ := multiaddr.NewMultiaddr(s)
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		DefaultBootstrapPeers[peerinfo.ID.Pretty()] = peerinfo
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
		addr, _ := multiaddr.NewMultiaddr(addr)
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Error("ConvertPeers", "err", err)
			continue
		}
		pinfos[peerinfo.ID.Pretty()] = peerinfo
	}
	return pinfos
}
