package p2pstore

import (
	"context"
	"encoding/hex"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

func (p *Protocol) refreshLocalChunk() {
	//优先处理同步任务
	if reply, err := p.API.IsSync(); err != nil || !reply.IsOk {
		return
	}
	start := time.Now()
	var saveNum, syncNum, deleteNum int
	for chunkNum := int64(0); ; chunkNum++ {
		records, err := p.getChunkRecordFromBlockchain(&types.ReqChunkRecords{Start: chunkNum, End: chunkNum})
		if err != nil || len(records.Infos) != 1 {
			log.Info("refreshLocalChunk", "break at chunk num", chunkNum, "error", err)
			break
		}
		info := records.Infos[0]
		msg := &types.ChunkInfoMsg{
			ChunkHash: info.ChunkHash,
			Start:     info.Start,
			End:       info.End,
		}
		if p.shouldSave(info.ChunkHash) {
			saveNum++
			// 本地处理
			// 1. p2pStore已保存数据则直接更新p2pStore
			// 2. p2pStore未保存数据则从blockchain获取数据
			if err := p.storeChunk(msg); err == nil {
				continue
			}

			// 通过网络从其他节点获取数据
			p.chunkInfoCacheMutex.Lock()
			p.chunkInfoCache[hex.EncodeToString(info.ChunkHash)] = msg
			p.chunkInfoCacheMutex.Unlock()
			syncNum++
			p.chunkToSync <- msg
			log.Info("refreshLocalChunk sync", "save num", saveNum, "sync num", syncNum, "delete num", deleteNum, "chunkHash", hex.EncodeToString(info.ChunkHash), "start", info.Start)

		} else {
			deleteNum++
			p.chunkToDelete <- msg
			log.Info("refreshLocalChunk delete", "save num", saveNum, "sync num", syncNum, "delete num", deleteNum, "chunkHash", hex.EncodeToString(info.ChunkHash), "start", info.Start)
		}
	}
	log.Info("refreshLocalChunk", "save num", saveNum, "sync num", syncNum, "delete num", deleteNum, "time cost", time.Since(start), "exRT size", p.getExtendRoutingTable().Size())
}

func (p *Protocol) shouldSave(chunkHash []byte) bool {
	if p.SubConfig.IsFullNode {
		return true
	}
	pids := p.getExtendRoutingTable().NearestPeers(genDHTID(chunkHash), 100)
	size := len(pids)
	if size == 0 {
		return true
	}
	if kbt.Closer(p.Host.ID(), pids[size*p.SubConfig.Percentage/100], genChunkNameSpaceKey(chunkHash)) {
		return true
	}
	return false
}

func (p *Protocol) getExtendRoutingTable() *kbt.RoutingTable {
	if p.extendRoutingTable.Size() < p.RoutingTable.Size() {
		return p.RoutingTable
	}
	return p.extendRoutingTable
}

func (p *Protocol) updateExtendRoutingTable() {
	const maxQueryPeerNum = 10
	start := time.Now()
	tmpRoutingTable, _ := kbt.NewRoutingTable(20, kbt.ConvertPeerID(p.Host.ID()), time.Minute, p.Host.Peerstore(), time.Hour, nil)
	peers := p.RoutingTable.ListPeers()
	for _, pid := range peers {
		if p.PeerInfoManager.PeerHeight(pid) < p.PeerInfoManager.PeerHeight(p.Host.ID())-1e4 {
			continue
		}
		_, _ = tmpRoutingTable.TryAddPeer(pid, true, true)
	}

	var count int // 只有通信成功的节点才会计数
	for _, pid := range p.RoutingTable.ListPeers() {
		if count > maxQueryPeerNum {
			break
		}
		extendPeers, err := p.fetchShardPeers(nil, 0, pid)
		if err != nil {
			log.Error("updateExtendRoutingTable", "fetchShardPeers error", err, "peer id", pid)
			continue
		}
		count++
		for _, cPid := range extendPeers {
			if cPid == p.Host.ID() {
				continue
			}
			_, _ = tmpRoutingTable.TryAddPeer(cPid, true, true)
			if len(p.Host.Network().Conns()) < 50 && len(p.Host.Network().ConnsToPeer(cPid)) == 0 {
				p.connect(cPid)
			}
		}
	}

	for _, pid := range p.extendRoutingTable.ListPeers() {
		p.extendRoutingTable.RemovePeer(pid)
	}
	for _, pid := range tmpRoutingTable.ListPeers() {
		_, _ = p.extendRoutingTable.TryAddPeer(pid, true, true)
	}
	_ = tmpRoutingTable.Close()
	log.Info("updateExtendRoutingTable", "pid", p.Host.ID(), "local peers count", p.RoutingTable.Size(), "extendRoutingTable peer count", p.extendRoutingTable.Size(), "time cost", time.Since(start))
}

func (p *Protocol) connect(pid peer.ID) {
	ctx, cancel := context.WithTimeout(p.Ctx, 3*time.Second)
	defer cancel()
	info := peer.AddrInfo{
		ID:    pid,
		Addrs: p.Host.Peerstore().Addrs(pid),
	}
	if err := p.Host.Connect(ctx, info); err != nil {
		log.Error("Host Connect", "err", err, "peer", info.ID)
	}

}
