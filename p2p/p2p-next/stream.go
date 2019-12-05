package p2pnext

import (
	"context"
	"sync"

	"github.com/33cn/chain33/p2p/p2p-next/service/broadcastTx"

	"github.com/libp2p/go-libp2p-core/peer"
	host "github.com/libp2p/go-libp2p-host"
	net "github.com/libp2p/go-libp2p-net"
)

type streamMange struct {
	StreamStore sync.Map
	Host        host.Host
}

func NewStreamManage(host host.Host) *streamMange {
	streamM := &streamMange{}
	streamM.Host = host
	return streamM

}

func (s *streamMange) newStream(ctx context.Context, pr peer.AddrInfo) (net.Stream, error) {
	//可以后续添加 block.ID,mempool.ID,header.ID
	stream, err := s.Host.NewStream(ctx, pr.ID, tx.ID)
	if err != nil {
		return nil, err
	}

	s.StreamStore.Store(pr.ID, stream)
	return stream, nil

}

func (s *streamMange) deleteStream(pid string) {
	s.StreamStore.Delete(pid)

}

func (s *streamMange) GetStream(id string) net.Stream {
	v, ok := s.StreamStore.Load(id)
	if ok {
		return v.(net.Stream)
	}
	return nil
}
func (s *streamMange) fetchStreams() []net.Stream {
	var streams []net.Stream

	s.StreamStore.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return streams
}

func (s *streamMange) Size() int {
	var streams []net.Stream
	s.StreamStore.Range(func(k, v interface{}) bool {
		streams = append(streams, v.(net.Stream))
		return true
	})

	return len(streams)
}
