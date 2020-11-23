package p2pstore

import (
	"context"
	"time"

	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

func (p *Protocol) republish() {
	//全节点的p2pstore保存所有chunk, 不进行republish操作
	if p.SubConfig.IsFullNode {
		return
	}
	reply, err := p.API.IsSync()
	if err != nil || !reply.IsOk {
		// 没有同步完，不进行republish操作
		return
	}
	m := make(map[string]LocalChunkInfo)
	p.localChunkInfoMutex.RLock()
	for k, v := range p.localChunkInfo {
		m[k] = v
	}
	p.localChunkInfoMutex.RUnlock()
	invertedIndex := make(map[peer.ID][]*types.ChunkInfoMsg)
	tmpRoutingTable := p.genTempRoutingTable(nil, 100)

	for hash, info := range m {
		if time.Since(info.Time) > types2.ExpiredTime {
			log.Info("republish deleteChunkBlock", "hash", hash, "start", info.Start)
			if err := p.deleteChunkBlock(info.ChunkHash); err != nil {
				log.Error("republish deleteChunkBlock error", "hash", hash, "error", err)
			}
			continue
		}
		if time.Since(info.Time) > types2.RefreshInterval*11/10 {
			continue
		}
		log.Info("local chunk", "hash", hash, "start", info.Start)
		peers := tmpRoutingTable.NearestPeers(genDHTID(info.ChunkHash), Backup-1)
		for _, pid := range peers {
			invertedIndex[pid] = append(invertedIndex[pid], info.ChunkInfoMsg)
		}
	}

	log.Info("republish", "invertedIndex length", len(invertedIndex))
	for pid, infos := range invertedIndex {
		log.Info("republish", "pid", pid, "info len", len(infos))
		if err := p.storeChunksOnPeer(pid, infos...); err != nil {
			log.Error("republish", "storeChunksOnPeer error", err, "pid", pid)
		}
	}
	log.Info("republish ok")
}

// 通知最近的 *BackUp-1* 个节点备份数据，加上本节点共Backup个
func (p *Protocol) notifyStoreChunk(req *types.ChunkInfoMsg) {
	tmpRoutingTable := p.genTempRoutingTable(req.ChunkHash, 100)
	for _, pid := range tmpRoutingTable.NearestPeers(genDHTID(req.ChunkHash), Backup-1) {
		err := p.storeChunksOnPeer(pid, req)
		if err != nil {
			log.Error("notifyStoreChunk", "peer id", pid, "error", err)
		}
	}
}

func (p *Protocol) storeChunksOnPeer(pid peer.ID, req ...*types.ChunkInfoMsg) error {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Minute)
	defer cancel()
	p.Host.ConnManager().Protect(pid, storeChunk)
	defer p.Host.ConnManager().Unprotect(pid, storeChunk)
	stream, err := p.Host.NewStream(ctx, pid, storeChunk)
	if err != nil {
		log.Error("new stream error when store chunk", "peer id", pid, "error", err)
		return err
	}
	defer protocol.CloseStream(stream)
	msg := types.P2PRequest{}
	msg.Request = &types.P2PRequest_ChunkInfoList{
		ChunkInfoList: &types.ChunkInfoList{
			Items: req,
		}}
	return protocol.SignAndWriteStream(&msg, stream)
}

func (p *Protocol) genTempRoutingTable(key []byte, count int) *kb.RoutingTable {
	tmpRoutingTable := kb.NewRoutingTable(dht.KValue, kb.ConvertPeerID(p.Host.ID()), time.Minute, p.Host.Peerstore())
	peers := p.ShardHealthyRoutingTable.ListPeers()
	for _, pid := range peers {
		_, _ = tmpRoutingTable.Update(pid)
	}
	if key != nil {
		peers = p.ShardHealthyRoutingTable.NearestPeers(genDHTID(key), Backup-1)
	}

	for i, pid := range peers {
		// 至少从 3 个节点上获取新节点，保证 tmpRoutingTable 至少有 3*Backup 个节点，但至多从 10 个节点上获取新节点
		if i+1 > 3 && (tmpRoutingTable.Size() > 3*Backup || i+1 > 10) {
			break
		}
		closerPeers, err := p.fetchCloserPeers(key, count, pid)
		if err != nil {
			log.Error("genTempRoutingTable", "fetchCloserPeers error", err, "peer id", pid)
			continue
		}
		for _, cPid := range closerPeers {
			if cPid == p.Host.ID() {
				continue
			}
			_, _ = tmpRoutingTable.Update(cPid)
		}
	}
	log.Info("genTempRoutingTable", "tmpRoutingTable peer count", tmpRoutingTable.Size())

	return tmpRoutingTable
}
