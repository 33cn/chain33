package manage

import (
	"sync"

	net "github.com/libp2p/go-libp2p-core/network"
)

type StreamManager struct {
	store sync.Map
}

func NewStreamManager() *StreamManager {
	streamM := &StreamManager{}
	return streamM

}

func (s *StreamManager) AddStream(pid string, stream net.Stream) {
	s.store.Store(pid, stream)
}
func (s *StreamManager) DeleteStream(pid string) {
	s.store.Delete(pid)

}

func (s *StreamManager) GetStream(id string) net.Stream {
	v, ok := s.store.Load(id)
	if ok {
		return v.(net.Stream)
	}
	return nil
}
func (s *StreamManager) FetchStreams() []net.Stream {
	var streams []net.Stream

	s.store.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return streams
}

func (s *StreamManager) Size() int {
	var streams []net.Stream
	s.store.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return len(streams)
}
