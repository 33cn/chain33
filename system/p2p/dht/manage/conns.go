package manage

import (
	"fmt"
	"sync"
	"time"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/net"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	//multiaddr "github.com/multiformats/go-multiaddr"
)

var (
	log = log15.New("module", "p2p.connManage")
)

const (
	MinBounds    = 15 //最少连接节点数，包含连接被连接
	MaxBounds    = 50 //最大连接数包含连接被连接
	MaxOutBounds = 30 //对外连接的最大节点数量
)

type ConnManager struct {
	neighborStore    sync.Map
	host             core.Host
	pstore           peerstore.Peerstore
	bandwidthTracker *metrics.BandwidthCounter
	discovery        *net.Discovery
	cfg              *p2pty.P2PSubConfig
	Done             chan struct{}
}

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

func (s *ConnManager) Close() {

	defer func() {
		if recover() != nil {
			log.Error("channel reclosed")
		}
	}()

	close(s.Done)
}

func (s *ConnManager) RecoredLatency(pid peer.ID, ttl time.Duration) {
	s.pstore.RecordLatency(pid, ttl)
}

func (s *ConnManager) GetLatencyByPeer(pids []peer.ID) map[string]time.Duration {
	var tds = make(map[string]time.Duration)
	for _, pid := range pids {
		duration := s.pstore.LatencyEWMA(pid)
		tds[pid.Pretty()] = duration
	}

	return tds
}

func (s *ConnManager) MonitorAllPeers(seeds []string, host core.Host) {
	bootstraps := net.ConvertPeers(s.cfg.BootStraps)
	for {
		select {
		case <-time.After(time.Minute):
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
			log.Info(LatencyInfo)
			insize, outsize := s.BoundSize()
			trackerInfo += fmt.Sprintln("peerstoreNum:", len(s.pstore.Peers()), ",conn num:", insize+outsize, "inbound num", insize, "outbound num", outsize,
				"dht size", s.discovery.RoutingTableSize())
			trackerInfo += fmt.Sprintln("-------------------------------------")
			log.Info(trackerInfo)

		case <-time.After(time.Minute * 10):
			//处理当前连接的节点问题

			if s.OutboundSize() <= MaxOutBounds {
				continue
			}
			nearestPeers := s.convertArrToMap(s.FetchNearestPeers())
			//close from seed
			for _, pid := range s.OutBounds() {
				if _, ok := bootstraps[pid.Pretty()]; ok {
					// 判断是否是最近nearest的30个节点
					if _, ok := nearestPeers[pid.Pretty()]; !ok {
						s.host.Network().ClosePeer(pid)
						if s.OutboundSize() <= MinBounds {
							break
						}
					}

				}
			}

		case <-s.Done:
			return

		}
	}
}

func (s *ConnManager) AddNeighbors(pr *peer.AddrInfo) {
	s.neighborStore.Store(pr.ID.Pretty(), pr)
}

func (s *ConnManager) IsNeighbors(pid peer.ID) bool {
	_, ok := s.neighborStore.Load(pid.Pretty())
	return ok
}

func (s *ConnManager) Delete(pid peer.ID) {
	s.discovery.Remove(pid)
}

func (s *ConnManager) Get(pid peer.ID) *peer.AddrInfo {

	peerinfo := s.discovery.FindLocalPeer(pid)
	return &peerinfo
}

func (s *ConnManager) FetchNearestPeers() []peer.ID {
	return s.discovery.FindNearestPeers(s.host.ID(), 50)

}

func (s *ConnManager) Size() int {

	return len(s.host.Network().Conns())

}

//FetchConnPeers 获取连接的Peer's ID 这个连接包含被连接的peer以及主动连接的peer.
func (s *ConnManager) FetchConnPeers() []peer.ID {
	var peers = make(map[string]peer.ID)
	for _, conn := range s.host.Network().Conns() {
		//peers=append(peers,conn.RemotePeer())
		peers[conn.RemotePeer().Pretty()] = conn.RemotePeer()
		log.Debug("FetchConnPeers", "ssssstream Num", len(conn.GetStreams()), "pid", conn.RemotePeer().Pretty())
		if len(peers) >= MaxBounds {
			break
		}
	}

	if len(peers) < MinBounds {
		nearpeers := s.FetchNearestPeers()
		for _, peer := range nearpeers {
			if _, ok := peers[peer.Pretty()]; !ok {
				peers[peer.Pretty()] = peer
			}
			if len(peers) >= MaxOutBounds {
				break
			}
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

func (s *ConnManager) convertArrToMap(in []peer.ID) map[string]peer.ID {
	var m = make(map[string]peer.ID)
	for _, pid := range in {
		m[pid.Pretty()] = pid
	}
	return m
}

//检查连接的节点ID是被连接的还是主动连接的
func (s *ConnManager) CheckDiraction(pid peer.ID) network.Direction {
	for _, conn := range s.host.Network().ConnsToPeer(pid) {
		return conn.Stat().Direction
	}
	return network.DirUnknown
}

func (s *ConnManager) InboundSize() int {
	var inboundSize int
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirInbound {
			inboundSize++
		}
	}
	return inboundSize

}

func (s *ConnManager) OutboundSize() int {
	var outboundSize int
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirOutbound {
			outboundSize++
		}
	}
	return outboundSize

}

func (s *ConnManager) OutBounds() []peer.ID {
	var pid []peer.ID
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirOutbound {
			pid = append(pid, con.RemotePeer())
		}
	}

	return pid
}

func (s *ConnManager) InBounds() []peer.ID {
	var pid []peer.ID
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirInbound {
			pid = append(pid, con.RemotePeer())
		}
	}

	return pid

}
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
