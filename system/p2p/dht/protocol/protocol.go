// Package protocol p2p protocol
package protocol

import (
	"context"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/extension"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	discovery "github.com/libp2p/go-libp2p-discovery"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

// P2PEnv p2p全局公共变量
type P2PEnv struct {
	Ctx             context.Context
	ChainCfg        *types.Chain33Config
	SubConfig       *types2.P2PSubConfig
	API             client.QueueProtocolAPI
	QueueClient     queue.Client
	Host            host.Host
	P2PManager      *p2p.Manager
	DB              ds.Datastore
	PeerInfoManager IPeerInfoManager
	ConnManager     IConnManager
	Pubsub          *extension.PubSub
	RoutingTable    *kbt.RoutingTable
	//普通路由表的一个子表，仅包含接近同步完成的节点
	HealthyRoutingTable *kbt.RoutingTable
	*discovery.RoutingDiscovery
}

type IPeerInfoManager interface {
	Refresh(info *types.Peer)
	Fetch(pid peer.ID) *types.Peer
	FetchAll() []*types.Peer
	PeerHeight(pid peer.ID) int64
}

type IConnManager interface {
	FetchConnPeers() []peer.ID
	BoundSize() (in int, out int)
	GetNetRate() metrics.Stats
	BandTrackerByProtocol() *types.NetProtocolInfos
	RateCalculate(ratebytes float64) string
}

func (p *P2PEnv) QueryModule(topic string, ty int64, data interface{}) (interface{}, error) {
	msg := p.QueueClient.NewMessage(topic, ty, data)
	err := p.QueueClient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := p.QueueClient.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}
