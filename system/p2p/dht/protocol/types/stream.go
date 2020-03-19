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

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/helpers"
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
	// PeerID peer id
	PeerID peer.ID
	// MsgID stream msg id
	MsgID string
	// Data request data
	Data types.Message
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

//SignProtoMessage sign data
func (s *BaseStreamHandler) SignProtoMessage(message types.Message, host core.Host) ([]byte, error) {
	return SignProtoMessage(message, host)
}

//VerifyRequest verify data
func (s *BaseStreamHandler) VerifyRequest(message types.Message, messageComm *types.MessageComm) bool {
	//基类统一验证数据, 不需要验证,重写该方法直接返回true

	return AuthenticateMessage(message, messageComm)

}

// GetProtocol get protocol
func (s *BaseStreamHandler) GetProtocol() IProtocol {
	return s.Protocol
}

// CloseStream 关闭流， 存在超时阻塞情况
func (s *BaseStreamHandler) CloseStream(stream core.Stream) {
	err := helpers.FullClose(stream)
	if err != nil {
		//这个错误不影响流程，只做记录
		log.Debug("CloseStream", "err", err)
	}
}

// HandleStream stream事件预处理函数
func (s *BaseStreamHandler) HandleStream(stream core.Stream) {
	log.Debug("BaseStreamHandler", "HandlerStream", stream.Conn().RemoteMultiaddr().String(), "proto", stream.Protocol())
	//TODO verify校验放在这里

	//defer stream.Close()
	s.child.Handle(stream)
	s.CloseStream(stream)

}

// SendPeer send data to peer with peer id
func (s *BaseStreamHandler) SendPeer(req *StreamRequest) error {
	stream, err := s.NewStream(req.PeerID, req.MsgID)
	if err != nil {
		return err
	}
	err = s.WriteStream(req.Data, stream)
	if err != nil {
		return err
	}

	s.CloseStream(stream)
	return nil
}

//SendRecvPeer send request to peer and wait response
func (s *BaseStreamHandler) SendRecvPeer(req *StreamRequest, resp types.Message) error {

	stream, err := s.NewStream(req.PeerID, req.MsgID)
	if err != nil {
		return err
	}
	err = s.WriteStream(req.Data, stream)
	if err != nil {
		return err
	}
	err = s.ReadStream(resp, stream)
	if err != nil {
		return err
	}

	s.CloseStream(stream)
	return nil
}

//SendToStream send data
func (s *BaseStreamHandler) NewStream(pid core.PeerID, msgID string) (core.Stream, error) {

	stream, err := s.GetProtocol().GetP2PEnv().Host.NewStream(context.Background(), pid, core.ProtocolID(msgID))
	if err != nil {
		log.Error("NewStream", "pid", pid.Pretty(), "msgID", msgID, " err", err)
		return nil, err
	}
	return stream, nil
}

//WriteStream send data to stream
func (s *BaseStreamHandler) WriteStream(data types.Message, stream core.Stream) error {
	_ = stream.SetWriteDeadline(time.Now().Add(30 * time.Second))
	writer := bufio.NewWriter(stream)
	enc := protobufCodec.Multicodec(nil).Encoder(writer)
	err := enc.Encode(data)
	if err != nil {
		log.Error("WriteStream", "pid", stream.Conn().RemotePeer().Pretty(), "msgID", stream.Protocol(), "encode err", err)
		return err
	}
	err = writer.Flush()
	if err != nil {
		log.Error("WriteStream", "pid", stream.Conn().RemotePeer().Pretty(), "msgID", stream.Protocol(), "flush err", err)
	}
	return nil
}

//ReadStream  read data from stream
func (s *BaseStreamHandler) ReadStream(data types.Message, stream core.Stream) error {

	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(stream))
	err := decoder.Decode(data)
	if err != nil {
		log.Error("ReadStream", "pid", stream.Conn().RemotePeer().Pretty(), "msgID", stream.Protocol(), "decode err", err)
		return err
	}
	return nil
}

func formatHandlerTypeID(protocolType, msgID string) string {
	return protocolType + "#" + msgID
}

func decodeHandlerTypeID(typeID string) (string, string) {

	arr := strings.Split(typeID, "#")
	return arr[0], arr[1]
}
