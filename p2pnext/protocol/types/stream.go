package types

import (
	"bufio"
	"context"
	"time"

	"reflect"
	"strings"

	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	proto "github.com/gogo/protobuf/proto"

	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

var (
	log                  = log15.New("module", "p2p.protocol.types")
	streamHandlerTypeMap = make(map[string]reflect.Type)
)

//注册typeName,msgID,处理函数
func RegisterStreamHandlerType(typeName, msgID string, handler StreamHandler) {

	if handler == nil {
		panic("RegisterStreamHandlerType, handler is nil, msgId=" + msgID)
	}

	if _, exist := protocolTypeMap[typeName]; !exist {
		panic("RegisterStreamHandlerType, protocol type not exist, msgId=" + msgID)
	}

	typeID := formatHandlerTypeID(typeName, msgID)

	if _, dup := streamHandlerTypeMap[typeID]; dup {
		panic("addStreamHandler, handler is nil, typeID=" + typeID)
	}
	//streamHandlerTypeMap[typeID] = reflect.TypeOf(handler)
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() == reflect.Ptr {
		handlerType = handlerType.Elem()
	}
	streamHandlerTypeMap[typeID] = handlerType
}

type StreamResponse struct {
	Stream  core.Stream
	protoID string
	Msg     types.Message
}

// StreamHandler stream handler
type StreamHandler interface {
	// GetProtocol get protocol
	GetProtocol() IProtocol
	// SetProtocol 初始化公共结构, 内部通过protocol获取外部依赖公共类, 如queue.client等
	SetProtocol(protocol IProtocol)
	// VerifyRequest  验证请求数据
	VerifyRequest(request []byte, messageComm *types.MessageComm) bool
	// Handle 处理请求
	Handle(stream core.Stream)
}

type BaseStreamHandler struct {
	Protocol IProtocol
	child    StreamHandler
}

func (s *BaseStreamHandler) SetProtocol(protocol IProtocol) {
	s.Protocol = protocol
}

func (s *BaseStreamHandler) Handle(core.Stream) {
	return
}

func (s *BaseStreamHandler) VerifyRequest(request []byte, messageComm *types.MessageComm) bool {
	//基类统一验证数据, 不需要验证,重写该方法直接返回true

	peerID, err := peer.IDB58Decode(messageComm.GetNodeId())
	if err != nil {
		return false
	}

	return verifyData(request, messageComm.GetSign(), peerID, messageComm.GetNodePubKey())

}

func (s *BaseStreamHandler) GetProtocol() IProtocol {
	return s.Protocol
}

//stream事件预处理函数
func (s *BaseStreamHandler) HandleStream(stream core.Stream) {
	log.Info("BaseStreamHandler", "HandlerStream", stream.Conn().RemoteMultiaddr().String(), "proto", stream.Protocol())
	//TODO verify校验放在这里

	s.child.Handle(stream)

}

func (s *BaseStreamHandler) SendToStream(pid string, data proto.Message, msgID protocol.ID, host core.Host) (core.Stream, error) {
	rID, err := peer.IDB58Decode(pid)
	if err != nil {
		return nil, err
	}
	stream, err := host.NewStream(context.Background(), rID, msgID)
	if err != nil {
		log.Error(" SendToStream NewStream", "err", err, "remoteID", rID)
		return nil, err
	}
	err = s.SendProtoMessage(data, stream)
	if err != nil {
		log.Error("SendToStream", "sendProtMessage err", err)
	}

	return stream, err
}

func (s *BaseStreamHandler) SendProtoMessage(data proto.Message, stream core.Stream) error {
	stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
	writer := bufio.NewWriter(stream)
	enc := protobufCodec.Multicodec(nil).Encoder(writer)
	err := enc.Encode(data)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}

func (s *BaseStreamHandler) ReadProtoMessage(data proto.Message, stream core.Stream) error {
	stream.SetReadDeadline(time.Now().Add(30 * time.Second))
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(stream))
	return decoder.Decode(data)
}

func formatHandlerTypeID(protocolType, msgID string) string {
	return protocolType + "#" + msgID
}

func decodeHandlerTypeID(typeID string) (string, string) {

	arr := strings.Split(typeID, "#")
	return arr[0], arr[1]
}
