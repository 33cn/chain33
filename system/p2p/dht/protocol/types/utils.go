// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bufio"
	"context"
	"io"
	"time"

	"github.com/libp2p/go-libp2p-core/helpers"

	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

// StreamRequest stream request
type StreamRequest struct {
	// PeerID peer id
	PeerID core.PeerID
	// MsgID stream msg id
	MsgID []core.ProtocolID
	// Data request data
	Data types.Message
}

// SendPeer send data to peer with peer id
func (base *BaseProtocol) SendPeer(req *StreamRequest) error {
	stream, err := NewStream(base.Host, req.PeerID, req.MsgID)
	if err != nil {
		return err
	}
	defer CloseStream(stream)
	err = WriteStream(req.Data, stream)
	if err != nil {
		return err
	}

	CloseStream(stream)
	return nil
}

//SendRecvPeer send request to peer and wait response
func (base *BaseProtocol) SendRecvPeer(req *StreamRequest, resp types.Message) error {

	stream, err := NewStream(base.Host, req.PeerID, req.MsgID)
	if err != nil {
		return err
	}
	defer CloseStream(stream)
	err = WriteStream(req.Data, stream)
	if err != nil {
		return err
	}
	err = ReadStream(resp, stream)
	if err != nil {
		return err
	}
	return nil
}

//NewStream new libp2p stream
func NewStream(host core.Host, pid core.PeerID, msgID []core.ProtocolID) (core.Stream, error) {

	stream, err := host.NewStream(context.Background(), pid, msgID...)
	// EOF表示底层连接断开， 增加一次重试
	if err == io.EOF {
		log.Debug("NewStream", "msg", "RetryConnectEOF")
		stream, err = host.NewStream(context.Background(), pid, msgID...)
	}
	if err != nil {
		log.Error("NewStream", "pid", pid.Pretty(), "msgID", msgID, " err", err)
		return nil, err
	}
	return stream, nil
}

// CloseStream 关闭流， 存在超时阻塞情况, 对于并行情况，可以优先处理读写，最后统一关闭
func CloseStream(stream core.Stream) {
	if stream == nil {
		return
	}
	err := helpers.FullClose(stream)
	if err != nil {
		//这个错误不影响流程，只做记录
		log.Debug("CloseStream", "err", err)
	}
}

//ReadStream  read data from stream
func ReadStream(data types.Message, stream core.Stream) error {

	_ = stream.SetReadDeadline(time.Now().Add(30 * time.Second))
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(stream))
	err := decoder.Decode(data)
	if err != nil {
		log.Error("ReadStream", "pid", stream.Conn().RemotePeer().Pretty(), "msgID", stream.Protocol(), "decode err", err)
		return err
	}
	return nil
}

//WriteStream send data to stream
func WriteStream(data types.Message, stream core.Stream) error {
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
