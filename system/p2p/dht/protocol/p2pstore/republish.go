package p2pstore

import (
	"context"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (p *Protocol) republish() {
	//全节点的p2pstore保存所有chunk, 不进行republish操作
	if p.SubConfig.IsFullNode {
		return
	}
	m := make(map[string]LocalChunkInfo)
	p.localChunkInfoMutex.RLock()
	for k, v := range p.localChunkInfo {
		m[k] = v
	}
	p.localChunkInfoMutex.RUnlock()
	log.Info("republish", ">>>>>>>>>>>>>> record amount:", len(m))
	log.Info("republish", "rt count", len(p.RoutingTable.ListPeers()), "healthy count", len(p.healthyRoutingTable.ListPeers()))
	for hash, info := range m {
		if time.Since(info.Time) > types2.ExpiredTime {
			if err := p.deleteChunkBlock(info.ChunkHash); err != nil {
				log.Error("republish deleteChunkBlock error", "hash", hash, "error", err)
			}
			continue
		}
		log.Info("local chunk", "hash", hash, "start", info.Start)
		p.notifyStoreChunk(info.ChunkInfoMsg)
	}
}

// 通知最近的 *BackUp-1* 个节点备份数据，加上本节点共Backup个
func (p *Protocol) notifyStoreChunk(req *types.ChunkInfoMsg) {
	peers := p.healthyRoutingTable.NearestPeers(genDHTID(req.ChunkHash), Backup-1)
	for _, pid := range peers {
		err := p.storeChunkOnPeer(req, pid)
		if err != nil {
			log.Error("notifyStoreChunk", "peer id", pid, "error", err)
		}
	}
}

func (p *Protocol) storeChunkOnPeer(req *types.ChunkInfoMsg, pid peer.ID) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, pid, protocol.StoreChunk)
	if err != nil {
		log.Error("new stream error when store chunk", "peer id", pid, "error", err)
		return err
	}
	defer protocol.CloseStream(stream)
	msg := types.P2PRequest{
		Request: &types.P2PRequest_ChunkInfoMsg{ChunkInfoMsg: req},
	}
	return protocol.SignAndWriteStream(&msg, stream)
}
