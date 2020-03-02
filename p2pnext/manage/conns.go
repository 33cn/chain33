package manage

import (
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/p2pnext/dht"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var (
	log = log15.New("module", "p2p.connManage")
)

type ConnManager struct {
	store            sync.Map
	host             core.Host
	pstore           peerstore.Peerstore
	bandwidthTracker *metrics.BandwidthCounter
	discovery        *dht.Discovery
}

func NewConnManager(host core.Host, discovery *dht.Discovery, tracker *metrics.BandwidthCounter) *ConnManager {
	connM := &ConnManager{}
	connM.pstore = host.Peerstore()
	connM.host = host
	connM.discovery = discovery
	connM.bandwidthTracker = tracker
	return connM

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

	for {
		select {
		case <-time.After(time.Second * 10):

			log.Info("--------------时延--------------------")
			for _, pid := range s.Fetch() {
				//统计每个节点的时延
				tduration := s.pstore.LatencyEWMA(pid)
				if tduration == 0 {
					continue
				}
				log.Info("MonitorAllPeers", "LatencyEWMA timeDuration", tduration, "pid", pid)
			}
			log.Info("------------BandTracker--------------")
			bandByPeer := s.bandwidthTracker.GetBandwidthByPeer()
			for pid, stat := range bandByPeer {
				log.Info("BandwidthTracker",
					"pid", pid,
					"RateIn bytes/seconds", stat.RateIn,
					"RateOut  bytes/seconds", stat.RateOut,
					"TotalIn", stat.TotalIn,
					"TotalOut", stat.TotalOut)
			}

			log.Info("-------------------------------------")
			log.Info("MounitorAllPeers", "peerstore peers", s.pstore.Peers(), "connPeer num", s.Size())

		}
	}
}

func (s *ConnManager) Add(pr *peer.AddrInfo) {

}

func (s *ConnManager) Delete(pid peer.ID) {
	s.discovery.Remove(pid)
}

func (s *ConnManager) Get(pid peer.ID) *peer.AddrInfo {

	peerinfo := s.discovery.FindLocalPeer(pid)
	return &peerinfo
}

func (s *ConnManager) Fetch() []peer.ID {
	return s.discovery.FindNearestPeers(s.host.ID(), 25)

}

func (s *ConnManager) Size() int {

	return len(s.host.Network().Conns())

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

func (s *ConnManager) BoundSize() (int, int) {
	var outboundSize int
	var inboundSize int
	for _, con := range s.host.Network().Conns() {
		if con.Stat().Direction == network.DirOutbound {
			outboundSize++
		}
		if con.Stat().Direction == network.DirInbound {
			inboundSize++
		}
	}

	return inboundSize, outboundSize

}

func convertPeers(peers []string) map[string]*peer.AddrInfo {
	pinfos := make(map[string]*peer.AddrInfo, len(peers))
	for _, addr := range peers {
		maddr := multiaddr.StringCast(addr)
		p, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			log.Error("convertPeers", "AddrInfoFromP2pAddr", err)
			continue
		}
		pinfos[p.ID.Pretty()] = p
	}
	return pinfos
}
