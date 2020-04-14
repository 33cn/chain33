package p2pstore

import (
	"bufio"
	"context"
	"fmt"
	"time"

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

	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	fmt.Println(">>>>>>>>>>>>>>>>>>>> local hash length:", len(chunkInfoMap), ">>>>>>>>>>>")
	fmt.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	for hash, info := range chunkInfoMap {
		_, err = s.getChunkBlock(info.ChunkHash)
		if err != nil && err != types2.ErrExpired {
			log.Error("republish get error", "hash", hash, "error", err)
			continue
		}
		fmt.Printf("m[\"%s\"]++\n", hash)
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
	rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))
	msg := types.P2PStoreRequest{
		ProtocolID: StoreChunk,
		Data:       &types.P2PStoreRequest_ChunkInfoMsg{ChunkInfoMsg: req},
	}
	return writeMessage(rw.Writer, &msg)
}
