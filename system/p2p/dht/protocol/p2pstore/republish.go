package p2pstore

import (
	"context"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (s *StoreProtocol) startRepublish() {
	time.Sleep(time.Second * 3)
	for range time.Tick(types2.RefreshInterval) {
		if err := s.republish(); err != nil {
			log.Error("cycling republish", "error", err)
		}
	}
}

func (s *StoreProtocol) republish() error {
	chunkInfoMap, err := s.getLocalChunkInfoMap()
	if err != nil {
		return err
	}

	for hash, info := range chunkInfoMap {
		_, err = s.getChunkBlock(info.ChunkHash)
		if err != nil && err != types2.ErrExpired {
			log.Error("republish get error", "hash", hash, "error", err)
			continue
		}
		s.notifyStoreChunk(info)
	}
	return nil
}

// 通知最近的 *BackUp* 个节点备份数据
func (s *StoreProtocol) notifyStoreChunk(req *types.ChunkInfoMsg) {
	peers := s.Discovery.FindNearestPeers(peer.ID(genChunkPath(req.ChunkHash)), Backup)
	for _, pid := range peers {
		err := s.storeChunkOnPeer(req, pid)
		if err != nil {
			log.Error("notifyStoreChunk", "peer id", pid, "error", err)
		}
	}
}

func (s *StoreProtocol) storeChunkOnPeer(req *types.ChunkInfoMsg, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	stream, err := s.Host.NewStream(ctx, pid, StoreChunk)
	if err != nil {
		log.Error("new stream error when store chunk", "peer id", pid, "error", err)
		return err
	}
	defer stream.Close()
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ChunkInfoMsg{ChunkInfoMsg: req},
	}
	return protocol.SignAndWriteStream(&msg, stream)
}
