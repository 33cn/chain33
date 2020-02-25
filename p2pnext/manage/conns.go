package manage

import (
	"sync"

	"time"

	net "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
)

type ConnManager struct {
	store  sync.Map
	pstore peerstore.Peerstore
}

func NewConnManager(ps peerstore.Peerstore) *ConnManager {
	connM := &ConnManager{}
	connM.pstore = ps
	return connM

}

func (s *ConnManager) Add(pr peer.AddrInfo, ttl time.Duration) {
	//s.store.Store(pid, conn)
	s.pstore.AddAddrs(pr.ID, pr.Addrs, ttl)
}
func (s *ConnManager) Delete(pid peer.ID) {
	//s.store.Delete(pid)
	s.pstore.ClearAddrs(pid)

}

func (s *ConnManager) Get(id string) net.Conn {
	v, ok := s.store.Load(id)
	if ok {
		return v.(net.Conn)
	}
	return nil
}

func (s *ConnManager) Fetch() []string {

	var pids []string
	/*s.store.Range(func(k, v interface{}) bool {
		pids = append(pids, k.(string))
		return true
	})

	return pids*/
	for _, pid := range s.pstore.PeersWithAddrs() {
		if pid.Validate() == nil {
			pids = append(pids, pid.Pretty())

		}
	}
	return pids
}

func (s *ConnManager) Size() int {
	/*	var pids []string
		s.store.Range(func(k, v interface{}) bool {
			pids = append(pids, k.(string))
			return true
		})

		return len(pids)*/
	return s.pstore.PeersWithAddrs().Len()

}
