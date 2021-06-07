package p2pstore

import (
	"time"

	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

func (p *Protocol) refreshLocalChunk() {
	start := time.Now()
	var saveNum, deleteNum int
	for chunkNum := int64(0); ; chunkNum++ {
		records, err := p.getChunkRecordFromBlockchain(&types.ReqChunkRecords{Start: chunkNum, End: chunkNum})
		if err != nil || len(records.Infos) != 1 {
			log.Info("refreshLocalChunk", "break at chunk num", chunkNum)
			break
		}
		info := records.Infos[0]
		msg := &types.ChunkInfoMsg{
			ChunkHash: info.ChunkHash,
			Start:     info.Start,
			End:       info.End,
		}
		if p.shouldSave(info.ChunkHash) {
			p.chunkToSync <- msg
			saveNum++
		} else {
			p.chunkToDelete <- msg
			deleteNum++
		}
	}
	log.Info("refreshLocalChunk", "save num", saveNum, "delete num", deleteNum, "time cost", time.Since(start))
}

func (p *Protocol) shouldSave(chunkHash []byte) bool {
	if p.SubConfig.IsFullNode {
		return true
	}
	pids := p.getExtendRoutingTable().NearestPeers(genDHTID(chunkHash), backup)
	if len(pids) >= backup && kbt.Closer(pids[backup-1], p.Host.ID(), genChunkNameSpaceKey(chunkHash)) {
		return true
	}
	return false
}

func (p *Protocol) getExtendRoutingTable() *kbt.RoutingTable {
	p.ertLock.Lock()
	defer p.ertLock.Unlock()
	return p.extendRoutingTable
}

func (p *Protocol) updateExtendRoutingTable() {
	key := []byte("temp")
	count := 100
	start := time.Now()
	extendRoutingTable := kbt.NewRoutingTable(dht.KValue, kbt.ConvertPeerID(p.Host.ID()), time.Minute, p.Host.Peerstore())
	peers := p.RoutingTable.ListPeers()
	for _, pid := range peers {
		_, _ = extendRoutingTable.Update(pid)
	}
	if key != nil {
		peers = p.RoutingTable.NearestPeers(genDHTID(key), backup-1)
	}

	searchedPeers := make(map[peer.ID]struct{})
	for i, pid := range peers {
		// 保证 extendRoutingTable 至少有 300 个节点，且至少从 3 个节点上获取新节点，
		if i+1 > 3 && extendRoutingTable.Size() > p.SubConfig.MaxExtendRoutingTableSize {
			break
		}
		searchedPeers[pid] = struct{}{}
		extendPeers, err := p.fetchShardPeers(key, count, pid)
		if err != nil {
			log.Error("updateExtendRoutingTable", "fetchShardPeers error", err, "peer id", pid)
			continue
		}
		for _, cPid := range extendPeers {
			if cPid == p.Host.ID() {
				continue
			}
			_, _ = extendRoutingTable.Update(cPid)
		}
	}

	// 如果扩展路由表节点数小于200，则迭代查询增加节点
	var lastSize int //如果经过一轮迭代节点数没有增加则结束迭代，防止节点数不到200导致无法退出
	for extendRoutingTable.Size() < p.SubConfig.MinExtendRoutingTableSize && extendRoutingTable.Size() > lastSize {
		lastSize = extendRoutingTable.Size()
		for _, pid := range extendRoutingTable.ListPeers() {
			if _, ok := searchedPeers[pid]; ok {
				continue
			}
			searchedPeers[pid] = struct{}{}
			closerPeers, err := p.fetchShardPeers(key, count, pid)
			if err != nil {
				log.Error("updateExtendRoutingTable", "fetchShardPeers error", err, "peer id", pid)
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
	log.Info("updateExtendRoutingTable", "local peers count", p.RoutingTable.Size(), "extendRoutingTable peer count", extendRoutingTable.Size(), "time cost", time.Since(start), "origin count", count)
	p.ertLock.Lock()
	p.extendRoutingTable = extendRoutingTable
	p.ertLock.Unlock()
}
