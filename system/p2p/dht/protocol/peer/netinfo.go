package peer

import (
	"net"
	"strings"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (p *peerInfoProtol) netinfoHandleEvent(msg *queue.Message) {
	log.Debug("peerInfoProtol", "net info", msg)
	insize, outsize := p.ConnManager.BoundSize()
	var netinfo types.NodeNetInfo
	//外网地址只有一个，显示太多，意义不大
	netinfo.Externaladdr = p.getExternalAddr()
	netinfo.Localaddr = strings.Split(p.GetHost().Addrs()[0].String(), "/")[2]
	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	netinfo.Service = false
	if netinfo.Inbounds != 0 {
		netinfo.Service = true
	}
	msg.Reply(p.GetQueueClient().NewMessage("rpc", types.EventReplyNetInfo, &netinfo))
}

/*
tcp/ip协议中，专门保留了三个IP地址区域作为私有地址，其地址范围如下：
10.0.0.0/8：10.0.0.0～10.255.255.255
172.16.0.0/12：172.16.0.0～172.31.255.255
192.168.0.0/16：192.168.0.0～192.168.255.255
*/
func isPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch true {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}
