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
	if p.SubConfig.DisableShard || p.SubConfig.IsFullNode {
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
	extendRoutingTable := p.genExtendRoutingTable(nil, 100)

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
		peers := extendRoutingTable.NearestPeers(genDHTID(info.ChunkHash), backup-1)
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
	extendRoutingTable := p.genExtendRoutingTable(req.ChunkHash, 100)
	for _, pid := range extendRoutingTable.NearestPeers(genDHTID(req.ChunkHash), backup-1) {
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

func (p *Protocol) genExtendRoutingTable(key []byte, count int) *kb.RoutingTable {
	if time.Since(p.refreshedTime) < time.Hour {
		return p.extendShardHealthyRoutingTable
	}
	start := time.Now()
	extendRoutingTable := kb.NewRoutingTable(dht.KValue*2, kb.ConvertPeerID(p.Host.ID()), time.Minute, p.Host.Peerstore())
	peers := p.ShardHealthyRoutingTable.ListPeers()
	for _, pid := range peers {
		_, _ = extendRoutingTable.Update(pid)
	}
	if key != nil {
		peers = p.ShardHealthyRoutingTable.NearestPeers(genDHTID(key), backup-1)
	}

	searchedPeers := make(map[peer.ID]struct{})
	for i, pid := range peers {
		// 保证 extendRoutingTable 至少有 300 个节点，且至少从 3 个节点上获取新节点，
		if i+1 > 3 && extendRoutingTable.Size() > 300 {
			break
		}
		searchedPeers[pid] = struct{}{}
		closerPeers, err := p.fetchCloserPeers(key, count, pid)
		if err != nil {
			log.Error("genExtendRoutingTable", "fetchCloserPeers error", err, "peer id", pid)
			continue
		}
		for _, cPid := range closerPeers {
			if cPid == p.Host.ID() {
				continue
			}
			_, _ = extendRoutingTable.Update(cPid)
		}
	}

	// 如果扩展路由表节点数小于200，则迭代查询增加节点
	var lastSize int //如果经过一轮迭代节点数没有增加则结束迭代，防止节点数不到200导致无法退出
	for extendRoutingTable.Size() < 200 && extendRoutingTable.Size() > lastSize {
		lastSize = extendRoutingTable.Size()
		for _, pid := range extendRoutingTable.ListPeers() {
			if _, ok := searchedPeers[pid]; ok {
				continue
			}
			searchedPeers[pid] = struct{}{}
			closerPeers, err := p.fetchCloserPeers(key, count, pid)
			if err != nil {
				log.Error("genExtendRoutingTable", "fetchCloserPeers error", err, "peer id", pid)
				continue
			}
			for _, cPid := range closerPeers {
				if cPid == p.Host.ID() {
					continue
				}
				_, _ = extendRoutingTable.Update(cPid)
			}
		}
	}
	log.Info("genExtendRoutingTable", "local peers count", len(peers), "extendRoutingTable peer count", extendRoutingTable.Size(), "time cost", time.Since(start), "origin count", count)
	p.extendShardHealthyRoutingTable = extendRoutingTable
	p.refreshedTime = start
	return extendRoutingTable
}
