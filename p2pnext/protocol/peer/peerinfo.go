package peer

import (
	"context"
	"strconv"
	"strings"

	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/common/log/log15"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	uuid "github.com/google/uuid"
	core "github.com/libp2p/go-libp2p-core"
	//net "github.com/libp2p/go-libp2p-core/network"
)

const (
	protoTypeID    = "PeerProtocolType"
	PeerInfoReq    = "/chain33/peerinfoReq/1.0.0"
	PeerVersionReq = "/chain33/peerVersion/1.0.0"
)

var log = log15.New("module", "p2p.peer")

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &PeerInfoProtol{})
	var hander = new(PeerInfoHandler)
	prototypes.RegisterStreamHandlerType(protoTypeID, PeerInfoReq, hander)
	prototypes.RegisterStreamHandlerType(protoTypeID, PeerVersionReq, hander)

}

//type Istream
type PeerInfoProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler
	p2pCfg *types.P2P
}

func (p *PeerInfoProtol) InitProtocol(data *prototypes.GlobalData) {
	//p.BaseProtocol = new(prototypes.BaseProtocol)
	//p.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	p.GlobalData = data
	p.p2pCfg = data.ChainCfg.GetModuleConfig().P2P
	prototypes.RegisterEventHandler(types.EventPeerInfo, p.handleEvent)
	go p.DetectNodeAddr()

}

func (p *PeerInfoProtol) OnResp(peerinfo *types.P2PPeerInfo, s core.Stream) {
	peerId := s.Conn().RemotePeer().Pretty()

	p.GetPeerInfoManager().Store(peerId, peerinfo)
	log.Info("OnResp Received peerinfo response ", "from", s.Conn().RemotePeer(), "to", s.Conn().LocalPeer())

}

func (p *PeerInfoProtol) getLoacalPeerInfo() *types.P2PPeerInfo {
	client := p.GetQueueClient()
	var peerinfo types.P2PPeerInfo

	msg := client.NewMessage("mempool", types.EventGetMempoolSize, nil)
	err := client.SendTimeout(msg, true, time.Second*10)
	if err != nil {
		log.Error("GetPeerInfo mempool", "Error", err.Error())
	}
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())

	} else {
		meminfo := resp.GetData().(*types.MempoolSize)
		peerinfo.MempoolSize = int32(meminfo.GetSize())
	}

	msg = client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err = client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())
		goto Jump

	}
	resp, err = client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())

		goto Jump

	}
Jump:
	header := resp.GetData().(*types.Header)
	peerinfo.Header = header
	peerinfo.Name = p.Host.ID().Pretty()
	splites := strings.Split(p.Host.Addrs()[0].String(), "/")
	port, _ := strconv.Atoi(splites[len(splites)-1])
	peerinfo.Port = int32(port)
	//TODO 需要返回自身的外网地址
	if externalAddr == "" {
		peerinfo.Addr = splites[2]

	} else {
		peerinfo.Addr = strings.Split(externalAddr, "/")[2]
	}
	return &peerinfo
}

//p2pserver 端接收处理事件
func (p *PeerInfoProtol) OnReq(req *types.MessagePeerInfoReq, s core.Stream) {

	log.Info("OnReq", "peerproto", s.Protocol(), "req", req)

	peerinfo := p.getLoacalPeerInfo()
	peerID := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(peerID).Bytes()

	resp := &types.MessagePeerInfoResp{MessageData: p.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
		Message: peerinfo}

	err := p.SendProtoMessage(resp, s)
	if err != nil {
		log.Error("SendProtoMessage", "err", err)
		return
	}

	log.Info(" OnReq", "localPeer", s.Conn().LocalPeer().String(), "remotePeer", s.Conn().RemotePeer().String())

}

// PeerInfo 向对方节点请求peerInfo信息
func (p *PeerInfoProtol) GetPeerInfo() []*types.P2PPeerInfo {

	pid := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()
	var peerinfos []*types.P2PPeerInfo
	for _, remoteId := range p.GetConnsManager().Fetch() {

		req := &types.MessagePeerInfoReq{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false)}
		s, err := p.Host.NewStream(context.Background(), peer.ID(remoteId), PeerInfoReq)
		if err != nil {
			log.Error("NewStream", "err", err)
			p.GetConnsManager().Delete(remoteId)
			continue
		}

		log.Info("peerInfo", "s.Proto", s.Protocol())
		err = p.SendProtoMessage(req, s)
		if err != nil {
			log.Error("PeerInfo", "sendProtMessage err", err)
			s.Close()
			continue
		}
		var resp types.MessagePeerInfoResp
		err = p.ReadProtoMessage(&resp, s)
		if err != nil {
			log.Error("PeerInfo", "ReadProtoMessage err", err)
			s.Close()
			continue
		}
		peerinfos = append(peerinfos, resp.GetMessage())
		//p.OnResp(resp.GetMessage(), s)
		s.Close()

	}
	return peerinfos

}
func (p *PeerInfoProtol) DetectNodeAddr() {
	for {
		if p.GetConnsManager().Size() == 0 {
			time.Sleep(time.Second)
			continue
		}
		break
	}
	pid := p.GetHost().ID()

	for _, remoteId := range p.GetConnsManager().Fetch() {
		var version types.P2PVersion
		version.AddrFrom = pid.Pretty()
		version.AddrRecv = remoteId
		pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()

		req := &types.MessageP2PVersionReq{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false),
			Message: &version}
		s, err := p.Host.NewStream(context.Background(), peer.ID(remoteId), PeerVersionReq)
		if err != nil {
			log.Error("NewStream", "err", err)
			p.GetConnsManager().Delete(remoteId)
			continue
		}

		err = p.SendProtoMessage(req, s)
		if err != nil {
			log.Error("DetectNodeAddr", "SendProtoMessage err", err)
			p.GetConnsManager().Delete(remoteId)
			continue
		}
		var resp types.MessageP2PVersionResp
		err = p.ReadProtoMessage(&resp, s)
		if err != nil {
			log.Error("DetectNodeAddr", "ReadProtoMessage err", err)
			p.GetConnsManager().Delete(remoteId)
			continue
		}
		log.Info("DetectAddr", "resp", resp)
		if externalAddr == "" {
			externalAddr = resp.GetMessage().GetAddrRecv()
		}

		break

	}

}

//接收chain33其他模块发来的请求消息
func (p *PeerInfoProtol) handleEvent(msg *queue.Message) {
	pinfos := p.GetPeerInfo()
	//peers := p.PeerInfoManager.FetchPeers()
	var peers []*types.Peer
	var peer types.Peer
	for _, pinfo := range pinfos {
		var peer types.Peer
		p.PeerInfoManager.Copy(&peer, pinfo)
		peers = append(peers, &peer)
	}
	peerinfo := p.getLoacalPeerInfo()
	p.PeerInfoManager.Copy(&peer, peerinfo)
	peer.Self = true
	peers = append(peers, &peer)
	msg.Reply(p.GetQueueClient().NewMessage("blockchain", types.EventPeerList, &types.PeerList{Peers: peers}))

}

type PeerInfoHandler struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (h *PeerInfoHandler) Handle(stream core.Stream) {
	protocol := h.GetProtocol().(*PeerInfoProtol)

	//解析处理
	log.Info("PeerInfo Handler", "stream proto", stream.Protocol())
	if stream.Protocol() == PeerInfoReq {
		var req types.MessagePeerInfoReq
		err := h.ReadProtoMessage(&req, stream)
		if err != nil {
			return
		}
		protocol.OnReq(&req, stream)
		return
	} else if stream.Protocol() == PeerVersionReq {
		var req types.MessageP2PVersionReq
		err := h.ReadProtoMessage(&req, stream)
		if err != nil {
			return
		}

		protocol.OnVersionReq(&req, stream)

	}

}

func (p *PeerInfoHandler) VerifyRequest(data []byte) bool {

	return true
}
func (p *PeerInfoHandler) SetProtocol(protocol prototypes.IProtocol) {
	p.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	p.Protocol = protocol
}
