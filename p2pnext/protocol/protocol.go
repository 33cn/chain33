package protocol

import (
	p2p "github.com/33cn/chain33/p2pnext"
	"github.com/33cn/chain33/p2pnext/manage"
	broadcast "github.com/33cn/chain33/p2pnext/protocol/broadcast"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
)

func init() {
	p2p.Register(p2p.PeerInfo, &PeerInfoProtol{})
	p2p.Register(p2p.Header, &HeaderInfoProtol{})
	p2p.Register(p2p.Download, &DownloadBlockProtol{})
	p2p.Register(p2p.BroadCast, &broadcast.Service{})

}


// Protocol store public data
type Protocol struct{

	chainCfg *types.Chain33Config
	queueClient queue.Client
	host core.Host
	streamManager *manage.StreamManger
	peerInfoManager *manage.PeerInfoManager

}


func New(host core.Host, chainCfg *types.Chain33Config, client queue.Client) *Protocol{

	proto := &Protocol{

	}


	return proto
}


func (p *Protocol)Init() {

	//

	//
	for id, handler := range streamHandlerMap {
		handler.Init(p)
		p.host.SetStreamHandler(core.ProtocolID(id), handleStream)
	}

}



func (p *Protocol)GetChainCfg() *types.Chain33Config{

	return p.chainCfg

}

func (p *Protocol)GetQueueClient() queue.Client {

	return p.queueClient
}

func (p *Protocol)GetHost() core.Host{

	return p.host

}

func (p *Protocol)GetStreamManager() *manage.StreamManger{
	return p.streamManager

}

func (p *Protocol)GetPeerInfoManager() *manage.PeerInfoManager{
	return p.peerInfoManager
}
