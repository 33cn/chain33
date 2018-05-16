package p2p

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	pb "gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
)

// NetAddress defines information about a peer on the network
// including its IP address, and port.
type NetAddress struct {
	IP   net.IP
	Port uint16
	str  string
}

// NewNetAddress returns a new NetAddress using the provided TCP
// address.
func NewNetAddress(addr net.Addr) *NetAddress {
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return nil
	}
	ip := tcpAddr.IP
	port := uint16(tcpAddr.Port)
	return NewNetAddressIPPort(ip, port)
}

// NewNetAddressString returns a new NetAddress using the provided
// address in the form of "IP:Port". Also resolves the host if host
// is not an IP.
func NewNetAddressString(addr string) (*NetAddress, error) {

	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		if len(host) > 0 {
			ips, err := net.LookupIP(host)
			if err != nil {
				return nil, err
			}
			ip = ips[0]
		}
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return nil, err
	}

	na := NewNetAddressIPPort(ip, uint16(port))
	return na, nil
}

// NewNetAddressStrings returns an array of NetAddress'es build using
// the provided strings.
func NewNetAddressStrings(addrs []string) ([]*NetAddress, error) {
	netAddrs := make([]*NetAddress, len(addrs))
	for i, addr := range addrs {
		netAddr, err := NewNetAddressString(addr)
		if err != nil {
			return nil, fmt.Errorf("error in address %s: %v", addr, err)
		}
		netAddrs[i] = netAddr
	}
	return netAddrs, nil
}

// NewNetAddressIPPort returns a new NetAddress using the provided IP
// and port number.
func NewNetAddressIPPort(ip net.IP, port uint16) *NetAddress {
	na := &NetAddress{
		IP:   ip,
		Port: port,
		str: net.JoinHostPort(
			ip.String(),
			strconv.FormatUint(uint64(port), 10),
		),
	}
	return na
}

// Equals reports whether na and other are the same addresses.
func (na *NetAddress) Equals(other interface{}) bool {
	if o, ok := other.(*NetAddress); ok {
		return na.String() == o.String()
	}

	return false
}

func (na *NetAddress) Less(other interface{}) bool {
	if o, ok := other.(*NetAddress); ok {
		return na.String() < o.String()
	}

	fmt.Println("Cannot compare unequal types")
	return false
}

// String representation.
func (na *NetAddress) String() string {
	if na.str == "" {
		na.str = net.JoinHostPort(
			na.IP.String(),
			strconv.FormatUint(uint64(na.Port), 10),
		)
	}
	return na.str
}

func (na *NetAddress) Copy() *NetAddress {
	copytmp := *na
	return &copytmp
}

// DialTimeout calls net.DialTimeout on the address.
func isCompressSupport(err error) bool {
	var errstr = `grpc: Decompressor is not installed for grpc-encoding "gzip"`
	if grpc.Code(err) == codes.Unimplemented && grpc.ErrorDesc(err) == errstr {
		return false
	}
	return true
}

func (na *NetAddress) DialTimeout(version int32) (*grpc.ClientConn, error) {
	ch := make(chan grpc.ServiceConfig, 1)
	ch <- P2pComm.GrpcConfig()

	var cliparm keepalive.ClientParameters
	cliparm.Time = 10 * time.Second    //10秒Ping 一次
	cliparm.Timeout = 10 * time.Second //等待10秒，如果Ping 没有响应，则超时
	cliparm.PermitWithoutStream = true //启动keepalive 进行检查
	keepaliveOp := grpc.WithKeepaliveParams(cliparm)
	timeoutOp := grpc.WithTimeout(time.Second * 3)
	log.Debug("NetAddress", "Dial", na.String())
	conn, err := grpc.Dial(na.String(), grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(grpc.UseCompressor("gzip")), grpc.WithServiceConfig(ch), keepaliveOp, timeoutOp)
	if err != nil {
		log.Debug("grpc DialCon", "did not connect", err, "addr", na.String())
		return nil, err
	}
	//判断是否对方是否支持压缩
	cli := pb.NewP2PgserviceClient(conn)
	_, err = cli.GetHeaders(context.Background(), &pb.P2PGetHeaders{StartHeight: 0, EndHeight: 0, Version: version}, grpc.FailFast(true))
	if err != nil && !isCompressSupport(err) {
		//compress not support
		log.Error("compress not supprot , rollback to uncompress version", "addr", na.String())
		conn.Close()
		ch2 := make(chan grpc.ServiceConfig, 1)
		ch2 <- P2pComm.GrpcConfig()
		log.Debug("NetAddress", "Dial with unCompressor", na.String())
		conn, err = grpc.Dial(na.String(), grpc.WithInsecure(), grpc.WithServiceConfig(ch2), keepaliveOp, timeoutOp)
	}

	if err != nil {
		log.Debug("grpc DialCon Uncompressor", "did not connect", err)
		if conn != nil {
			conn.Close()
		}
		return nil, err
	}
	return conn, nil
}

// Routable returns true if the address is routable.
func (na *NetAddress) Routable() bool {
	// TODO(oga) bitcoind doesn't include RFC3849 here, but should we?
	return na.Valid() && !(na.RFC1918() || na.RFC3927() || na.RFC4862() ||
		na.RFC4193() || na.RFC4843() || na.Local())
}

// For IPv4 these are either a 0 or all bits set address. For IPv6 a zero
// address or one that matches the RFC3849 documentation address format.
func (na *NetAddress) Valid() bool {
	return na.IP != nil && !(na.IP.IsUnspecified() || na.RFC3849() ||
		na.IP.Equal(net.IPv4bcast))
}

// Local returns true if it is a local address.
func (na *NetAddress) Local() bool {
	return na.IP.IsLoopback() || zero4.Contains(na.IP)
}

// ReachabilityTo checks whenever o can be reached from na.
func (na *NetAddress) ReachabilityTo(o *NetAddress) int {
	const (
		Unreachable = 0
		Default     = iota
		Teredo
		Ipv6Weak
		Ipv4
		Ipv6Strong
	)
	if !na.Routable() {
		return Unreachable
	} else if na.RFC4380() {
		if !o.Routable() {
			return Default
		} else if o.RFC4380() {
			return Teredo
		} else if o.IP.To4() != nil {
			return Ipv4
		} else { // ipv6
			return Ipv6Weak
		}
	} else if na.IP.To4() != nil {
		if o.Routable() && o.IP.To4() != nil {
			return Ipv4
		}
		return Default
	} else /* ipv6 */ {
		var tunnelled bool
		// Is our v6 is tunnelled?
		if o.RFC3964() || o.RFC6052() || o.RFC6145() {
			tunnelled = true
		}
		if !o.Routable() {
			return Default
		} else if o.RFC4380() {
			return Teredo
		} else if o.IP.To4() != nil {
			return Ipv4
		} else if tunnelled {
			// only prioritise ipv6 if we aren't tunnelling it.
			return Ipv6Weak
		}
		return Ipv6Strong
	}
}

// RFC1918: IPv4 Private networks (10.0.0.0/8, 192.168.0.0/16, 172.16.0.0/12)
// RFC3849: IPv6 Documentation address  (2001:0DB8::/32)
// RFC3927: IPv4 Autoconfig (169.254.0.0/16)
// RFC3964: IPv6 6to4 (2002::/16)
// RFC4193: IPv6 unique local (FC00::/7)
// RFC4380: IPv6 Teredo tunneling (2001::/32)
// RFC4843: IPv6 ORCHID: (2001:10::/28)
// RFC4862: IPv6 Autoconfig (FE80::/64)
// RFC6052: IPv6 well known prefix (64:FF9B::/96)
// RFC6145: IPv6 IPv4 translated address ::FFFF:0:0:0/96
var (
	rfc1918_10  = net.IPNet{IP: net.ParseIP("10.0.0.0"), Mask: net.CIDRMask(8, 32)}
	rfc1918_192 = net.IPNet{IP: net.ParseIP("192.168.0.0"), Mask: net.CIDRMask(16, 32)}
	rfc1918_172 = net.IPNet{IP: net.ParseIP("172.16.0.0"), Mask: net.CIDRMask(12, 32)}
	rfc3849     = net.IPNet{IP: net.ParseIP("2001:0DB8::"), Mask: net.CIDRMask(32, 128)}
	rfc3927     = net.IPNet{IP: net.ParseIP("169.254.0.0"), Mask: net.CIDRMask(16, 32)}
	rfc3964     = net.IPNet{IP: net.ParseIP("2002::"), Mask: net.CIDRMask(16, 128)}
	rfc4193     = net.IPNet{IP: net.ParseIP("FC00::"), Mask: net.CIDRMask(7, 128)}
	rfc4380     = net.IPNet{IP: net.ParseIP("2001::"), Mask: net.CIDRMask(32, 128)}
	rfc4843     = net.IPNet{IP: net.ParseIP("2001:10::"), Mask: net.CIDRMask(28, 128)}
	rfc4862     = net.IPNet{IP: net.ParseIP("FE80::"), Mask: net.CIDRMask(64, 128)}
	rfc6052     = net.IPNet{IP: net.ParseIP("64:FF9B::"), Mask: net.CIDRMask(96, 128)}
	rfc6145     = net.IPNet{IP: net.ParseIP("::FFFF:0:0:0"), Mask: net.CIDRMask(96, 128)}
	zero4       = net.IPNet{IP: net.ParseIP("0.0.0.0"), Mask: net.CIDRMask(8, 32)}
)

func (na *NetAddress) RFC1918() bool {
	return rfc1918_10.Contains(na.IP) ||
		rfc1918_192.Contains(na.IP) ||
		rfc1918_172.Contains(na.IP)
}
func (na *NetAddress) RFC3849() bool { return rfc3849.Contains(na.IP) }
func (na *NetAddress) RFC3927() bool { return rfc3927.Contains(na.IP) }
func (na *NetAddress) RFC3964() bool { return rfc3964.Contains(na.IP) }
func (na *NetAddress) RFC4193() bool { return rfc4193.Contains(na.IP) }
func (na *NetAddress) RFC4380() bool { return rfc4380.Contains(na.IP) }
func (na *NetAddress) RFC4843() bool { return rfc4843.Contains(na.IP) }
func (na *NetAddress) RFC4862() bool { return rfc4862.Contains(na.IP) }
func (na *NetAddress) RFC6052() bool { return rfc6052.Contains(na.IP) }
func (na *NetAddress) RFC6145() bool { return rfc6145.Contains(na.IP) }
