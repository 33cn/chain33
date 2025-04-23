// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	log = log15.New("module", "p2p.manage")
)

/** 处理系统发送至p2p模块的事件, 包含下载, 交易及区块广播等
 * 主要是为了兼容多种类型p2p, 为不同的事件指定路由策略
 */
func (mgr *Manager) handleSysEvent() {
	mgr.Client.Sub("p2p")
	log.Debug("Manager handleSysEvent start")
	for msg := range mgr.Client.Recv() {
		switch msg.Ty {

		case types.EventTxBroadcast, types.EventBlockBroadcast: //广播
			mgr.pub2All(msg)
		case types.EventFetchBlocks, types.EventGetMempool, types.EventFetchBlockHeaders:
			mgr.pub2P2P(msg, mgr.p2pCfg.Types[0])
		case types.EventPeerInfo:
			// 采用默认配置
			p2pTy := mgr.p2pCfg.Types[0]
			req, _ := msg.Data.(*types.P2PGetPeerReq)
			for _, ty := range mgr.p2pCfg.Types {
				if ty == req.GetP2PType() {
					p2pTy = req.GetP2PType()
				}
			}
			mgr.pub2P2P(msg, p2pTy)

		case types.EventGetNetInfo:
			// 采用默认配置
			p2pTy := mgr.p2pCfg.Types[0]
			req, _ := msg.Data.(*types.P2PGetNetInfoReq)
			for _, ty := range mgr.p2pCfg.Types {
				if ty == req.GetP2PType() {
					p2pTy = req.GetP2PType()
				}
			}
			mgr.pub2P2P(msg, p2pTy)

		case types.EventPubTopicMsg, types.EventFetchTopics, types.EventRemoveTopic, types.EventSubTopic, types.EventAddBlacklist, types.EventDelBlacklist, types.EventShowBlacklist:
			p2pTy := mgr.p2pCfg.Types[0]
			mgr.pub2P2P(msg, p2pTy)

		default:
			mgr.pub2P2P(msg, "dht")
			//log.Warn("unknown msgtype", "msg", msg)
			//msg.Reply(mgr.Client.NewMessage("", msg.Ty, types.Reply{Msg: []byte("unknown msgtype")}))
			//continue
		}
	}
	log.Debug("Manager handleSysEvent stop")
}

// 处理p2p内部向外发送的消息, 主要是为了兼容多种类型p2p广播消息, 避免重复接交易或者区块
func (mgr *Manager) handleP2PSub() {

	//mgr.subChan = mgr.PubSub.Sub("p2p")
	log.Debug("Manager handleP2PSub start")
	//for msg := range mgr.subChan {
	//
	//}

}

// PubBroadCast 兼容多种类型p2p广播消息, 避免重复接交易或者区块
func (mgr *Manager) PubBroadCast(hash string, data interface{}, eventTy int) (*queue.Message, error) {

	if len(mgr.p2pCfg.Types) > 1 {
		exist, _ := mgr.broadcastFilter.ContainsOrAdd(hash, struct{}{})
		if exist {
			return nil, nil
		}
	}

	var err error
	var msg *queue.Message
	if eventTy == types.EventTx {
		//同步模式发送交易，但不需要进行等待回复，目的是为了在消息队列内部使用高速模式
		msg = mgr.Client.NewMessage("mempool", types.EventTx, data)
		err = mgr.Client.Send(msg, true)
	} else if eventTy == types.EventBroadcastAddBlock {
		msg = mgr.Client.NewMessage("blockchain", types.EventBroadcastAddBlock, data)
		err = mgr.Client.Send(msg, true)
	}
	if err != nil {
		log.Error("PubBroadCast", "eventTy", eventTy, "sendMsgErr", err)
		return nil, err
	}
	return msg, nil
}

func (mgr *Manager) pub2All(msg *queue.Message) {

	for _, ty := range mgr.p2pCfg.Types {
		mgr.PubSub.Pub(msg, ty)
	}

}

func (mgr *Manager) pub2P2P(msg *queue.Message, p2pType string) {

	mgr.PubSub.Pub(msg, p2pType)
}
