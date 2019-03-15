package grpcclient

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc/resolver"
)

// Scheme 自定义grpc负载局衡名
const scheme = "multiple"

// Separator url分隔符
const separator = ":///"

// MultiPleHostsBalancerPrefix url前缀
const multiPleHostsBalancerPrefix = scheme + separator

// defaultGrpcPort 默认grpc端口
const defaultGrpcPort = "8802"

func parseTarget(target, defaultPort string) (host, port string, err error) {
	if target == "" {
		return "", "", errors.New("multiple resolver: missing address")
	}
	if ip := net.ParseIP(target); ip != nil {
		return target, defaultPort, nil
	}
	if host, port, err = net.SplitHostPort(target); err == nil {
		if port == "" {
			return "", "", errors.New("multiple resolver: missing port after port-separator colon")
		}
		if host == "" {
			host = "localhost"
		}
		return host, port, nil
	}
	if host, port, err = net.SplitHostPort(target + ":" + defaultPort); err == nil {
		return host, port, nil
	}
	return "", "", fmt.Errorf("invalid target address %v, error info: %v", target, err)
}

type multipleBuilder struct{}

func (*multipleBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {
	r := &multipleResolver{
		target: target,
		cc:     cc,
	}

	urls := strings.Split(target.Endpoint, ",")
	if len(urls) < 1 {
		return nil, fmt.Errorf("invalid target address %v", target)
	}

	for _, url := range urls {
		host, port, err := parseTarget(url, defaultGrpcPort)
		if err != nil {
			return nil, err
		}

		if host != "" {
			r.addrs = append(r.addrs, resolver.Address{Addr: host + ":" + port})
		}
	}
	r.start()
	return r, nil
}

func (*multipleBuilder) Scheme() string {
	return scheme
}

type multipleResolver struct {
	target resolver.Target
	cc     resolver.ClientConn
	addrs  []resolver.Address
}

func (r *multipleResolver) start() {
	r.cc.NewAddress(r.addrs)
}

func (*multipleResolver) ResolveNow(o resolver.ResolveNowOption) {}

func (*multipleResolver) Close() {}

func init() {
	resolver.Register(&multipleBuilder{})
}

//NewMultipleURL 创建url
func NewMultipleURL(url string) string {
	return multiPleHostsBalancerPrefix + url
}
