package manage

import (
	"container/list"
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p-core/control"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	net "github.com/multiformats/go-multiaddr-net"
)

const (
	// limit for rate limiter when processing new inbound dials.
	ipLimit = 4
	// burst limit for inbound dials.
	ipBurst = 8
	//缓存的临时的节点连接数量，虽然达到了最大限制，但是有的节点连接是查询需要，开辟缓冲区
)

//CacheLimit cachebuffer
var CacheLimit int32 = 50

//Conngater gater struct data
type Conngater struct {
	host          *host.Host
	maxConnectNum int32
	ipLimiter     *leakybucket.Collector
	blacklist     *TimeCache
	whitPeerList  map[peer.ID]multiaddr.Multiaddr
}

//NewConnGater connect gater
func NewConnGater(h *host.Host, limit int32, cache *TimeCache, whitPeers []*peer.AddrInfo) *Conngater {
	gater := &Conngater{}
	gater.host = h
	gater.maxConnectNum = limit
	gater.blacklist = cache
	if gater.blacklist == nil {
		gater.blacklist = NewTimeCache(context.Background(), time.Minute*5)
	}
	gater.ipLimiter = leakybucket.NewCollector(ipLimit, ipBurst, true)

	for _, pr := range whitPeers {
		if gater.whitPeerList == nil {
			gater.whitPeerList = make(map[peer.ID]multiaddr.Multiaddr)
		}
		gater.whitPeerList[pr.ID] = pr.Addrs[0]
	}
	return gater
}

// InterceptPeerDial tests whether we're permitted to Dial the specified peer.
func (s *Conngater) InterceptPeerDial(p peer.ID) (allow bool) {
	//具体的拦截策略
	//黑名单检查
	//1.增加校验p2p白名单节点列表
	if !s.checkWhitePeerList(p) {
		return false
	}
	return !s.blacklist.Has(p.Pretty())

}

func (s *Conngater) checkWhitePeerList(p peer.ID) bool {
	if s.whitPeerList != nil {
		if _, ok := s.whitPeerList[p]; !ok {
			return false
		}
	}
	return true
}

// InterceptAddrDial tests whether we're permitted to dial the specified
// multiaddr for the given peer.
func (s *Conngater) InterceptAddrDial(_ peer.ID, m multiaddr.Multiaddr) (allow bool) {
	return true
}

// InterceptAccept tests whether an incipient inbound connection is allowed.
func (s *Conngater) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	if !s.validateDial(n.RemoteMultiaddr()) { //对连进来的节点进行速率限制
		// Allow other go-routines to run in the event
		// we receive a large amount of junk connections.
		runtime.Gosched()
		return false
	}
	//增加校验p2p白名单节点列表
	if !s.checkWhitAddr(n.RemoteMultiaddr()) {
		return false
	}
	return !s.isPeerAtLimit(network.DirInbound)

}

func (s *Conngater) checkWhitAddr(addr multiaddr.Multiaddr) bool {
	if s.whitPeerList == nil {
		return true
	}
	iswhiteIP := false
	checkIP, _ := net.ToIP(addr)
	for _, maddr := range s.whitPeerList {
		ip, err := net.ToIP(maddr)
		if err != nil {
			continue
		}
		if ip.String() == checkIP.String() {
			iswhiteIP = true
		}
	}

	return iswhiteIP
}

// InterceptSecured tests whether a given connection, now authenticated,
// is allowed.
func (s *Conngater) InterceptSecured(_ network.Direction, _ peer.ID, n network.ConnMultiaddrs) (allow bool) {
	return true
}

// InterceptUpgraded tests whether a fully capable connection is allowed.
func (s *Conngater) InterceptUpgraded(n network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}

func (s *Conngater) validateDial(addr multiaddr.Multiaddr) bool {
	ip, err := net.ToIP(addr)
	if err != nil {
		return false
	}
	remaining := s.ipLimiter.Remaining(ip.String())
	if remaining <= 0 {
		return false
	}
	s.ipLimiter.Add(ip.String(), 1)
	return true
}

func (s *Conngater) isPeerAtLimit(direction network.Direction) bool {
	if s.maxConnectNum == 0 { //不对连接节点数量进行限制
		return false
	}
	numOfConns := len((*s.host).Network().Peers())
	var maxPeers int
	if direction == network.DirInbound { //inbound connect
		maxPeers = int(s.maxConnectNum + CacheLimit/2)
	} else {
		maxPeers = int(s.maxConnectNum + CacheLimit)
	}
	return numOfConns >= maxPeers
}

//TimeCache data struct
type TimeCache struct {
	cacheLock sync.Mutex
	Q         *list.List
	M         map[string]time.Time
	ctx       context.Context
	span      time.Duration
}

//NewTimeCache new time cache obj.
func NewTimeCache(ctx context.Context, span time.Duration) *TimeCache {
	cache := &TimeCache{
		Q:    list.New(),
		M:    make(map[string]time.Time),
		span: span,
		ctx:  ctx,
	}
	go cache.sweep()
	return cache
}

//Add add key
func (tc *TimeCache) Add(s string, lifetime time.Duration) {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()
	_, ok := tc.M[s]
	if ok {
		return
	}
	if lifetime == 0 {
		lifetime = tc.span
	}
	tc.M[s] = time.Now().Add(lifetime)
	tc.Q.PushFront(s)
}

func (tc *TimeCache) sweep() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			tc.checkOvertimekey()
		case <-tc.ctx.Done():
			return
		}
	}

}

func (tc *TimeCache) checkOvertimekey() {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()

	back := tc.Q.Back()
	if back == nil {
		return
	}
	v := back.Value.(string)
	t, ok := tc.M[v]
	if !ok {
		return
	}
	//if time.Since(t) > tc.span {
	if time.Now().After(t) {
		tc.Q.Remove(back)
		delete(tc.M, v)
	}
}

//Has check key
func (tc *TimeCache) Has(s string) bool {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()

	_, ok := tc.M[s]
	return ok
}
