package p2pstore

// TODO: debug, to delete soon
import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
)

func (p *Protocol) debugLog() {
	ticker := time.NewTicker(time.Minute * 30)
	defer ticker.Stop()

	for {
		select {
		case <-p.Ctx.Done():
			return
		case <-ticker.C:
			//debug info
			p.localChunkInfoMutex.Lock()
			log.Info("debugLocalChunk", "local chunk hash len", len(p.localChunkInfo))
			p.localChunkInfoMutex.Unlock()
			p.debugFullNode()
			log.Info("debug rt peers", "======== amount", p.RoutingTable.Size())
			log.Info("debug length", "notifying msg len", len(p.chunkToSync))
			log.Info("debug peers and conns", "peers len", len(p.Host.Network().Peers()), "conns len", len(p.Host.Network().Conns()))
			var streamCount int
			for _, conn := range p.Host.Network().Conns() {
				streams := conn.GetStreams()
				streamCount += len(streams)
				log.Info("debug new conn", "remote peer", conn.RemotePeer(), "len streams", len(streams))
				for _, s := range streams {
					log.Info("debug new stream", "protocol id", s.Protocol())
				}
			}
			log.Info("debug all stream", "count", streamCount)
			p.localChunkInfoMutex.RLock()
			for hash, info := range p.localChunkInfo {
				log.Info("localChunkInfo", "hash", hash, "start", info.Start, "end", info.End)
			}
			p.localChunkInfoMutex.RUnlock()

			log.Info("debug msg chan", "chunkToSync", len(p.chunkToSync), "chunkToDelete", len(p.chunkToDelete), "chunkToDownload", len(p.chunkToDownload))
		}
	}
}

func (p *Protocol) debugFullNode() {
	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*3)
	defer cancel()
	peerInfos, err := p.Discovery.FindPeers(ctx, fullNode)
	if err != nil {
		log.Error("debugFullNode", "FindPeers error", err)
		return
	}
	var count int
	for peerInfo := range peerInfos {
		count++
		log.Info("debugFullNode", "ID", peerInfo.ID, "maddrs", peerInfo.Addrs)
	}
	log.Info("debugFullNode", "total count", count)
}

func (p *Protocol) advertiseFullNode(opts ...discovery.Option) {
	if !p.SubConfig.IsFullNode {
		return
	}
	reply, err := p.API.IsSync()
	if err != nil || !reply.IsOk {
		// 没有同步完，不进行Advertise操作
		return
	}
	_, err = p.Discovery.Advertise(p.Ctx, fullNode, opts...)
	if err != nil {
		log.Error("advertiseFullNode", "error", err)
	}
}
