package p2pstore

import (
	"time"

	"github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/libp2p/go-libp2p/core/peer"
)

func (p *Protocol) cleanTrace() {
	start := time.Now()

	p.peerAddrRequestTraceMutex.Lock()
	for _, trace := range p.peerAddrRequestTrace {
		for pid, t := range trace {
			if time.Since(t) > time.Minute*10 {
				delete(trace, pid)
			}
		}
	}
	p.peerAddrRequestTraceMutex.Unlock()

	p.chunkRequestTraceMutex.Lock()
	for _, trace := range p.chunkRequestTrace {
		for pid, t := range trace {
			if time.Since(t) > time.Minute*10 {
				delete(trace, pid)
			}
		}
	}
	p.chunkRequestTraceMutex.Unlock()

	log.Info("cleanTrace", "time cost", time.Since(start))
}

func (p *Protocol) cleanCache() {

	start := time.Now()

	p.chunkProviderCacheMutex.Lock()
	for _, cache := range p.chunkProviderCache {
		for pid, t := range cache {
			if time.Since(t) > types.RefreshInterval {
				delete(cache, pid)
			}
		}
	}
	p.chunkProviderCacheMutex.Unlock()

	log.Info("cleanCache", "time cost", time.Since(start))
}

func (p *Protocol) addPeerAddrRequestTrace(pid, from peer.ID) (exist bool) {
	p.peerAddrRequestTraceMutex.Lock()
	if p.peerAddrRequestTrace[pid] == nil {
		p.peerAddrRequestTrace[pid] = make(map[peer.ID]time.Time)
	}
	exist = len(p.peerAddrRequestTrace[pid]) != 0
	p.peerAddrRequestTrace[pid][from] = time.Now()
	p.peerAddrRequestTraceMutex.Unlock()
	return
}

func (p *Protocol) getPeerAddrRequestTrace(pid peer.ID) []peer.ID {
	var peers []peer.ID
	p.peerAddrRequestTraceMutex.RLock()
	for pid := range p.peerAddrRequestTrace[pid] {
		peers = append(peers, pid)
	}
	p.peerAddrRequestTraceMutex.RUnlock()
	return peers
}

func (p *Protocol) removePeerAddrRequestTrace(pid, from peer.ID) {
	p.peerAddrRequestTraceMutex.Lock()
	delete(p.peerAddrRequestTrace[pid], from)
	p.peerAddrRequestTraceMutex.Unlock()
}

func (p *Protocol) addChunkRequestTrace(chunkHash []byte, pid peer.ID) (exist bool) {
	p.chunkRequestTraceMutex.Lock()
	if p.chunkRequestTrace[string(chunkHash)] == nil {
		p.chunkRequestTrace[string(chunkHash)] = make(map[peer.ID]time.Time)
	}
	exist = len(p.chunkRequestTrace[string(chunkHash)]) != 0
	p.chunkRequestTrace[string(chunkHash)][pid] = time.Now()
	p.chunkRequestTraceMutex.Unlock()
	return
}

func (p *Protocol) getChunkRequestTrace(chunkHash []byte) []peer.ID {
	var peers []peer.ID
	p.chunkRequestTraceMutex.RLock()
	for pid := range p.chunkRequestTrace[string(chunkHash)] {
		peers = append(peers, pid)
	}
	p.chunkRequestTraceMutex.RUnlock()
	return peers
}

func (p *Protocol) removeChunkRequestTrace(chunkHash []byte, from peer.ID) {
	p.chunkRequestTraceMutex.Lock()
	delete(p.chunkRequestTrace[string(chunkHash)], from)
	p.chunkRequestTraceMutex.Unlock()
}

func (p *Protocol) addChunkProviderCache(chunkHash []byte, provider peer.ID) (exist bool) {
	p.chunkProviderCacheMutex.Lock()
	if p.chunkProviderCache[string(chunkHash)] == nil {
		p.chunkProviderCache[string(chunkHash)] = make(map[peer.ID]time.Time)
	}
	exist = len(p.chunkProviderCache[string(chunkHash)]) != 0
	p.chunkProviderCache[string(chunkHash)][provider] = time.Now()
	p.chunkProviderCacheMutex.Unlock()
	return
}

// TODO: 按时延对节点排序
func (p *Protocol) getChunkProviderCache(chunkHash []byte) []peer.ID {
	var peers []peer.ID
	p.chunkProviderCacheMutex.RLock()
	for pid := range p.chunkProviderCache[string(chunkHash)] {
		peers = append(peers, pid)
	}
	p.chunkProviderCacheMutex.RUnlock()
	return peers
}
