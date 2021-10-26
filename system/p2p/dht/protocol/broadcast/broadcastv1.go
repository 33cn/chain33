// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"context"
	"sync/atomic"

	"github.com/33cn/chain33/p2p/utils"

	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Handle 处理请求
func (p *broadcastProtocol) handleStreamBroadcastV1(stream core.Stream) {

	pid := stream.Conn().RemotePeer()
	sPid := pid.String()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	var data types.MessageBroadCast
	err := prototypes.ReadStream(&data, stream)
	if err != nil {
		log.Error("handleStreamBroadcastV1", "pid", sPid, "addr", peerAddr, "read err", err)
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
				prototypes.CloseStream(stream)
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

// 发送广播数据到节点, 支持延迟关闭内部stream，主要考虑多个节点并行发送情况，不需要等待关闭
func (p *broadcastProtocol) sendPeer(data interface{}, pid, version string) error {

	// broadcast v1 TODO 版本升级后移除代码
	sendData, doSend := p.handleSend(data, pid)
	if !doSend {
		return nil
	}
	//包装一层MessageBroadCast
	broadData := &types.MessageBroadCast{
		Message: sendData}

	rawPid, err := peer.Decode(pid)
	if err != nil {
		log.Error("sendPeer", "id", pid, "decode pid err", err)
		return err
	}
	stream, err := p.Host.NewStream(p.Ctx, rawPid, broadcastV1)
	if err != nil {
		log.Error("sendPeer", "id", pid, "NewStreamErr", err)
		return err
	}
	defer prototypes.CloseStream(stream)
	err = prototypes.WriteStream(broadData, stream)
	if err != nil {
		log.Error("sendPeer", "pid", pid, "WriteStream err", err)
		return err
	}
	return nil
}

// handleSend 对数据进行处理，包装成BroadCast结构
func (p *broadcastProtocol) handleSend(rawData interface{}, pid string) (sendData *types.BroadCastData, doSend bool) {
	//出错处理
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleSend_Panic", "sendData", rawData, "pid", pid, "recoverErr", r)
			doSend = false
		}
	}()
	sendData = &types.BroadCastData{}

	doSend = false
	if tx, ok := rawData.(*types.P2PTx); ok {
		doSend = p.sendTx(tx, sendData, pid)
	} else if blc, ok := rawData.(*types.P2PBlock); ok {
		doSend = p.sendBlock(blc, sendData, pid)
	} else if query, ok := rawData.(*types.P2PQueryData); ok {
		doSend = p.sendQueryData(query, sendData, pid)
	} else if rep, ok := rawData.(*types.P2PBlockTxReply); ok {
		doSend = p.sendQueryReply(rep, sendData, pid)
	}
	return
}

func (p *broadcastProtocol) handleReceive(data *types.BroadCastData, pid, peerAddr, version string) (err error) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleReceive_Panic", "recvData", data, "pid", pid, "addr", peerAddr, "recoverErr", r)
		}
	}()
	if tx := data.GetTx(); tx != nil {
		err = p.recvTx(tx, pid)
	} else if ltTx := data.GetLtTx(); ltTx != nil {
		err = p.recvLtTx(ltTx, pid, peerAddr, version)
	} else if ltBlc := data.GetLtBlock(); ltBlc != nil {
		err = p.recvLtBlock(ltBlc, pid, peerAddr, version)
	} else if blc := data.GetBlock(); blc != nil {
		err = p.recvBlock(blc, pid, peerAddr)
	} else if query := data.GetQuery(); query != nil {
		err = p.recvQueryData(query, pid, peerAddr, version)
	} else if rep := data.GetBlockRep(); rep != nil {
		err = p.recvQueryReply(rep, pid, peerAddr, version)
	}
	if err != nil {
		log.Error("handleReceive", "pid", pid, "addr", peerAddr, "recvData", data.Value, "err", err)
	}
	return
}

// Deprecated, TODO 全网升级后可删除
func (p *broadcastProtocol) postBlockChainV1(blockHash, pid string, block *types.Block) error {
	//新老广播网络存在隔离情况，将接收的老版本广播数据再次分发到新网络(重复的数据在新网络会自动屏蔽)
	p.ps.FIFOPub(block, psBlockTopic)
	return p.postBlockChain(blockHash, pid, block)
}

type sendFilterInfo struct {
	//记录广播交易或区块时需要忽略的节点, 这些节点可能是交易的来源节点,也可能节点间维护了多条连接, 冗余发送
	ignoreSendPeers map[string]bool
}

//检测是否冗余发送, 或者添加到发送过滤(内部存在直接修改读写保护的数据, 对filter lru的读写需要外层锁保护)
func addIgnoreSendPeerAtomic(filter *utils.Filterdata, key string, pid string) (exist bool) {

	filter.GetAtomicLock()
	defer filter.ReleaseAtomicLock()
	var info *sendFilterInfo
	if !filter.Contains(key) { //之前没有收到过这个key
		info = &sendFilterInfo{ignoreSendPeers: make(map[string]bool)}
		filter.Add(key, info)
	} else {
		data, _ := filter.Get(key)
		info = data.(*sendFilterInfo)
	}
	_, exist = info.ignoreSendPeers[pid]
	info.ignoreSendPeers[pid] = true
	return exist
}

// 删除发送过滤器记录
func removeIgnoreSendPeerAtomic(filter *utils.Filterdata, key, pid string) {

	filter.GetAtomicLock()
	defer filter.ReleaseAtomicLock()
	if filter.Contains(key) {
		data, _ := filter.Get(key)
		info := data.(*sendFilterInfo)
		delete(info.ignoreSendPeers, pid)
	}
}
