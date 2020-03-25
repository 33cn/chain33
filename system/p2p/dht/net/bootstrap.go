package net

import (
	"context"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"

	"github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var DefaultBootstrapPeers = make(map[string]*peer.AddrInfo)

func initMainNet() {
	//上线后添加种子节点
	for _, s := range []string{
		"/ip4/116.62.14.25/tcp/13803/p2p/16Uiu2HAmTLe2UuhiEiXkMBzbMsEUSJSqRCrc2rDJou855amo1JGD",
		"/ip4/39.106.166.159/tcp/13803/p2p/16Uiu2HAmM7iiHwerHR63CpG71sd2Y3jmC6QrPuAT1hRPWL9wx4eB",
		"/ip4/47.106.114.93/tcp/13803/p2p/16Uiu2HAmAvHCoV1i9U3MHorQGZNRUhms1FXpd57g12fpovg3owRS",
		"/ip4/120.76.100.165/tcp/13803/p2p/16Uiu2HAmGoQ8Sp9Tk42ftV6TjrMaSjSNuYJ6xNKVoAw18AcNJ6dy",
		"/ip4/120.24.85.66/tcp/13803/p2p/16Uiu2HAkwfpLqNcPpCgv6Pfewk58mV4mKF9xQJJdNY392wFa5bGC",
		"/ip4/120.24.92.123/tcp/13803/p2p/16Uiu2HAkzabkVNN9WPu63R27i7449cAYLkT7FgumDyYUYa9j4fQ8",
		"/ip4/116.62.14.25/tcp/13803/p2p/16Uiu2HAmTLe2UuhiEiXkMBzbMsEUSJSqRCrc2rDJou855amo1JGD",
		"/ip4/39.106.166.159/tcp/13803/p2p/16Uiu2HAmM7iiHwerHR63CpG71sd2Y3jmC6QrPuAT1hRPWL9wx4eB",
		"/ip4/39.106.193.172/tcp/13803/p2p/16Uiu2HAm19JAcWxkryuYShj6RWn3TGP4Y4Er1DwFe7AC4TevbuU6",
	} {
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
