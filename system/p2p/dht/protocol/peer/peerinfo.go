package peer

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/google/uuid"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
)

const (
	protoTypeID    = "PeerProtocolType"
	peerInfoReq    = "/chain33/peerinfoReq/1.0.0"
	peerVersionReq = "/chain33/peerVersion/1.0.0"
	pubsubTypeID   = "PubSubProtoType"
)

var log = log15.New("module", "p2p.peer")

func init() {
	prototypes.RegisterProtocol(protoTypeID, &peerInfoProtol{})
	prototypes.RegisterStreamHandler(protoTypeID, peerInfoReq, &peerInfoHandler{})
	prototypes.RegisterStreamHandler(protoTypeID, peerVersionReq, &peerInfoHandler{})
	prototypes.RegisterProtocol(pubsubTypeID, &peerPubSub{})

}

//type Istream
type peerInfoProtol struct {
	*prototypes.BaseProtocol
	p2pCfg       *p2pty.P2PSubConfig
	externalAddr string
	mutex        sync.Mutex
}

// InitProtocol init protocol
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
		select {
		case <-time.After(time.Second * 10):
			p.getPeerInfo()
		case <-p.Ctx.Done():
			return
		}

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
		if p.checkDone() {
			log.Warn("getPeerInfo", "process", "done+++++++")
			return
		}

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
				MsgID:  peerInfoReq,
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

	defer func() { //防止出错，数组索引越界
		if r := recover(); r != nil {
			log.Error("setExternalAddr", "recoverErr", r)
		}
	}()

	p.mutex.Lock()
	defer p.mutex.Unlock()
	var spliteAddr string
	splites := strings.Split(addr, "/")
	if len(splites) == 1 {
		spliteAddr = splites[0]
	} else {
		spliteAddr = splites[2]
	}
	if spliteAddr == p.externalAddr {
		return
	}
	p.externalAddr = spliteAddr
	//增加外网地址

	selfExternalAddr := fmt.Sprintf("/ip4/%v/tcp/%d", p.externalAddr, p.p2pCfg.Port)
	newmAddr, err := multiaddr.NewMultiaddr(selfExternalAddr)
	if err != nil {
		return
	}

	p.Host.Peerstore().AddAddr(p.GetHost().ID(), newmAddr, peerstore.AddressTTL)
	log.Debug("setExternalAddr", "afterSetAddrs", p.Host.Addrs(), "peerstoreAddr", p.Host.Peerstore().Addrs(p.Host.ID()))
}

func (p *peerInfoProtol) getExternalAddr() string {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.externalAddr
}

func (p *peerInfoProtol) checkDone() bool {
	select {
	case <-p.Ctx.Done():
		return true
	default:
		return false
	}

}

func (p *peerInfoProtol) detectNodeAddr() {

	//通常libp2p监听的地址列表，第一个为局域网地址，最后一个为外部，先进行外部地址预设置
	addrs := p.GetHost().Addrs()
	if len(addrs) > 0 {
		p.setExternalAddr(addrs[len(addrs)-1].String())
	}
	preExternalAddr := p.getExternalAddr()
	netIP := net.ParseIP(preExternalAddr)
	if isPublicIP(netIP) { //检测是PubIp不用继续通过其他节点获取
		log.Debug("detectNodeAddr", "testPubIp", preExternalAddr)
	}

	log.Info("detectNodeAddr", "+++++++++++++++", preExternalAddr, "addrs", addrs)
	localID := p.GetHost().ID()
	var rangeCount int
	for {

		if p.checkDone() {
			break
		}

		if len(p.GetConnsManager().FetchConnPeers()) == 0 {
			time.Sleep(time.Second)
			continue
		}

		//启动后间隔1分钟，以充分获得节点外网地址
		rangeCount++
		if rangeCount > 2 {
			time.Sleep(time.Minute)
		}

		for _, pid := range p.GetConnsManager().FetchConnPeers() {

			var version types.P2PVersion

			pubkey, _ := p.GetHost().Peerstore().PubKey(localID).Bytes()
			req := &types.MessageP2PVersionReq{MessageData: p.NewMessageCommon(uuid.New().String(), localID.Pretty(), pubkey, false),
				Message: &version}

			s, err := prototypes.NewStream(p.Host, pid, peerVersionReq)
			if err != nil {
				log.Error("NewStream", "err", err, "remoteID", pid)
				continue
			}

			version.Version = p.p2pCfg.Channel

			//real localport
			version.AddrFrom = fmt.Sprintf("/ip4/%v/tcp/%d", p.getExternalAddr(), p.p2pCfg.Port)
			version.AddrRecv = s.Conn().RemoteMultiaddr().String()
			err = prototypes.WriteStream(req, s)
			if err != nil {
				log.Error("DetectNodeAddr", "WriteStream err", err)
				prototypes.CloseStream(s)
				continue
			}
			var resp types.MessageP2PVersionResp
			err = prototypes.ReadStream(&resp, s)
			if err != nil {
				log.Error("DetectNodeAddr", "ReadStream err", err)
				prototypes.CloseStream(s)
				continue
			}
			prototypes.CloseStream(s)
			addr := resp.GetMessage().GetAddrRecv()
			if len(strings.Split(addr, "/")) < 3 {
				continue
			}
			spliteAddr := strings.Split(addr, "/")[2]
			if isPublicIP(net.ParseIP(spliteAddr)) {
				p.setExternalAddr(addr)
			}
		}
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

// Handle 处理请求
func (h *peerInfoHandler) Handle(stream core.Stream) {
	protocol := h.GetProtocol().(*peerInfoProtol)

	//解析处理
	log.Debug("PeerInfo Handler", "stream proto", stream.Protocol())
	if stream.Protocol() == peerInfoReq {
		var req types.MessagePeerInfoReq
		err := prototypes.ReadStream(&req, stream)
		if err != nil {
			return
		}
		protocol.onReq(&req, stream)
		return
	} else if stream.Protocol() == peerVersionReq {
		var req types.MessageP2PVersionReq
		err := prototypes.ReadStream(&req, stream)
		if err != nil {
			return
		}

		protocol.onVersionReq(&req, stream)

	}

}
