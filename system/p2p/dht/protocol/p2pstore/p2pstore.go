package p2pstore

import (
	"encoding/hex"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	kbt "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	fetchActivePeer = "/chain33/fetch-active-peer/1.0.0"
	fetchShardPeer  = "/chain33/fetch-shard-peer/1.0.0"
	fetchChunk      = "/chain33/fetch-chunk/1.0.0"
	//storeChunk      = "/chain33/store-chunk/1.0.0"
	getHeader      = "/chain33/headers/1.0.0"
	getChunkRecord = "/chain33/chunk-record/1.0.0"
	fullNode       = "/chain33/full-node/1.0.0"
	fetchPeerAddr  = "/chain33/fetch-peer-addr/1.0.0"
	// Deprecated: old version, use getHeader instead
	getHeaderOld = "/chain33/headerinfoReq/1.0.0"

	// 异步接口
	requestPeerAddr          = "/chain33/request-peer-addr/1.0.0"
	responsePeerAddr         = "/chain33/response-peer-addr/1.0.0"
	requestPeerInfoForChunk  = "/chain33/request-peer-info-for-chunk/1.0.0"
	responsePeerInfoForChunk = "/chain33/response-peer-info-for-chunk/1.0.0"
)

const maxConcurrency = 10

var log = log15.New("module", "protocol.p2pstore")

// Protocol ...
type Protocol struct {
	concurrency      int64
	*protocol.P2PEnv //协议共享接口变量

	chunkToSync     chan *types.ChunkInfoMsg
	chunkToDelete   chan *types.ChunkInfoMsg
	chunkToDownload chan *types.ChunkInfoMsg

	wakeup      map[string]chan struct{}
	wakeupMutex sync.Mutex

	chunkInfoCache      map[string]*types.ChunkInfoMsg
	chunkInfoCacheMutex sync.Mutex

	//本节点保存的chunk的索引表，会随着网络拓扑结构的变化而变化
	localChunkInfo      map[string]LocalChunkInfo
	localChunkInfoMutex sync.RWMutex

	// 扩展路由表，每小时更新一次
	extendRoutingTable *kbt.RoutingTable

	peerAddrRequestTrace      map[peer.ID]map[peer.ID]time.Time // peerID requested ==>
	peerAddrRequestTraceMutex sync.RWMutex

	chunkRequestTrace      map[string]map[peer.ID]time.Time // chunkHash ==> request peers
	chunkRequestTraceMutex sync.RWMutex

	chunkProviderCache      map[string]map[peer.ID]time.Time // chunkHash ==> provider peers
	chunkProviderCacheMutex sync.RWMutex
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

// InitProtocol initials the protocol.
func InitProtocol(env *protocol.P2PEnv) {
	exRT, _ := kbt.NewRoutingTable(20, kbt.ConvertPeerID(env.Host.ID()), time.Minute, env.Host.Peerstore(), time.Hour, nil)
	p := &Protocol{
		P2PEnv:               env,
		chunkToSync:          make(chan *types.ChunkInfoMsg, 1024),
		chunkToDelete:        make(chan *types.ChunkInfoMsg, 1024),
		chunkToDownload:      make(chan *types.ChunkInfoMsg, 1024),
		wakeup:               make(map[string]chan struct{}),
		chunkInfoCache:       make(map[string]*types.ChunkInfoMsg),
		peerAddrRequestTrace: make(map[peer.ID]map[peer.ID]time.Time),
		chunkRequestTrace:    make(map[string]map[peer.ID]time.Time),
		chunkProviderCache:   make(map[string]map[peer.ID]time.Time),
		extendRoutingTable:   exRT,
	}
	//
	if env.SubConfig.Percentage < types2.MinPercentage || env.SubConfig.Percentage > types2.MaxPercentage {
		env.SubConfig.Percentage = types2.DefaultPercentage
	}

	p.initLocalChunkInfoMap()

	//注册p2p通信协议，用于处理节点之间请求
	protocol.RegisterStreamHandler(p.Host, requestPeerInfoForChunk, p.handleStreamRequestPeerInfoForChunk)
	protocol.RegisterStreamHandler(p.Host, responsePeerInfoForChunk, p.handleStreamResponsePeerInfoForChunk)
	protocol.RegisterStreamHandler(p.Host, requestPeerAddr, p.handleStreamRequestPeerAddr)
	protocol.RegisterStreamHandler(p.Host, responsePeerAddr, p.handleStreamResponsePeerAddr)

	protocol.RegisterStreamHandler(p.Host, fetchPeerAddr, protocol.HandlerWithRW(p.handleStreamPeerAddr))
	protocol.RegisterStreamHandler(p.Host, fetchActivePeer, protocol.HandlerWithWrite(p.handleStreamFetchActivePeer))
	protocol.RegisterStreamHandler(p.Host, fetchShardPeer, protocol.HandlerWithRW(p.handleStreamFetchShardPeers))
	protocol.RegisterStreamHandler(p.Host, getHeaderOld, p.handleStreamGetHeaderOld)
	protocol.RegisterStreamHandler(p.Host, getHeader, protocol.HandlerWithAuthAndSign(p.Host, p.handleStreamGetHeader))
	if !p.SubConfig.DisableShard {
		protocol.RegisterStreamHandler(p.Host, fullNode, protocol.HandlerWithWrite(p.handleStreamIsFullNode))
		protocol.RegisterStreamHandler(p.Host, fetchChunk, p.handleStreamFetchChunk) //数据较大，采用特殊写入方式
		protocol.RegisterStreamHandler(p.Host, getChunkRecord, protocol.HandlerWithAuthAndSign(p.Host, p.handleStreamGetChunkRecord))
	}
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, p.handleEventNotifyStoreChunk)
	protocol.RegisterEventHandler(types.EventGetChunkBlock, p.handleEventGetChunkBlock)
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, p.handleEventGetChunkBlockBody)
	protocol.RegisterEventHandler(types.EventGetChunkRecord, p.handleEventGetChunkRecord)
	protocol.RegisterEventHandler(types.EventFetchBlockHeaders, p.handleEventGetHeaders)

	go p.processLocalChunk()
	go p.updateRoutine()
	go p.refreshRoutine()
	go p.debugLog()
	go p.processLocalChunkOldVersion()
}

func (p *Protocol) refreshRoutine() {
	ticker := time.NewTicker(types2.RefreshInterval)
	defer ticker.Stop()
	ticker2 := time.NewTicker(time.Minute * 3)
	defer ticker2.Stop()
	for {
		select {
		case <-p.Ctx.Done():
			return
		case <-ticker.C:
			p.refreshLocalChunk()
			p.cleanCache()
		case <-ticker2.C:
			p.cleanTrace()
		}
	}
}

func (p *Protocol) updateRoutine() {
	ticker1 := time.NewTicker(time.Minute)
	defer ticker1.Stop()
	ticker2 := time.NewTicker(time.Minute * 30)
	defer ticker2.Stop()

	p.updateExtendRoutingTable()
	for {
		select {
		case <-p.Ctx.Done():
			return

		case <-ticker1.C:
			p.advertiseFullNode()

		case <-ticker2.C:
			p.updateExtendRoutingTable()

		}
	}
}

// TODO: to delete next version
func (p *Protocol) processLocalChunkOldVersion() {
	for {
		select {
		case <-p.Ctx.Done():
			return
		case info := <-p.chunkToSync:
			if err := p.storeChunk(info); err == nil {
				break
			}
			bodys, _, err := p.mustFetchChunk(info)
			if err != nil {
				log.Error("processLocalChunk", "mustFetchChunk error", err)
				break
			}
			if err := p.addChunkBlock(info, bodys); err != nil {
				log.Error("processLocalChunk", "addChunkBlock error", err)
				break
			}
			log.Info("processLocalChunk save chunk success")
		}
	}
}

func (p *Protocol) processLocalChunk() {
	for {
		select {
		case <-p.Ctx.Done():
			return

		// TODO: open in next version
		//case info := <-p.chunkToSync:
		//	for _, pid := range p.RoutingTable.NearestPeers(genDHTID(info.ChunkHash), AlphaValue) {
		//		if err := p.requestPeerInfoForChunk(info, pid); err != nil {
		//			log.Error("processLocalChunk", "requestPeerInfoForChunk error", err, "pid", pid, "chunkHash", hex.EncodeToString(info.ChunkHash))
		//		}
		//	}

		case info := <-p.chunkToDelete:
			if localInfo, ok := p.getChunkInfoByHash(info.ChunkHash); ok && time.Since(localInfo.Time) > types2.RefreshInterval*3 {
				if err := p.deleteChunkBlock(localInfo.ChunkInfoMsg); err != nil {
					log.Error("processLocalChunk", "deleteChunkBlock error", err, "chunkHash", hex.EncodeToString(localInfo.ChunkHash), "start", localInfo.Start)
					break
				}
			}
		case info := <-p.chunkToDownload:
			//TODO: 1.根据节点时延排序 2.并发获取数据 3.把一个chunk分成多份分别到多个节点上去获取数据
			for _, pid := range p.getChunkProviderCache(info.ChunkHash) {
				bodys, _, err := p.fetchChunkFromPeer(info, pid)
				if err != nil {
					log.Error("processLocalChunk", "fetchChunkFromPeer error", err, "pid", pid, "chunkHash", hex.EncodeToString(info.ChunkHash))
					continue
				}
				if bodys == nil {
					continue
				}
				if err := p.addChunkBlock(info, bodys); err != nil {
					log.Error("processLocalChunk", "addChunkBlock error", err)
					continue
				}
				break
			}
			p.chunkInfoCacheMutex.Lock()
			delete(p.chunkInfoCache, hex.EncodeToString(info.ChunkHash))
			p.chunkInfoCacheMutex.Unlock()
		}
	}
}
