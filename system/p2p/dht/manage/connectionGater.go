package manage

import (
	"container/list"
	"context"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/33cn/chain33/types"

	"github.com/kevinms/leakybucket-go"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	//net "github.com/multiformats/go-multiaddr-net"
	net "github.com/multiformats/go-multiaddr/net"
)

const (
	// limit for rate limiter when processing new inbound dials.
	ipLimit = 4
	// burst limit for inbound dials.
	ipBurst = 8
	//缓存的临时的节点连接数量，虽然达到了最大限制，但是有的节点连接是查询需要，开辟缓冲区
)

// CacheLimit cachebuffer
var CacheLimit int32 = 20

// Conngater gater struct data
type Conngater struct {
	host          *host.Host
	maxConnectNum int32
	ipLimiter     *leakybucket.Collector
	blacklist     *TimeCache
	whitPeerList  map[peer.ID]multiaddr.Multiaddr
}

// NewConnGater connect gater
func NewConnGater(h *host.Host, limit int32, cache *TimeCache, whitPeers []*peer.AddrInfo) *Conngater {
	gater := &Conngater{}
	gater.host = h
	if limit == 0 {
		limit = 4096
	}

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
	if s.blacklist.Has(p.Pretty()) {
		return false
	}

	return !s.isPeerAtLimit(network.DirOutbound)
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
func (s *Conngater) InterceptAddrDial(p peer.ID, m multiaddr.Multiaddr) (allow bool) {
	return !s.blacklist.Has(p.Pretty())
}

// InterceptAccept tests whether an incipient inbound connection is allowed.
func (s *Conngater) InterceptAccept(n network.ConnMultiaddrs) (allow bool) {
	if !s.validateDial(n.RemoteMultiaddr()) { //对连进来的节点进行速率限制
		log.Error("InterceptAccept: query too frequent", "multiAddr", n.RemoteMultiaddr())
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
func (s *Conngater) InterceptSecured(_ network.Direction, p peer.ID, n network.ConnMultiaddrs) (allow bool) {
	return !s.blacklist.Has(p.Pretty())
}

// InterceptUpgraded tests whether a fully capable connection is allowed.
func (s *Conngater) InterceptUpgraded(n network.Conn) (allow bool, reason control.DisconnectReason) {
	if n == nil {
		return false, 0
	}
	return !s.blacklist.Has(n.RemotePeer().Pretty()), 0
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
	host := (*s.host)
	if host == nil {
		return false
	}
	var inboundNum, outboundNum int32
	for _, conn := range host.Network().Conns() {
		if conn.Stat().Direction == network.DirInbound {
			inboundNum++
		} else {
			outboundNum++
		}
	}

	if direction == network.DirInbound { //inbound connect
		return inboundNum >= s.maxConnectNum+CacheLimit
	}
	return outboundNum >= s.maxConnectNum+CacheLimit
}

// TimeCache data struct
type TimeCache struct {
	cacheLock sync.Mutex
	Q         *list.List
	M         map[string]time.Time
	ctx       context.Context
	span      time.Duration
}

// 对系统的连接时长按照从大到小的顺序排序
type blacklist []*types.BlackInfo

// Len return size of blackinfo
func (b blacklist) Len() int { return len(b) }

// Swap swap data between i,j
func (b blacklist) Swap(i, j int) { b[i], b[j] = b[j], b[i] }

// Less check lifetime
func (b blacklist) Less(i, j int) bool { //从小到大排序，即index=0 ，表示数值最大
	return b[i].Lifetime < b[j].Lifetime
}

// NewTimeCache new time cache obj.
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

// Add add key
func (tc *TimeCache) Add(s string, lifetime time.Duration) {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()

	if lifetime == 0 {
		lifetime = tc.span
	}
	tc.M[s] = time.Now().Add(lifetime) //update lifetime
	tc.Q.PushFront(s)

}

func (tc *TimeCache) sweep() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	tc.checkOvertimekey()

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
	for e := tc.Q.Front(); e != nil; e = e.Next() {
		v := e.Value.(string)
		t, ok := tc.M[v]
		if !ok {
			tc.Q.Remove(e)
			continue
		}

		if time.Now().After(t) {
			tc.Q.Remove(e)
			delete(tc.M, v)
		}
	}

}

// Has check key
func (tc *TimeCache) Has(s string) bool {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()

	_, ok := tc.M[s]
	return ok
}

// List show all peers
func (tc *TimeCache) List() *types.Blacklist {
	tc.cacheLock.Lock()
	defer tc.cacheLock.Unlock()
	var list blacklist
	for pid, p := range tc.M {
		list = append(list, &types.BlackInfo{PeerName: pid, Lifetime: int64(^(time.Since(p) / time.Second) + 1)})
	}
	sort.Sort(list)
	return &types.Blacklist{Blackinfo: list}

}
