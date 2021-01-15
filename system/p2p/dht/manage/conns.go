// Package manage p2p manage
package manage

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
}

// NewConnManager new connection manager
func NewConnManager(ctx context.Context, host core.Host, rt *kb.RoutingTable, tracker *metrics.BandwidthCounter, cfg *p2pty.P2PSubConfig) *ConnManager {
	connM := &ConnManager{}
	connM.ctx = ctx
	connM.host = host
	connM.routingTable = rt
	connM.bandwidthTracker = tracker
	connM.cfg = cfg

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
	var infos netprotocols
	for id, stat := range bandprotocols {
		if id == "" || stat.RateIn+stat.RateOut == 0 {
			continue
		}
		var info sortNetProtocols
		info.Protocol = string(id)
		info.Ratein = s.RateCalculate(stat.RateIn)
		info.Rateout = s.RateCalculate(stat.RateOut)
		info.Ratetotal = s.RateCalculate(stat.RateIn + stat.RateOut)
		infos = append(infos, &info)
	}

	sort.Sort(infos) //对Ratetotal 进行排序
	var netinfoArr []*types.ProtocolInfo
	for _, info := range infos {
		var protoinfo types.ProtocolInfo
		protoinfo.Ratetotal = info.Ratetotal
		protoinfo.Rateout = info.Rateout
		protoinfo.Ratein = info.Ratein
		protoinfo.Protocol = info.Protocol
		if strings.Contains(protoinfo.Ratetotal, "0.000") {
			continue
		}
		netinfoArr = append(netinfoArr, &protoinfo)
	}

	return &types.NetProtocolInfos{Protoinfo: netinfoArr}

}

// MonitorAllPeers monitory all peers
func (s *ConnManager) MonitorAllPeers() {
	ticker1 := time.NewTicker(time.Minute)
	ticker2 := time.NewTicker(time.Minute * 2)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker1.C:
			s.printMonitorInfo()
		case <-ticker2.C:
			s.procConnections()
		}
	}
}

func (s *ConnManager) printMonitorInfo() {
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
}

func (s *ConnManager) procConnections() {
	//处理当前连接的节点问题
	_, outBoundSize := s.BoundSize()
	if outBoundSize > maxOutBounds || s.Size() > maxBounds {
		return
	}
	//如果连接的节点数较少，尝试连接内置的和配置的种子节点
	//无须担心重新连接的问题，底层会自己判断是否已经连接了此节点，如果已经连接了就会忽略
	for _, seed := range s.cfg.Seeds {
		info, err := genAddrInfo(seed)
		if err != nil {
			panic(`invalid seeds format in config, use format of "/ip4/118.89.190.76/tcp/13803/p2p/16Uiu2HAmRao56AsxpobLBvbNfDttheQxnke9y1uWQRMWW7XaEdk5"`)
		}
		_ = s.host.Connect(context.Background(), *info)
	}

	for _, node := range s.cfg.BootStraps {
		info, err := genAddrInfo(node)
		if err != nil {
			panic(`invalid bootStraps format in config, use format of "/ip4/118.89.190.76/tcp/13803/p2p/16Uiu2HAmRao56AsxpobLBvbNfDttheQxnke9y1uWQRMWW7XaEdk5"`)
		}
		_ = s.host.Connect(context.Background(), *info)
	}
	if s.cfg.RelayEnable {
		//对relay中中继服务器要长期保持连接
		for _, node := range s.cfg.RelayNodeAddr {
			info, err := genAddrInfo(node)
			if err != nil {
				panic(`invalid relayNodeAddr in config, use format of "/ip4/118.89.190.76/tcp/13803/p2p/16Uiu2HAmRao56AsxpobLBvbNfDttheQxnke9y1uWQRMWW7XaEdk5"`)
			}
			if len(s.host.Network().ConnsToPeer(info.ID)) == 0 {
				s.host.Connect(context.Background(), *info)
			}
		}
	}

}

func genAddrInfo(addr string) (*peer.AddrInfo, error) {
	mAddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, err
	}
	return peer.AddrInfoFromP2pAddr(mAddr)
}

// AddNeighbors add neighbors by peer info
func (s *ConnManager) AddNeighbors(pr *peer.AddrInfo) {
	if pr == nil {
		return
	}
	s.neighborStore.Store(pr.ID.Pretty(), pr)
}

// IsNeighbors check is neighbors by id
func (s *ConnManager) IsNeighbors(pid peer.ID) bool {
	_, ok := s.neighborStore.Load(pid.Pretty())
	return ok
}

// Delete delete peer by id
func (s *ConnManager) Delete(pid peer.ID) {
	_ = s.host.Network().ClosePeer(pid)
	s.routingTable.Remove(pid)
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
	return convertMapToArr(peers)
}

func convertMapToArr(in map[string]peer.ID) []peer.ID {
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

type sortNetProtocols struct {
	Protocol  string
	Ratetotal string
	Ratein    string
	Rateout   string
}

type netprotocols []*sortNetProtocols

//Len
func (n netprotocols) Len() int { return len(n) }

//Swap
func (n netprotocols) Swap(i, j int) { n[i], n[j] = n[j], n[i] }

//Less
func (n netprotocols) Less(i, j int) bool { //从小到大排序，即index=0 ，表示数值最小
	return n[i].Ratetotal < n[j].Ratetotal
}
