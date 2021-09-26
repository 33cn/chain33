package statistical

import (
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
)

//handlerStreamStatistical 返回当前连接的节点信息以及网络带宽信息
func (p *Protocol)handlerStreamStatistical(stream network.Stream){
	defer protocol.CloseStream(stream)
	//TODO check the requst signature
	/*
		remotePeer:=stream.Conn().RemotePeer()
		if remotePeer!= p.SubConfig.ScanPeer{
			return nil
		}
	*/

	peersInfo:= convertPeerInfo(p.PeerInfoManager.FetchAll())
	var statistical types.Statistical
	statistical.Peerinfo=peersInfo
	insize,outsize:=p.ConnManager.BoundSize()
	statistical.Nodeinfo.Inbounds= int32(insize)
	statistical.Nodeinfo.Outbounds= int32(outsize)
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

func convertPeerInfo(peers []*types.Peer)[]*types.P2PPeerInfo{
	var infos []*types.P2PPeerInfo
	for _,peer:=range peers{
		info:=&types.P2PPeerInfo{
			Addr:           peer.GetAddr(),
			Port:           peer.GetPort(),
			Name:           peer.GetName(),
			MempoolSize:    peer.GetMempoolSize(),
			Header:         peer.GetHeader(),
			Version:        peer.GetVersion(),
			LocalDBVersion: peer.GetLocalDBVersion(),
			StoreDBVersion: peer.GetStoreDBVersion(),
			Runningtime:    peer.GetRunningtime(),
			FullNode:       peer.GetFullNode(),
		}

		infos=append(infos,info)
	}

	return infos
}