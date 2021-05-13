package peer

import (
	"encoding/json"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
)

func (p *Protocol) handleStreamPeerInfo(stream network.Stream) {
	peerInfo := p.getLocalPeerInfo()
	if peerInfo == nil {
		return
	}
	err := protocol.WriteStream(peerInfo, stream)
	if err != nil {
		log.Error("handleStreamPeerInfo", "WriteStream error", err)
		return
	}
}

func (p *Protocol) handleStreamVersion(stream network.Stream) {
	var req types.P2PVersion
	err := protocol.ReadStream(&req, stream)
	if err != nil {
		log.Error("handleStreamVersion", "read stream error", err)
		return
	}
	if req.GetVersion() != p.SubConfig.Channel {
		// 不是同一条链，拉黑且断开连接
		p.ConnBlackList.Add(stream.Conn().RemotePeer().Pretty(), time.Hour*24)
		_ = stream.Conn().Close()
		return
	}

	if ip, _ := parseIPAndPort(req.GetAddrFrom()); isPublicIP(ip) {
		remoteMAddr, err := multiaddr.NewMultiaddr(req.GetAddrFrom())
		if err != nil {
			return
		}
		p.Host.Peerstore().AddAddr(stream.Conn().RemotePeer(), remoteMAddr, time.Hour*24)
	}

	p.setExternalAddr(req.GetAddrRecv())
	resp := &types.P2PVersion{
		AddrFrom:  p.getExternalAddr(),
		AddrRecv:  stream.Conn().RemoteMultiaddr().String(),
		Timestamp: time.Now().Unix(),
	}
	err = protocol.WriteStream(resp, stream)
	if err != nil {
		log.Error("handleStreamVersion", "WriteStream error", err)
		return
	}
}

func (p *Protocol) handleStreamPeerInfoOld(stream network.Stream) {
	var req types.MessagePeerInfoReq
	err := protocol.ReadStream(&req, stream)
	if err != nil {
		log.Error("handleStreamPeerInfoOld", "read stream error", err)
		return
	}

	peerInfo := p.getLocalPeerInfo()
	if peerInfo == nil {
		return
	}
	pInfo := &types.P2PPeerInfo{
		Addr:           peerInfo.Addr,
		Port:           peerInfo.Port,
		Name:           peerInfo.Name,
		MempoolSize:    peerInfo.MempoolSize,
		Header:         peerInfo.Header,
		Version:        peerInfo.Version,
		LocalDBVersion: peerInfo.LocalDBVersion,
		StoreDBVersion: peerInfo.StoreDBVersion,
	}
	err = protocol.WriteStream(&types.MessagePeerInfoResp{
		Message: pInfo,
	}, stream)
	if err != nil {
		log.Error("handleStreamPeerInfo", "WriteStream error", err)
		return
	}
}

func (p *Protocol) handleStreamVersionOld(stream network.Stream) {
	var req types.MessageP2PVersionReq
	err := protocol.ReadStream(&req, stream)
	if err != nil {
		log.Error("handleStreamVersion", "read stream error", err)
		return
	}
	msg := req.Message
	if msg.GetVersion() != p.SubConfig.Channel {
		// 不是同一条链，拉黑且断开连接
		p.ConnBlackList.Add(stream.Conn().RemotePeer().Pretty(), time.Hour*24)
		_ = stream.Conn().Close()
		return
	}
	if ip, _ := parseIPAndPort(msg.GetAddrFrom()); isPublicIP(ip) {
		remoteMAddr, err := multiaddr.NewMultiaddr(msg.GetAddrFrom())
		if err != nil {
			return
		}
		p.Host.Peerstore().AddAddr(stream.Conn().RemotePeer(), remoteMAddr, time.Hour*24)
	}

	p.setExternalAddr(msg.GetAddrRecv())
	resp := &types.MessageP2PVersionResp{
		Message: &types.P2PVersion{
			AddrFrom:  p.getExternalAddr(),
			AddrRecv:  stream.Conn().RemoteMultiaddr().String(),
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
	peers := p.PeerInfoManager.FetchAll()
	msg.Reply(p.QueueClient.NewMessage("blockchain", types.EventPeerList, &types.PeerList{Peers: peers}))
}

func (p *Protocol) handleEventNetProtocols(msg *queue.Message) {
	//all protocols net info
	bandProtocols := p.ConnManager.BandTrackerByProtocol()
	allProtocolNetInfo, _ := json.MarshalIndent(bandProtocols, "", "\t")
	log.Debug("handleEventNetInfo", "allProtocolNetInfo", string(allProtocolNetInfo))
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventNetProtocols, bandProtocols))
}

func (p *Protocol) handleEventNetInfo(msg *queue.Message) {
	insize, outsize := p.ConnManager.BoundSize()
	var netinfo types.NodeNetInfo

	netinfo.Externaladdr = p.getPublicIP()
	localips, _ := localIPv4s()
	if len(localips) != 0 {
		log.Info("handleEventNetInfo", "localIps", localips)
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

	netinfo.Ratein = p.ConnManager.RateCalculate(netstat.RateIn)
	netinfo.Rateout = p.ConnManager.RateCalculate(netstat.RateOut)
	netinfo.Ratetotal = p.ConnManager.RateCalculate(netstat.RateOut + netstat.RateIn)
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReplyNetInfo, &netinfo))
}
