// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"bytes"
	"encoding/hex"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

func (protocol *broadCastProtocol) sendBlock(block *types.P2PBlock, p2pData *types.BroadCastData, pid peer.ID, peerAddr string) (doSend bool) {
	byteHash := block.Block.Hash(protocol.GetChainCfg())
	blockHash := hex.EncodeToString(byteHash)
	//检测冗余发送
	if addIgnoreSendPeerAtomic(protocol.blockSendFilter, blockHash, pid) {
		return false
	}
	blockSize := types.Size(block.Block)
	//log.Debug("P2PSendBlock", "blockHash", blockHash, "peerAddr", peerAddr, "blockSize(KB)", float32(blockSize)/1024)
	//区块内交易采用哈希广播
	if blockSize >= int(protocol.p2pCfg.MinLtBlockSize*1024) {
		ltBlock := &types.LightBlock{}
		ltBlock.Size = int64(blockSize)
		ltBlock.Header = block.Block.GetHeader(protocol.GetChainCfg())
		ltBlock.Header.Hash = byteHash[:]
		ltBlock.Header.Signature = block.Block.Signature
		ltBlock.MinerTx = block.Block.Txs[0]
		for _, tx := range block.Block.Txs[1:] {
			//tx short hash
			ltBlock.STxHashes = append(ltBlock.STxHashes, types.CalcTxShortHash(tx.Hash()))
		}

		p2pData.Value = &types.BroadCastData_LtBlock{LtBlock: ltBlock}
	} else {
		p2pData.Value = &types.BroadCastData_Block{Block: block}
	}

	return true
}

func (protocol *broadCastProtocol) recvBlock(block *types.P2PBlock, pid peer.ID, peerAddr string) error {

	if block.GetBlock() == nil {
		return types.ErrInvalidParam
	}
	blockHash := hex.EncodeToString(block.GetBlock().Hash(protocol.GetChainCfg()))
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(protocol.blockSendFilter, blockHash, pid)
	//如果重复接收, 则不再发到blockchain执行
	if protocol.blockFilter.AddWithCheckAtomic(blockHash, true) {
		return nil
	}
	log.Debug("recvBlock", "height", block.GetBlock().GetHeight(), "size(KB)", float32(types.Size(block.GetBlock()))/1024)
	//发送至blockchain执行
	if err := protocol.postBlockChain(blockHash, pid, block.GetBlock()); err != nil {
		log.Error("recvBlock", "send block to blockchain Error", err.Error())
		return errSendBlockChain
	}
	return nil
}

func (protocol *broadCastProtocol) recvLtBlock(ltBlock *types.LightBlock, pid peer.ID, peerAddr string) error {

	blockHash := hex.EncodeToString(ltBlock.Header.Hash)
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(protocol.blockSendFilter, blockHash, pid)
	//检测是否已经收到此block
	if protocol.blockFilter.AddWithCheckAtomic(blockHash, true) {
		return nil
	}

	//组装block
	block := &types.Block{}
	block.TxHash = ltBlock.Header.TxHash
	block.Signature = ltBlock.Header.Signature
	block.ParentHash = ltBlock.Header.ParentHash
	block.Height = ltBlock.Header.Height
	block.BlockTime = ltBlock.Header.BlockTime
	block.Difficulty = ltBlock.Header.Difficulty
	block.Version = ltBlock.Header.Version
	block.StateHash = ltBlock.Header.StateHash
	//add miner tx
	block.Txs = append(block.Txs, ltBlock.MinerTx)

	txList := &types.ReplyTxList{}
	ok := false
	//get tx list from mempool
	if len(ltBlock.STxHashes) > 0 {
		resp, err := protocol.QueryMempool(types.EventTxListByHash,
			&types.ReqTxHashList{Hashes: ltBlock.STxHashes, IsShortHash: true})
		if err != nil {
			log.Error("recvLtBlock", "queryTxListByHashErr", err)
			return errRecvMempool
		}

		txList, ok = resp.(*types.ReplyTxList)
		if !ok {
			log.Error("recvLtBlock", "queryMemPool", "nilReplyTxList")
		}
	}
	nilTxIndices := make([]int32, 0)
	for i := 0; ok && i < len(txList.Txs); i++ {
		tx := txList.Txs[i]
		if tx == nil {
			//tx not exist in mempool
			nilTxIndices = append(nilTxIndices, int32(i+1))
			tx = &types.Transaction{}
		} else if count := tx.GetGroupCount(); count > 0 {

			group, err := tx.GetTxGroup()
			if err != nil {
				log.Error("recvLtBlock", "getTxGroupErr", err)
				//触发请求所有
				nilTxIndices = nilTxIndices[:0]
				break
			}
			block.Txs = append(block.Txs, group.Txs...)
			//跳过遍历
			i += len(group.Txs) - 1
			continue
		}
		block.Txs = append(block.Txs, tx)
	}
	nilTxLen := len(nilTxIndices)

	//需要比较交易根哈希是否一致, 不一致需要请求区块内所有的交易
	if nilTxLen == 0 && bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(protocol.BaseProtocol.ChainCfg, block.GetHeight(), block.Txs)) {

		log.Debug("recvLtBlock", "height", block.GetHeight(), "txCount", ltBlock.Header.TxCount, "size(KB)", float32(ltBlock.Size)/1024)
		//发送至blockchain执行
		if err := protocol.postBlockChain(blockHash, pid, block); err != nil {
			log.Error("recvLtBlock", "send block to blockchain Error", err.Error())
			return errSendBlockChain
		}
		return nil
	}
	//本地缺失交易或者根哈希不同(nilTxLen==0)
	log.Debug("recvLtBlock", "height", ltBlock.Header.Height, "hash", blockHash,
		"txCount", ltBlock.Header.TxCount, "missTxCount", nilTxLen,
		"blockSize(KB)", float32(ltBlock.Size)/1024, "buildBlockSize(KB)", float32(block.Size())/1024)
	// 缺失的交易个数大于总数1/3 或者缺失数据大小大于2/3, 触发请求区块所有交易数据
	if nilTxLen > 0 && (float32(nilTxLen) > float32(ltBlock.Header.TxCount)/3 ||
		float32(block.Size()) < float32(ltBlock.Size)/3) {
		//空的TxIndices表示请求区块内所有交易
		nilTxIndices = nilTxIndices[:0]
	}

	// query not exist txs
	query := &types.P2PQueryData{
		Value: &types.P2PQueryData_BlockTxReq{
			BlockTxReq: &types.P2PBlockTxReq{
				BlockHash: blockHash,
				TxIndices: nilTxIndices,
			},
		},
	}

	//需要将不完整的block预存
	protocol.ltBlockCache.Add(blockHash, block, block.Size())
	//pub to specified peer
	_, err := protocol.sendPeer(pid, query, false)
	if err != nil {
		log.Error("recvLtBlock", "pid", pid, "addr", peerAddr, "err", err)
		protocol.blockFilter.Remove(blockHash)
		protocol.ltBlockCache.Remove(blockHash)
		return errSendStream
	}
	return nil
}
