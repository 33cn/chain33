package grpcclient

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver"
)

// scheme 自定义grpc负载局衡名
const syncScheme = "sync"

// sync url前缀
const syncPrefix = syncScheme + separator

// 节点同步状态
const (
	UNKNOWN = 0
	NOTSYNC = 1
	SYNC    = 2
)

func init() {
	resolver.Register(&syncBuilder{})
}

// NewSyncURL 创建url
func NewSyncURL(url string) string {
	return syncPrefix + url
}

type syncBuilder struct{}

// Build 为给定目标创建一个新的`resolver`，当调用`grpc.Dial()`时执行
func (*syncBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &syncResolver{
		cc:    cc,
		stopC: make(chan struct{}, 1),
	}

	urls := strings.Split(target.Endpoint(), ",")
	if len(urls) < 1 {
		return nil, fmt.Errorf("invalid target address %v", target)
	}

	for _, url := range urls {
		host, port, err := parseTarget(url, defaultGrpcPort)
		if err != nil {
			return nil, err
		}
		if host != "" {
			addr := resolver.Address{Addr: host + ":" + port}
			state := getSyncState(addr)
			r.SetServiceList(addr.Addr, state)
		}
	}
	r.start()
	return r, nil
}

// Scheme return syncScheme
func (*syncBuilder) Scheme() string {
	return syncScheme
}

type syncResolver struct {
	cc         resolver.ClientConn
	serverList sync.Map //服务列表
	stopC      chan struct{}
}

func (s *syncResolver) start() {
	s.cc.UpdateState(resolver.State{Addresses: s.getServices()})
	go s.watcher()
}

// ResolveNow 监视目标更新
func (s *syncResolver) ResolveNow(rn resolver.ResolveNowOptions) {}

// Close 关闭
func (s *syncResolver) Close() {
	s.stopC <- struct{}{}
}

// watcher 监听节点同步状态
func (s *syncResolver) watcher() {
	hint := time.NewTicker(10 * time.Second)
	defer hint.Stop()
	for {
		select {
		case <-s.stopC:
			return
		case <-hint.C:
			for _, addr := range s.getServices() {
				state := getSyncState(addr)
				s.SetServiceList(addr.Addr, state)
			}
			s.cc.UpdateState(resolver.State{Addresses: s.getServices()})
		}
	}
}

// SetServiceList 设置服务地址
func (s *syncResolver) SetServiceList(key string, val int) {
	//获取服务地址
	addr := resolver.Address{Addr: key}
	//把节点同步状态存储到resolver.Address的元数据中
	addr = SetAddrInfo(addr, AddrInfo{State: val})
	s.serverList.Store(key, addr)
}

// GetServices 获取服务地址
func (s *syncResolver) getServices() []resolver.Address {
	addrs := make([]resolver.Address, 0, 10)
	s.serverList.Range(func(k, v interface{}) bool {
		addrs = append(addrs, v.(resolver.Address))
		return true
	})
	return addrs
}

func getSyncState(addr resolver.Address) int {
	conn, err := grpc.Dial(addr.Addr, grpc.WithInsecure())
	defer conn.Close()

	if err != nil {
		return UNKNOWN
	}
	grpcClient := types.NewChain33Client(conn)
	req := &types.ReqNil{}
	reply, err := grpcClient.IsSync(context.Background(), req)
	if err != nil {
		return UNKNOWN
	}
	if !reply.IsOk {
		return NOTSYNC
	}
	return SYNC
}
