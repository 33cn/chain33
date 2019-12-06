package p2pnext

import (
	"io"
	"io/ioutil"
	"time"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	net "github.com/libp2p/go-libp2p-core/network"
)

const (
	peerInfoReq  = "/chain33/peerinfoReq/1.0.0"
	peerInfoResp = "/chain33/peerinfoResp/1.0.0"
)

//type Istream
type PeerInfoProtol struct {
	client   queue.Client
	done     chan struct{}
	node     *Node                                // local host
	requests map[string]*types.MessagePeerInfoReq // used to access request data from response handlers
}

func NewPeerInfoProtol(node *Node, cli queue.Client, done chan struct{}) *PeerInfoProtol {

	Server := &PeerInfoProtol{}
	node.host.SetStreamHandler(peerInfoReq, Server.ReqHandler)
	node.host.SetStreamHandler(peerInfoResp, Server.onPeerInfoResp)
	Server.requests = make(map[string]*types.MessagePeerInfoReq)
	Server.node = node
	Server.client = cli
	Server.done = done
	return Server
}

func (p *PeerInfoProtol) onPeerInfoResp(s net.Stream) {
	for {

		data := &types.MessagePeerInfoResp{}
		buf, err := ioutil.ReadAll(s)
		if err != nil {
			s.Reset()
			logger.Error(err)
			continue
		}

		// unmarshal it
		proto.Unmarshal(buf, data)
		if err != nil {
			logger.Error(err)
			continue
		}

		valid := p.node.authenticateMessage(data, data.MessageData)

		if !valid {
			logger.Error("Failed to authenticate message")
			continue
		}

		// locate request data and remove it if found
		_, ok := p.requests[data.MessageData.Id]
		if ok {
			// remove request from map as we have processed it here
			delete(p.requests, data.MessageData.Id)
		} else {
			logger.Error("Failed to locate request data boject for response")
			continue
		}
		p.node.peersInfo.Store(data.GetMessage().GetName(), data.GetMessage())
		logger.Debug("%s: Received ping response from %s. Message id:%s. Message: %s.", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.MessageData.Id, data.Message)

	}
}

func (p *PeerInfoProtol) getLoacalPeerInfo() *types.P2PPeerInfo {
	client := p.client
	var peerinfo types.P2PPeerInfo

	msg := client.NewMessage("mempool", types.EventGetMempoolSize, nil)
	err := client.SendTimeout(msg, true, time.Second*10)
	if err != nil {
		logger.Error("GetPeerInfo mempool", "Error", err.Error())
	}
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		logger.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())

	} else {
		meminfo := resp.GetData().(*types.MempoolSize)
		peerinfo.MempoolSize = int32(meminfo.GetSize())
	}

	msg = client.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err = client.SendTimeout(msg, true, time.Minute)
	if err != nil {
		logger.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())
		goto Jump

	}
	resp, err = client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		logger.Error("GetPeerInfo EventGetLastHeader", "Error", err.Error())

		goto Jump

	}
Jump:
	header := resp.GetData().(*types.Header)
	peerinfo.Header = header
	peerinfo.Name = p.node.host.ID().Pretty()

	peerinfo.Addr = p.node.host.Addrs()[0].String()
	return &peerinfo
}

//p2pserver 端接收处理事件
func (p *PeerInfoProtol) ReqHandler(s net.Stream) {
	for {

		var buf []byte

		_, err := io.ReadFull(s, buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			s.Close()
			continue

		}

		//解析处理
		var data types.MessagePeerInfoReq
		err = types.Decode(buf, &data)
		if err != nil {
			continue
		}

		peerinfo := p.getLoacalPeerInfo()

		resp := &types.MessagePeerInfoResp{MessageData: p.node.NewMessageData(data.MessageData.Id, false),
			Message: peerinfo}

		// sign the data
		signature, err := p.node.signProtoMessage(resp)
		if err != nil {
			logger.Error("failed to sign response")
			continue
		}

		// add the signature to the message
		resp.MessageData.Sign = signature

		ok := p.node.sendProtoMessage(s, peerInfoResp, resp)

		if ok {
			logger.Info("%s: Ping response to %s sent.", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())
		}

	}
}

// PeerInfo 向对方节点请求peerInfo信息
func (p *PeerInfoProtol) PeerInfo() {

	for _, s := range p.node.fetchStreams() {
		req := &types.MessagePeerInfoReq{MessageData: p.node.NewMessageData(uuid.New().String(), false)}

		// sign the data
		signature, err := p.node.signProtoMessage(req)
		if err != nil {
			logger.Error("failed to sign pb data")
			return
		}

		req.MessageData.Sign = signature

		ok := p.node.sendProtoMessage(s, peerInfoReq, req)
		if !ok {
			return
		}

		// store ref request so response handler has access to it
		p.requests[req.MessageData.Id] = req

	}

}

func (p *PeerInfoProtol) copy(dest *types.Peer, source *types.P2PPeerInfo) {
	dest.Addr = source.GetAddr()
	dest.Name = source.GetName()
	dest.Header = source.GetHeader()
	dest.Self = false
	dest.MempoolSize = source.GetMempoolSize()
	dest.Port = source.GetPort()
}

//接收chain33其他模块发来的请求消息
func (p *PeerInfoProtol) GetPeersInfo(msg *queue.Message) {

	_, ok := msg.GetData().(*types.P2PGetPeerInfo)
	if !ok {
		return
	}
	var peers []*types.Peer

	p.node.peersInfo.Range(func(key interface{}, value interface{}) bool {
		peerinfo := value.(*types.P2PPeerInfo)
		var peer types.Peer
		p.copy(&peer, peerinfo)
		peers = append(peers, &peer)
		return true
	})

	var peer types.Peer
	peerinfo := p.getLoacalPeerInfo()
	p.copy(&peer, peerinfo)
	peer.Self = true
	peers = append(peers, &peer)
	msg.Reply(p.client.NewMessage("blockchain", types.EventPeerList, &types.PeerList{Peers: peers}))

}
