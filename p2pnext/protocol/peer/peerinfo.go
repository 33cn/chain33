package peer

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	p2pty "github.com/33cn/chain33/p2pnext/types"
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
	prototypes.RegisterProtocolType(protoTypeID, &peerInfoProtol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, PeerInfoReq, &peerInfoHandler{})
	prototypes.RegisterStreamHandlerType(protoTypeID, PeerVersionReq, &peerInfoHandler{})

}

//type Istream
type peerInfoProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler
	p2pCfg       *p2pty.P2PSubConfig
	externalAddr string
	mutex        sync.Mutex
}

func (p *peerInfoProtol) InitProtocol(env *prototypes.P2PEnv) {
	p.P2PEnv = env
	p.p2pCfg = env.SubConfig
	prototypes.RegisterEventHandler(types.EventPeerInfo, p.handleEvent)
	prototypes.RegisterEventHandler(types.EventGetNetInfo, p.netinfoHandleEvent)
	go p.DetectNodeAddr()

}

func (p *peerInfoProtol) getLoacalPeerInfo() *types.P2PPeerInfo {
	client := p.GetQueueClient()
	var peerinfo types.P2PPeerInfo

	msg := client.NewMessage("mempool", types.EventGetMempoolSize, nil)
	err := client.SendTimeout(msg, true, time.Second*10)
	if err != nil {
		log.Error("GetPeerInfo mempool", "Error", err.Error())
		return nil
	}
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())
		return nil
	}
	meminfo := resp.GetData().(*types.MempoolSize)
	peerinfo.MempoolSize = int32(meminfo.GetSize())

	msg = client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err = client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())
		return nil

	}
	resp, err = client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		log.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())

		return nil

	}
	header := resp.GetData().(*types.Header)
	peerinfo.Header = header
	peerinfo.Name = p.Host.ID().Pretty()
	splites := strings.Split(p.Host.Addrs()[0].String(), "/")
	port, _ := strconv.Atoi(splites[len(splites)-1])
	peerinfo.Port = int32(port)
	//TODO 需要返回自身的外网地址
	if p.GetExternalAddr() == "" {
		peerinfo.Addr = splites[2]

	} else {
		peerinfo.Addr = p.GetExternalAddr()
	}
	return &peerinfo
}

//p2pserver 端接收处理事件
func (p *peerInfoProtol) OnReq(req *types.MessagePeerInfoReq, s core.Stream) {
	defer s.Close()

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
func (p *peerInfoProtol) GetPeerInfo() []*types.P2PPeerInfo {

	pid := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()
	var peerinfos []*types.P2PPeerInfo
	for _, remoteID := range p.GetConnsManager().FindNearestPeers() {
		if remoteID.Pretty() == p.GetHost().ID().Pretty() {
			continue
		}

		msgReq := &types.MessagePeerInfoReq{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false)}

		req := &prototypes.StreamRequest{
			PeerID:  remoteID,
			Host:    p.GetHost(),
			Data:    msgReq,
			ProtoID: PeerInfoReq,
		}
		var resp types.MessagePeerInfoResp
		err := p.StreamSendHandler(req, &resp)
		if err != nil {
			log.Error("handleEvent", "SendProtoMessage", err)
			continue
		}

		if resp.GetMessage() == nil {
			continue
		}
		peerinfos = append(peerinfos, resp.GetMessage())

	}
	return peerinfos

}

func (p *peerInfoProtol) SetExternalAddr(addr string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if len(strings.Split(addr, "/")) < 2 {
		return
	}
	p.externalAddr = strings.Split(addr, "/")[2]
}

func (p *peerInfoProtol) GetExternalAddr() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.externalAddr
}

func (p *peerInfoProtol) DetectNodeAddr() {

	var seedMap = make(map[string]interface{})
	for _, seed := range p.p2pCfg.Seeds {
		seedSplit := strings.Split(seed, "/")
		seedMap[seedSplit[len(seedSplit)-1]] = seed
	}
	pid := p.GetHost().ID()

	for {
		if len(p.GetConnsManager().FindNearestPeers()) == 0 {
			time.Sleep(time.Second)
			continue
		}
		break
	}

	for _, remoteID := range p.GetConnsManager().FindNearestPeers() {
		if remoteID.Pretty() == pid.Pretty() {
			continue
		}

		if p.GetConnsManager().IsNeighbors(remoteID) {
			continue
		}

		var version types.P2PVersion

		pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()

		req := &types.MessageP2PVersionReq{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false),
			Message: &version}

		s, err := p.Host.NewStream(context.Background(), remoteID, PeerVersionReq)
		if err != nil {
			log.Error("NewStream", "err", err, "remoteID", remoteID)
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

		p.SetExternalAddr(resp.GetMessage().GetAddrRecv())
		log.Info("DetectNodeAddr", "externalAddr", resp.GetMessage().GetAddrRecv())
		//要判断是否是自身局域网的其他节点
		if _, ok := seedMap[remoteID.Pretty()]; !ok {
			continue
		}

		break

	}

}

//接收chain33其他模块发来的请求消息
func (p *peerInfoProtol) handleEvent(msg *queue.Message) {
	pinfos := p.GetPeerInfo()
	var peers []*types.Peer
	var peer types.Peer
	for _, pinfo := range pinfos {
		if pinfo == nil {
			continue
		}
		var peer types.Peer
		p.PeerInfoManager.Copy(&peer, pinfo)
		peers = append(peers, &peer)
		//增加peerInfo 到peerInfoManager
		p.PeerInfoManager.Add(peer.GetName(), &peer)
	}
	peerinfo := p.getLoacalPeerInfo()
	p.PeerInfoManager.Copy(&peer, peerinfo)
	peer.Self = true
	peers = append(peers, &peer)

	msg.Reply(p.GetQueueClient().NewMessage("blockchain", types.EventPeerList, &types.PeerList{Peers: peers}))

}

type peerInfoHandler struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (h *peerInfoHandler) Handle(stream core.Stream) {
	protocol := h.GetProtocol().(*peerInfoProtol)

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
