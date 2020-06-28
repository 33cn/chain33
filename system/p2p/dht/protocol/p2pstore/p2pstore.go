package p2pstore

import (
	"math/rand"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	kb "github.com/libp2p/go-libp2p-kbucket"
	"github.com/multiformats/go-multiaddr"
)

var log = log15.New("module", "protocol.p2pstore")

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

	//full nodes
	fullNodes []peer.ID
}

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

func InitProtocol(env *protocol.P2PEnv) {
	p := &Protocol{
		P2PEnv:              env,
		healthyRoutingTable: kb.NewRoutingTable(dht.KValue, kb.ConvertPeerID(env.Host.ID()), time.Minute, env.Host.Peerstore()),
		retryInterval:       30 * time.Second,
	}
	//routing table更新时同时更新healthy routing table
	rm := p.RoutingTable.RoutingTable().PeerRemoved
	p.RoutingTable.RoutingTable().PeerRemoved = func(id peer.ID) {
		rm(id)
		p.healthyRoutingTable.Remove(id)
	}
	p.initLocalChunkInfoMap()

	//注册p2p通信协议，用于处理节点之间请求
	p.Host.SetStreamHandler(protocol.FetchChunk, protocol.HandlerWithAuth(p.HandleStreamFetchChunk)) //数据较大，采用特殊写入方式
	p.Host.SetStreamHandler(protocol.StoreChunk, protocol.HandlerWithAuth(p.HandleStreamStoreChunk))
	p.Host.SetStreamHandler(protocol.GetHeader, protocol.HandlerWithAuthAndSign(p.HandleStreamGetHeader))
	p.Host.SetStreamHandler(protocol.GetChunkRecord, protocol.HandlerWithAuthAndSign(p.HandleStreamGetChunkRecord))
	//同时注册eventHandler，用于处理blockchain模块发来的请求
	protocol.RegisterEventHandler(types.EventNotifyStoreChunk, protocol.EventHandlerWithRecover(p.HandleEventNotifyStoreChunk))
	protocol.RegisterEventHandler(types.EventGetChunkBlock, protocol.EventHandlerWithRecover(p.HandleEventGetChunkBlock))
	protocol.RegisterEventHandler(types.EventGetChunkBlockBody, protocol.EventHandlerWithRecover(p.HandleEventGetChunkBlockBody))
	protocol.RegisterEventHandler(types.EventGetChunkRecord, protocol.EventHandlerWithRecover(p.HandleEventGetChunkRecord))

	//全节点的p2pstore保存所有chunk, 不进行republish操作
	err := p.initFullNodes()
	if err != nil {
		panic(err)
	}
	go func() {
		for i := 0; i < 3; i++ { //节点启动后充分初始化 healthy routing table
			p.updateHealthyRoutingTable()
			time.Sleep(time.Second * 1)
		}
		ticker1 := time.Tick(time.Minute)
		ticker2 := time.Tick(types2.RefreshInterval)
		ticker3 := time.Tick(types2.CheckHealthyInterval)
		for {
			select {
			case <-ticker1:
				p.updateChunkWhiteList()
				p.localChunkInfoMutex.Lock()
				log.Info("debugLocalChunk", "local chunk hash len", len(p.localChunkInfo))
				p.localChunkInfoMutex.Unlock()
			case <-ticker2:
				p.republish()
			case <-ticker3:
				p.updateHealthyRoutingTable()
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

func (p *Protocol) initFullNodes() error {
	if p.SubConfig.IsFullNode {
		return nil
	}

	for _, node := range p.SubConfig.FullNodes {
		// Turn the destination into a multiaddr.
		maddr, err := multiaddr.NewMultiaddr(node)
		if err != nil {
			return err
		}
		// Extract the peer ID from the multiaddr.
		info, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			return err
		}
		p.Host.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
		p.fullNodes = append(p.fullNodes, info.ID)
	}
	shuffle(p.fullNodes)
	return nil
}

func shuffle(slice []peer.ID) {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	for len(slice) > 0 {
		n := len(slice)
		randIndex := r.Intn(n)
		slice[n-1], slice[randIndex] = slice[randIndex], slice[n-1]
		slice = slice[:n-1]
	}
}
