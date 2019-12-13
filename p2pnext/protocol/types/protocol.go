package types

import (
	"github.com/33cn/chain33/p2pnext/manage"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/network"
)




// Protocol store public data
type Protocol struct{

	ChainCfg        *types.Chain33Config
	QueueClient     queue.Client
	Host            core.Host
	StreamManager   *manage.StreamManager
	PeerInfoManager *manage.PeerInfoManager

}



func (p *Protocol)Init(streamHandler network.StreamHandler) {

	//

	//
	for id, handler := range streamHandlerMap {
		handler.Init(p)
		p.Host.SetStreamHandler(core.ProtocolID(id), streamHandler)
	}

}



func (p *Protocol)GetChainCfg() *types.Chain33Config{

	return p.ChainCfg

}

func (p *Protocol)GetQueueClient() queue.Client {

	return p.QueueClient
}

func (p *Protocol)GetHost() core.Host{

	return p.Host

}

func (p *Protocol)GetStreamManager() *manage.StreamManager {
	return p.StreamManager

}

func (p *Protocol)GetPeerInfoManager() *manage.PeerInfoManager{
	return p.PeerInfoManager
}
