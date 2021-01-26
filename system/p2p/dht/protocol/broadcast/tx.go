// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/hex"

	"github.com/33cn/chain33/types"
)

func (p *broadcastProtocol) sendTx(tx *types.P2PTx, p2pData *types.BroadCastData, pid string) (doSend bool) {

	txHash := hex.EncodeToString(tx.Tx.Hash())
	ttl := tx.GetRoute().GetTTL()
	isLightSend := ttl >= p.p2pCfg.LightTxTTL

	//检测冗余发送, 已经发送或者接收过此Tx
	if addIgnoreSendPeerAtomic(p.txSendFilter, txHash, pid) {
		return false
	}

	//log.Debug("P2PSendTx", "txHash", txHash, "ttl", ttl, "peerAddr", peerAddr)
	//超过最大的ttl, 不再发送
	if ttl > p.p2pCfg.MaxTTL { //超过最大发送次数
		return false
	}

	//新版本且ttl达到设定值
	if isLightSend {
		p2pData.Value = &types.BroadCastData_LtTx{ //超过最大的ttl, 不再发送
			LtTx: &types.LightTx{
				TxHash: tx.Tx.Hash(),
				Route:  tx.GetRoute(),
			},
		}
	} else {
		p2pData.Value = &types.BroadCastData_Tx{Tx: tx} //完整Tx发送
	}
	return true
}

func (p *broadcastProtocol) recvTx(tx *types.P2PTx, pid string) (err error) {
	if tx.GetTx() == nil {
		return
	}
	txHash := hex.EncodeToString(tx.GetTx().Hash())
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(p.txSendFilter, txHash, pid)
	//避免重复接收
	if p.txFilter.AddWithCheckAtomic(txHash, true) {
		return
	}
	//log.Debug("recvTx", "tx", txHash, "ttl", tx.GetRoute().GetTTL(), "peerAddr", peerAddr)
	//有可能收到老版本的交易路由,此时route是空指针
	if tx.GetRoute() == nil {
		tx.Route = &types.P2PRoute{TTL: 1}
	}
	p.txFilter.Add(txHash, tx.GetRoute())
	return p.postMempool(txHash, tx.GetTx())

}

func (p *broadcastProtocol) recvLtTx(tx *types.LightTx, pid, peerAddr, version string) (err error) {

	txHash := hex.EncodeToString(tx.TxHash)
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(p.txSendFilter, txHash, pid)
	//存在则表示本地已经接收过此交易, 不做任何操作
	if p.txFilter.Contains(txHash) {
		return nil
	}

	//log.Debug("recvLtTx", "txHash", txHash, "ttl", tx.GetRoute().GetTTL(), "peerAddr", peerAddr)
	//本地不存在, 需要向对端节点发起完整交易请求
	query := &types.P2PQueryData{}
	query.Value = &types.P2PQueryData_TxReq{
		TxReq: &types.P2PTxReq{
			TxHash: tx.TxHash,
		},
	}
	//发布到指定的节点
	if p.sendPeer(query, pid, version) != nil {
		log.Error("recvLtTx", "pid", pid, "addr", peerAddr, "err", err)
		return errSendPeer
	}

	return nil
}
