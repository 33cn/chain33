package types

import (
	"bufio"
	"context"

	"reflect"
	"strings"

	"github.com/libp2p/go-libp2p-core/protocol"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"

	"github.com/golang/protobuf/proto"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
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
	VerifyRequest(request []byte) bool
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

func (s *BaseStreamHandler) VerifyRequest(request []byte) bool {
	//基类统一验证数据, 不需要验证,重写该方法直接返回true
	//TODO, verify request
	//获取对应的消息结构package

	return true

}

func (s *BaseStreamHandler) verifyData(data []byte, signature []byte, peerId peer.ID, pubKeyData []byte) bool {
	key, err := crypto.UnmarshalPublicKey(pubKeyData)
	if err != nil {
		log.Error("verifyData", err, "Failed to extract key from message key data")
		return false
	}

	// extract node id from the provided public key
	idFromKey, err := peer.IDFromPublicKey(key)

	if err != nil {
		log.Error("verifyData", err, "Failed to extract peer id from public key")
		return false
	}

	// verify that message author node id matches the provided node public key
	if idFromKey != peerId {
		log.Error("verifyData", err, "Node id and provided public key mismatch")
		return false
	}

	res, err := key.Verify(data, signature)
	if err != nil {
		log.Error("verifyData", err, "Error authenticating data")
		return false
	}

	return res
}

func (s *BaseStreamHandler) GetProtocol() IProtocol {
	return s.Protocol
}

//stream事件预处理函数
func (s *BaseStreamHandler) HandleStream(stream core.Stream) {
	log.Info("BaseStreamHandler", "HandlerStream", stream.Conn().RemoteMultiaddr().String(), "proto", stream.Protocol())
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
