package p2pstore

import (
	"context"
	"math/rand"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (p *Protocol) startRepublish() {
	rand.Seed(time.Now().UnixNano())
	//避免多个节点同时启动时，republish时间过于接近导致网络拥堵
	time.Sleep(types2.RefreshInterval * time.Duration(rand.Intn(10000)) / 10000)
	for range time.Tick(types2.RefreshInterval) {
		if err := p.republish(); err != nil {
			log.Error("cycling republish", "error", err)
		}
	}
}

func (p *Protocol) republish() error {
	chunkInfoMap, err := p.getLocalChunkInfoMap()
	if err != nil {
		return err
	}
	log.Info("republish", ">>>>>>>>>>>>>> record amount:", len(chunkInfoMap))
	for hash, info := range chunkInfoMap {
		_, err = p.getChunkBlock(info.ChunkHash)
		if err != nil && err != types2.ErrExpired {
			log.Error("republish get error", "hash", hash, "error", err)
			continue
		}
		p.notifyStoreChunk(info.ChunkInfoMsg)
	}
	return nil
}

// 通知最近的 *BackUp* 个节点备份数据
func (p *Protocol) notifyStoreChunk(req *types.ChunkInfoMsg) {
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), Backup)
	for _, pid := range peers {
		err := p.storeChunkOnPeer(req, pid)
		if err != nil {
			log.Error("notifyStoreChunk", "peer id", pid, "error", err)
		}
	}
}

func (p *Protocol) storeChunkOnPeer(req *types.ChunkInfoMsg, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, protocol.StoreChunk)
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
