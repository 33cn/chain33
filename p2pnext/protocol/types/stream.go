// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
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
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

var (
	log                  = log15.New("module", "p2p.protocol.types")
	streamHandlerTypeMap = make(map[string]reflect.Type)
)

// RegisterStreamHandlerType 注册typeName,msgID,处理函数
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

// StreamRequest stream request
type StreamRequest struct {
	PeerID  peer.ID
	Host    core.Host
	Data    types.Message
	ProtoID protocol.ID
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

func (s *BaseStreamHandler) SignProtoMessage(message types.Message, host core.Host) ([]byte, error) {
	return SignProtoMessage(message, host)
}

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
	log.Debug("BaseStreamHandler", "HandlerStream", stream.Conn().RemoteMultiaddr().String(), "proto", stream.Protocol())
	//TODO verify校验放在这里

	s.child.Handle(stream)

}

func (s *BaseStreamHandler) StreamSendHandler(in *StreamRequest, result types.Message) error {
	stream, err := s.SendToStream(in.PeerID.Pretty(), in.Data, in.ProtoID, in.Host)
	if err != nil {
		return err
	}
	return s.ReadProtoMessage(result, stream)
}

func (s *BaseStreamHandler) SendToStream(pid string, data types.Message, msgID protocol.ID, host core.Host) (core.Stream, error) {
	rID, err := peer.IDB58Decode(pid)
	if err != nil {
		return nil, err
	}
	stream, err := host.NewStream(context.Background(), rID, msgID)
	if err != nil {
		log.Error("SendToStream NewStream", "err", err, "remoteID", rID)
		return nil, err
	}
	err = s.SendProtoMessage(data, stream)
	if err != nil {
		log.Error("SendToStream", "sendProtMessage err", err, "remoteID", rID)
	}

	return stream, err
}

func (s *BaseStreamHandler) SendProtoMessage(data types.Message, stream core.Stream) error {
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

func (s *BaseStreamHandler) ReadProtoMessage(data types.Message, stream core.Stream) error {
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
