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

func (n *Node) processSendP2P(rawData interface{}, peerVersion int32, pid, peerAddr string) (sendData *types.BroadCastData, doSend bool) {
	//出错处理
	defer func() {
		if r := recover(); r != nil {
			log.Error("processSendP2P_Panic", "sendData", rawData, "peerAddr", peerAddr, "recoverErr", r)
			doSend = false
		}
	}()
	log.Debug("ProcessSendP2PBegin", "peerID", pid, "peerAddr", peerAddr)
	sendData = &types.BroadCastData{}
	doSend = false
	if tx, ok := rawData.(*types.P2PTx); ok {
		doSend = n.sendTx(tx, sendData, peerVersion, pid, peerAddr)
	} else if blc, ok := rawData.(*types.P2PBlock); ok {
		doSend = n.sendBlock(blc, sendData, peerVersion, pid, peerAddr)
	} else if query, ok := rawData.(*types.P2PQueryData); ok {
		doSend = n.sendQueryData(query, sendData, peerAddr)
	} else if rep, ok := rawData.(*types.P2PBlockTxReply); ok {
		doSend = n.sendQueryReply(rep, sendData, peerAddr)
	} else if ping, ok := rawData.(*types.P2PPing); ok {
		doSend = true
		sendData.Value = &types.BroadCastData_Ping{Ping: ping}
	}
	log.Debug("ProcessSendP2PEnd", "peerAddr", peerAddr, "doSend", doSend)
	return
}

func (n *Node) processRecvP2P(data *types.BroadCastData, pid string, pubPeerFunc pubFuncType, peerAddr string) (handled bool) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("ProcessRecvP2P_Panic", "recvData", data, "peerAddr", peerAddr, "recoverErr", r)
		}
	}()
	log.Debug("ProcessRecvP2P", "peerID", pid, "peerAddr", peerAddr)
	if pid == "" {
		return false
	}
	handled = true
	if tx := data.GetTx(); tx != nil {
		n.recvTx(tx, pid, peerAddr)
	} else if ltTx := data.GetLtTx(); ltTx != nil {
		n.recvLtTx(ltTx, pid, peerAddr, pubPeerFunc)
	} else if ltBlc := data.GetLtBlock(); ltBlc != nil {
		n.recvLtBlock(ltBlc, pid, peerAddr, pubPeerFunc)
	} else if blc := data.GetBlock(); blc != nil {
		n.recvBlock(blc, pid, peerAddr)
	} else if query := data.GetQuery(); query != nil {
		n.recvQueryData(query, pid, peerAddr, pubPeerFunc)
	} else if rep := data.GetBlockRep(); rep != nil {
		n.recvQueryReply(rep, pid, peerAddr, pubPeerFunc)
	} else {
		handled = false
	}
	log.Debug("ProcessRecvP2P", "peerAddr", peerAddr, "handled", handled)
	return
}

func (n *Node) sendBlock(block *types.P2PBlock, p2pData *types.BroadCastData, peerVersion int32, pid, peerAddr string) (doSend bool) {

	byteHash := block.Block.Hash(n.cfg)
	blockHash := hex.EncodeToString(byteHash)
	//检测冗余发送
	ignoreSend := n.addIgnoreSendPeerAtomic(blockSendFilter, blockHash, pid)
	log.Debug("P2PSendBlock", "blockHash", blockHash, "peerIsLtVersion", peerVersion >= lightBroadCastVersion,
		"peerAddr", peerAddr, "ignoreSend", ignoreSend)
	if ignoreSend {
		return false
	}

	if peerVersion >= lightBroadCastVersion {

		if len(block.Block.Txs) < 1 {
			return false
		}
		ltBlock := &types.LightBlock{}
		ltBlock.Size = int64(types.Size(block.Block))
		ltBlock.Header = block.Block.GetHeader(n.cfg)
		ltBlock.Header.Hash = byteHash[:]
		ltBlock.Header.Signature = block.Block.Signature
		ltBlock.MinerTx = block.Block.Txs[0]
		for _, tx := range block.Block.Txs[1:] {
			//tx short hash
			ltBlock.STxHashes = append(ltBlock.STxHashes, types.CalcTxShortHash(tx.Hash()))
		}

		// cache block
		if !totalBlockCache.contains(blockHash) {
			totalBlockCache.add(blockHash, block.Block, ltBlock.Size)
		}

		p2pData.Value = &types.BroadCastData_LtBlock{LtBlock: ltBlock}
	} else {
		p2pData.Value = &types.BroadCastData_Block{Block: block}
	}

	return true
}

func (n *Node) sendQueryData(query *types.P2PQueryData, p2pData *types.BroadCastData, peerAddr string) bool {
	log.Debug("P2PSendQueryData", "peerAddr", peerAddr)
	p2pData.Value = &types.BroadCastData_Query{Query: query}
	return true
}

func (n *Node) sendQueryReply(rep *types.P2PBlockTxReply, p2pData *types.BroadCastData, peerAddr string) bool {
	log.Debug("P2PSendQueryReply", "peerAddr", peerAddr)
	p2pData.Value = &types.BroadCastData_BlockRep{BlockRep: rep}
	return true
}

func (n *Node) sendTx(tx *types.P2PTx, p2pData *types.BroadCastData, peerVersion int32, pid, peerAddr string) (doSend bool) {

	txHash := hex.EncodeToString(tx.Tx.Hash())
	ttl := tx.GetRoute().GetTTL()
	isLightSend := peerVersion >= lightBroadCastVersion && ttl >= n.nodeInfo.cfg.LightTxTTL
	//检测冗余发送
	ignoreSend := false

	//短哈希广播不记录发送过滤
	if !isLightSend {
		ignoreSend = n.addIgnoreSendPeerAtomic(txSendFilter, txHash, pid)
	}

	log.Debug("P2PSendTx", "txHash", txHash, "ttl", ttl, "isLightSend", isLightSend,
		"peerAddr", peerAddr, "ignoreSend", ignoreSend)

	if ignoreSend {
		return false
	}
	//超过最大的ttl, 不再发送
	if ttl > n.nodeInfo.cfg.MaxTTL {
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
		p2pData.Value = &types.BroadCastData_Tx{Tx: tx}
	}
	return true
}

func (n *Node) recvTx(tx *types.P2PTx, pid, peerAddr string) {
	if tx.GetTx() == nil {
		return
	}
	txHash := hex.EncodeToString(tx.GetTx().Hash())
	//将节点id添加到发送过滤, 避免冗余发送
	n.addIgnoreSendPeerAtomic(txSendFilter, txHash, pid)
	//重复接收
	isDuplicate := n.checkAndRegFilterAtomic(txHashFilter, txHash)
	log.Debug("recvTx", "tx", txHash, "ttl", tx.GetRoute().GetTTL(), "peerAddr", peerAddr, "duplicateTx", isDuplicate)
	if isDuplicate {
		return
	}
	//有可能收到老版本的交易路由,此时route是空指针
	if tx.GetRoute() == nil {
		tx.Route = &types.P2PRoute{TTL: 1}
	}
	txHashFilter.Add(txHash, tx.GetRoute())

	msg := n.nodeInfo.client.NewMessage("mempool", types.EventTx, tx.GetTx())
	errs := n.nodeInfo.client.Send(msg, false)
	if errs != nil {
		log.Error("recvTx", "process EventTx msg Error", errs.Error())
	}

}

func (n *Node) recvLtTx(tx *types.LightTx, pid, peerAddr string, pubPeerFunc pubFuncType) {

	txHash := hex.EncodeToString(tx.TxHash)
	//将节点id添加到发送过滤, 避免冗余发送
	n.addIgnoreSendPeerAtomic(txSendFilter, txHash, pid)
	exist := txHashFilter.QueryRecvData(txHash)
	log.Debug("recvLtTx", "txHash", txHash, "ttl", tx.GetRoute().GetTTL(), "peerAddr", peerAddr, "exist", exist)
	//本地不存在, 需要向对端节点发起完整交易请求. 如果存在则表示本地已经接收过此交易, 不做任何操作
	if !exist {

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

func (n *Node) recvBlock(block *types.P2PBlock, pid, peerAddr string) {

	if block.GetBlock() == nil {
		return
	}
	blockHash := hex.EncodeToString(block.GetBlock().Hash(n.cfg))
	//将节点id添加到发送过滤, 避免冗余发送
	n.addIgnoreSendPeerAtomic(blockSendFilter, blockHash, pid)
	//如果重复接收, 则不再发到blockchain执行
	isDuplicate := n.checkAndRegFilterAtomic(blockHashFilter, blockHash)
	log.Debug("recvBlock", "blockHeight", block.GetBlock().GetHeight(), "peerAddr", peerAddr,
		"block size(KB)", float32(block.Block.Size())/1024, "blockHash", blockHash, "duplicateBlock", isDuplicate)
	if isDuplicate {
		return
	}
	//发送至blockchain执行
	if err := n.postBlockChain(block.GetBlock(), pid); err != nil {
		log.Error("recvBlock", "send block to blockchain Error", err.Error())
	}

}

func (n *Node) recvLtBlock(ltBlock *types.LightBlock, pid, peerAddr string, pubPeerFunc pubFuncType) {

	blockHash := hex.EncodeToString(ltBlock.Header.Hash)
	//将节点id添加到发送过滤, 避免冗余发送
	n.addIgnoreSendPeerAtomic(blockSendFilter, blockHash, pid)
	//检测是否已经收到此block
	isDuplicate := n.checkAndRegFilterAtomic(blockHashFilter, blockHash)
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
		resp, err := n.queryMempool(types.EventTxListByHash, &types.ReqTxHashList{Hashes: ltBlock.STxHashes, IsShortHash: true})
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
	if nilTxLen == 0 && len(block.Txs) == int(ltBlock.Header.TxCount) &&
		bytes.Equal(block.TxHash, merkle.CalcMerkleRoot(block.Txs)) {

		log.Debug("recvLtBlock", "height", block.GetHeight(), "peerAddr", peerAddr,
			"blockHash", blockHash, "block size(KB)", float32(ltBlock.Size)/1024)
		//发送至blockchain执行
		if err := n.postBlockChain(block, pid); err != nil {
			log.Error("recvLtBlock", "send block to blockchain Error", err.Error())
		}

		return
	}
	// 缺失的交易个数大于总数1/3 或者缺失数据大小大于2/3, 触发请求区块所有交易数据
	if nilTxLen > 0 && (float32(nilTxLen) > float32(ltBlock.Header.TxCount)/3 ||
		float32(block.Size()) < float32(ltBlock.Size)/3) {
		nilTxIndices = nilTxIndices[:0]
	}
	log.Debug("recvLtBlock", "queryBlockHash", blockHash, "queryHeight", ltBlock.GetHeader().GetHeight(), "queryTxNum", len(nilTxIndices))

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

func (n *Node) recvQueryData(query *types.P2PQueryData, pid, peerAddr string, pubPeerFunc pubFuncType) {

	if txReq := query.GetTxReq(); txReq != nil {

		txHash := hex.EncodeToString(txReq.TxHash)
		log.Debug("recvQueryTx", "txHash", txHash, "peerAddr", peerAddr)
		//向mempool请求交易
		resp, err := n.queryMempool(types.EventTxListByHash, &types.ReqTxHashList{Hashes: []string{string(txReq.TxHash)}})
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
		pubPeerFunc(p2pTx, pid)

	} else if blcReq := query.GetBlockTxReq(); blcReq != nil {

		log.Debug("recvQueryBlockTx", "blockHash", blcReq.BlockHash, "queryTxCount", len(blcReq.TxIndices), "peerAddr", peerAddr)
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

func (n *Node) recvQueryReply(rep *types.P2PBlockTxReply, pid, peerAddr string, pubPeerFunc pubFuncType) {

	log.Debug("recvQueryReplyBlock", "blockHash", rep.GetBlockHash(), "queryTxsCount", len(rep.GetTxIndices()), "peerAddr", peerAddr)
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

		log.Debug("recvQueryReplyBlock", "blockHeight", block.GetHeight(), "peerAddr", peerAddr,
			"block size(KB)", float32(block.Size())/1024, "blockHash", rep.BlockHash)
		//发送至blockchain执行
		if err := n.postBlockChain(block, pid); err != nil {
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

//同时收到多个节点相同交易, 需要加锁保证原子操作, 检测是否存在以及添加过滤
func (n *Node) checkAndRegFilterAtomic(filter *Filterdata, key string) (exist bool) {

	filter.GetLock()
	defer filter.ReleaseLock()
	if filter.QueryRecvData(key) {
		return true
	}
	filter.RegRecvData(key)
	return false
}

//检测是否冗余发送, 或者添加到发送过滤(内部存在直接修改读写保护的数据, 对filter lru的读写需要外层锁保护)
func (n *Node) addIgnoreSendPeerAtomic(filter *Filterdata, key, pid string) (exist bool) {

	filter.GetLock()
	defer filter.ReleaseLock()
	var info *sendFilterInfo
	if !filter.QueryRecvData(key) {
		info = &sendFilterInfo{ignoreSendPeers: make(map[string]bool)}
		filter.Add(key, info)
	} else {
		info = filter.Get(key).(*sendFilterInfo)
	}
	_, exist = info.ignoreSendPeers[pid]
	info.ignoreSendPeers[pid] = true
	return exist
}
