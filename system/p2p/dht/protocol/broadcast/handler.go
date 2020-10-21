package broadcast

import (
	"encoding/hex"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/network"
)

func (p *Protocol) handleStreamOld(stream network.Stream) {
	var oldData types.MessageBroadCast
	err := protocol.ReadStream(&oldData, stream)
	if err != nil {
		log.Error("handleStreamBroadcastTx", "read stream error", err)
		return
	}
	msg := oldData.Message
	if msg.GetTx() != nil {
		tx := msg.GetTx().Tx
		txHash := hex.EncodeToString(tx.Hash())
		//将节点id添加到发送过滤, 避免冗余发送
		addIgnoreSendPeerAtomic(p.txSendFilter, txHash, stream.Conn().RemotePeer())
		//避免重复接收
		if p.txFilter.AddWithCheckAtomic(txHash, struct{}{}) {
			return
		}
		err = p.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
		if err != nil {
			log.Error("handleStreamBroadcastTx", "PubBroadCast error", err)
		}
	} else if msg.GetBlock() != nil {
		block := msg.GetBlock().Block
		pid := stream.Conn().RemotePeer()
		blockHash := hex.EncodeToString(block.Hash(p.ChainCfg))
		//将节点id添加到发送过滤, 避免冗余发送
		addIgnoreSendPeerAtomic(p.blockSendFilter, blockHash, pid)
		//如果重复接收, 则不再发到blockchain执行
		if p.blockFilter.AddWithCheckAtomic(blockHash, true) {
			return
		}
		log.Debug("recvBlock", "height", block.GetHeight(), "size(KB)", float32(types.Size(block))/1024)
		//发送至blockchain执行
		if err := p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid.Pretty(), Block: block}, types.EventBroadcastAddBlock); err != nil {
			log.Error("handleStreamBroadcastBlock", "send block to blockchain error", err)
			return
		}
	}
}

//func (p *Protocol) handleStreamBroadcastTx(stream network.Stream) {
//	var data types.Transactions
//	err := protocol.ReadStream(&data, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastTx", "read stream error", err)
//		return
//	}
//	pid := stream.Conn().RemotePeer()
//	for _, tx := range data.Txs {
//		txHash := hex.EncodeToString(tx.Hash())
//		//将节点id添加到发送过滤, 避免冗余发送
//		addIgnoreSendPeerAtomic(p.txSendFilter, txHash, pid)
//		//避免重复接收
//		if p.txFilter.AddWithCheckAtomic(txHash, struct{}{}) {
//			return
//		}
//		err = p.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
//		if err != nil {
//			log.Error("handleStreamBroadcastTx", "PubBroadCast error", err)
//		}
//	}
//}

//func (p *Protocol) handleStreamBroadcastLightTx(stream network.Stream) {
//	var data types.BroadCastData
//	err := protocol.ReadStream(&data, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastTx", "read stream error", err)
//		return
//	}
//	txHash := hex.EncodeToString(data.GetLtTx().TxHash)
//	pid := stream.Conn().RemotePeer()
//	//将节点id添加到发送过滤, 避免冗余发送
//	addIgnoreSendPeerAtomic(p.txSendFilter, txHash, pid)
//	//存在则表示本地已经接收过此交易, 不做任何操作
//	if p.txFilter.Contains(txHash) {
//		return
//	}
//
//	//本地不存在, 需要向对端节点发起完整交易请求
//	err = protocol.WriteStream(&types.HashList{
//		Hashes: [][]byte{data.GetLtTx().TxHash},
//	}, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastLightTx", "write stream error", err)
//		return
//	}
//	var resp types.Transaction
//	err = protocol.ReadStream(&resp, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastLightTx", "read stream error", err)
//		return
//	}
//	p.txFilter.Add(txHash, struct{}{})
//	err = p.P2PManager.PubBroadCast(txHash, &resp, types.EventTx)
//	if err != nil {
//		log.Error("handleStreamBroadcastTx", "PubBroadCast error", err)
//	}
//}

//func (p *Protocol) handleStreamBroadcastBlock(stream network.Stream) {
//	var block types.Block
//	err := protocol.ReadStream(&block, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastBlock", "read stream error", err)
//		return
//	}
//
//	pid := stream.Conn().RemotePeer()
//	blockHash := hex.EncodeToString(block.Hash(p.ChainCfg))
//	//将节点id添加到发送过滤, 避免冗余发送
//	addIgnoreSendPeerAtomic(p.blockSendFilter, blockHash, pid)
//	//如果重复接收, 则不再发到blockchain执行
//	if p.blockFilter.AddWithCheckAtomic(blockHash, true) {
//		return
//	}
//	log.Debug("recvBlock", "height", block.GetHeight(), "size(KB)", float32(types.Size(&block))/1024)
//	//发送至blockchain执行
//	if err := p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid.Pretty(), Block: &block}, types.EventBroadcastAddBlock); err != nil {
//		log.Error("handleStreamBroadcastBlock", "send block to blockchain error", err)
//		return
//	}
//}

//func (p *Protocol) handleStreamBroadcastLtBlock(stream network.Stream) {
//	var ltBlock types.LightBlock
//	err := protocol.ReadStream(&ltBlock, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastBlock", "read stream error", err)
//		return
//	}
//
//	pid := stream.Conn().RemotePeer()
//	blockHash := hex.EncodeToString(ltBlock.Header.Hash)
//	//将节点id添加到发送过滤, 避免冗余发送
//	addIgnoreSendPeerAtomic(p.blockSendFilter, blockHash, pid)
//	//如果重复接收, 则不再发到blockchain执行
//	if p.blockFilter.AddWithCheckAtomic(blockHash, true) {
//		return
//	}
//	//组装block
//	block := types.Block{
//		TxHash:     ltBlock.Header.TxHash,
//		Signature:  ltBlock.Header.Signature,
//		ParentHash: ltBlock.Header.ParentHash,
//		Height:     ltBlock.Header.Height,
//		BlockTime:  ltBlock.Header.BlockTime,
//		Difficulty: ltBlock.Header.Difficulty,
//		Version:    ltBlock.Header.Version,
//		StateHash:  ltBlock.Header.StateHash,
//	}
//	//add miner tx
//	block.Txs = append(block.Txs, ltBlock.MinerTx)
//	if len(ltBlock.STxHashes) == 0 {
//		//发送至blockchain执行
//		if err := p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid.Pretty(), Block: &block}, types.EventBroadcastAddBlock); err != nil {
//			log.Error("handleStreamBroadcastBlock", "send block to blockchain error", err)
//		}
//		return
//	}
//
//	//get tx list from mempool
//	res, err := p.QueryModule("mempool", types.EventTxListByHash, &types.ReqTxHashList{Hashes: ltBlock.STxHashes, IsShortHash: true})
//	if err != nil {
//		log.Error("handleStreamBroadcastLtBlock", "query mempool error", err)
//		return
//	}
//
//	txList, ok := res.(*types.ReplyTxList)
//	if !ok {
//		log.Error("handleStreamBroadcastLtBlock", "queryMemPool", "nilReplyTxList")
//		return
//	}
//	var nilTxIndices []int32
//	var i int
//	for _, tx := range txList.Txs {
//		if tx == nil {
//			//tx not exist in mempool
//			nilTxIndices = append(nilTxIndices, int32(i+1))
//			block.Txs = append(block.Txs, nil)
//			i++
//		} else if count := tx.GetGroupCount(); count > 0 {
//			group, err := tx.GetTxGroup()
//			if err != nil {
//				log.Error("handleStreamBroadcastLtBlock", "getTxGroupErr", err, "block hash", blockHash)
//				//触发请求所有
//				nilTxIndices = nil
//				break
//			}
//			block.Txs = append(block.Txs, group.Txs...)
//			//跳过遍历
//			i += len(group.Txs)
//		} else {
//			block.Txs = append(block.Txs, tx)
//			i++
//		}
//	}
//	nilTxLen := len(nilTxIndices)
//
//	//需要比较交易根哈希是否一致, 不一致需要请求区块内所有的交易
//	if nilTxLen == 0 && bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(p.ChainCfg, block.GetHeight(), block.Txs)) {
//		log.Debug("handleStreamBroadcastLtBlock", "height", block.GetHeight(), "txCount", ltBlock.Header.TxCount, "size(KB)", float32(ltBlock.Size)/1024)
//		//发送至blockchain执行
//		if err := p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid.Pretty(), Block: &block}, types.EventBroadcastAddBlock); err != nil {
//			log.Error("handleStreamBroadcastBlock", "send block to blockchain error", err)
//		}
//		return
//	}
//	//本地缺失交易或者根哈希不同(nilTxLen==0)
//	log.Debug("handleStreamBroadcastLtBlock", "height", ltBlock.Header.Height, "hash", blockHash,
//		"txCount", ltBlock.Header.TxCount, "missTxCount", nilTxLen,
//		"blockSize(KB)", float32(ltBlock.Size)/1024, "buildBlockSize(KB)", float32(block.Size())/1024)
//	// 缺失的交易个数大于总数1/3 或者缺失数据大小大于2/3, 触发请求区块所有交易数据
//	if nilTxLen > 0 && (int64(nilTxLen)*3 > ltBlock.Header.TxCount) || (int64(block.Size())*3 < ltBlock.Size) {
//		//空的TxIndices表示请求区块内所有交易
//		nilTxIndices = nil
//	}
//
//	// query not exist txs
//	req := types.P2PBlockTxReq{
//		BlockHash: blockHash,
//		TxIndices: nilTxIndices,
//	}
//	err = protocol.WriteStream(&req, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastLtBlock", "write stream error", err)
//		return
//	}
//	var reply types.P2PBlockTxReply
//	err = protocol.ReadStream(&reply, stream)
//	if err != nil {
//		log.Error("handleStreamBroadcastLtBlock", "read stream error", err)
//		return
//	}
//
//	for i, idx := range reply.TxIndices {
//		block.Txs[idx] = reply.Txs[i]
//	}
//
//	//所有交易覆盖
//	if len(reply.TxIndices) == 0 {
//		block.Txs = reply.Txs
//	}
//
//	if bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(p.ChainCfg, block.GetHeight(), block.Txs)) {
//		log.Debug("handleStreamBroadcastLtBlock", "height", block.GetHeight(), "txCount", ltBlock.Header.TxCount, "size(KB)", float32(ltBlock.Size)/1024)
//		//发送至blockchain执行
//		if err := p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid.Pretty(), Block: &block}, types.EventBroadcastAddBlock); err != nil {
//			log.Error("handleStreamBroadcastBlock", "send block to blockchain error", err)
//		}
//		return
//	}
//}

func (p *Protocol) handleEventBroadcastTx(m *queue.Message) {
	tx := m.Data.(*types.Transaction)
	hash := hex.EncodeToString(tx.Hash())
	// pub sub只需要转发本节点产生的交易或区块
	if !p.txFilter.Contains(hash) {
		p.txFilter.Add(hash, struct{}{})
		p.ps.FIFOPub(tx, psTxTopic)
	}
	p.txQueue <- tx
}

func (p *Protocol) handleEventBroadcastBlock(m *queue.Message) {
	block := m.Data.(*types.Block)
	hash := hex.EncodeToString(block.Hash(p.ChainCfg))
	// pub sub只需要转发本节点产生的交易或区块
	if !p.blockFilter.Contains(hash) {
		p.blockFilter.Add(hash, struct{}{})
		p.ps.FIFOPub(block, psBlockTopic)
	}
	p.blockQueue <- block
}

//func (p *Protocol) handleEventBroadcastTxOld(m *queue.Message) {
//	tx := m.GetData().(*types.Transaction)
//	p.txQueue <- tx
//	// pub sub只需要转发本节点产生的交易或区块
//	hash := hex.EncodeToString(tx.Hash())
//	if !p.txFilter.Contains(hash) {
//		p.txFilter.Add(hash, struct{}{})
//		p.ps.FIFOPub(tx, psTxTopic)
//	}
//}
//
//func (p *Protocol) handleEventBroadcastBlockOld(m *queue.Message) {
//	block := m.GetData().(*types.Block)
//	p.blockQueue <- block
//	// pub sub只需要转发本节点产生的交易或区块
//	hash := hex.EncodeToString(block.Hash(p.ChainCfg))
//	if !p.blockFilter.Contains(hash) {
//		p.blockFilter.Add(hash, struct{}{})
//		p.ps.FIFOPub(block, psBlockTopic)
//	}
//}
