// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"encoding/hex"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/types"
)

func (protocol *broadCastProtocol) sendTx(tx *types.P2PTx, p2pData *types.BroadCastData, pid peer.ID, peerAddr string) (doSend bool) {

	txHash := hex.EncodeToString(tx.Tx.Hash())
	ttl := tx.GetRoute().GetTTL()
	isLightSend := ttl >= protocol.p2pCfg.LightTxTTL

	//检测冗余发送, 短哈希广播不记录发送过滤, 已经发送或者接收过此Tx
	if !isLightSend && addIgnoreSendPeerAtomic(protocol.txSendFilter, txHash, pid) {
		return false
	}

	//log.Debug("P2PSendTx", "txHash", txHash, "ttl", ttl, "peerAddr", peerAddr)
	//超过最大的ttl, 不再发送
	if ttl > protocol.p2pCfg.MaxTTL { //超过最大发送次数
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

func (protocol *broadCastProtocol) recvTx(tx *types.P2PTx, pid peer.ID, peerAddr string) (err error) {
	if tx.GetTx() == nil {
		return
	}
	txHash := hex.EncodeToString(tx.GetTx().Hash())
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(protocol.txSendFilter, txHash, pid)
	//避免重复接收
	if protocol.txFilter.AddWithCheckAtomic(txHash, true) {
		return
	}
	//log.Debug("recvTx", "tx", txHash, "ttl", tx.GetRoute().GetTTL(), "peerAddr", peerAddr)
	//有可能收到老版本的交易路由,此时route是空指针
	if tx.GetRoute() == nil {
		tx.Route = &types.P2PRoute{TTL: 1}
	}
	protocol.txFilter.Add(txHash, tx.GetRoute())
	return protocol.postMempool(txHash, tx.GetTx())

}

func (protocol *broadCastProtocol) recvLtTx(tx *types.LightTx, pid peer.ID, peerAddr string) (err error) {

	txHash := hex.EncodeToString(tx.TxHash)
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(protocol.txSendFilter, txHash, pid)
	//存在则表示本地已经接收过此交易, 不做任何操作
	if protocol.txFilter.Contains(txHash) {
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
	_, err = protocol.sendPeer(pid, query, false)
	if err != nil {
		log.Error("recvLtTx", "pid", pid, "addr", peerAddr, "err", err)
		return errSendStream
	}

	return nil
}
