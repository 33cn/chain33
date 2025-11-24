package extension

import (
	"strings"

	"github.com/libp2p/go-libp2p/config"
	ma "github.com/multiformats/go-multiaddr"
)

// WithRelayAddrs 会自动把relay的地址加入到自己的addrs 中，然后进行广播，这样其他节点收到包含有relay地址格式的节点地址时，
//如果接收到的节点打开了启用relay服务功能，就会触发relay协议进行尝试连接。
/*
				比如 中继服务器R的地址：/ip4/116.63.171.186/tcp/13801/p2p/16Uiu2HAm1Qgogf1vHj7WV7d8YmVNPw6PzRuYEnJbQsMDmDWtLoEM
				NAT 后面的一个普通节点A是：/ip4/192.168.1.101/tcp/13801/p2p/16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg
			    第一步：A--->R  A连接到中继节点R
				第二步：A开始组装自己的中继地址：
			[/ip4/116.63.171.186/tcp/13801/p2p/16Uiu2HAm1Qgogf1vHj7WV7d8YmVNPw6PzRuYEnJbQsMDmDWtLoEM/p2p-circuit/ip4/192.168.1.101/tcp
		    /13801/p2p/16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg]
		        第三步：A广播这个拼接后的带有p2p-circuit的地址
		        第四步：网络中的节点不论是NAT前面的或者NAT后面的节点，如果想连接节点PID为16Uiu2HAkw6w2YVenbCLAXHv2XVo1kh945GVxQrpm5Y6z2kE3eFSg的A节点，
	                   只需要通过上述组装的带有p2p-circuit的地址就可以建立到与A的连接

*/

// MakeRelayAddrs 把中继ID和与目的节点的ID 组装为一个新的地址
func MakeRelayAddrs(relayID, destID string) (ma.Multiaddr, error) {
	return ma.NewMultiaddr("/p2p/" + relayID + "/p2p-circuit/p2p/" + destID)
}

// WithRelayAddrs relay address to addrs
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
