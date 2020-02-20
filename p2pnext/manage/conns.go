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

func (s *ConnManager) Fetch() []net.Conn {
	var conns []net.Conn

	s.store.Range(func(k, v interface{}) bool {
		conns = append(conns, v.(net.Conn))
		return true
	})

	return conns
}

func (s *ConnManager) Size() int {
	var conns []net.Conn
	s.store.Range(func(k, v interface{}) bool {
		conns = append(conns, v.(net.Conn))
		return true
	})

	return len(conns)
}
