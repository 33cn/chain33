// Package manage p2p manage
package manage

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/multiformats/go-multiaddr"
)

var (
	log = log15.New("module", "p2p.connManage")
)

const (
	maxBounds    = 30 //最大连接数包含连接被连接
	maxOutBounds = 15 //对外连接的最大节点数量
)

// ConnManager p2p connection manager
type ConnManager struct {
	ctx              context.Context
	neighborStore    sync.Map
	host             core.Host
	bandwidthTracker *metrics.BandwidthCounter
	routingTable     *kb.RoutingTable
	cfg              *p2pty.P2PSubConfig

	Done chan struct{}
}

// NewConnManager new connection manager
func NewConnManager(ctx context.Context, host core.Host, rt *kb.RoutingTable, tracker *metrics.BandwidthCounter, cfg *p2pty.P2PSubConfig) *ConnManager {
	connM := &ConnManager{}
	connM.ctx = ctx
	connM.host = host
	connM.routingTable = rt
	connM.bandwidthTracker = tracker
	connM.cfg = cfg
	connM.Done = make(chan struct{}, 1)

	return connM

}

// RateCalculate means bytes sent / received per second.
func (s *ConnManager) RateCalculate(ratebytes float64) string {
	kbytes := ratebytes / 1024
	rate := fmt.Sprintf("%.3f KB/s", kbytes)
	if kbytes/1024 > 0.1 {
		rate = fmt.Sprintf("%.3f MB/s", kbytes/1024)
	}

	return rate
}

// BandTrackerByProtocol returns all protocols band info
func (s *ConnManager) BandTrackerByProtocol() *types.NetProtocolInfos {
	bandprotocols := s.bandwidthTracker.GetBandwidthByProtocol()
	var infos types.NetProtocolInfos
	for id, stat := range bandprotocols {
		if id == "" {
			continue
		}

		var info types.ProtocolInfo
		info.Protocol = string(id)
		info.Ratein = s.RateCalculate(stat.RateIn)
		info.Rateout = s.RateCalculate(stat.RateOut)
		info.Ratetotal = s.RateCalculate(stat.RateIn + stat.RateOut)
		infos.Protoinfo = append(infos.Protoinfo, &info)

	}
	return &infos

}

// MonitorAllPeers monitory all peers
func (s *ConnManager) MonitorAllPeers() {
	ticker1 := time.NewTicker(time.Minute)
	ticker2 := time.NewTicker(time.Minute * 2)
	ticker3 := time.NewTicker(time.Hour * 6)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker1.C:
			var LatencyInfo = fmt.Sprintln("--------------时延--------------------")
			peers := s.FetchConnPeers()
			bandByPeer := s.bandwidthTracker.GetBandwidthByPeer()
			var trackerInfo = fmt.Sprintln("------------BandTracker--------------")
			for _, pid := range peers {
				//统计每个节点的时延,统计最多MaxBounds个
				tduration := s.host.Peerstore().LatencyEWMA(pid)
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
				//protocols rate
				log.Debug(LatencyInfo)

				insize, outsize := s.BoundSize()
				trackerInfo += fmt.Sprintln("peerstoreNum:", len(s.host.Peerstore().Peers()), ",conn num:", insize+outsize, "inbound num", insize, "outbound num", outsize,
					"dht size", s.routingTable.Size())
				trackerInfo += fmt.Sprintln("-------------------------------------")
				log.Debug(trackerInfo)
			}

		case <-ticker2.C:
			//处理当前连接的节点问题
			_, outBoundSize := s.BoundSize()
			if outBoundSize > maxOutBounds || s.Size() > maxBounds {
				continue
			}
			//如果连接的节点数较少，尝试连接内置的和配置的种子节点
			//无须担心重新连接的问题，底层会自己判断是否已经连接了此节点，如果已经连接了就会忽略
			for _, seed := range ConvertPeers(s.cfg.Seeds) {
				_ = s.host.Connect(context.Background(), *seed)
			}

			for _, node := range ConvertPeers(s.cfg.BootStraps) {
				_ = s.host.Connect(context.Background(), *node)
			}

		//debug
		case <-ticker3.C:
			for _, pid := range s.routingTable.ListPeers() {
				log.Debug("debug routing table", "pid", pid, "maddrs", s.host.Peerstore().Addrs(pid))
			}
		}
	}
}

// ConvertPeers convert peers to addr info
func ConvertPeers(peers []string) map[string]*peer.AddrInfo {
	pinfos := make(map[string]*peer.AddrInfo, len(peers))
	for _, addr := range peers {
		addr, _ := multiaddr.NewMultiaddr(addr)
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			log.Error("ConvertPeers", "err", err)
			continue
		}
		pinfos[peerinfo.ID.Pretty()] = peerinfo
	}
	return pinfos
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

// FetchNearestPeers fetch nearest peer ids
func (s *ConnManager) FetchNearestPeers(count int) []peer.ID {
	if s.routingTable == nil {
		return nil
	}
	return s.routingTable.NearestPeers(kb.ConvertPeerID(s.host.ID()), count)
}

// Size connections size
func (s *ConnManager) Size() int {
	return len(s.host.Network().Conns())
}

// FetchConnPeers 获取连接的Peer's ID 这个连接包含被连接的peer以及主动连接的peer.
func (s *ConnManager) FetchConnPeers() []peer.ID {
	var peers = make(map[string]peer.ID)
	var allconns conns

	for _, conn := range s.host.Network().Conns() {
		allconns = append(allconns, conn)
	}

	//对当前连接的节点时长进行排序
	sort.Sort(allconns)
	//log.Debug("FetchConnPeers", "stream Num", len(conn.GetStreams()), "pid", conn.RemotePeer().Pretty())
	for _, conn := range allconns {
		peers[conn.RemotePeer().Pretty()] = conn.RemotePeer()
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

// CheckDirection 检查连接的节点ID是被连接的还是主动连接的
func (s *ConnManager) CheckDirection(pid peer.ID) network.Direction {
	for _, conn := range s.host.Network().ConnsToPeer(pid) {
		return conn.Stat().Direction
	}
	return network.DirUnknown
}

// OutBounds get out bounds conn peers
func (s *ConnManager) OutBounds() []peer.ID {
	var peers []peer.ID
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirOutbound {
			peers = append(peers, con.RemotePeer())
		}
	}
	return peers
}

// InBounds get in bounds conn peers
func (s *ConnManager) InBounds() []peer.ID {
	var peers []peer.ID
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirInbound {
			peers = append(peers, con.RemotePeer())
		}
	}
	return peers
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

//对系统的连接时长按照从大到小的顺序排序
type conns []network.Conn

//Len
func (c conns) Len() int { return len(c) }

//Swap
func (c conns) Swap(i, j int) { c[i], c[j] = c[j], c[i] }

//Less
func (c conns) Less(i, j int) bool { //从大到小排序，即index=0 ，表示数值最大
	return c[i].Stat().Opened.After(c[j].Stat().Opened)
}
