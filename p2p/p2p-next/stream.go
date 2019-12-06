package p2p_next

import (
	"context"
	"sync"

	"github.com/libp2p/go-libp2p-core/peer"
	host "github.com/libp2p/go-libp2p-core/host"
	net "github.com/libp2p/go-libp2p-core/network"
)

type StreamMange struct {
	StreamStore sync.Map
	Host        host.Host
}

func NewStreamManage(host host.Host) *StreamMange {
	streamM := &StreamMange{}
	streamM.Host = host
	return streamM

}

func (s *StreamMange) newStream(ctx context.Context, pr peer.AddrInfo) (net.Stream, error) {
	//可以后续添加 block.ID,mempool.ID,header.ID

	stream, err := s.Host.NewStream(ctx, pr.ID)
	if err != nil {
		return nil, err
	}

	s.StreamStore.Store(pr.ID, stream)
	return stream, nil

}

func (s *StreamMange) deleteStream(pid string) {
	s.StreamStore.Delete(pid)

}

func (s *StreamMange) GetStream(id string) net.Stream {
	v, ok := s.StreamStore.Load(id)
	if ok {
		return v.(net.Stream)
	}
	return nil
}
func (s *StreamMange) FetchStreams() []net.Stream {
	var streams []net.Stream

	s.StreamStore.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return streams
}

func (s *StreamMange) Size() int {
	var streams []net.Stream
	s.StreamStore.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return len(streams)
}
