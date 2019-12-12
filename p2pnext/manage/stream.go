package manage

import (
	"sync"

	net "github.com/libp2p/go-libp2p-core/network"
)

type StreamManger struct {
	store sync.Map
}

func NewStreamManager() *StreamManger {
	streamM := &StreamManger{}
	return streamM

}

func (s *StreamManger) AddStream(pid string, stream net.Stream) {
	s.store.Store(pid, stream)
}
func (s *StreamManger) DeleteStream(pid string) {
	s.store.Delete(pid)

}

func (s *StreamManger) GetStream(id string) net.Stream {
	v, ok := s.store.Load(id)
	if ok {
		return v.(net.Stream)
	}
	return nil
}
func (s *StreamManger) FetchStreams() []net.Stream {
	var streams []net.Stream

	s.store.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return streams
}

func (s *StreamManger) Size() int {
	var streams []net.Stream
	s.store.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return len(streams)
}
