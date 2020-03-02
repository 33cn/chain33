package broadcast

import (
	"bytes"
	"encoding/hex"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

func (s *broadCastProtocol) sendQueryData(query *types.P2PQueryData, p2pData *types.BroadCastData, peerAddr string) bool {
	log.Debug("P2PSendQueryData", "peerAddr", peerAddr)
	p2pData.Value = &types.BroadCastData_Query{Query: query}
	return true
}

func (s *broadCastProtocol) sendQueryReply(rep *types.P2PBlockTxReply, p2pData *types.BroadCastData, peerAddr string) bool {
	log.Debug("P2PSendQueryReply", "peerAddr", peerAddr)
	p2pData.Value = &types.BroadCastData_BlockRep{BlockRep: rep}
	return true
}

func (s *broadCastProtocol) recvQueryData(query *types.P2PQueryData, pid, peerAddr string) {

	if txReq := query.GetTxReq(); txReq != nil {

		txHash := hex.EncodeToString(txReq.TxHash)
		log.Debug("recvQueryTx", "txHash", txHash, "peerAddr", peerAddr)
		//向mempool请求交易
		resp, err := s.sendToMempool(types.EventTxListByHash, &types.ReqTxHashList{Hashes: []string{string(txReq.TxHash)}})
		if err != nil {
			log.Error("recvQuery", "queryMempoolErr", err)
			return
		}

		p2pTx := &types.P2PTx{}
		txList, ok := resp.(*types.ReplyTxList)
		//返回的交易数组实际大小只有一个, for循环能省略空值等情况判断
		for i := 0; ok && i < len(txList.Txs); i++ {
			p2pTx.Tx = txList.Txs[i]
		}

		if p2pTx.GetTx() == nil {
			log.Error("recvQueryTx", "txHash", txHash, "err", "recvNilTxFromMempool")
			return
		}
		//再次发送完整交易至节点, ttl重设为1
		p2pTx.Route = &types.P2PRoute{TTL: 1}
		s.sendStream(pid, p2pTx)

	} else if blcReq := query.GetBlockTxReq(); blcReq != nil {

		log.Debug("recvQueryBlockTx", "blockHash", blcReq.BlockHash, "queryTxCount", len(blcReq.TxIndices), "peerAddr", peerAddr)
		if block, ok := s.totalBlockCache.Get(blcReq.BlockHash).(*types.Block); ok {

			blockRep := &types.P2PBlockTxReply{BlockHash: blcReq.BlockHash}

			blockRep.TxIndices = blcReq.TxIndices
			for _, idx := range blcReq.TxIndices {
				blockRep.Txs = append(blockRep.Txs, block.Txs[idx])
			}
			//请求所有的交易
			if len(blockRep.TxIndices) == 0 {
				blockRep.Txs = block.Txs
			}
			s.sendStream(pid, blockRep)
		}
	}
}

func (s *broadCastProtocol) recvQueryReply(rep *types.P2PBlockTxReply, pid, peerAddr string) {

	log.Debug("recvQueryReplyBlock", "blockHash", rep.GetBlockHash(), "queryTxsCount", len(rep.GetTxIndices()), "peerAddr", peerAddr)
	val, exist := s.ltBlockCache.Del(rep.BlockHash)
	block, _ := val.(*types.Block)
	//not exist in cache or nil block
	if !exist || block == nil {
		return
	}
	for i, idx := range rep.TxIndices {
		block.Txs[idx] = rep.Txs[i]
	}

	//所有交易覆盖
	if len(rep.TxIndices) == 0 {
		block.Txs = rep.Txs
	}

	//计算的root hash是否一致
	if bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(s.BaseProtocol.ChainCfg, block.GetHeight(), block.Txs)) {

		log.Debug("recvQueryReplyBlock", "blockHeight", block.GetHeight(), "peerAddr", peerAddr,
			"block size(KB)", float32(block.Size())/1024, "blockHash", rep.BlockHash)
		//发送至blockchain执行
		if err := s.postBlockChain(rep.BlockHash, pid, block); err != nil {
			log.Error("recvQueryReplyBlock", "send block to blockchain Error", err.Error())
		}
	} else if len(rep.TxIndices) != 0 {
		log.Debug("recvQueryReplyBlock", "GetTotalBlock", block.GetHeight())
		//不一致尝试请求整个区块的交易, 且判定是否已经请求过完整交易
		query := &types.P2PQueryData{
			Value: &types.P2PQueryData_BlockTxReq{
				BlockTxReq: &types.P2PBlockTxReq{
					BlockHash: rep.BlockHash,
					TxIndices: nil,
				},
			},
		}
		//pub to specified peer
		s.sendStream(pid, query)
		block.Txs = nil
		s.ltBlockCache.Add(rep.BlockHash, block, int64(block.Size()))
	}
}
