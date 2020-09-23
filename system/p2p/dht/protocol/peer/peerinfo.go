package peer

import (
	"fmt"

	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/multiformats/go-multiaddr"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/google/uuid"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
)

const (
	protoTypeID = "PeerProtocolType"
	peerInfoReq = "/chain33/peerinfoReq/1.0.0" //老版本
	//peerInfoReqV2  = "/chain33/peerinfoReq/1.0.1" //新版本，增加chain33 version 信息反馈
	peerVersionReq = "/chain33/peerVersion/1.0.0"
	pubsubTypeID   = "PubSubProtoType"
)

var log = log15.New("module", "p2p.peer")

func init() {
	prototypes.RegisterProtocol(protoTypeID, &peerInfoProtol{})
	prototypes.RegisterStreamHandler(protoTypeID, peerInfoReq, &peerInfoHandler{})
	//prototypes.RegisterStreamHandler(protoTypeID, peerInfoReqV2, &peerInfoHandler{})
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
	prototypes.RegisterEventHandler(types.EventNetProtocols, p.netprotocolsHandleEvent)

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
	defer func() {
		if r := recover(); r != nil {
			log.Error("getLoacalPeerInfo", "recoverErr", r)
		}
	}()
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
	peerinfo.Version = version.GetVersion() + "@" + version.GetAppVersion()
	peerinfo.StoreDBVersion = version.GetStoreDBVersion()
	peerinfo.LocalDBVersion = version.GetLocalDBVersion()
	return &peerinfo
}

//p2pserver 端接收处理事件
func (p *peerInfoProtol) onReq(req *types.MessagePeerInfoReq, s core.Stream) {
	log.Debug(" OnReq", "localPeer", s.Conn().LocalPeer().String(), "remotePeer", s.Conn().RemotePeer().String(), "peerproto", s.Protocol())
	peerID := p.GetHost().ID()
	pubkey, _ := p.GetHost().Peerstore().PubKey(peerID).Bytes()
	peerinfo := p.getLoacalPeerInfo()
	resp := &types.MessagePeerInfoResp{MessageData: p.NewMessageCommon(req.MessageData.GetId(), peerID.Pretty(), pubkey, false),
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
				MsgID:  []core.ProtocolID{peerInfoReq},
			}
			var resp types.MessagePeerInfoResp
			err := p.SendRecvPeer(req, &resp)
			if err != nil {
				log.Error("handleEvent", "WriteStream", err)
				return
			}
			var dest types.Peer
			if resp.GetMessage() != nil {
				p.PeerInfoManager.Copy(&dest, resp.GetMessage())
				p.PeerInfoManager.Add(peerid.Pretty(), &dest)
			}

		}(remoteID)

	}
	wg.Wait()
}

func (p *peerInfoProtol) setExternalAddr(addr string) {
	var spliteAddr string
	splites := strings.Split(addr, "/")
	if len(splites) == 1 {
		spliteAddr = splites[0]
	} else if len(splites) > 2 {
		spliteAddr = splites[2]
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if spliteAddr != p.externalAddr && isPublicIP(net.ParseIP(spliteAddr)) {

		p.externalAddr = spliteAddr

		//设置外部地址时同时保存到peerstore里
		addr := fmt.Sprintf("/ip4/%s/tcp/%d", spliteAddr, p.SubConfig.Port)
		ma, _ := multiaddr.NewMultiaddr(addr)
		p.Host.Peerstore().AddAddr(p.Host.ID(), ma, peerstore.PermanentAddrTTL)
	}
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
	defer func() {
		if r := recover(); r != nil {
			log.Error("detectNodeAddr", "recover", r)
		}
	}()
	//通常libp2p监听的地址列表，第一个为局域网地址，最后一个为外部，先进行外部地址预设置
	addrs := p.GetHost().Addrs()
	preExternalAddr := strings.Split(addrs[len(addrs)-1].String(), "/")[2]
	p.mutex.Lock()
	p.externalAddr = preExternalAddr
	p.mutex.Unlock()

	netIP := net.ParseIP(preExternalAddr)
	if isPublicIP(netIP) { //检测是PubIp不用继续通过其他节点获取
		log.Debug("detectNodeAddr", "testPubIp", preExternalAddr)
	}
	log.Info("detectNodeAddr", "+++++++++++++++", preExternalAddr, "addrs", addrs)
	localID := p.GetHost().ID()
	var rangeCount int
	queryInterval := time.Minute
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
			time.Sleep(queryInterval)
		}
		log.Info("detectNodeAddr", "conns amount", len(p.Host.Network().Conns()))
		for _, conn := range p.Host.Network().Conns() {
			if rangeCount > 2 {
				time.Sleep(time.Second / 10)
			}
			var version types.P2PVersion
			pid := conn.RemotePeer()
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
			p.setExternalAddr(addr)
			remoteMAddr, err := multiaddr.NewMultiaddr(resp.GetMessage().GetAddrFrom())
			if err != nil {
				continue
			}
			p.Host.Peerstore().AddAddr(pid, remoteMAddr, queryInterval*3)
		}
	}

}

//接收chain33其他模块发来的请求消息
func (p *peerInfoProtol) handleEvent(msg *queue.Message) {
	pinfos := p.PeerInfoManager.FetchPeerInfosInMin()
	var peers []*types.Peer
	localinfo := p.getLoacalPeerInfo()
	var checkHeight int64
	var lcoalPeer types.Peer
	if localinfo != nil {

		checkHeight = localinfo.Header.Height - 512
		p.PeerInfoManager.Copy(&lcoalPeer, localinfo)
		lcoalPeer.Self = true
	}

	for _, pinfo := range pinfos {
		if pinfo == nil {
			continue
		}
		//过滤比自身节点低很多的节点，传送给blockchain或者rpc模块
		if pinfo.Header.Height >= checkHeight {
			peers = append(peers, pinfo)
		}

	}

	peers = append(peers, &lcoalPeer)

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

	switch stream.Protocol() {
	case peerInfoReq:
		var req types.MessagePeerInfoReq
		err := prototypes.ReadStream(&req, stream)
		if err != nil {
			return
		}
		protocol.onReq(&req, stream)
	case peerVersionReq:
		var req types.MessageP2PVersionReq
		err := prototypes.ReadStream(&req, stream)
		if err != nil {
			return
		}

		protocol.onVersionReq(&req, stream)

	}

}
