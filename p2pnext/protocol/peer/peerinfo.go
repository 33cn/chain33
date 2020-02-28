package peer

import (
	"context"
	"strconv"
	"strings"

	"time"

	//"github.com/libp2p/go-libp2p-core/peerstore"
	//multiaddr "github.com/multiformats/go-multiaddr"
	//"github.com/golang/protobuf/proto"
	//"github.com/libp2p/go-libp2p-core/protocol"

	//"fmt"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/common/log/log15"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	uuid "github.com/google/uuid"
	core "github.com/libp2p/go-libp2p-core"
)

const (
	protoTypeID    = "PeerProtocolType"
	PeerInfoReq    = "/chain33/peerinfoReq/1.0.0"
	PeerVersionReq = "/chain33/peerVersion/1.0.0"
)

var log = log15.New("module", "p2p.peer")

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &PeerInfoProtol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, PeerInfoReq, &PeerInfoHandler{})
	prototypes.RegisterStreamHandlerType(protoTypeID, PeerVersionReq, &PeerInfoHandler{})

}

//type Istream
type PeerInfoProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler
	p2pCfg *types.P2P
}

func (p *PeerInfoProtol) InitProtocol(data *prototypes.GlobalData) {
	p.GlobalData = data
	p.p2pCfg = data.ChainCfg.GetModuleConfig().P2P
	prototypes.RegisterEventHandler(types.EventPeerInfo, p.handleEvent)
	prototypes.RegisterEventHandler(types.EventGetNetInfo, p.netinfoHandleEvent)
	go p.DetectNodeAddr()

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
	log.Info(" OnReq", "localPeer", s.Conn().LocalPeer().String(), "remotePeer", s.Conn().RemotePeer().String(), "peerproto", s.Protocol())

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

}

// PeerInfo 向对方节点请求peerInfo信息
func (p *PeerInfoProtol) GetPeerInfo() []*types.P2PPeerInfo {

	pid := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()
	var peerinfos []*types.P2PPeerInfo
	for _, remoteId := range p.GetConnsManager().Fetch() {
		if remoteId == p.GetHost().ID().Pretty() {
			continue
		}

		rID, err := peer.IDB58Decode(remoteId)
		if err != nil {
			continue
		}
		req := &types.MessagePeerInfoReq{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false)}

		recordStart := time.Now().UnixNano()
		s, err := p.SendToStream(remoteId, req, PeerInfoReq, p.GetHost())
		if err != nil {
			log.Error("GetPeerInfo NewStream", "err", err, "remoteID", remoteId)
			p.GetConnsManager().Delete(rID.Pretty())
			continue
		}
		var resp types.MessagePeerInfoResp
		err = p.ReadProtoMessage(&resp, s)
		if err != nil {
			log.Error("PeerInfo", "ReadProtoMessage err", err)
			continue
		}

		recordEnd := time.Now().UnixNano()
		p.GetConnsManager().RecoredLatency(rID, time.Duration((recordEnd-recordStart)/1e6))
		peerinfos = append(peerinfos, resp.GetMessage())

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

	var seedMap = make(map[string]interface{})
	for _, seed := range p.p2pCfg.Seeds {
		seedSplit := strings.Split(seed, "/")
		seedMap[seedSplit[len(seedSplit)-1]] = seed
	}
	pid := p.GetHost().ID()
	for _, remoteId := range p.GetConnsManager().Fetch() {
		if remoteId == p.GetHost().ID().Pretty() {
			continue
		}

		var version types.P2PVersion

		pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()

		req := &types.MessageP2PVersionReq{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false),
			Message: &version}

		rID, err := peer.IDB58Decode(remoteId)
		if err != nil {
			continue
		}

		s, err := p.Host.NewStream(context.Background(), rID, PeerVersionReq)
		if err != nil {
			log.Error("NewStream", "err", err, "remoteID", rID)
			p.GetConnsManager().Delete(rID.Pretty())
			continue
		}
		version.AddrFrom = s.Conn().LocalMultiaddr().String()
		version.AddrRecv = s.Conn().RemoteMultiaddr().String()
		err = p.SendProtoMessage(req, s)
		if err != nil {
			log.Error("DetectNodeAddr", "SendProtoMessage err", err)
			continue
		}
		var resp types.MessageP2PVersionResp
		err = p.ReadProtoMessage(&resp, s)
		if err != nil {
			log.Error("DetectNodeAddr", "ReadProtoMessage err", err)
			continue
		}
		log.Info("DetectAddr", "resp", resp)

		//if externalAddr == "" {
		externalAddr = resp.GetMessage().GetAddrRecv()
		log.Info("DetectNodeAddr", "externalAddr", externalAddr)
		//要判断是否是自身局域网的其他节点
		if _, ok := seedMap[remoteId]; !ok {
			continue
		}

		break
		//	}

	}

}

//接收chain33其他模块发来的请求消息
func (p *PeerInfoProtol) handleEvent(msg *queue.Message) {
	pinfos := p.GetPeerInfo()
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
