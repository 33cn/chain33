package headers

import (
	"time"

	"github.com/33cn/chain33/common/log/log15"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"

	uuid "github.com/google/uuid"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	log = log15.New("module", "p2p.headers")
)

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &HeaderInfoProtol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, HeaderInfoReq, &HeaderInfoHander{})
}

const (
	protoTypeID   = "HeadersProtocolType"
	HeaderInfoReq = "/chain33/headerinfoReq/1.0.0"
)

//type Istream
type HeaderInfoProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler
}

func (h *HeaderInfoProtol) InitProtocol(data *prototypes.GlobalData) {
	h.GlobalData = data
	prototypes.RegisterEventHandler(types.EventFetchBlockHeaders, h.handleEvent)
}

func (h *HeaderInfoProtol) OnReq(id string, getheaders *types.P2PGetHeaders, s core.Stream) {

	//获取headers 信息
	if getheaders.GetEndHeight()-getheaders.GetStartHeight() > 2000 || getheaders.GetEndHeight() < getheaders.GetStartHeight() {
		return
	}

	log.Info("OnReq", "Start", "start", getheaders.GetStartHeight(), "end", getheaders.GetEndHeight())
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
		log.Info(" Header response ", "from", s.Conn().LocalPeer().String(), "to", s.Conn().RemotePeer().String(), "height", getheaders.GetStartHeight())
	} else {
		log.Error("OnReq", "SendProtoMessage", err)
		return
	}

}

//GetHeaders 接收来自chain33 blockchain模块发来的请求
func (h *HeaderInfoProtol) handleEvent(msg *queue.Message) {
	req := msg.GetData().(*types.ReqBlocks)
	pids := req.GetPid()
	log.Info("handleEvent", "msg", msg, "pid", pids, "req header start", req.GetStart(), "req header end", req.GetEnd())

	if len(pids) == 0 { //根据指定的pidlist 获取对应的block header
		log.Debug("GetHeaders:pid is nil")
		msg.Reply(h.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{Msg: []byte("no pid")}))
		return
	}

	msg.Reply(h.GetQueueClient().NewMessage("blockchain", types.EventReply, types.Reply{IsOk: true, Msg: []byte("ok")}))

	for _, pid := range pids {

		log.Info("handleEvent", "pid", pid, "start", req.GetStart(), "end", req.GetEnd())

		p2pgetheaders := &types.P2PGetHeaders{StartHeight: req.GetStart(), EndHeight: req.GetEnd(),
			Version: 0}
		peerID := h.GetHost().ID()
		pubkey, _ := h.GetHost().Peerstore().PubKey(peerID).Bytes()
		headerReq := &types.MessageHeaderReq{MessageData: h.NewMessageCommon(uuid.New().String(), peerID.Pretty(), pubkey, false),
			Message: p2pgetheaders}

		// headerReq.MessageData.Sign = signature

		stream, err := h.SendToStream(pid, headerReq, HeaderInfoReq, h.GetHost())
		if err != nil {
			log.Error("handleEvent", "SendProtoMessage", err)
			rID, _ := peer.IDB58Decode(pid)
			h.GetConnsManager().Delete(rID.Pretty())

			continue
		}
		var resp types.MessageHeaderResp
		err = h.ReadProtoMessage(&resp, stream)
		if err != nil {
			log.Error("handleEvent", "ReadProtoMessage", err)
			continue
		}
		log.Info("handleEvent EventAddBlockHeaders", "pid", pid, "headers", resp.GetMessage().GetHeaders()[0])
		client := h.GetQueueClient()
		msg := client.NewMessage("blockchain", types.EventAddBlockHeaders, &types.HeadersPid{Pid: pid, Headers: &types.Headers{Items: resp.GetMessage().GetHeaders()}})
		err = client.Send(msg, false)
		if err != nil {
			log.Error("handleEvent send", "to blockchain EventAddBlockHeaders msg Err", err.Error())
			continue
		}

		break

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
		log.Info("Handler", "protoID", "HeaderInfoReq")
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
