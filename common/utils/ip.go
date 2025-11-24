package utils

import (
	"net"
)

// IsPublicIP ...
/*
tcp/ip协议中，专门保留了三个IP地址区域作为私有地址，其地址范围如下：
10.0.0.0/8：10.0.0.0～10.255.255.255
172.16.0.0/12：172.16.0.0～172.31.255.255
192.168.0.0/16：192.168.0.0～192.168.255.255
*/
func IsPublicIP(ip string) bool {
	IP := net.ParseIP(ip)
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

// LocalIPv4s ...
// LocalIPs return all non-loopback IPv4 addresses
func LocalIPv4s() ([]string, error) {
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
