// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

//
//import (
//	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
//	core "github.com/libp2p/go-libp2p-core"
//)
//
//// broadcast v2
//type broadcastHandlerV2 struct {
//	prototypes.BaseStreamHandler
//}
//
//// ReuseStream 复用stream, 框架不会进行关闭
//func (b *broadcastHandlerV2) ReuseStream() bool {
//	return true
//}
//
//// Handle 处理广播接收
//func (b *broadcastHandlerV2) Handle(stream core.Stream) {
//	protocol := b.GetProtocol().(*broadcastProtocol)
//	protocol.newStream <- stream
//}

// 更新广播节点， broadcast v2相关代码， 暂时保留
/*func (protocol *broadcastProtocol) manageBroadcastPeers() {

	// 节点启动前十分钟，网络还未稳定，每5秒更新一次连接信息, 后续每分钟更新一次
	tick := time.NewTicker(time.Second * 5)
	timer := time.NewTimer(time.Minute * 10)

	for {
		select {
		//定时获取节点， 加入广播节点列表
		case <-tick.C:
			morePeers := protocol.SubConfig.MaxBroadcastPeers - len(protocol.broadcastPeers)
			peers := protocol.GetConnsManager().FetchConnPeers()
			for i := 0; i < len(peers) && morePeers > 0; i++ {
				pid := peers[i]
				_, ok := protocol.broadcastPeers[pid]
				if !ok {
					protocol.addBroadcastPeer(pid, nil)
					morePeers--
				}
			}

		case <-timer.C:
			tick.Stop()
			timer.Stop()
			tick = time.NewTicker(time.Minute)

		//被动接收到节点 加入广播节点列表
		case stream := <-protocol.newStream:
			pid := stream.Conn().RemotePeer()
			//广播节点列表已达上限
			if len(protocol.broadcastPeers) >= protocol.SubConfig.MaxBroadcastPeers {
				_ = stream.Reset()
				break
			}
			pCancel, ok := protocol.broadcastPeers[pid]
			// 该节点已经在广播列表中, 双方同时发起的情况导致双向重复, 这里采用直接比较节点id策略，保留节点id大的的一方主动发起的广播流
			if ok {

				if strings.Compare(pid.String(), stream.Conn().LocalPeer().String()) > 0 {
					//关闭本地已存在的该节点广播协程
					pCancel()
					//上述操作会将节点从相关结构删除，需要做记录避免误删除
					protocol.simultaneousPeers[pid] = struct{}{}
				} else {
					//保留本地已存在的广播协程， 直接关闭stream并退出
					_ = stream.Reset()
					break
				}
			}
			protocol.addBroadcastPeer(pid, stream)
		case pid := <-protocol.exitPeer:
			protocol.removeBroadcastPeer(pid)
		case pid := <-protocol.errPeer:
			//错误节点减少tag值， 这样在内部连接超额时会优先断开
			protocol.Host.ConnManager().UpsertTag(pid, broadcastTag, func(oldVal int) int { return oldVal - 1 })
		case <-protocol.Ctx.Done():
			tick.Stop()
			timer.Stop()
			return
		}
	}
}*/

/* broadcast v2 相关代码，暂时保留
func (protocol *broadcastProtocol) broadcastV2(pid peer.ID, stream core.Stream, peerCtx context.Context, peerCancel context.CancelFunc) {

	sPid := pid.String()
	outgoing := protocol.ps.Sub(bcTopic, sPid)
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	log.Debug("broadcastV2Start", "pid", sPid, "addr", peerAddr)
	errChan := make(chan error, 2)
	defer func() {
		protocol.ps.Unsub(outgoing)
		err := <-errChan
		if err != nil {
			protocol.errPeer <- pid
		}
		protocol.exitPeer <- pid
		_ = stream.Reset()
		log.Debug("broadcastV2End", "pid", sPid, "addr", peerAddr)
	}()

	// handle broadcast recv
	go func() {
		data := &types.BroadCastData{}
		for {
			//阻塞等待
			err := prototypes.ReadStreamTimeout(data, stream, -1)
			if err != nil {
				log.Error("broadcastV2", "pid", sPid, "addr", peerAddr, "read stream err", err)
				errChan <- err
				peerCancel()
				return
			}
			_ = protocol.handleReceive(data, sPid, peerAddr, broadcastV2)
		}
	}()

	// handle broadcast send
	for {
		var err error
		select {
		case data := <-outgoing:
			sendData, doSend := protocol.handleSend(data, sPid, peerAddr)
			if !doSend {
				break //ignore send
			}
			err = prototypes.WriteStream(sendData, stream)
			if err != nil {
				errChan <- err
				log.Error("broadcastV2", "pid", sPid, "WriteStream err", err)
				return
			}

		case <-peerCtx.Done():
			errChan <- nil
			return
		}
	}
}
*/
