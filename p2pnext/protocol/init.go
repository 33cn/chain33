package protocol

import (
	_ "github.com/33cn/chain33/p2pnext/protocol/broadcast"
	_ "github.com/33cn/chain33/p2pnext/protocol/download"
	_ "github.com/33cn/chain33/p2pnext/protocol/headers"
	_ "github.com/33cn/chain33/p2pnext/protocol/peer"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	_ "github.com/libp2p/go-libp2p-core/protocol"
)

// Init init p2p protocol
func Init(data *prototypes.P2PEnv) {
	manager := &prototypes.ProtocolManager{}
	manager.Init(data)
}
