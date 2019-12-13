package protocol

import (
	_ "github.com/33cn/chain33/p2pnext/protocol/broadcast"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
)




func Init(host core.Host, chainCfg *types.Chain33Config, client queue.Client){

	proto := &prototypes.Protocol{
		ChainCfg:chainCfg,
		QueueClient:client,
		Host:host,
	}

	proto.Init(HandleStream)
}




