package p2pstore

import (
	"context"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
)

var log = log15.New("module", "protocol.p2pstore")

//Protocol ...
type Protocol struct {
	*protocol.P2PEnv //协议共享接口变量

	notifying      sync.Map
	chunkWhiteList sync.Map

	//普通路由表的一个子表，仅包含接近同步完成的节点
	healthyRoutingTable *kb.RoutingTable

	//本节点保存的chunk的索引表，会随着网络拓扑结构的变化而变化
	localChunkInfo      map[string]LocalChunkInfo
	localChunkInfoMutex sync.RWMutex

	retryInterval time.Duration
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

//InitProtocol initials the protocol.
func InitProtocol(env *protocol.P2PEnv) {
	p := &Protocol{
		P2PEnv:              env,
		healthyRoutingTable: kb.NewRoutingTable(dht.KValue, kb.ConvertPeerID(env.Host.ID()), time.Minute, env.Host.Peerstore()),
		retryInterval:       30 * time.Second,
	}
	//routing table更新时同时更新healthy routing table
	rm := p.RoutingTable.PeerRemoved
	p.RoutingTable.PeerRemoved = func(id peer.ID) {
		rm(id)
		p.healthyRoutingTable.Remove(id)
	}
	p.initLocalChunkInfoMap()

	//注册p2p通信协议，用于处理节点之间请求
	p.Host.SetStreamHandler(protocol.FetchChunk, protocol.HandlerWithAuth(p.handleStreamFetchChunk)) //数据较大，采用特殊写入方式
	p.Host.SetStreamHandler(protocol.StoreChunk, protocol.HandlerWithAuth(p.handleStreamStoreChunk))
	p.Host.SetStreamHandler(protocol.GetHeader, protocol.HandlerWithAuthAndSign(p.handleStreamGetHeader))
	p.Host.SetStreamHandler(protocol.GetChunkRecord, protocol.HandlerWithAuthAndSign(p.handleStreamGetChunkRecord))
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, protocol.EventHandlerWithRecover(p.handleEventNotifyStoreChunk))
	protocol.RegisterEventHandler(types.EventGetChunkBlock, protocol.EventHandlerWithRecover(p.handleEventGetChunkBlock))
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, protocol.EventHandlerWithRecover(p.handleEventGetChunkBlockBody))
	protocol.RegisterEventHandler(types.EventGetChunkRecord, protocol.EventHandlerWithRecover(p.handleEventGetChunkRecord))

	go func() {
		for i := 0; i < 3; i++ { //节点启动后充分初始化 healthy routing table
			p.updateHealthyRoutingTable()
			time.Sleep(time.Second * 1)
		}
		if p.SubConfig.IsFullNode {
			discovery.Advertise(context.Background(), p.RoutingDiscovery, protocol.BroadcastFullNode)
		}

		ticker1 := time.NewTicker(time.Minute)
		ticker2 := time.NewTicker(types2.RefreshInterval)
		ticker3 := time.NewTicker(types2.CheckHealthyInterval)
		ticker4 := time.NewTicker(time.Hour)
		for {
			select {
			case <-ticker1.C:
				p.updateChunkWhiteList()
			case <-ticker2.C:
				p.republish()
			case <-ticker3.C:
				p.updateHealthyRoutingTable()
			case <-ticker4.C:
				//debug info
				p.localChunkInfoMutex.Lock()
				log.Info("debugLocalChunk", "local chunk hash len", len(p.localChunkInfo))
				p.localChunkInfoMutex.Unlock()
				p.debugFullNode()
			}
		}
	}()
}

func (p *Protocol) updateChunkWhiteList() {
	p.chunkWhiteList.Range(func(k, v interface{}) bool {
		t := v.(time.Time)
		if time.Since(t) > time.Minute*10 {
			p.chunkWhiteList.Delete(k)
		}
		return true
	})
}

//TODO
//debug info, to delete soon
func (p *Protocol) debugFullNode() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	peerInfos, err := discovery.FindPeers(ctx, p.RoutingDiscovery, protocol.BroadcastFullNode)
	if err != nil {
		log.Error("debugFullNode", "FindPeers error", err)
		return
	}
	for _, peerInfo := range peerInfos {
		log.Info("debugFullNode", "ID", peerInfo.ID, "maddrs", peerInfo.Addrs)
	}
	log.Info("debugFullNode", "total count", len(peerInfos))
}
