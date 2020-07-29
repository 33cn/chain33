// Package protocol p2p protocol
package protocol

import (
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
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
