// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"sync/atomic"

	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Handle 处理请求
func (p *broadcastProtocol) handleStreamBroadcastV1(stream core.Stream) {

	pid := stream.Conn().RemotePeer()
	sPid := pid.Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	var data types.MessageBroadCast
	err := prototypes.ReadStream(&data, stream)
	if err != nil {
		log.Error("Handle", "pid", pid.Pretty(), "addr", peerAddr, "err", err)
		return
	}
	_ = p.handleReceive(data.Message, sPid, peerAddr, broadcastV1)
	sendNonBlocking(p.peerV1, pid)
}

// 增加广播节点， 每个节点分配一个协程处理广播逻辑
func (p *broadcastProtocol) addBroadcastPeer(id peer.ID) {
	// 广播节点加入保护， 避免被连接管理误删除
	pCtx, pCancel := context.WithCancel(p.Ctx)
	p.broadcastPeers[id] = pCancel
	atomic.AddInt32(&p.peerV1Num, 1)
	p.Host.ConnManager().Protect(id, broadcastTag)
	go p.broadcastV1(pCtx, id)
}

// 移除广播节点
func (p *broadcastProtocol) removeBroadcastPeer(id peer.ID) {
	p.Host.ConnManager().Unprotect(id, broadcastTag)
	delete(p.broadcastPeers, id)
	atomic.AddInt32(&p.peerV1Num, -1)
}

// 兼容处理老版本的广播
func (p *broadcastProtocol) manageBroadcastV1Peer() {

	for {

		select {
		case pid := <-p.peerV1:
			// 老版本限制广播广播数量
			if len(p.broadcastPeers) >= p.p2pCfg.MaxBroadcastPeers {
				break
			}
			_, ok := p.broadcastPeers[pid]
			// 已经存在
			if ok {
				break
			}
			p.addBroadcastPeer(pid)

		case pid := <-p.exitPeer:
			p.removeBroadcastPeer(pid)
		case pid := <-p.errPeer:
			//错误节点减少tag值， 这样在内部连接超额时会优先断开
			p.Host.ConnManager().UpsertTag(pid, broadcastTag, func(oldVal int) int { return oldVal - 1 })
		case <-p.Ctx.Done():
			return

		}
	}
}

//TODO 老版本广播后期全网升级后，可以移除
func (p *broadcastProtocol) broadcastV1(peerCtx context.Context, pid peer.ID) {

	var stream core.Stream
	var err error
	outgoing := p.ps.Sub(bcTopic)
	sPid := pid.String()
	log.Debug("broadcastV1Start", "pid", sPid)
	defer func() {
		p.ps.Unsub(outgoing)
		sendNonBlocking(p.exitPeer, pid)
		if stream != nil {
			_ = stream.Reset()
		}
		if err != nil {
			sendNonBlocking(p.errPeer, pid)
		}
		log.Debug("broadcastV1End", "pid", sPid)
	}()

	for {
		select {
		case data := <-outgoing:
			sendData, doSend := p.handleSend(data, sPid)
			if !doSend {
				break //ignore send
			}
			//包装一层MessageBroadCast
			broadData := &types.MessageBroadCast{
				Message: sendData}

			stream, err = p.Host.NewStream(p.Ctx, pid, broadcastV1)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "NewStreamErr", err)
				return
			}

			err = prototypes.WriteStream(broadData, stream)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "WriteStream err", err)
				return
			}
			prototypes.CloseStream(stream)

		case <-peerCtx.Done():
			return

		}
	}

}

// 相关协程退出时有顺序依赖，统一使用非阻塞模式
func sendNonBlocking(ch chan peer.ID, id peer.ID) {
	select {
	case ch <- id:
	default:
	}
}
