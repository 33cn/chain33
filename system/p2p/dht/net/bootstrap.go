// Package net net utils
package net

import (
	"context"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	host "github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peerstore"

	"github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

func initInnerPeers(host host.Host, peersInfo []peer.AddrInfo, cfg *p2pty.P2PSubConfig) {

	for _, peer := range ConvertPeers(cfg.BootStraps) {
		host.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.PermanentAddrTTL)
		err := host.Connect(context.Background(), *peer)
		if err != nil {
			log.Error("Host Connect", "err", err)
			continue
		}
	}

	for _, seed := range ConvertPeers(cfg.Seeds) {
		if seed.ID == host.ID() {
			continue
		}
		host.Peerstore().AddAddrs(seed.ID, seed.Addrs, peerstore.PermanentAddrTTL)
		err := host.Connect(context.Background(), *seed)
		if err != nil {
			log.Error("Host Connect", "err", err, "peer", seed.ID)
			continue
		}
		//加保护
		host.ConnManager().Protect(seed.ID, "seed")
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

// ConvertPeers conver peers to addr info
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
