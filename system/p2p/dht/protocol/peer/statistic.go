package peer

import (
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/network"
)

const (
	//p2p statistical information
	statisticalInfo = "/chain33/statistical/1.0.0"
)

// handlerStreamStatistical 返回当前连接的节点信息以及网络带宽信息
func (p *Protocol) handlerStreamStatistical(stream network.Stream) {
	defer protocol.CloseStream(stream)
	var statistical types.Statistical
	statistical.Peers = p.PeerInfoManager.FetchAll()
	insize, outsize := p.ConnManager.BoundSize()
	statistical.Nodeinfo = &types.NodeNetInfo{}
	statistical.Nodeinfo.Inbounds = int32(insize)
	statistical.Nodeinfo.Outbounds = int32(outsize)
	statistical.Nodeinfo.Peerstore = int32(len(p.Host.Peerstore().PeersWithAddrs()))
	statistical.Nodeinfo.Routingtable = int32(p.RoutingTable.Size())
	netstat := p.ConnManager.GetNetRate()
	statistical.Nodeinfo.Ratein = p.ConnManager.RateCalculate(netstat.RateIn)
	statistical.Nodeinfo.Rateout = p.ConnManager.RateCalculate(netstat.RateOut)
	statistical.Nodeinfo.Ratetotal = p.ConnManager.RateCalculate(netstat.RateOut + netstat.RateIn)

	err := protocol.WriteStream(&statistical, stream)
	if err != nil {
		log.Error("handleStreamPeerInfoOld", "WriteStream error", err)
		return
	}

}
