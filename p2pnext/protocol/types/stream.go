package types

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"io/ioutil"
	"reflect"
	"strings"
)



var (

	log                  = log15.New("module", "p2p.protocol.types")
	streamHandlerTypeMap = make(map[string]reflect.Type)
)


func RegisterStreamHandlerType(typeName, msgID string, handler StreamHandler) {

	if handler == nil {
		panic("RegisterStreamHandlerType, handler is nil, msgId="+msgID)
	}

	if _, exist := protocolTypeMap[typeName]; !exist{
		panic("RegisterStreamHandlerType, protocol type not exist, msgId="+msgID)
	}

	typeID := formatHandlerTypeID(typeName, msgID)

	if _, dup := streamHandlerTypeMap[typeID]; dup {
		panic("addStreamHandler, handler is nil, typeID=" + typeID)
	}
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() == reflect.Ptr {
		handlerType = handlerType.Elem()
	}
	streamHandlerTypeMap[typeID] = handlerType
}

type StreamResponse struct{
	Stream core.Stream
	MsgID string
	Msg   types.Message
}

// StreamHandler stream handler
type StreamHandler interface {

	// GetProtocol get protocol
	GetProtocol() IProtocol
	// SetProtocol 初始化公共结构, 内部通过protocol获取外部依赖公共类, 如queue.client等
	SetProtocol(protocol IProtocol)
	// VerifyRequest  验证请求数据
	VerifyRequest(request []byte) bool
	// Handle 处理请求, 有返回需要设置具体的response结构
	Handle(request []byte, stream core.Stream) (*StreamResponse, error)
}

type BaseStreamHandler struct {
	protocol IProtocol
	child StreamHandler
}

func (s *BaseStreamHandler) SetProtocol(protocol IProtocol) {
	s.protocol = protocol
}

func (s *BaseStreamHandler) Handle([]byte, core.Stream) (*StreamResponse, error) {
	return nil, nil
}


func (s *BaseStreamHandler) VerifyRequest(request []byte) bool {
	//基类统一验证数据, 不需要验证,重写该方法直接返回true
	//TODO, verify request
	return true
}

func (s *BaseStreamHandler) GetProtocol() IProtocol {
	return s.protocol
}

func (s *BaseStreamHandler) HandleStream(stream core.Stream) {

	for {

		buf, err := ioutil.ReadAll(stream)
		if err != nil {
			stream.Reset()
			log.Error("HandleStream", "err", err)
			continue
		}

		if !s.child.VerifyRequest(buf) {
			//invalid request
			continue
		}

		resp, err := s.child.Handle(buf, stream)
		if err != nil {
			continue
		}
		if resp.Msg != nil {
			//TODO, send response message
		}
	}
}



func formatHandlerTypeID(protocolType, msgID string) string {
	return protocolType + "#" + msgID
}

func decodeHandlerTypeID(typeID string) (string, string) {

	arr := strings.Split(typeID, "#")
	return arr[0], arr[1]
}
