package protocol

import (
	_ "github.com/33cn/chain33/p2pnext/protocol/broadcast" //广播协议
	_ "github.com/33cn/chain33/p2pnext/protocol/download"  //区块下载协议
	_ "github.com/33cn/chain33/p2pnext/protocol/headers"   //区块头拉取
	_ "github.com/33cn/chain33/p2pnext/protocol/peer"      //邻居节点维护
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
)

// Init init p2p protocol
func Init(data *prototypes.P2PEnv) {
	manager := &prototypes.ProtocolManager{}
	manager.Init(data)
}
