// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
)

type broadcastHandler struct {
	*prototypes.BaseStreamHandler
}

// Handle 处理请求
func (handler *broadcastHandler) Handle(stream core.Stream) {

	protocol := handler.GetProtocol().(*broadcastProtocol)
	pid := stream.Conn().RemotePeer()
	sPid := pid.Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	var data types.MessageBroadCast
	err := prototypes.ReadStream(&data, stream)
	if err != nil {
		log.Error("Handle", "pid", pid.Pretty(), "addr", peerAddr, "err", err)
		return
	}

	_ = protocol.handleReceive(data.Message, sPid, peerAddr, broadcastV1)
}

// VerifyRequest verify request
func (handler *broadcastHandler) VerifyRequest(types.Message, *types.MessageComm) bool {
	return true
}

type broadcastHandlerV2 struct {
	prototypes.BaseStreamHandler
}

// ReuseStream 复用stream, 框架不会进行关闭
func (b *broadcastHandlerV2) ReuseStream() bool {
	return true
}

// Handle 处理广播接收
func (b *broadcastHandlerV2) Handle(stream core.Stream) {
	protocol := b.GetProtocol().(*broadcastProtocol)
	protocol.newStream <- stream
}
