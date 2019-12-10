package protos

import (
	p2p "github.com/33cn/chain33/p2pnext"
	broadcast "github.com/33cn/chain33/p2pnext/protos/broadcast"
)

func init() {
	p2p.Register(p2p.PeerInfo, &PeerInfoProtol{})
	p2p.Register(p2p.Header, &HeaderInfoProtol{})
	p2p.Register(p2p.Download, &DownloadBlockProtol{})
	p2p.Register(p2p.BroadCast, &broadcast.Service{})

}
