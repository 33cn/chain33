package headers

import (
	"time"

	"github.com/33cn/chain33/common/log/log15"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	core "github.com/libp2p/go-libp2p-core"

	uuid "github.com/google/uuid"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	net "github.com/libp2p/go-libp2p-core/network"
)

var (
	log = log15.New("module", "p2p.headers")
)

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &HeaderInfoProtol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, headerInfoReq, &HeaderInfoHander{})
	prototypes.RegisterStreamHandlerType(protoTypeID, headerInfoResp, &HeaderInfoHander{})
}

const (
	protoTypeID    = "HeadersProtocolType"
	headerInfoReq  = "/chain33/headerinfoReq/1.0.0"
	headerInfoResp = "/chain33/headerinfoResp/1.0.0"
)

//type Istream
type HeaderInfoProtol struct {
	*prototypes.BaseProtocol
	requests map[string]*types.MessageHeaderReq // used to access request data from response handlers
}

func (h *HeaderInfoProtol) InitProtocol(data *prototypes.GlobalData) {
	h.GlobalData = data
	h.ChainCfg = data.ChainCfg
	//注册事件处理函数
	h.QueueClient = data.QueueClient
	prototypes.RegisterEventHandler(types.EventGetHeaders, h.handleEvent)
}

//接收RESP消息
func (h *HeaderInfoProtol) OnResp(headers *types.P2PHeaders, s net.Stream) {

	client := h.GetQueueClient()
	msg := client.NewMessage("blockchain", types.EventAddBlockHeaders, &types.HeadersPid{Pid: s.Conn().LocalPeer().String(), Headers: &types.Headers{Items: headers.GetHeaders()}})
	client.Send(msg, false)

}

func (h *HeaderInfoProtol) OnReq(id string, getheaders *types.P2PGetHeaders, s net.Stream) {

	//获取headers 信息
	if getheaders.GetEndHeight()-getheaders.GetStartHeight() > 2000 || getheaders.GetEndHeight() < getheaders.GetStartHeight() {
		return

	}
	client := h.GetQueueClient()
	msg := client.NewMessage("blockchain", types.EventGetHeaders, &types.ReqBlocks{Start: getheaders.GetStartHeight(), End: getheaders.GetEndHeight()})
	err := client.SendTimeout(msg, true, time.Second*30)
	if err != nil {
		log.Error("GetHeaders", "Error", err.Error())
		return
	}
	blockResp, err := client.WaitTimeout(msg, time.Second*30)
	if err != nil {
		log.Error("EventGetHeaders WaitTimeout", "Error", err.Error())
		return
	}

	headers := blockResp.GetData().(*types.Headers)
	peerID := h.GetHost().ID()
	pubkey, _ := h.GetHost().Peerstore().PubKey(peerID).Bytes()
	resp := &types.MessageHeaderResp{MessageData: h.NewMessageCommon(id, peerID.Pretty(), pubkey, false),
		Message: &types.P2PHeaders{Headers: headers.GetItems()}}

	ok := h.StreamManager.SendProtoMessage(resp, s)
	if ok {
		log.Info("%s: Ping response to %s sent.", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())
	}

}

//GetHeaders 接收来自chain33 blockchain模块发来的请求
func (h *HeaderInfoProtol) handleEvent(msg *queue.Message) {

	req := msg.GetData().(*types.ReqBlocks)
	pids := req.GetPid()
	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		log.Debug("GetHeaders:pid is nil")
		msg.Reply(h.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(h.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))

	for _, pid := range pids {
		data := h.PeerInfoManager.Load(pid)
		if data == nil {
			continue
		}
		peerinfo := data.(*types.P2PPeerInfo)
		//去指定的peer上获取对应的blockHeader
		peerId := peerinfo.GetName()

		pstream := h.StreamManager.GetStream(peerId)
		if pstream == nil {
			continue
		}

		p2pgetheaders := &types.P2PGetHeaders{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
			Version: 0}
		peerID := h.GetHost().ID()
		pubkey, _ := h.GetHost().Peerstore().PubKey(peerID).Bytes()
		headerReq := &types.MessageHeaderReq{MessageData: h.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
			Message: p2pgetheaders}

		// sign the data
		// signature, err := h.node.SignProtoMessage(req)
		// if err != nil {
		// 	logger.Error("failed to sign pb data")
		// 	return
		// }

		// headerReq.MessageData.Sign = signature
		if h.StreamManager.SendProtoMessage(headerReq, pstream) {
			h.requests[headerReq.MessageData.Id] = headerReq
		}

	}

}

type HeaderInfoHander struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (d *HeaderInfoHander) Handle(req []byte, stream core.Stream) {

	protocol := d.GetProtocol().(*HeaderInfoProtol)

	//解析处理
	if stream.Protocol() == headerInfoReq {
		var data types.MessageHeaderReq
		err := types.Decode(req, &data)
		if err != nil {

			return
		}
		protocol.CheckMessage(data.GetMessageData().GetId())
		recvData := data.Message

		protocol.OnReq(data.MessageData.Id, recvData, stream)
		return
	}
	var data types.MessageHeaderResp
	err := types.Decode(req, &data)
	if err != nil {
		return
	}
	protocol.CheckMessage(data.GetMessageData().GetId())
	recvData := data.Message
	protocol.OnResp(recvData, stream)
	return

}

func (h *HeaderInfoHander) VerifyRequest(data []byte) bool {

	return true
}
func (h *HeaderInfoProtol) CheckMessage(id string) bool {
	_, ok := h.requests[id]
	if ok {
		delete(h.requests, id)
		return true
	}

	log.Error("Failed to locate request data boject for response")
	return false
}
