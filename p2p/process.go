package p2p

import (
	"bytes"
	"encoding/hex"
	"time"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

type pubFuncType func(interface{}, string)

func (n *Node) pubToPeer(data interface{}, pid string) {
	n.pubsub.FIFOPub(data, pid)
}

func (n *Node) processSendP2P(rawData interface{}, peerVersion int32, peerAddr string) (sendData *types.BroadCastData, doSend bool) {
	//出错处理
	defer func() {
		if r := recover(); r != nil {
			log.Error("processSendP2P_Panic", "recoverErr", r)
			doSend = false
		}
	}()
	log.Debug("processSendP2PBegin", "peerAddr", peerAddr)
	sendData = &types.BroadCastData{}
	doSend = false
	if tx, ok := rawData.(*types.P2PTx); ok {
		doSend = n.sendTx(tx, sendData, peerVersion)
	} else if blc, ok := rawData.(*types.P2PBlock); ok {
		doSend = n.sendBlock(blc, sendData, peerVersion)
	} else if query, ok := rawData.(*types.P2PQueryData); ok {
		doSend = n.sendQueryData(query, sendData)
	} else if rep, ok := rawData.(*types.P2PBlockTxReply); ok {
		doSend = n.sendQueryReply(rep, sendData)
	} else if ping, ok := rawData.(*types.P2PPing); ok {
		doSend = true
		sendData.Value = &types.BroadCastData_Ping{Ping: ping}
	}
	log.Debug("processSendP2PEnd", "doSend", doSend)
	return
}

func (n *Node) processRecvP2P(data *types.BroadCastData, pid string, pubPeerFunc pubFuncType, peerAddr string) (handled bool) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("processRecvP2P_Panic", "recoverErr", r)
		}
	}()
	log.Debug("processRecvP2P", "peerAddr", peerAddr)
	if pid == "" {
		return false
	}
	handled = true
	if tx := data.GetTx(); tx != nil {
		n.recvTx(tx)
	} else if ltTx := data.GetLtTx(); ltTx != nil {
		n.recvLtTx(ltTx, pid, pubPeerFunc)
	} else if ltBlc := data.GetLtBlock(); ltBlc != nil {
		n.recvLtBlock(ltBlc, pid, pubPeerFunc)
	} else if blc := data.GetBlock(); blc != nil {
		n.recvBlock(blc, pid)
	} else if query := data.GetQuery(); query != nil {
		n.recvQueryData(query, pid, pubPeerFunc)
	} else if rep := data.GetBlockRep(); rep != nil {
		n.recvQueryReply(rep, pid, pubPeerFunc)
	} else {
		handled = false
	}
	log.Debug("processRecvP2P", "peerAddr", peerAddr, "handled", handled)
	return
}

func (n *Node) sendBlock(block *types.P2PBlock, p2pData *types.BroadCastData, peerVersion int32) (doSend bool) {

	if block.Block == nil {
		return false
	}
	byteHash := block.Block.Hash()
	blockHash := hex.EncodeToString(byteHash)
	log.Debug("sendStream", "will send block", blockHash)
	if peerVersion >= lightBroadCastVersion {

		if len(block.Block.Txs) < 1 {
			return false
		}
		ltBlock := &types.LightBlock{}
		ltBlock.Size = int64(types.Size(block.Block))
		ltBlock.Header = block.Block.GetHeader()
		ltBlock.Header.Hash = byteHash[:]
		ltBlock.Header.Signature = block.Block.Signature
		ltBlock.MinerTx = block.Block.Txs[0]
		for _, tx := range block.Block.Txs[1:] {
			//tx short hash
			ltBlock.STxHashes = append(ltBlock.STxHashes, types.CalcTxShortHash(tx.Hash()))
		}

		// cache block
		totalBlockCache.add(blockHash, block.Block, ltBlock.Size)

		p2pData.Value = &types.BroadCastData_LtBlock{LtBlock: ltBlock}
	} else {
		p2pData.Value = &types.BroadCastData_Block{Block: block}
	}

	blockHashFilter.RegRecvData(blockHash)
	return true
}

func (n *Node) sendQueryData(query *types.P2PQueryData, p2pData *types.BroadCastData) bool {
	p2pData.Value = &types.BroadCastData_Query{Query: query}
	return true
}

func (n *Node) sendQueryReply(rep *types.P2PBlockTxReply, p2pData *types.BroadCastData) bool {
	p2pData.Value = &types.BroadCastData_BlockRep{BlockRep: rep}
	return true
}

func (n *Node) sendTx(tx *types.P2PTx, p2pData *types.BroadCastData, peerVersion int32) (doSend bool) {

	log.Debug("P2PSendStream", "will send tx", hex.EncodeToString(tx.Tx.Hash()), "ttl", tx.Route.TTL)
	//超过最大的ttl, 不再发送
	if tx.Route.TTL > n.nodeInfo.cfg.MaxTTL {
		return false
	}

	//新版本且ttl达到设定值
	if peerVersion >= lightBroadCastVersion && tx.Route.TTL >= n.nodeInfo.cfg.LightTxTTL {
		p2pData.Value = &types.BroadCastData_LtTx{
			LtTx: &types.LightTx{
				TxHash: tx.Tx.Hash(),
				Route:  tx.Route,
			},
		}
	} else {
		p2pData.Value = &types.BroadCastData_Tx{Tx: tx}
	}
	return true
}

func (n *Node) recvTx(tx *types.P2PTx) {
	if tx.GetTx() == nil {
		return
	}
	txHash := hex.EncodeToString(tx.GetTx().Hash())
	if n.checkAndRegFilterAtomic(txHashFilter, txHash) {
		return
	}
	log.Debug("recvTx", "tx", txHash)
	txHashFilter.Add(txHash, tx.Route)

	msg := n.nodeInfo.client.NewMessage("mempool", types.EventTx, tx.GetTx())
	errs := n.nodeInfo.client.Send(msg, false)
	if errs != nil {
		log.Error("recvTx", "process EventTx msg Error", errs.Error())
	}

}

func (n *Node) recvLtTx(tx *types.LightTx, pid string, pubPeerFunc pubFuncType) {

	txHash := hex.EncodeToString(tx.TxHash)
	log.Debug("recvLtTx", "peerID", pid, "txHash", txHash)
	//本地不存在, 需要向对端节点发起完整交易请求. 如果存在则表示本地已经接收过此交易, 不做任何操作
	if !txHashFilter.QueryRecvData(txHash) {

		query := &types.P2PQueryData{}
		query.Value = &types.P2PQueryData_TxReq{
			TxReq: &types.P2PTxReq{
				TxHash: tx.TxHash,
			},
		}
		//发布到指定的节点
		pubPeerFunc(query, pid)
	}
}

func (n *Node) recvBlock(block *types.P2PBlock, pid string) {

	if block.GetBlock() == nil {
		return
	}
	//如果已经有登记过的消息记录，则不发送给本地blockchain
	blockHash := hex.EncodeToString(block.GetBlock().Hash())
	if n.checkAndRegFilterAtomic(blockHashFilter, blockHash) {
		return
	}

	log.Info("recvBlock", "block==+======+====+=>Height", block.GetBlock().GetHeight(), "fromPeer", pid,
		"block size(KB)", float32(block.Block.Size())/1024, "blockHash", blockHash)
	//发送至blockchain执行
	if err := n.postBlockChain(block.GetBlock(), pid); err != nil {
		log.Error("recvBlock", "send block to blockchain Error", err.Error())
	}

}

func (n *Node) recvLtBlock(ltBlock *types.LightBlock, pid string, pubPeerFunc pubFuncType) {

	blockHash := hex.EncodeToString(ltBlock.Header.Hash)
	//检测是否已经收到此block
	if n.checkAndRegFilterAtomic(blockHashFilter, blockHash) {
		return
	}
	log.Debug("recvLtBlock", "blockHash", blockHash)
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

	//get tx list from mempool
	resp, err := n.queryMempool(types.EventTxListByHash, &types.ReqTxHashList{Hashes: ltBlock.STxHashes, IsShortHash: true})
	if err != nil {
		log.Error("queryMempoolTxWithHash", "err", err)
		return
	}

	txList, ok := resp.(*types.ReplyTxList)
	if !ok {
		log.Error("recvLtBlock", "queryMemPool", "nilReplyTxList")
		txList = &types.ReplyTxList{}
	}
	nilTxIndices := make([]int32, 0)
	for i := 0; ok && i < len(txList.Txs); i++ {
		tx := txList.Txs[i]
		if tx == nil {
			//tx not exist in mempool
			nilTxIndices = append(nilTxIndices, int32(i+1))
		} else if tx.GetGroupCount() > 0 {

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

	//需要比较交易根哈希是否一致, 不一致需要请求区块内所有的交易
	if len(block.Txs) == int(ltBlock.Header.TxCount) && bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(block.Txs)) {

		log.Info("recvBlock", "block==+======+====+=>Height", block.GetHeight(), "fromPeer", pid,
			"block size(KB)", float32(ltBlock.Size)/1024, "blockHash", blockHash)
		//发送至blockchain执行
		if err = n.postBlockChain(block, pid); err != nil {
			log.Error("recvBlock", "send block to blockchain Error", err.Error())
		}

		return
	}

	nilTxLen := len(nilTxIndices)
	// 缺失的交易个数大于总数1/3 或者缺失数据大小大于2/3, 触发请求区块所有交易数据
	if nilTxLen > 0 && (float32(nilTxLen) > float32(ltBlock.Header.TxCount)/3 ||
		float32(block.Size()) < float32(ltBlock.Size)/3) {
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
	//pub to specified peer
	pubPeerFunc(query, pid)
	//需要将不完整的block预存
	ltBlockCache.add(blockHash, block, int64(block.Size()))
}

func (n *Node) recvQueryData(query *types.P2PQueryData, pid string, pubPeerFunc pubFuncType) {

	if txReq := query.GetTxReq(); txReq != nil {

		txHash := hex.EncodeToString(txReq.TxHash)
		log.Debug("recvQueryTx", "txHash", txHash)
		//向mempool请求交易
		//get tx from mempool
		resp, err := n.queryMempool(types.EventTxListByHash, &types.ReqTxHashList{Hashes: []string{string(txReq.TxHash)}})
		if err != nil {
			log.Error("queryMempoolTxWithHash", "err", err)
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
		pubPeerFunc(p2pTx, pid)

	} else if blcReq := query.GetBlockTxReq(); blcReq != nil {

		log.Debug("recvQueryBlockTx", "blockHash", blcReq.BlockHash, "queryTxCount", len(blcReq.TxIndices))
		if block, ok := totalBlockCache.get(blcReq.BlockHash).(*types.Block); ok {

			blockRep := &types.P2PBlockTxReply{BlockHash: blcReq.BlockHash}

			blockRep.TxIndices = blcReq.TxIndices
			for _, idx := range blcReq.TxIndices {
				blockRep.Txs = append(blockRep.Txs, block.Txs[idx])
			}
			//请求所有的交易
			if len(blockRep.TxIndices) == 0 {
				blockRep.Txs = block.Txs
			}
			pubPeerFunc(blockRep, pid)
		}
	}
}

func (n *Node) recvQueryReply(rep *types.P2PBlockTxReply, pid string, pubPeerFunc pubFuncType) {
	val, exist := ltBlockCache.del(rep.BlockHash)
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
	if bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(block.Txs)) {

		log.Info("recvBlock", "block==+======+====+=>Height", block.GetHeight(), "fromPeer", pid,
			"block size(KB)", float32(block.Size())/1024, "blockHash", rep.BlockHash)
		//发送至blockchain执行
		if err := n.postBlockChain(block, pid); err != nil {
			log.Error("recvBlock", "send block to blockchain Error", err.Error())
		}
	} else if len(rep.TxIndices) != 0 {
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
		pubPeerFunc(query, pid)
		block.Txs = nil
		ltBlockCache.add(rep.BlockHash, block, int64(block.Size()))
	}
}

func (n *Node) queryMempool(ty int64, data interface{}) (interface{}, error) {

	client := n.nodeInfo.client

	msg := client.NewMessage("mempool", ty, data)
	err := client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (n *Node) postBlockChain(block *types.Block, pid string) error {

	msg := n.nodeInfo.client.NewMessage("blockchain", types.EventBroadcastAddBlock, &types.BlockPid{Pid: pid, Block: block})
	err := n.nodeInfo.client.Send(msg, false)
	if err != nil {
		log.Error("postBlockChain", "send to blockchain Error", err.Error())
		return err
	}
	return nil
}

func (n *Node) checkAndRegFilterAtomic(filter *Filterdata, key string) (exist bool) {

	filter.GetLock()
	defer filter.ReleaseLock()
	if filter.QueryRecvData(key) {
		return true
	}
	filter.RegRecvData(key)
	return false
}
