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
	"github.com/libp2p/go-libp2p-core/peer"
)

func (protocol *broadCastProtocol) sendQueryData(query *types.P2PQueryData, p2pData *types.BroadCastData, peerAddr string) bool {
	log.Debug("P2PSendQueryData", "peerAddr", peerAddr)
	p2pData.Value = &types.BroadCastData_Query{Query: query}
	return true
}

func (protocol *broadCastProtocol) sendQueryReply(rep *types.P2PBlockTxReply, p2pData *types.BroadCastData, peerAddr string) bool {
	log.Debug("P2PSendQueryReply", "peerAddr", peerAddr)
	p2pData.Value = &types.BroadCastData_BlockRep{BlockRep: rep}
	return true
}

func (protocol *broadCastProtocol) recvQueryData(query *types.P2PQueryData, pid peer.ID, peerAddr string) error {

	var reply interface{}
	if txReq := query.GetTxReq(); txReq != nil {

		txHash := hex.EncodeToString(txReq.TxHash)
		log.Debug("recvQueryTx", "txHash", txHash, "pid", pid.Pretty())
		//向mempool请求交易
		resp, err := protocol.QueryMempool(types.EventTxListByHash, &types.ReqTxHashList{Hashes: []string{string(txReq.TxHash)}})
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

	} else if blcReq := query.GetBlockTxReq(); blcReq != nil {

		log.Debug("recvQueryBlockTx", "hash", blcReq.BlockHash, "queryCount", len(blcReq.TxIndices), "pid", pid)
		blcHash, _ := common.FromHex(blcReq.BlockHash)
		if blcHash != nil {
			resp, err := protocol.QueryBlockChain(types.EventGetBlockByHashes, &types.ReqHashes{Hashes: [][]byte{blcHash}})
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
		_, err := protocol.sendPeer(pid, reply, false)
		if err != nil {
			log.Error("recvQueryData", "pid", pid, "addr", peerAddr, "err", err)
			return errSendStream
		}
	}
	return nil
}

func (protocol *broadCastProtocol) recvQueryReply(rep *types.P2PBlockTxReply, pid peer.ID, peerAddr string) (err error) {

	log.Debug("recvQueryReply", "hash", rep.BlockHash, "queryTxsCount", len(rep.GetTxIndices()), "pid", pid.Pretty())
	val, exist := protocol.ltBlockCache.Remove(rep.BlockHash)
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
	if bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(protocol.BaseProtocol.ChainCfg, block.GetHeight(), block.Txs)) {
		log.Debug("recvQueryReply", "height", block.GetHeight())
		//发送至blockchain执行
		if err := protocol.postBlockChain(rep.BlockHash, pid, block); err != nil {
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
	protocol.ltBlockCache.Add(rep.BlockHash, block, block.Size())
	//query peer
	_, err = protocol.sendPeer(pid, query, false)
	if err != nil {
		log.Error("recvQueryReply", "pid", pid, "addr", peerAddr, "err", err)
		protocol.ltBlockCache.Remove(rep.BlockHash)
		protocol.blockFilter.Remove(rep.BlockHash)
		return errSendStream
	}
	return
}
