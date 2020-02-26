package manage

import (
	"context"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	core "github.com/libp2p/go-libp2p-core"
	multiaddr "github.com/multiformats/go-multiaddr"

	//net "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
)

var (
	log = log15.New("module", "p2p.manage")
)

type ConnManager struct {
	host   core.Host
	pstore peerstore.Peerstore
}

func NewConnManager(host core.Host) *ConnManager {
	connM := &ConnManager{}
	connM.pstore = host.Peerstore()
	connM.host = host
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
		for _, pid := range s.pstore.PeersWithAddrs() {
			//统计每个节点的时延
			tduration := s.pstore.LatencyEWMA(pid)
			log.Info("MonitorAllPeers", "LatencyEWMA timeDuration", tduration, "pid", pid)
		}
		log.Info("---------------------------------")

		time.Sleep(time.Second * 5)
		if s.Size() == 0 {
			s.connectSeeds(seeds, host)
		}
	}
}

func (s *ConnManager) connectSeeds(seeds []string, host core.Host) {

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

func (s *ConnManager) Add(pr peer.AddrInfo, ttl time.Duration) {
	s.pstore.AddAddrs(pr.ID, pr.Addrs, ttl)
}

func (s *ConnManager) Delete(pid peer.ID) {
	s.pstore.ClearAddrs(pid)

}

func (s *ConnManager) Get(pid peer.ID) peer.AddrInfo {
	return s.pstore.PeerInfo(pid)
}

func (s *ConnManager) Fetch() []string {

	var pids []string

	for _, pid := range s.pstore.PeersWithAddrs() {
		if pid.Validate() == nil {
			pids = append(pids, pid.Pretty())

		}
	}
	return pids
}

func (s *ConnManager) Size() int {

	return s.pstore.PeersWithAddrs().Len()

}
