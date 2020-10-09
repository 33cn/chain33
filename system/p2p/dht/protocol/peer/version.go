package peer

import (
	"fmt"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
	"net"
	"strings"
)

// MainNet Channel = 0x0000

//true means: RemotoAddr, false means:LAN addr
func (p *Protocol) checkRemotePeerExternalAddr(remoteMAddr string) bool {

	//存储对方的外网地址道peerstore中
	//check remoteMaddr isPubAddr 示例： /ip4/192.168.0.1/tcp/13802
	if len(strings.Split(remoteMAddr, "/")) < 5 {
		return false
	}

	return isPublicIP(strings.Split(remoteMAddr, "/")[2]))

}

func (p *Protocol) setAddrToPeerStore(pid core.PeerID, remoteAddr, addrFrom string) {

	defer func() { //防止出错，数组索引越界
		if r := recover(); r != nil {
			log.Error("setAddrToPeerStore", "recoverErr", r)
		}
	}()

	//改造RemoteAddr的端口，使之为对方绑定的端口
	realPort := strings.Split(addrFrom, "/")[4]
	realExternalIP := strings.Split(remoteAddr, "/")[2]
	var err error
	remoteMAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%v/tcp/%v", realExternalIP, realPort))
	if err != nil {
		return
	}

	p.Host.Peerstore().AddAddr(pid, remoteMAddr, peerstore.AddressTTL)
}
