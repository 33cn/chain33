package manage

import (
	"sync"

	net "github.com/libp2p/go-libp2p-core/network"
)

type ConnManager struct {
	store sync.Map
}

func NewConnManager() *ConnManager {
	streamM := &ConnManager{}
	return streamM

}

func (s *ConnManager) Add(pid string, conn net.Conn) {
	s.store.Store(pid, conn)
}
func (s *ConnManager) Delete(pid string) {
	s.store.Delete(pid)

}

func (s *ConnManager) Get(id string) net.Conn {
	v, ok := s.store.Load(id)
	if ok {
		return v.(net.Conn)
	}
	return nil
}

func (s *ConnManager) Fetch() []string {
	//var conns []net.Conn
	var pids []string
	s.store.Range(func(k, v interface{}) bool {
		pids = append(pids, k.(string))
		return true
	})

	return pids
}

func (s *ConnManager) Size() int {
	var pids []string
	s.store.Range(func(k, v interface{}) bool {
		pids = append(pids, k.(string))
		return true
	})

	return len(pids)
}
