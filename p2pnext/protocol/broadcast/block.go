package broadcast

import (
	"bytes"
	"encoding/hex"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

func (s *broadCastProtocol) sendBlock(block *types.P2PBlock, p2pData *types.BroadCastData, pid, peerAddr string) (doSend bool) {
	byteHash := block.Block.Hash(s.GetChainCfg())
	blockHash := hex.EncodeToString(byteHash)
	//检测冗余发送
	ignoreSend := addIgnoreSendPeerAtomic(s.blockSendFilter, blockHash, pid)
	log.Debug("P2PSendBlock", "blockHash", blockHash, "peerAddr", peerAddr, "ignoreSend", ignoreSend)
	if ignoreSend {
		return false
	}

	if len(block.Block.Txs) < 1 {
		return false
	}
	//首先发起liteBlock broadcast
	ltBlock := &types.LightBlock{}
	ltBlock.Size = int64(types.Size(block.Block))
	ltBlock.Header = block.Block.GetHeader(s.GetChainCfg())
	ltBlock.Header.Hash = byteHash[:]
	ltBlock.Header.Signature = block.Block.Signature
	ltBlock.MinerTx = block.Block.Txs[0]
	for _, tx := range block.Block.Txs[1:] {
		//tx short hash
		ltBlock.STxHashes = append(ltBlock.STxHashes, types.CalcTxShortHash(tx.Hash()))
	}

	// cache block
	if !s.totalBlockCache.Contains(blockHash) {
		s.totalBlockCache.Add(blockHash, block.Block, ltBlock.Size)
	}

	p2pData.Value = &types.BroadCastData_LtBlock{LtBlock: ltBlock}

	return true
}

func (s *broadCastProtocol) recvBlock(block *types.P2PBlock, pid, peerAddr string) {

	if block.GetBlock() == nil {
		return
	}
	blockHash := hex.EncodeToString(block.GetBlock().Hash(s.GetChainCfg()))
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(s.blockSendFilter, blockHash, pid)
	//如果重复接收, 则不再发到blockchain执行
	isDuplicate := checkAndRegFilterAtomic(s.blockFilter, blockHash)
	log.Debug("recvBlock", "blockHeight", block.GetBlock().GetHeight(), "peerAddr", peerAddr,
		"block size(KB)", float32(block.Block.Size())/1024, "blockHash", blockHash, "duplicateBlock", isDuplicate)
	if isDuplicate {
		return
	}
	//发送至blockchain执行
	if err := s.postBlockChain(block.GetBlock(), pid); err != nil {
		log.Error("recvBlock", "send block to blockchain Error", err.Error())
	}

}

func (s *broadCastProtocol) recvLtBlock(ltBlock *types.LightBlock, pid, peerAddr string) {

	blockHash := hex.EncodeToString(ltBlock.Header.Hash)
	//将节点id添加到发送过滤, 避免冗余发送
	addIgnoreSendPeerAtomic(s.blockSendFilter, blockHash, pid)
	//检测是否已经收到此block
	isDuplicate := checkAndRegFilterAtomic(s.blockFilter, blockHash)
	log.Debug("recvLtBlock", "blockHash", blockHash, "blockHeight", ltBlock.GetHeader().GetHeight(),
		"peerAddr", peerAddr, "duplicateBlock", isDuplicate)
	if isDuplicate {
		return
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
		resp, err := s.sendToMempool(types.EventTxListByHash,
			&types.ReqTxHashList{Hashes: ltBlock.STxHashes, IsShortHash: true})
		if err != nil {
			log.Error("recvLtBlock", "queryTxListByHashErr", err)
			return
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
			block.Txs = append(block.Txs, tx)
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
		}

	}
	nilTxLen := len(nilTxIndices)
	//需要比较交易根哈希是否一致, 不一致需要请求区块内所有的交易
	if nilTxLen == 0 && len(block.Txs) == int(ltBlock.Header.TxCount) &&
		bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(block.Txs)) {

		log.Debug("recvLtBlock", "height", block.GetHeight(), "peerAddr", peerAddr,
			"blockHash", blockHash, "block size(KB)", float32(ltBlock.Size)/1024)
		//发送至blockchain执行
		if err := s.postBlockChain(block, pid); err != nil {
			log.Error("recvLtBlock", "send block to blockchain Error", err.Error())
		}

		return
	}
	// 缺失的交易个数大于总数1/3 或者缺失数据大小大于2/3, 触发请求区块所有交易数据
	if nilTxLen > 0 && (float32(nilTxLen) > float32(ltBlock.Header.TxCount)/3 ||
		float32(block.Size()) < float32(ltBlock.Size)/3) {
		nilTxIndices = nilTxIndices[:0]
	}
	log.Debug("recvLtBlock", "queryBlockHash", blockHash,
		"queryHeight", ltBlock.GetHeader().GetHeight(), "queryTxNum", len(nilTxIndices))

	// query not exist txs
	query := &types.P2PQueryData{
		Value: &types.P2PQueryData_BlockTxReq{
			BlockTxReq: &types.P2PBlockTxReq{
				BlockHash: blockHash,
				TxIndices: nilTxIndices,
			},
		},
	}
	//pub to specified peer
	s.queryStream(pid, query)
	//需要将不完整的block预存
	s.ltBlockCache.Add(blockHash, block, int64(block.Size()))
}
