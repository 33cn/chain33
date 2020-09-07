package peer

import (
	"fmt"
	"net"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (p *peerInfoProtol) netinfoHandleEvent(msg *queue.Message) {
	log.Debug("peerInfoProtol", "net info", msg)
	insize, outsize := p.ConnManager.BoundSize()
	var netinfo types.NodeNetInfo

	netinfo.Externaladdr = p.getExternalAddr()
	localips, err := localIPv4s()
	if err == nil {
		log.Debug("netinfoHandleEvent", "localIps", localips)
		netinfo.Localaddr = localips[0]
	} else {
		netinfo.Localaddr = netinfo.Externaladdr
	}

	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	netinfo.Service = false
	if netinfo.Inbounds != 0 {
		netinfo.Service = true
	}
	netinfo.Peerstore = int32(len(p.Host.Peerstore().PeersWithAddrs()))
	netinfo.Routingtable = int32(p.Discovery.RoutingTableSize())
	netstat := p.ConnManager.GetNetRate()
	netinfo.Ratein = rateCaculate(netstat.RateIn)
	netinfo.Rateout = rateCaculate(netstat.RateOut)
	netinfo.Ratetotal = rateCaculate(netstat.RateOut + netstat.RateIn)
	msg.Reply(p.GetQueueClient().NewMessage("rpc", types.EventReplyNetInfo, &netinfo))
}

// ratebytes means bytes sent / received per second.
func rateCaculate(ratebytes float64) string {
	kbytes := ratebytes / 1024
	rate := fmt.Sprintf("%.3f KB/s", kbytes)
	if kbytes/1024 > 0.1 {
		rate = fmt.Sprintf("%.3f MB/s", kbytes/1024)
	}

	return rate
}

/*
tcp/ip协议中，专门保留了三个IP地址区域作为私有地址，其地址范围如下：
10.0.0.0/8：10.0.0.0～10.255.255.255
172.16.0.0/12：172.16.0.0～172.31.255.255
192.168.0.0/16：192.168.0.0～192.168.255.255
*/
func isPublicIP(IP net.IP) bool {
	if IP == nil || IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
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

// LocalIPs return all non-loopback IPv4 addresses
func localIPv4s() ([]string, error) {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			ips = append(ips, ipnet.IP.String())
		}
	}

	return ips, nil
}
