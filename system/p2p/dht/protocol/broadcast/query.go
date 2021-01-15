// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"bytes"
	"encoding/hex"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

func (p *broadcastProtocol) sendQueryData(query *types.P2PQueryData, p2pData *types.BroadCastData, pid string) bool {
	log.Debug("P2PSendQueryData", "pid", pid)
	p2pData.Value = &types.BroadCastData_Query{Query: query}
	return true
}

func (p *broadcastProtocol) sendQueryReply(rep *types.P2PBlockTxReply, p2pData *types.BroadCastData, pid string) bool {
	log.Debug("P2PSendQueryReply", "pid", pid)
	p2pData.Value = &types.BroadCastData_BlockRep{BlockRep: rep}
	return true
}

func (p *broadcastProtocol) recvQueryData(query *types.P2PQueryData, pid, peerAddr, version string) error {

	var reply interface{}
	if txReq := query.GetTxReq(); txReq != nil {

		txHash := hex.EncodeToString(txReq.TxHash)
		log.Debug("recvQueryTx", "txHash", txHash, "pid", pid)
		//向mempool请求交易
		resp, err := p.P2PEnv.QueryModule("mempool", types.EventTxListByHash, &types.ReqTxHashList{Hashes: []string{string(txReq.TxHash)}})
		if err != nil {
			log.Error("recvQuery", "queryMempoolErr", err)
			return errQueryMempool
		}

		txList, _ := resp.(*types.ReplyTxList)
		//返回的数据检测
		if len(txList.GetTxs()) != 1 || txList.GetTxs()[0] == nil {
			log.Error("recvQueryTx", "txHash", txHash, "err", "recvNilTxFromMempool")
			return errRecvMempool
		}
		p2pTx := &types.P2PTx{Tx: txList.Txs[0]}
		//再次发送完整交易至节点, ttl重设为1
		p2pTx.Route = &types.P2PRoute{TTL: 1}
		reply = p2pTx
		// 轻广播交易需要再次发送交易，删除发送过滤相关记录，确保不被拦截
		removeIgnoreSendPeerAtomic(p.txSendFilter, txHash, pid)

	} else if blcReq := query.GetBlockTxReq(); blcReq != nil {

		log.Debug("recvQueryBlockTx", "hash", blcReq.BlockHash, "queryCount", len(blcReq.TxIndices), "pid", pid)
		blcHash, _ := common.FromHex(blcReq.BlockHash)
		if blcHash != nil {
			resp, err := p.P2PEnv.QueryModule("blockchain", types.EventGetBlockByHashes, &types.ReqHashes{Hashes: [][]byte{blcHash}})
			if err != nil {
				log.Error("recvQueryBlockTx", "queryBlockChainErr", err)
				return errQueryBlockChain
			}
			blocks, ok := resp.(*types.BlockDetails)
			if !ok || len(blocks.Items) != 1 || blocks.Items[0].Block == nil {
				log.Error("recvQueryBlockTx", "blockHash", blcReq.BlockHash, "err", "blockNotExist")
				return errRecvBlockChain
			}
			block := blocks.Items[0].Block
			blockRep := &types.P2PBlockTxReply{BlockHash: blcReq.BlockHash}
			blockRep.TxIndices = blcReq.TxIndices
			for _, idx := range blcReq.TxIndices {
				blockRep.Txs = append(blockRep.Txs, block.Txs[idx])
			}
			//请求所有的交易
			if len(blockRep.TxIndices) == 0 {
				blockRep.Txs = block.Txs
			}
			reply = blockRep
		}
	}

	if reply != nil {
		if err := p.sendPeer(reply, pid, version); err != nil {
			log.Error("recvQueryData", "pid", pid, "addr", peerAddr, "err", err)
			return errSendPeer
		}
	}
	return nil
}

func (p *broadcastProtocol) recvQueryReply(rep *types.P2PBlockTxReply, pid, peerAddr, version string) (err error) {

	log.Debug("recvQueryReply", "hash", rep.BlockHash, "queryTxsCount", len(rep.GetTxIndices()), "pid", pid)
	val, exist := p.ltBlockCache.Remove(rep.BlockHash)
	block, _ := val.(*types.Block)
	//not exist in cache or nil block
	if !exist || block == nil {
		log.Error("recvQueryReply", "hash", rep.BlockHash, "exist", exist, "isBlockNil", block == nil)
		return errLtBlockNotExist
	}
	for i, idx := range rep.TxIndices {
		block.Txs[idx] = rep.Txs[i]
	}

	//所有交易覆盖
	if len(rep.TxIndices) == 0 {
		block.Txs = rep.Txs
	}

	//计算的root hash是否一致
	if bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(p.ChainCfg, block.GetHeight(), block.Txs)) {
		log.Debug("recvQueryReply", "height", block.GetHeight())
		//发送至blockchain执行
		if err := p.postBlockChain(rep.BlockHash, pid, block); err != nil {
			log.Error("recvQueryReply", "height", block.GetHeight(), "send block to blockchain Error", err.Error())
			return errSendBlockChain
		}
		return nil
	}

	// 区块校验仍然不通过，则尝试向对端请求整个区块 ， txIndices空表示请求整个区块, 已请求过不再重复请求
	if len(rep.TxIndices) == 0 {
		log.Error("recvQueryReply", "height", block.GetHeight(), "hash", rep.BlockHash, "err", errBuildBlockFailed)
		return errBuildBlockFailed
	}

	log.Debug("recvQueryReply", "getBlockRetry", block.GetHeight(), "hash", rep.BlockHash)

	query := &types.P2PQueryData{
		Value: &types.P2PQueryData_BlockTxReq{
			BlockTxReq: &types.P2PBlockTxReq{
				BlockHash: rep.BlockHash,
				TxIndices: nil,
			},
		},
	}
	block.Txs = nil
	p.ltBlockCache.Add(rep.BlockHash, block, block.Size())
	//query peer
	if err = p.sendPeer(query, pid, version); err != nil {
		log.Error("recvQueryReply", "pid", pid, "addr", peerAddr, "err", err)
		p.ltBlockCache.Remove(rep.BlockHash)
		p.blockFilter.Remove(rep.BlockHash)
		return errSendPeer
	}
	return
}
