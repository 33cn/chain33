package p2pnext

import (
	"io"
	"io/ioutil"
	"time"

	proto "github.com/gogo/protobuf/proto"
	uuid "github.com/google/uuid"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	net "github.com/libp2p/go-libp2p-net"
)

const (
	headerInfoReq  = "/chain33/headerinfoReq/1.0.0"
	headerInfoResp = "/chain33/headerinfoResp/1.0.0"
)

//type Istream
type HeaderInfoProtol struct {
	client   queue.Client
	done     chan struct{}
	node     *Node                              // local host
	requests map[string]*types.MessageHeaderReq // used to access request data from response handlers
}

func NewHeaderInfoProtol(node *Node, cli queue.Client, done chan struct{}) *HeaderInfoProtol {

	Server := &HeaderInfoProtol{}
	node.host.SetStreamHandler(headerInfoReq, Server.onHeaderReq)
	node.host.SetStreamHandler(headerInfoResp, Server.onHeaderResp)
	Server.requests = make(map[string]*types.MessageHeaderReq)
	Server.node = node
	Server.client = cli
	Server.done = done
	return Server
}

//接收RESP消息

func (h *HeaderInfoProtol) onHeaderResp(s net.Stream) {
	for {

		data := &types.MessageHeaderResp{}
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

		valid := h.node.authenticateMessage(data, data.MessageData)

		if !valid {
			logger.Error("Failed to authenticate message")
			continue
		}

		// locate request data and remove it if found
		_, ok := h.requests[data.MessageData.Id]
		if ok {
			// remove request from map as we have processed it here
			delete(h.requests, data.MessageData.Id)
		} else {
			logger.Error("Failed to locate request data boject for response")
			continue
		}
		logger.Debug("%s: Received ping response from %s. Message id:%s. Message: %s.", s.Conn().LocalPeer(), s.Conn().RemotePeer(), data.MessageData.Id, data.Message)

		//TODO
		p2pheaders := data.Message

		msg := h.client.NewMessage("blockchain", types.EventAddBlockHeaders, &types.HeadersPid{Pid: s.Conn().LocalPeer().String(), Headers: &types.Headers{Items: p2pheaders.GetHeaders()}})
		h.client.Send(msg, false)

	}
}

func (h *HeaderInfoProtol) onHeaderReq(s net.Stream) {
	for {

		var buf []byte

		_, err := io.ReadFull(s, buf)
		if err != nil {
			if err == io.EOF {
				continue
			}
			s.Close()
			return

		}

		//解析处理
		var data types.MessageHeaderReq
		err = types.Decode(buf, &data)
		if err != nil {
			continue
		}

		valid := h.node.authenticateMessage(&data, data.MessageData)
		if !valid {
			logger.Error("Failed to authenticate message")
			continue
		}

		//获取headers 信息
		data.GetMessage().GetStartHeight()
		message := data.GetMessage()
		if message.GetEndHeight()-message.GetStartHeight() > 2000 || message.GetEndHeight() < message.GetStartHeight() {
			continue

		}

		msg := h.client.NewMessage("blockchain", types.EventGetHeaders, &types.ReqBlocks{Start: message.GetStartHeight(), End: message.GetEndHeight()})
		err = h.client.SendTimeout(msg, true, time.Second*30)
		if err != nil {
			logger.Error("GetHeaders", "Error", err.Error())
			continue
		}
		blockResp, err := h.client.WaitTimeout(msg, time.Second*30)
		if err != nil {
			logger.Error("EventGetHeaders WaitTimeout", "Error", err.Error())
			continue
		}

		headers := blockResp.GetData().(*types.Headers)

		resp := &types.MessageHeaderResp{MessageData: h.node.NewMessageData(data.MessageData.Id, false),
			Message: &types.P2PHeaders{Headers: headers.GetItems()}}

		// sign the data
		signature, err := h.node.signProtoMessage(resp)
		if err != nil {
			logger.Error("failed to sign response")
			continue
		}

		// add the signature to the message
		resp.MessageData.Sign = signature

		ok := h.node.sendProtoMessage(s, headerInfoResp, resp)

		if ok {
			logger.Info("%s: Ping response to %s sent.", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())
		}

	}
}

//GetHeaders 接收来自chain33 blockchain模块发来的请求
func (h *HeaderInfoProtol) GetHeaders(msg *queue.Message) {

	req := msg.GetData().(*types.ReqBlocks)
	pids := req.GetPid()
	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		logger.Debug("GetHeaders:pid is nil")
		msg.Reply(h.client.NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(h.client.NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	peersMap := h.node.peersInfo

	for _, pid := range pids {
		data, ok := peersMap.Load(pid)
		if !ok {
			continue
		}
		peerinfo := data.(*types.P2PPeerInfo)
		//去指定的peer上获取对应的blockHeader
		peerId := peerinfo.GetName()

		pstream := h.node.streamMange.GetStream(peerId)
		if pstream == nil {
			continue
		}

		p2pgetheaders := &types.P2PGetHeaders{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
			Version: 0}
		headerReq := &types.MessageHeaderReq{MessageData: h.node.NewMessageData(uuid.New().String(), false),
			Message: p2pgetheaders}

		// sign the data
		signature, err := h.node.signProtoMessage(req)
		if err != nil {
			logger.Error("failed to sign pb data")
			return
		}

		headerReq.MessageData.Sign = signature
		if h.node.sendProtoMessage(pstream, headerInfoReq, headerReq) {
			h.requests[headerReq.MessageData.Id] = headerReq
		}

	}

}
