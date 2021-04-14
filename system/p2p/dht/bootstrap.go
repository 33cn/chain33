package dht

import (
	"context"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

func initInnerPeers(host host.Host, peersInfo []peer.AddrInfo, cfg *p2pty.P2PSubConfig) {

	for _, node := range cfg.Seeds {
		info := genAddrInfo(node)
		if info == nil || info.ID == host.ID() {
			continue
		}
		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		err := host.Connect(context.Background(), *info)
		if err != nil {
			log.Error("Host Connect", "err", err, "peer", info.ID)
			continue
		}
		//加保护
		host.ConnManager().Protect(info.ID, "seed")
	}

	for _, node := range cfg.BootStraps {
		info := genAddrInfo(node)
		if info == nil || info.ID == host.ID() {
			continue
		}
		host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		err := host.Connect(context.Background(), *info)
		if err != nil {
			log.Error("Host Connect", "err", err)
			continue
		}
	}

	//连接配置的relay中继服务器
	if cfg.RelayEnable {
		for _, relay := range cfg.RelayNodeAddr {
			info := genAddrInfo(relay)
			if info == nil || info.ID == host.ID() {
				continue
			}
			host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
			err := host.Connect(context.Background(), *info)
			if err != nil {
				log.Error("Host Connect", "err", err, "peer", info.ID)
				continue
			}
			//加保护
			host.ConnManager().Protect(info.ID, "relayNode")
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

func genAddrInfo(addr string) *peer.AddrInfo {
	mAddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil
	}
	peerInfo, err := peer.AddrInfoFromP2pAddr(mAddr)
	if err != nil {
		return nil
	}
	return peerInfo
}

func genAddrInfos(addrs []string) []*peer.AddrInfo {
	if len(addrs) == 0 {
		return nil
	}
	var infos []*peer.AddrInfo
	for _, addr := range addrs {
		info := genAddrInfo(addr)
		if info != nil {
			infos = append(infos, info)
		}
	}
	return infos
}
