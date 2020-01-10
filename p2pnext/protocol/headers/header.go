package headers

import (
	"context"

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
	var hander = new(HeaderInfoHander)
	prototypes.RegisterStreamHandlerType(protoTypeID, HeaderInfoReq, hander)
}

const (
	protoTypeID   = "HeadersProtocolType"
	HeaderInfoReq = "/chain33/headerinfoReq/1.0.0"
)

//type Istream
type HeaderInfoProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler

	requests map[string]*types.MessageHeaderReq // used to access request data from response handlers
}

func (h *HeaderInfoProtol) InitProtocol(data *prototypes.GlobalData) {
	h.BaseProtocol = new(prototypes.BaseProtocol)
	h.requests = make(map[string]*types.MessageHeaderReq)
	h.GlobalData = data
	h.requests = make(map[string]*types.MessageHeaderReq)
	prototypes.RegisterEventHandler(types.EventFetchBlockHeaders, h.handleEvent)
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

	err = h.SendProtoMessage(resp, s)
	if err == nil {
		log.Info("%s: Ping response to %s sent.", s.Conn().LocalPeer().String(), s.Conn().RemotePeer().String())
	}

	log.Info("OnReq", "SendProtoMessage", "ok")

}

//GetHeaders 接收来自chain33 blockchain模块发来的请求
func (h *HeaderInfoProtol) handleEvent(msg *queue.Message) {
	req := msg.GetData().(*types.ReqBlocks)
	pids := req.GetPid()
	log.Info("handleEvent", "msgxxxxxx", msg, "pid", pids)

	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		log.Debug("GetHeaders:pid is nil")
		msg.Reply(h.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(h.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))
	log.Info("handlerEvent", "conns num", p.GetConnsManager().Fetch())

	for _, pid := range pids {

		log.Info("handleEvent", "pid", pid, "start", req.GetStart(), "end", req.GetEnd())
		pConn := h.GetConnsManager().Get(pid)
		if pConn == nil {
			log.Error("handleEvent", "no is conn from ", pid)
			continue
		}

		p2pgetheaders := &types.P2PGetHeaders{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
			Version: 0}
		peerID := h.GetHost().ID()
		pubkey, _ := h.GetHost().Peerstore().PubKey(peerID).Bytes()
		headerReq := &types.MessageHeaderReq{MessageData: h.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
			Message: p2pgetheaders}

		// headerReq.MessageData.Sign = signature
		stream, err := h.Host.NewStream(context.Background(), pConn.RemotePeer(), HeaderInfoReq)
		if err != nil {
			log.Error("NewStream", "err", err, "peerID", pConn.RemotePeer())
			h.BaseProtocol.ConnManager.Delete(pConn.RemotePeer().Pretty())
			continue
		}
		//发送请求
		if err := h.SendProtoMessage(headerReq, stream); err != nil {
			log.Error("handleEvent", "SendProtoMessage", err)
		}

		var resp types.MessageHeaderResp
		err = h.ReadProtoMessage(&resp, stream)
		if err != nil {
			log.Error("handleEvent", "SendProtoMessage", err)
		}

		client := h.GetQueueClient()
		msg := client.NewMessage("blockchain", types.EventAddBlockHeaders, &types.HeadersPid{Pid: pid, Headers: &types.Headers{Items: resp.GetMessage().GetHeaders()}})
		err = client.Send(msg, false)
		if err != nil {
			log.Error("send", "to blockchain EventAddBlockHeaders msg Err", err.Error())
		}

		stream.Close()

	}

}

type HeaderInfoHander struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (d *HeaderInfoHander) Handle(stream core.Stream) {

	protocol := d.GetProtocol().(*HeaderInfoProtol)

	//解析处理
	if stream.Protocol() == HeaderInfoReq {
		var data types.MessageHeaderReq

		err := d.ReadProtoMessage(&data, stream)
		if err != nil {
			return
		}
		//TODO checkCommonData
		//protocol.CheckMessage(data.GetMessageData().GetId())
		recvData := data.Message
		protocol.OnReq(data.GetMessageData().GetId(), recvData, stream)
		return
	}

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
