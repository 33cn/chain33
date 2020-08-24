// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
	"strings"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
)

var (
	log                  = log15.New("module", "p2p.protocol.types")
	streamHandlerTypeMap = make(map[string]reflect.Type)
)

// RegisterStreamHandler 注册typeName,msgID,处理函数
func RegisterStreamHandler(typeName, msgID string, handler StreamHandler) {

	if handler == nil {
		panic("RegisterStreamHandler, handler is nil, msgId=" + msgID)
	}

	if _, exist := protocolTypeMap[typeName]; !exist {
		panic("RegisterStreamHandler, protocol type not exist, msgId=" + msgID)
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

// StreamHandler stream handler
type StreamHandler interface {
	// GetProtocol get protocol
	GetProtocol() IProtocol
	// SetProtocol 初始化公共结构, 内部通过protocol获取外部依赖公共类, 如queue.client等
	SetProtocol(protocol IProtocol)
	// VerifyRequest  验证请求数据
	VerifyRequest(message types.Message, messageComm *types.MessageComm) bool
	//SignMessage
	SignProtoMessage(message types.Message, host core.Host) ([]byte, error)
	// Handle 处理请求
	Handle(stream core.Stream)
}

// BaseStreamHandler base stream handler
type BaseStreamHandler struct {
	Protocol IProtocol
	child    StreamHandler
}

// SetProtocol set protocol
func (s *BaseStreamHandler) SetProtocol(protocol IProtocol) {
	s.Protocol = protocol
}

// Handle handle stream
func (s *BaseStreamHandler) Handle(core.Stream) {
}

// SignProtoMessage sign data
func (s *BaseStreamHandler) SignProtoMessage(message types.Message, host core.Host) ([]byte, error) {
	return SignProtoMessage(message, host)
}

// VerifyRequest verify data
func (s *BaseStreamHandler) VerifyRequest(message types.Message, messageComm *types.MessageComm) bool {
	//基类统一验证数据, 不需要验证,重写该方法直接返回true

	return AuthenticateMessage(message, messageComm)

}

// GetProtocol get protocol
func (s *BaseStreamHandler) GetProtocol() IProtocol {
	return s.Protocol
}

// HandleStream stream事件预处理函数
func (s *BaseStreamHandler) HandleStream(stream core.Stream) {
	//log.Debug("BaseStreamHandler", "HandlerStream", stream.Conn().RemoteMultiaddr().String(), "proto", stream.Protocol())
	//TODO verify校验放在这里
	s.child.Handle(stream)
	CloseStream(stream)
}

func formatHandlerTypeID(protocolType, msgID string) string {
	return protocolType + "#" + msgID
}

func decodeHandlerTypeID(typeID string) (string, string) {

	arr := strings.Split(typeID, "#")
	return arr[0], arr[1]
}
