package protocol

import (
	_ "github.com/33cn/chain33/p2pnext/protocol/broadcast"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
)




func Init(data *prototypes.GlobalData){

	manager := &prototypes.ProtocolManager{}
	manager.Init(data)
}




