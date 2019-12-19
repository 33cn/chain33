package types

import (
	"bufio"
	//"io/ioutil"
	"reflect"
	"strings"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
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
	streamHandlerTypeMap[typeID] = reflect.TypeOf(handler)
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
	Handle(buf []byte, stream core.Stream)
}

type BaseStreamHandler struct {
	Protocol IProtocol
	child    StreamHandler
}

func (s *BaseStreamHandler) SetProtocol(protocol IProtocol) {
	s.Protocol = protocol
}

func (s *BaseStreamHandler) Handle([]byte, core.Stream) (*StreamResponse, error) {
	return nil, nil
}

func (s *BaseStreamHandler) VerifyRequest(request []byte, msgId string) bool {
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
	var readbuf []byte
	r := bufio.NewReader(bufio.NewReader(stream))

	rlen, err := r.Read(readbuf)
	if err != nil {
		log.Error("HandleStream", "err", err)
		return
	}

	log.Info("BaseStreamHandler", "read size", len(readbuf))
	s.child.Handle(readbuf[:rlen], stream)

}

func formatHandlerTypeID(protocolType, msgID string) string {
	return protocolType + "#" + msgID
}

func decodeHandlerTypeID(typeID string) (string, string) {

	arr := strings.Split(typeID, "#")
	return arr[0], arr[1]
}
