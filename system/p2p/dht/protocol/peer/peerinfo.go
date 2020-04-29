package peer

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	uuid "github.com/google/uuid"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	protoTypeID    = "PeerProtocolType"
	PeerInfoReq    = "/chain33/peerinfoReq/1.0.0"
	PeerVersionReq = "/chain33/peerVersion/1.0.0"
)

var log = log15.New("module", "p2p.peer")

func init() {
	prototypes.RegisterProtocol(protoTypeID, &peerInfoProtol{})
	prototypes.RegisterStreamHandler(protoTypeID, PeerInfoReq, &peerInfoHandler{})
	prototypes.RegisterStreamHandler(protoTypeID, PeerVersionReq, &peerInfoHandler{})

}

//type Istream
type peerInfoProtol struct {
	*prototypes.BaseProtocol
	p2pCfg       *p2pty.P2PSubConfig
	externalAddr string
	mutex        sync.Mutex
}

func (p *peerInfoProtol) InitProtocol(env *prototypes.P2PEnv) {
	p.P2PEnv = env
	p.p2pCfg = env.SubConfig
	prototypes.RegisterEventHandler(types.EventPeerInfo, p.handleEvent)
	prototypes.RegisterEventHandler(types.EventGetNetInfo, p.netinfoHandleEvent)
	go p.detectNodeAddr()
	go p.fetchPeersInfo()

}

func (p *peerInfoProtol) fetchPeersInfo() {
	for {
		<-time.After(time.Second * 10)
		p.getPeerInfo()

	}
}
func (p *peerInfoProtol) getLoacalPeerInfo() *types.P2PPeerInfo {
	var peerinfo types.P2PPeerInfo

	resp, err := p.QueryMempool(types.EventGetMempoolSize, nil)
	if err != nil {
		log.Error("getlocalPeerInfo", "sendToMempool", err)
		return nil
	}

	meminfo := resp.(*types.MempoolSize)
	peerinfo.MempoolSize = int32(meminfo.GetSize())

	resp, err = p.QueryBlockChain(types.EventGetLastHeader, nil)
	if err != nil {
		log.Error("getlocalPeerInfo", "sendToBlockChain", err)
		return nil
	}

	header := resp.(*types.Header)
	peerinfo.Header = header
	peerinfo.Name = p.Host.ID().Pretty()
	splites := strings.Split(p.Host.Addrs()[0].String(), "/")
	port, _ := strconv.Atoi(splites[len(splites)-1])
	peerinfo.Port = int32(port)
	//TODO 需要返回自身的外网地址
	if p.getExternalAddr() == "" {
		peerinfo.Addr = splites[2]

	} else {
		peerinfo.Addr = p.getExternalAddr()
	}
	return &peerinfo
}

//p2pserver 端接收处理事件
func (p *peerInfoProtol) onReq(req *types.MessagePeerInfoReq, s core.Stream) {
	log.Debug(" OnReq", "localPeer", s.Conn().LocalPeer().String(), "remotePeer", s.Conn().RemotePeer().String(), "peerproto", s.Protocol())

	peerinfo := p.getLoacalPeerInfo()
	peerID := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(peerID).Bytes()

	resp := &types.MessagePeerInfoResp{MessageData: p.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
		Message: peerinfo}

	err := prototypes.WriteStream(resp, s)
	if err != nil {
		log.Error("WriteStream", "err", err)
		return
	}

}

// PeerInfo 向对方节点请求peerInfo信息
func (p *peerInfoProtol) getPeerInfo() {

	pid := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(pid).Bytes()
	var wg sync.WaitGroup
	for _, remoteID := range p.GetConnsManager().FetchConnPeers() {
		if remoteID.Pretty() == p.GetHost().ID().Pretty() {
			continue
		}
		//修改为并发获取peerinfo信息
		wg.Add(1)
		go func(peerid peer.ID) {
			defer wg.Done()
			msgReq := &types.MessagePeerInfoReq{MessageData: p.NewMessageCommon(uuid.New().String(), pid.Pretty(), pubkey, false)}

			req := &prototypes.StreamRequest{
				PeerID: peerid,
				Data:   msgReq,
				MsgID:  PeerInfoReq,
			}
			var resp types.MessagePeerInfoResp
			err := p.SendRecvPeer(req, &resp)
			if err != nil {
				log.Error("handleEvent", "WriteStream", err)
				return
			}

			if resp.GetMessage() == nil {
				return
			}
			var dest types.Peer
			p.PeerInfoManager.Copy(&dest, resp.GetMessage())
			p.PeerInfoManager.Add(peerid.Pretty(), &dest)
		}(remoteID)

	}
	wg.Wait()
}

func (p *peerInfoProtol) setExternalAddr(addr string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if len(strings.Split(addr, "/")) < 2 {
		return
	}
	p.externalAddr = strings.Split(addr, "/")[2]
}

func (p *peerInfoProtol) getExternalAddr() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.externalAddr
}

func (p *peerInfoProtol) detectNodeAddr() {
	//通常libp2p监听的地址列表，第一个为127，第二个为外部，先进行外部地址预设置
	addrs := p.GetHost().Addrs()
	if len(addrs) > 0 {
		p.setExternalAddr(addrs[len(addrs)-1].String())
	}
	var seedMap = make(map[string]interface{})
	for _, seed := range p.p2pCfg.Seeds {
		seedSplit := strings.Split(seed, "/")
		seedMap[seedSplit[len(seedSplit)-1]] = seed
	}
	pid := p.GetHost().ID()

	for {
		if len(p.GetConnsManager().FetchConnPeers()) == 0 {
			time.Sleep(time.Second)
			continue
		}
		break
	}

	openedStreams := make([]core.Stream, 0)
	for _, remoteID := range p.GetConnsManager().FetchConnPeers() {
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

		s, err := prototypes.NewStream(p.Host, remoteID, PeerVersionReq)
		if err != nil {
			log.Error("NewStream", "err", err, "remoteID", remoteID)
			continue
		}
		openedStreams = append(openedStreams, s)
		version.Version = p.p2pCfg.Channel
		version.AddrFrom = s.Conn().LocalMultiaddr().String()
		version.AddrRecv = s.Conn().RemoteMultiaddr().String()
		err = prototypes.WriteStream(req, s)
		if err != nil {
			log.Error("DetectNodeAddr", "WriteStream err", err)
			continue
		}
		var resp types.MessageP2PVersionResp
		err = prototypes.ReadStream(&resp, s)
		if err != nil {
			log.Error("DetectNodeAddr", "ReadStream err", err)
			continue
		}
		log.Debug("DetectAddr", "resp", resp)

		p.setExternalAddr(resp.GetMessage().GetAddrRecv())
		log.Debug("DetectNodeAddr", "externalAddr", resp.GetMessage().GetAddrRecv())
		//要判断是否是自身局域网的其他节点
		if _, ok := seedMap[remoteID.Pretty()]; !ok {
			continue
		}

		break

	}
	// 统一关闭stream
	for _, stream := range openedStreams {
		prototypes.CloseStream(stream)
	}

}

//接收chain33其他模块发来的请求消息
func (p *peerInfoProtol) handleEvent(msg *queue.Message) {
	pinfos := p.PeerInfoManager.FetchPeerInfosInMin()
	var peers []*types.Peer
	var peer types.Peer
	for _, pinfo := range pinfos {
		if pinfo == nil {
			continue
		}

		peers = append(peers, pinfo)

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
	log.Debug("PeerInfo Handler", "stream proto", stream.Protocol())
	if stream.Protocol() == PeerInfoReq {
		var req types.MessagePeerInfoReq
		err := prototypes.ReadStream(&req, stream)
		if err != nil {
			return
		}
		protocol.onReq(&req, stream)
		return
	} else if stream.Protocol() == PeerVersionReq {
		var req types.MessageP2PVersionReq
		err := prototypes.ReadStream(&req, stream)
		if err != nil {
			return
		}

		protocol.onVersionReq(&req, stream)

	}

}
