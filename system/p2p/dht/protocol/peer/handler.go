package peer

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/multiformats/go-multiaddr"
)

func (p *Protocol) handleStreamPeerInfo(stream network.Stream) {

	peerInfo := p.getLocalPeerInfo()
	resp := &types.MessagePeerInfoResp{Message: peerInfo}
	err := protocol.WriteStream(resp, stream)
	if err != nil {
		log.Error("handleStreamPeerInfo", "WriteStream error", err)
		return
	}
}

func (p *Protocol) handleStreamVersion(stream network.Stream) {
	var req types.MessageP2PVersionReq
	err := protocol.ReadStream(&req, stream)
	if err != nil {
		log.Error("handleStreamVersion", "read stream error", err)
		return
	}
	remoteMAddr, err := multiaddr.NewMultiaddr(req.GetMessage().GetAddrFrom())
	if err != nil {
		return
	}
	p.Host.Peerstore().AddAddr(stream.Conn().RemotePeer(), remoteMAddr, peerstore.TempAddrTTL*2)

	p.setExternalAddr(req.GetMessage().GetAddrRecv())
	rand.Seed(time.Now().Unix())
	resp := &types.MessageP2PVersionResp{
		Message: &types.P2PVersion{
			AddrFrom:  p.getExternalAddr(),
			AddrRecv:  remoteMAddr.String(),
			Nonce:     rand.Int63n(102400),
			Timestamp: time.Now().Unix(),
		},
	}
	err = protocol.WriteStream(resp, stream)
	if err != nil {
		log.Error("handleStreamVersion", "WriteStream error", err)
		return
	}
}

func (p *Protocol) handleEventPeerInfo(msg *queue.Message) {
	peerInfos := p.PeerInfoManager.FetchAll()
	var peers []*types.Peer
	for _, pinfo := range peerInfos {
		if pinfo == nil {
			continue
		}
		peers = append(peers, pinfo)
	}
	msg.Reply(p.QueueClient.NewMessage("blockchain", types.EventPeerList, &types.PeerList{Peers: peers}))
}

func (p *Protocol) handleEventNetProtocols(msg *queue.Message) {
	//all protocols net info
	bandProtocols := p.ConnManager.BandTrackerByProtocol()
	allProtocolNetInfo, _ := json.MarshalIndent(bandProtocols, "", "\t")
	log.Debug("handleEventNetInfo", string(allProtocolNetInfo))
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventNetProtocols, bandProtocols))
}

func (p *Protocol) handleEventNetInfo(msg *queue.Message) {
	insize, outsize := p.ConnManager.BoundSize()
	var netinfo types.NodeNetInfo

	netinfo.Externaladdr = p.getExternalAddr()
	localips, err := localIPv4s()
	if err == nil {
		log.Debug("handleEventNetInfo", "localIps", localips)
		netinfo.Localaddr = localips[0]
	} else {
		netinfo.Localaddr = netinfo.Externaladdr
	}

	netinfo.Outbounds = int32(outsize)
	netinfo.Inbounds = int32(insize)
	netinfo.Service = false
	if netinfo.Inbounds != 0 {
		netinfo.Service = true
	}
	netinfo.Peerstore = int32(len(p.Host.Peerstore().PeersWithAddrs()))
	netinfo.Routingtable = int32(p.RoutingTable.Size())
	netstat := p.ConnManager.GetNetRate()

	netinfo.Ratein = p.ConnManager.RateCaculate(netstat.RateIn)
	netinfo.Rateout = p.ConnManager.RateCaculate(netstat.RateOut)
	netinfo.Ratetotal = p.ConnManager.RateCaculate(netstat.RateOut + netstat.RateIn)
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReplyNetInfo, &netinfo))
}
