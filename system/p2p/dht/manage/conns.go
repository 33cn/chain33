// Package manage p2p manage
package manage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/net"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
)

var (
	log = log15.New("module", "p2p.connManage")
)

const (
	maxBounds    = 50 //最大连接数包含连接被连接
	maxOutBounds = 30 //对外连接的最大节点数量
)

// ConnManager p2p connection manager
type ConnManager struct {
	neighborStore    sync.Map
	host             core.Host
	pstore           peerstore.Peerstore
	bandwidthTracker *metrics.BandwidthCounter
	discovery        *net.Discovery
	cfg              *p2pty.P2PSubConfig
	Done             chan struct{}
}

// NewConnManager new connection manager
func NewConnManager(host core.Host, discovery *net.Discovery, tracker *metrics.BandwidthCounter, cfg *p2pty.P2PSubConfig) *ConnManager {
	connM := &ConnManager{}
	connM.pstore = host.Peerstore()
	connM.host = host
	connM.discovery = discovery
	connM.bandwidthTracker = tracker
	connM.cfg = cfg
	connM.Done = make(chan struct{}, 1)
	return connM

}

// Close close connection manager
func (s *ConnManager) Close() {

	defer func() {
		if recover() != nil {
			log.Error("channel reclosed")
		}
	}()

	close(s.Done)
}

// RecordLatency record latency
func (s *ConnManager) RecordLatency(pid peer.ID, ttl time.Duration) {
	s.pstore.RecordLatency(pid, ttl)
}

// GetLatencyByPeer get peer latency by id
func (s *ConnManager) GetLatencyByPeer(pids []peer.ID) map[string]time.Duration {
	var tds = make(map[string]time.Duration)
	for _, pid := range pids {
		duration := s.pstore.LatencyEWMA(pid)
		tds[pid.Pretty()] = duration
	}

	return tds
}

// MonitorAllPeers monitory all peers
func (s *ConnManager) MonitorAllPeers(seeds []string, host core.Host) {
	bootstraps := net.ConvertPeers(s.cfg.BootStraps)
	ticker1 := time.NewTicker(time.Minute)
	ticker2 := time.NewTicker(time.Minute * 2)
	ticker3 := time.NewTicker(time.Hour * 6)
	for {
		select {
		case <-ticker1.C:
			var LatencyInfo = fmt.Sprintln("--------------时延--------------------")
			peers := s.FetchConnPeers()
			bandByPeer := s.bandwidthTracker.GetBandwidthByPeer()
			var trackerInfo = fmt.Sprintln("------------BandTracker--------------")
			for _, pid := range peers {
				//统计每个节点的时延,统计最多MaxBounds个
				tduration := s.pstore.LatencyEWMA(pid)
				if tduration == 0 {
					continue
				}
				LatencyInfo += fmt.Sprintln("PeerID:", pid.Pretty(), "LatencyEWMA:", tduration)
				if stat, ok := bandByPeer[pid]; ok {
					trackerInfo += fmt.Sprintf("PeerID:%s,RateIn:%f bytes/s,RateOut:%f bytes/s,totalIn:%d bytes,totalOut:%d\n",
						pid,
						stat.RateIn,
						stat.RateOut,
						stat.TotalIn,
						stat.TotalOut)
				}
			}
			log.Debug(LatencyInfo)
			insize, outsize := s.BoundSize()
			trackerInfo += fmt.Sprintln("peerstoreNum:", len(s.pstore.Peers()), ",conn num:", insize+outsize, "inbound num", insize, "outbound num", outsize,
				"dht size", s.discovery.RoutingTableSize())
			trackerInfo += fmt.Sprintln("-------------------------------------")
			log.Debug(trackerInfo)

		case <-ticker2.C:

			//处理当前连接的节点问题
			if s.OutboundSize() > maxOutBounds || s.Size() > maxBounds {
				continue
			}
			//如果连接的节点数较少，尝试连接内置的和配置的种子节点
			//无须担心重新连接的问题，底层会自己判断是否已经连接了此节点，如果已经连接了就会忽略
			for _, seed := range net.ConvertPeers(seeds) {
				_ = s.host.Connect(context.Background(), *seed)
			}

			for _, node := range bootstraps {
				_ = s.host.Connect(context.Background(), *node)
			}

		//debug
		case <-ticker3.C:
			for _, pid := range s.discovery.ListPeers() {
				log.Info("debug routing table", "pid", pid, "maddrs", s.host.Peerstore().Addrs(pid))
			}

		case <-s.Done:
			return
		}
	}
}

// AddNeighbors add neighbors by peer info
func (s *ConnManager) AddNeighbors(pr *peer.AddrInfo) {
	s.neighborStore.Store(pr.ID.Pretty(), pr)
}

// IsNeighbors check is neighbors by id
func (s *ConnManager) IsNeighbors(pid peer.ID) bool {
	_, ok := s.neighborStore.Load(pid.Pretty())
	return ok
}

// Delete delete peer by id
func (s *ConnManager) Delete(pid peer.ID) {
	s.discovery.Remove(pid)
}

// Get get peer add info by id
func (s *ConnManager) Get(pid peer.ID) *peer.AddrInfo {
	peerinfo := s.discovery.FindLocalPeer(pid)
	return &peerinfo
}

// FetchNearestPeers fetch nearest peer ids
func (s *ConnManager) FetchNearestPeers() []peer.ID {
	return s.discovery.FindNearestPeers(s.host.ID(), 50)
}

// Size connections size
func (s *ConnManager) Size() int {
	return len(s.host.Network().Conns())
}

// FetchConnPeers 获取连接的Peer's ID 这个连接包含被连接的peer以及主动连接的peer.
func (s *ConnManager) FetchConnPeers() []peer.ID {
	var peers = make(map[string]peer.ID)

	nearpeers := s.FetchNearestPeers()
	for _, peer := range nearpeers {
		if _, ok := peers[peer.Pretty()]; !ok {
			peers[peer.Pretty()] = peer
		}
		if len(peers) >= maxBounds {
			return s.convertMapToArr(peers)
		}
	}

	for _, conn := range s.host.Network().Conns() {
		peers[conn.RemotePeer().Pretty()] = conn.RemotePeer()
		//log.Debug("FetchConnPeers", "stream Num", len(conn.GetStreams()), "pid", conn.RemotePeer().Pretty())
		if len(peers) >= maxBounds {
			break
		}

	}

	return s.convertMapToArr(peers)
}

func (s *ConnManager) convertMapToArr(in map[string]peer.ID) []peer.ID {
	var pids []peer.ID
	for _, id := range in {
		pids = append(pids, id)
	}
	return pids
}

// CheckDiraction 检查连接的节点ID是被连接的还是主动连接的
func (s *ConnManager) CheckDiraction(pid peer.ID) network.Direction {
	for _, conn := range s.host.Network().ConnsToPeer(pid) {
		return conn.Stat().Direction
	}
	return network.DirUnknown
}

// InboundSize get inbound conn size
func (s *ConnManager) InboundSize() int {
	var inboundSize int
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirInbound {
			inboundSize++
		}
	}
	return inboundSize

}

// OutboundSize get outbound conn size
func (s *ConnManager) OutboundSize() int {
	var outboundSize int
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirOutbound {
			outboundSize++
		}
	}
	return outboundSize

}

// OutBounds get out bounds conn peers
func (s *ConnManager) OutBounds() []peer.ID {
	var pid []peer.ID
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirOutbound {
			pid = append(pid, con.RemotePeer())
		}
	}

	return pid
}

// InBounds get in bounds conn peers
func (s *ConnManager) InBounds() []peer.ID {
	var pid []peer.ID
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirInbound {
			pid = append(pid, con.RemotePeer())
		}
	}

	return pid

}

// BoundSize get in out conn bound size
func (s *ConnManager) BoundSize() (insize int, outsize int) {

	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirOutbound {
			outsize++
		}
		if con.Stat().Direction == network.DirInbound {
			insize++
		}
	}

	return insize, outsize

}

// GetNetRate get rateinfo
func (s *ConnManager) GetNetRate() metrics.Stats {

	return s.bandwidthTracker.GetBandwidthTotals()
}
