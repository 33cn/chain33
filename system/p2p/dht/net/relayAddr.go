package net

import (
	"github.com/libp2p/go-libp2p/config"
	ma "github.com/multiformats/go-multiaddr"
	"strings"
)

// withRelayAddrs returns an AddrFactory which will return Multiaddr via
// specified relay string in addition to existing MultiAddr.
func WithRelayAddrs(relays []string) config.AddrsFactory { //添加多个relay地址
	return func(addrs []ma.Multiaddr) []ma.Multiaddr {
		if len(relays) == 0 {
			return addrs
		}

		var relayAddrs []ma.Multiaddr
		for _, a := range addrs {
			if strings.Contains(a.String(), "/p2p-circuit") {
				continue
			}
			for _, relay := range relays {
				relayAddr, err := ma.NewMultiaddr(relay + "/p2p-circuit" + a.String())
				if err != nil {
					log.Error("Failed to create multiaddress for relay node: %v", err)
				} else {
					relayAddrs = append(relayAddrs, relayAddr)
				}
			}

		}

		if len(relayAddrs) == 0 {
			log.Warn("no relay addresses")
			return addrs
		}
		return append(addrs, relayAddrs...)
	}
}
