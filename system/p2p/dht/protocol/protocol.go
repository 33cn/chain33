// Package protocol p2p protocol
package protocol

import (
	"github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	ds "github.com/ipfs/go-datastore"
	core "github.com/libp2p/go-libp2p-core"
	discovery "github.com/libp2p/go-libp2p-discovery"
	kbt "github.com/libp2p/go-libp2p-kbucket"
)

// all protocols
const (
	//p2pstore protocols
	FetchChunk        = "/chain33/fetch-chunk/" + types2.Version
	StoreChunk        = "/chain33/store-chunk/" + types2.Version
	GetHeader         = "/chain33/headers/" + types2.Version
	GetChunkRecord    = "/chain33/chunk-record/" + types2.Version
	BroadcastFullNode = "/chain33/full-node/" + types2.Version

	//sync protocols
	IsSync        = "/chain33/is-sync/" + types2.Version
	IsHealthy     = "/chain33/is-healthy/" + types2.Version
	GetLastHeader = "/chain33/last-header/" + types2.Version
)

// P2PEnv p2p全局公共变量
type P2PEnv struct {
	ChainCfg    *types.Chain33Config
	QueueClient queue.Client
	Host        core.Host
	P2PManager  *p2p.Manager
	SubConfig   *p2pty.P2PSubConfig
	DB          ds.Datastore

	*discovery.RoutingDiscovery

	RoutingTable *kbt.RoutingTable
}
