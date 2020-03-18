package headers

import (
	"errors"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	"github.com/33cn/chain33/types"
	uuid "github.com/google/uuid"
	core "github.com/libp2p/go-libp2p-core"
)

var (
	log = log15.New("module", "p2p.headers")
)

const (
	protoTypeID   = "HeadersProtocolType"
	HeaderInfoReq = "/chain33/headerinfoReq/1.0.0"
)

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &headerInfoProtol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, HeaderInfoReq, &headerInfoHander{})
}

//type Istream
type headerInfoProtol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler
}

func (h *headerInfoProtol) InitProtocol(env *prototypes.P2PEnv) {
	h.P2PEnv = env
	prototypes.RegisterEventHandler(types.EventFetchBlockHeaders, h.handleEvent)
}

func (h *headerInfoProtol) processReq(id string, getheaders *types.P2PGetHeaders) (*types.MessageHeaderResp, error) {
	//获取headers 信息
	log.Debug("processReq", "start", getheaders.GetStartHeight(), "end", getheaders.GetEndHeight())

	if getheaders.GetEndHeight()-getheaders.GetStartHeight() > 2000 || getheaders.GetEndHeight() < getheaders.GetStartHeight() {
		return nil, errors.New("param err")
	}

	blockResp, err := h.SendToBlockChain(types.EventGetHeaders, &types.ReqBlocks{Start: getheaders.GetStartHeight(), End: getheaders.GetEndHeight()})
	if err != nil {
		return nil, err
	}
	headers := blockResp.(*types.Headers)
	peerID := h.GetHost().ID()
	pubkey, _ := h.GetHost().Peerstore().PubKey(peerID).Bytes()
	resp := &types.MessageHeaderResp{MessageData: h.NewMessageCommon(id, peerID.Pretty(), pubkey, false),
		Message: &types.P2PHeaders{Headers: headers.GetItems()}}
	return resp, nil
}

func (h *headerInfoProtol) onReq(id string, getheaders *types.P2PGetHeaders, s core.Stream) {
	senddata, err := h.processReq(id, getheaders)
	if err != nil {
		log.Error("onReq", "processReq", err)
		return
	}

	err = h.SendProtoMessage(senddata, s)
	if err == nil {
		log.Info(" Header response ", "from", s.Conn().LocalPeer().String(), "to", s.Conn().RemotePeer().String(), "height", getheaders.GetStartHeight())
	} else {
		log.Error("OnReq", "SendProtoMessage", err)
		return
	}

}

//GetHeaders 接收来自chain33 blockchain模块发来的请求
func (h *headerInfoProtol) handleEvent(msg *queue.Message) {
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
		rID, err := peer.IDB58Decode(pid)
		if err != nil {
			log.Error("handleEvent", "err", err)
			continue
		}
		req := &prototypes.StreamRequest{
			PeerID:  rID,
			Host:    h.GetHost(),
			Data:    headerReq,
			ProtoID: HeaderInfoReq,
		}
		var resp types.MessageHeaderResp
		err = h.StreamSendHandler(req, &resp)
		if err != nil {
			log.Error("handleEvent", "SendProtoMessage", err)
			continue
		}

		h.SendToBlockChain(types.EventAddBlockHeaders, &types.HeadersPid{Pid: pid, Headers: &types.Headers{Items: resp.GetMessage().GetHeaders()}})

		break

	}

}

type headerInfoHander struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (d *headerInfoHander) Handle(stream core.Stream) {

	protocol := d.GetProtocol().(*headerInfoProtol)
	//解析处理
	if stream.Protocol() == HeaderInfoReq {
		var data types.MessageHeaderReq
		err := d.ReadProtoMessage(&data, stream)
		if err != nil {
			return
		}
		//TODO checkCommonData
		recvData := data.Message

		protocol.onReq(data.GetMessageData().GetId(), recvData, stream)
	}
}
