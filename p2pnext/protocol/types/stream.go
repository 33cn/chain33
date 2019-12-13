package types

import (
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
)



var (

	streamHandlerMap map[string]StreamHandler
)


func RegisterStreamHandler(msgID string, handler StreamHandler) {

	if handler == nil {
		panic("addStreamHandler, handler is nil, msgId="+msgID)
	}
	if _, dup := streamHandlerMap[msgID]; dup {
		panic("addStreamHandler, handler is nil, msgId="+msgID)
	}
	streamHandlerMap[msgID] = handler
}

type StreamResponse struct{
	Stream core.Stream
	MsgID string
	Msg   types.Message
}

// StreamHandler stream handler
type StreamHandler interface {

	// Init 初始化公共结构, 内部通过protocol获取外部依赖公共类, 如queue.client等
	Init(protocol *Protocol)
	// VerifyRequest  验证请求数据
	VerifyRequest(request []byte) bool
	// 处理请求, 有返回需要设置具体的response结构
	Handle(request []byte, stream core.Stream) (*StreamResponse, error)
}

type BaseStreamHandler struct {
	protocol *Protocol
}

func (s *BaseStreamHandler) Init (protocol *Protocol) {
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


func GetStreamHandler(msgID string) (StreamHandler, bool) {

	handler, ok := streamHandlerMap[msgID]
	return handler, ok
}





