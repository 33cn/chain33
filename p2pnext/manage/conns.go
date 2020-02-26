package manage

import (
	"context"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var (
	log = log15.New("module", "p2p.manage")
)

type ConnManager struct {
	store            sync.Map
	host             core.Host
	pstore           peerstore.Peerstore
	bandwidthTracker *metrics.BandwidthCounter
}

func NewConnManager(host core.Host, tracker *metrics.BandwidthCounter) *ConnManager {
	connM := &ConnManager{}
	connM.pstore = host.Peerstore()
	connM.host = host
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
		log.Info("--------------时延--------------------")
		for _, pid := range s.pstore.Peers() {
			//统计每个节点的时延
			tduration := s.pstore.LatencyEWMA(pid)
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
		time.Sleep(time.Second * 5)
		if s.Size() == 0 {
			s.connectSeeds(seeds)
		}
	}
}

func (s *ConnManager) connectSeeds(seeds []string) {

	for _, seed := range seeds {
		addr, _ := multiaddr.NewMultiaddr(seed)
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}

		err = s.host.Connect(context.Background(), *peerinfo)
		if err != nil {
			log.Error("ConnectSeeds  Connect", "err", err)
			continue
		}
		s.pstore.AddAddrs(peerinfo.ID, peerinfo.Addrs, peerstore.PermanentAddrTTL)
	}
}

func (s *ConnManager) Add(pr *peer.AddrInfo) {
	s.store.Store(pr.ID.Pretty(), pr)
	//s.pstore.AddAddrs(pr.ID, pr.Addrs, ttl)
}

func (s *ConnManager) Delete(pid string) {
	s.store.Delete(pid)

}

func (s *ConnManager) Get(pid string) *peer.AddrInfo {
	v, ok := s.store.Load(pid)
	if ok {
		return v.(*peer.AddrInfo)
	}
	return nil
}

func (s *ConnManager) Fetch() []string {

	var pids []string

	s.store.Range(func(k interface{}, v interface{}) bool {
		pids = append(pids, k.(string))
		return true

	})
	// bandByPeer := s.bandwidthTracker.GetBandwidthByPeer()

	// for pid, _ := range bandByPeer {

	// 	if pid.Validate() == nil {
	// 		pids = append(pids, pid.Pretty())

	// 	}
	// }
	return pids
}

func (s *ConnManager) Size() int {
	//bandByPeer := s.bandwidthTracker.GetBandwidthByPeer()

	return len(s.Fetch())

}
