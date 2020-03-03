package protocol

import (
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
)

// Init init p2p protocol
func Init(data *prototypes.GlobalData) {
	manager := &prototypes.ProtocolManager{}
	manager.Init(data)
}
