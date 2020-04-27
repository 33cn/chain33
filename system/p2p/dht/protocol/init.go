package protocol

import (
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/broadcast" //广播协议
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/download"  //区块下载协议
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/headers"   //区块头拉取
	_ "github.com/33cn/chain33/system/p2p/dht/protocol/peer"      //邻居节点维护
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
)

// Init init p2p protocol
func Init(data *prototypes.P2PEnv) {
	manager := &prototypes.ProtocolManager{}
	manager.Init(data)
}
