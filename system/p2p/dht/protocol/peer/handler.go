package peer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/33cn/chain33/common/utils"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p/core/network"
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

	if ip, _ := parseIPAndPort(req.GetAddrFrom()); utils.IsPublicIP(ip) {
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
		RunningTime:    peerInfo.GetRunningTime(),
		FullNode:       peerInfo.GetFullNode(),
		Blocked:        peerInfo.GetBlocked(),
	}
	err = protocol.WriteStream(&types.MessagePeerInfoResp{
		Message: pInfo,
	}, stream)
	if err != nil {
		log.Error("handleStreamPeerInfoOld", "WriteStream error", err)
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
	if ip, _ := parseIPAndPort(msg.GetAddrFrom()); utils.IsPublicIP(ip) {
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
	// no more than 40 peers
	peers := p.RoutingTable.ListPeers()
	if len(peers) > maxPeers*2 {
		peers = peers[:maxPeers*2]
	}
	var peerList types.PeerList
	for _, pid := range peers {
		if info := p.PeerInfoManager.Fetch(pid); info != nil {
			peerList.Peers = append(peerList.Peers, info)
		}
	}
	// add self at last
	if info := p.PeerInfoManager.Fetch(p.Host.ID()); info != nil {
		peerList.Peers = append(peerList.Peers, info)
	}
	msg.Reply(p.QueueClient.NewMessage("blockchain", types.EventPeerList, &peerList))
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
	localips, _ := utils.LocalIPv4s()
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

// add peerName to blacklist
func (p *Protocol) handleEventAddBlacklist(msg *queue.Message) {
	var err error
	defer func() {
		if err != nil {
			msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
		}

	}()
	blackPeer, ok := msg.GetData().(*types.BlackPeer)
	if !ok {
		err = types.ErrInvalidParam
		return
	}
	lifeTime, err := CaculateLifeTime(blackPeer.GetLifetime())
	if err != nil {
		err = errors.New("invalid lifetime")
		return
	}
	var timeduration time.Duration
	if lifeTime == 0 {
		//default 1 year
		timeduration = time.Hour * 24 * 365
	} else {
		timeduration = lifeTime
	}
	//check peerID format
	var pid peer.ID
	pid, err = peer.Decode(blackPeer.GetPeerName())
	if err != nil {
		return
	}
	//close this peer
	err = p.P2PEnv.Host.Network().ClosePeer(pid)
	if err != nil {
		log.Error("handleEventAddBlacklist", "close peer", err)
	}
	p.P2PEnv.ConnBlackList.Add(blackPeer.GetPeerName(), timeduration)

	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: true, Msg: []byte("success")}))

}

// delete peerName from blacklist
func (p *Protocol) handleEventDelBlacklist(msg *queue.Message) {
	var err error
	defer func() {
		if err != nil {
			msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
		}

	}()

	blackPeer, ok := msg.GetData().(*types.BlackPeer)
	if !ok {
		err = types.ErrInvalidParam
		return
	}
	if p.P2PEnv.ConnBlackList.Has(blackPeer.GetPeerName()) {
		p.P2PEnv.ConnBlackList.Add(blackPeer.GetPeerName(), time.Millisecond)
		msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: true, Msg: []byte("success")}))
		return
	}
	err = errors.New("no this peerName")
}

// show all peers from blacklist
func (p *Protocol) handleEventShowBlacklist(msg *queue.Message) {
	peers := p.P2PEnv.ConnBlackList.List()
	//添加peer remoteAddr
	for _, blackPeer := range peers.GetBlackinfo() {
		info := p.P2PEnv.Host.Peerstore().PeerInfo(peer.ID(blackPeer.GetPeerName()))
		if len(info.Addrs) > 0 {
			blackPeer.RemoteAddr = info.Addrs[0].String()
		}
	}
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventShowBlacklist, peers))

}

func (p *Protocol) handleEventDialPeer(msg *queue.Message) {
	maddr, addrinfo, err := p.setPeerCheck(msg)
	if err != nil {
		msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
		return
	}

	ctx, cancel := context.WithTimeout(p.Ctx, time.Second*2)
	defer cancel()
	paddr := msg.GetData().(*types.SetPeer).GetPeerAddr()

	err = p.Host.Connect(ctx, *addrinfo)
	if err != nil {
		log.Error("handleEventDialPeer", "host.Connect", err.Error())
		msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
		return
	}

	if msg.GetData().(*types.SetPeer).GetSeed() { //标记为seed节点
		p.Host.Peerstore().AddAddr(addrinfo.ID, maddr, time.Hour)
		p.Host.ConnManager().Protect(addrinfo.ID, "seed")
		//加入种子列表
		seedNum := len(p.SubConfig.Seeds)
		var index int
		var seed string
		for index, seed = range p.SubConfig.Seeds {
			if seed == paddr {
				break
			}
		}
		if index == seedNum-1 && seed != paddr {
			p.SubConfig.Seeds = append(p.SubConfig.Seeds, paddr)
		}
	}

	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: true, Msg: []byte("dial success")}))

}

func (p *Protocol) handleEventClosePeer(msg *queue.Message) {
	var err error
	defer func() {
		if err != nil {
			msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: false, Msg: []byte(err.Error())}))
		}
	}()

	setPeer, ok := msg.GetData().(*types.SetPeer)
	if !ok {
		err = types.ErrInvalidParam
		return
	}
	pid, err := peer.Decode(setPeer.GetPid())
	if err != nil {
		return
	}

	//close peer
	err = p.Host.Network().ClosePeer(pid)
	if err != nil {
		return
	}

	//delete peer from seedConf
	if p.Host.ConnManager().IsProtected(peer.ID(setPeer.GetPid()), "seed") {
		p.Host.ConnManager().Unprotect(peer.ID(setPeer.GetPid()), "seed")
		//从种子列表中删除
		for index, seed := range p.SubConfig.Seeds {
			a, err := multiaddr.NewMultiaddr(seed)
			if err == nil {
				addrinfo, err := peer.AddrInfoFromP2pAddr(a)
				if err == nil {
					if addrinfo.ID.String() == setPeer.GetPid() {
						//delete
						p.SubConfig.Seeds = append(p.SubConfig.Seeds[:index], p.SubConfig.Seeds[index+1])
					}
				}
			}

		}
	}
	msg.Reply(p.QueueClient.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: true, Msg: []byte(fmt.Sprintf("%v closed success", setPeer.GetPid()))}))
}

func (p *Protocol) setPeerCheck(msg *queue.Message) (multiaddr.Multiaddr, *peer.AddrInfo, error) {
	var err error
	setPeer, ok := msg.GetData().(*types.SetPeer)
	if !ok {
		err = types.ErrInvalidParam
		return nil, nil, err
	}

	maddr, merr := multiaddr.NewMultiaddr(setPeer.GetPeerAddr())
	if merr != nil {
		log.Error("setPeerCheck", "NewMultiaddr", merr, "peerAddr", setPeer.GetPeerAddr())
		err = merr
		return nil, nil, err
	}

	addrinfo, perr := peer.AddrInfoFromP2pAddr(maddr)
	if perr != nil {
		log.Error("setPeerCheck", "AddrInfoFromP2pAddr", perr, "peerAddr", maddr.String())
		err = perr
		return nil, nil, err
	}
	return maddr, addrinfo, nil

}

func (p *Protocol) checkVersionLimit(version string) (isallow bool) {
	if p.SubConfig.VerLimit == "" { //允许所有版本
		return true
	}
	//@分割版本号 1.68.0-a612c9a6@6.8.9
	nodeVers := strings.Split(version, "@")
	if len(nodeVers) != 2 {
		return false
	}
	verLimits := strings.Split(p.SubConfig.VerLimit, ".")
	checkVers := strings.Split(nodeVers[1], ".")
	if len(checkVers) < len(verLimits) {
		return false
	}
	for i, verLimitN := range verLimits {
		limit, _ := strconv.Atoi(verLimitN)
		checkVer, _ := strconv.Atoi(checkVers[i])
		if limit > checkVer {
			return false
		}
	}
	return true

}
