package protocol

import (
	_ "github.com/33cn/chain33/p2pnext/protocol/broadcast"
	_ "github.com/33cn/chain33/p2pnext/protocol/download"
	_ "github.com/33cn/chain33/p2pnext/protocol/headers"
	_ "github.com/33cn/chain33/p2pnext/protocol/peer"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
)

func Init(data *prototypes.GlobalData) {

	manager := &prototypes.ProtocolManager{}
	manager.Init(data)
}
