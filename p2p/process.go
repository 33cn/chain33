package p2p

import (
	"bytes"
	"encoding/hex"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
	"time"
)

type pubFuncType func(interface{}, string)

func (n *Node)pubToPeer(data interface{}, pid string) {
	n.pubsub.Pub(data, pid)
}

func (n *Node)processSendP2P(rawData interface{}, peerVersion int32) (sendData *types.BroadCastData, doSend bool){

	sendData = &types.BroadCastData{}
	doSend = false
	if tx, ok := rawData.(*types.P2PTx); ok {
		doSend = n.sendTx(tx, sendData, peerVersion)
	}else if blc, ok := rawData.(*types.P2PBlock); ok {
		doSend = n.sendBlock(blc, sendData, peerVersion)
	}else if query, ok := rawData.(*types.P2PQueryData); ok {
		doSend = n.sendQueryData(query, sendData)
	}else if rep, ok := rawData.(*types.P2PBlockTxReply); ok {
		doSend = n.sendQueryReply(rep, sendData)
	}

	return
}

func (n *Node)processRecvP2P(data *types.BroadCastData, pid string, pubPeerFunc pubFuncType) (handled bool){

	//TODO: 网络接收数据不可靠, 有必要统一recovery异常
	if pid == "" {
		return false
	}
	handled = true
	if tx := data.GetTx(); tx != nil {
		n.recvTx(tx)
	}else if ltTx := data.GetLtTx(); ltTx != nil {
		n.recvLtTx(ltTx, pid, pubPeerFunc)
	}else if ltBlc := data.GetLtBlock(); ltBlc != nil {
		n.recvLtBlock(ltBlc, pid, pubPeerFunc)
	}else if blc := data.GetBlock(); blc != nil {
		n.recvBlock(blc, pid)
	}else if query := data.GetQuery(); query != nil {
		n.recvQueryData(query, pid, pubPeerFunc)
	}else if rep := data.GetBlockRep(); rep != nil {
		n.recvQueryReply(rep, pid, pubPeerFunc)
	}else {
		handled = false
	}
	return
}

func (n *Node)sendBlock(block *types.P2PBlock, p2pData *types.BroadCastData, peerVersion int32) (doSend bool) {

	byteHash := block.Block.Hash()
	blockHash := hex.EncodeToString(byteHash)
	log.Debug("sendStream", "will send block", blockHash)
	if peerVersion >= LightBroadCastVersion {

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
			//TODO: replace with short hash
			ltBlock.STxHashes = append(ltBlock.STxHashes, hex.EncodeToString(tx.Hash()))
		}

		// cache block
		totalBlockCache.add(blockHash, block.Block, ltBlock.Size)

		p2pData.Value = &types.BroadCastData_LtBlock{LtBlock:ltBlock}
	} else {
		p2pData.Value = &types.BroadCastData_Block{Block:block}
	}

	blockHashFilter.RegRecvData(blockHash)
	return true
}

func (n *Node)sendQueryData(query *types.P2PQueryData, p2pData *types.BroadCastData) bool{
	p2pData.Value = &types.BroadCastData_Query{Query: query}
	return true
}

func (n *Node)sendQueryReply(rep *types.P2PBlockTxReply, p2pData *types.BroadCastData) bool {
	p2pData.Value = &types.BroadCastData_BlockRep{BlockRep:rep}
	return true
}

func (n *Node)sendTx(tx *types.P2PTx, p2pData *types.BroadCastData, peerVersion int32) (doSend bool){

	byteHash := tx.GetTx().Hash()
	txHash := hex.EncodeToString(byteHash)
	tx.Route = &types.P2PRoute{}
	if oldInfo, ok := txHashFilter.Get(txHash).(types.P2PRoute);ok {
		tx.Route.TTL = oldInfo.TTL + 1
	}else {
		//本节点发起的交易, 记录哈希
		txHashFilter.Add(txHash, tx.Route)
	}
	log.Debug("P2PSendStream", "will send tx", txHash, "ttl", tx.Route.TTL)

	//超过最大的ttl, 不再发送
	if tx.Route.TTL > n.nodeInfo.cfg.MaxTTL {
		return false
	}

	//新版本且ttl达到设定值
	if peerVersion >= LightBroadCastVersion && tx.Route.TTL >= n.nodeInfo.cfg.StartLightTxTTL{
		p2pData.Value = &types.BroadCastData_LtTx{
			LtTx: &types.LightTx{
				TxHash: byteHash,
				Route:  tx.Route,
			},
		}
	} else {
		p2pData.Value = &types.BroadCastData_Tx{Tx:tx}
	}
	return true
}

func (n *Node)recvTx(tx *types.P2PTx) {
	if tx.GetTx() == nil {
		return
	}
	txHash := hex.EncodeToString(tx.GetTx().Hash())
	if n.checkAndRegFilterAtomic(txHashFilter, txHash) {
		return
	}
	log.Debug("recvTx", "tx", txHash)
	txHashFilter.Add(txHash, tx.Route)
	txHashFilter.ReleaseLock()

	msg := n.nodeInfo.client.NewMessage("mempool", types.EventTx, tx.GetTx())
	errs := n.nodeInfo.client.Send(msg, false)
	if errs != nil {
		log.Error("recvTx", "process EventTx msg Error", errs.Error())
	}

}

func (n *Node)recvLtTx(tx *types.LightTx, pid string, pubPeerFunc pubFuncType) {

	txHash := hex.EncodeToString(tx.TxHash)
	log.Debug("recvLtTx", "peerID", pid, "txHash", txHash)
	//本地不存在, 需要向对端节点发起完整交易请求. 如果存在则表示本地已经接收过此交易, 不做任何操作
	if !txHashFilter.QueryRecvData(txHash) {

		query := &types.P2PQueryData{}
		query.Value = &types.P2PQueryData_TxReq{
			TxReq:&types.P2PTxReq{
				TxHash:tx.TxHash,
			},
		}
		//发布到指定的节点
		pubPeerFunc(query, pid)
	}
}

func (n *Node)recvBlock(block *types.P2PBlock, pid string) {

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

func (n *Node)recvLtBlock(ltBlock *types.LightBlock, pid string, pubPeerFunc pubFuncType) {

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

	//get tx list from mempool, TODO:
	resp, err := n.queryMempool(0, ltBlock.STxHashes)
	if err != nil {
		log.Error("queryMempoolTxWithHash", "err", err)
		return
	}

	txList := resp.([]*types.Transaction)
	nilTxIndices := make([]int32, 0)
	for i, tx := range txList {
		if tx == nil {
			//tx not exist in mempool
			nilTxIndices = append(nilTxIndices, int32(i+1))
		} else if tx.GroupCount > 0{

			//TODO
		}


		block.Txs = append(block.Txs, tx)
	}

	//需要比较交易根哈希是否一致, 不一致需要请求区块内所有的交易
	if len(nilTxIndices) == 0 && bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(block.Txs)){

		log.Info("recvBlock", "block==+======+====+=>Height", block.GetHeight(), "fromPeer",
			"block size(KB)", float32(ltBlock.Size)/1024, "blockHash", blockHash)
		//发送至blockchain执行
		if err = n.postBlockChain(block, pid); err != nil {
			log.Error("recvBlock", "send block to blockchain Error", err.Error())
		}

		return
	}
	// query not exist txs, TODO:增加是否请求整个block判定逻辑
	query := &types.P2PQueryData{
		Value: &types.P2PQueryData_BlockTxReq{
			BlockTxReq: &types.P2PBlockTxReq{
				BlockHash: blockHash,
				TxIndices:nilTxIndices,
			},
		},
	}
	//pub to specified peer
	pubPeerFunc(query, pid)
	//需要将不完整的block预存
	ltBlockCache.add(blockHash, block, int64(block.Size()))
}

func (n *Node)recvQueryData(query *types.P2PQueryData, pid string, pubPeerFunc pubFuncType) {

	if txReq := query.GetTxReq(); txReq != nil {

		txHash := hex.EncodeToString(txReq.TxHash)
		log.Debug("recvQueryTx", "txHash", txHash)
		//向mempool请求交易
		//get tx from mempool, TODO:
		resp, err := n.queryMempool(0, txReq.TxHash)
		if err != nil {
			log.Error("queryMempoolTxWithHash", "err", err)
			return
		}

		p2pTx := &types.P2PTx{}
		//返回的交易数组实际大小只有一个, for循环能省略空值等情况判断
		for _, tx := range resp.([]*types.Transaction) {
			p2pTx.Tx = tx
		}

		if p2pTx.GetTx() == nil {
			log.Error("recvQueryTx", "txHash", txHash, "err", "recvNilTxFromMempool")
			return
		}
		//需要重设交易的ttl
		if info, ok := txHashFilter.Get(txHash).(*types.P2PRoute); ok {
			info.TTL = -1
		}

		n.pubsub.Pub(p2pTx, pid)

	}else if blcReq := query.GetBlockTxReq(); blcReq != nil {

		log.Debug("recvQueryBlockTx", "blockHash", blcReq.BlockHash, "queryTxCount", len(blcReq.TxIndices))
		if block, ok := blockHashFilter.Get(blcReq.BlockHash).(*types.Block); ok {

			blockRep := &types.P2PBlockTxReply{BlockHash:blcReq.BlockHash}

			for i, idx := range blcReq.TxIndices {
				blockRep.TxIndices[i] = idx
				blockRep.Txs = append(blockRep.Txs, block.Txs[idx])
			}
			//请求所有的交易
			if len(blockRep.Txs) == 0 {
				blockRep.Txs = block.Txs
			}
			pubPeerFunc(blockRep, pid)
		}
	}
	return
}

func (n *Node)recvQueryReply(rep *types.P2PBlockTxReply, pid string, pubPeerFunc pubFuncType) {
	block := ltBlockCache.get(rep.BlockHash).(*types.Block)
	//not exist in cache
	if block == nil {
		return
	}

	for i, idx := range rep.TxIndices {
		block.Txs[idx] = rep.Txs[i]
	}

	//所有交易覆盖
	if len(rep.TxIndices) == 0 {
		block.Txs = rep.Txs
	}

	ltBlockCache.del(rep.BlockHash)
	//计算的root hash是否一致
	if bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(block.Txs)) {

		log.Info("recvBlock", "block==+======+====+=>Height", block.GetHeight(), "fromPeer",
			"block size(KB)", float32(block.Size())/1024, "blockHash", rep.BlockHash)
		//发送至blockchain执行
		if err := n.postBlockChain(block, pid); err != nil {
			log.Error("recvBlock", "send block to blockchain Error", err.Error())
		}
	}else if len(rep.TxIndices) != 0 {
		//不一致尝试请求整个区块的交易, 且判定是否已经请求过完整交易
		query := &types.P2PQueryData{
			Value: &types.P2PQueryData_BlockTxReq{
				BlockTxReq: &types.P2PBlockTxReq{
					BlockHash: rep.BlockHash,
					TxIndices:nil,
				},
			},
		}
		//pub to specified peer
		pubPeerFunc(query, pid)
		block.Txs = nil
		ltBlockCache.add(rep.BlockHash, block, int64(block.Size()))
	}

	return

}

func (n *Node)queryMempool(ty int64, data interface{}) (interface{}, error) {

	client := n.nodeInfo.client

	msg := client.NewMessage("mempool", ty, data)
	err := client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.WaitTimeout(msg, time.Second * 10)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (n *Node)postBlockChain(block *types.Block, pid string) error {

	msg := n.nodeInfo.client.NewMessage("blockchain", types.EventBroadcastAddBlock, &types.BlockPid{Pid: pid, Block: block})
	err := n.nodeInfo.client.Send(msg, false)
	if err != nil {
		log.Error("recvBlock", "send to blockchain Error", err.Error())
		return err
	}
	return nil
}

func (n *Node)checkAndRegFilterAtomic(filter *Filterdata, key string) (exist bool) {

	filter.GetLock()
	defer filter.ReleaseLock()
	if filter.QueryRecvData(key) {
		return true
	}
	filter.RegRecvData(key)
	return false
}
