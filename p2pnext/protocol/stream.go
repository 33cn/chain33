package protocol

import (
	"github.com/33cn/chain33/types"
	net "github.com/libp2p/go-libp2p-core/network"
	"io/ioutil"
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
	msgID string
	msg   types.Message
}

// StreamHandler stream handler
type StreamHandler interface {

	// Init 初始化公共结构, 内部通过protocol获取外部依赖公共类, 如queue.client等
	Init(protocol *Protocol)
	// VerifyRequest  验证请求数据
	VerifyRequest(request []byte) bool
	// 处理请求, 有返回需要设置具体的response结构
	Handle(request []byte) (*StreamResponse, error)
}

type BaseStreamHandler struct {
	protocol *Protocol
}

func (s *BaseStreamHandler) Init (protocol *Protocol) {
	s.protocol = protocol
}

func (s *BaseStreamHandler) Handle(request []byte) (*StreamResponse, error) {
	return nil, nil
}


func (s *BaseStreamHandler) VerifyRequest(request []byte) bool {
	//基类统一验证数据, 不需要验证,重写该方法直接返回true
	//TODO, verify request
	return true
}

func handleStream(stream net.Stream) {

	msgID := stream.Protocol()
	for {

		buf, err := ioutil.ReadAll(stream)
		if err != nil {
			stream.Reset()
			logger.Error(err)
			continue
		}

		if handler, ok := streamHandlerMap[string(msgID)]; ok {

			if !handler.VerifyRequest(buf) {
				//invalid request
				continue
			}

			resp, err := handler.Handle(buf)
		    if err != nil {
		    	continue
			}
			if resp.msg != nil {
				//TODO, send response message
			}
		}
	}
}



