// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

//message callback
import (
	"fmt"
	"sync/atomic"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

//ProcRecvMsg blockchain模块的消息接收处理
func (chain *BlockChain) ProcRecvMsg() {
	defer chain.recvwg.Done()
	reqnum := make(chan struct{}, 1000)
	for msg := range chain.client.Recv() {
		chainlog.Debug("blockchain recv", "msg", types.GetEventName(int(msg.Ty)), "id", msg.ID, "cap", len(reqnum))
		msgtype := msg.Ty
		reqnum <- struct{}{}
		atomic.AddInt32(&chain.runcount, 1)
		if chain.procLocalDB(msgtype, msg, reqnum) {
			continue
		}
		switch msgtype {
		case types.EventQueryTx:
			go chain.processMsg(msg, reqnum, chain.queryTx)
		case types.EventGetBlocks:
			go chain.processMsg(msg, reqnum, chain.getBlocks)
		case types.EventSyncBlock: // block
			go chain.processMsg(msg, reqnum, chain.addBlock)
		case types.EventGetBlockHeight:
			go chain.processMsg(msg, reqnum, chain.getBlockHeight)
		case types.EventTxHashList:
			go chain.processMsg(msg, reqnum, chain.txHashList)
		case types.EventGetHeaders:
			go chain.processMsg(msg, reqnum, chain.getHeaders)
		case types.EventGetLastHeader:
			go chain.processMsg(msg, reqnum, chain.getLastHeader)
		case types.EventAddBlockDetail:
			go chain.processMsg(msg, reqnum, chain.addBlockDetail)
		case types.EventBroadcastAddBlock: //block
			go chain.processMsg(msg, reqnum, chain.broadcastAddBlock)
		case types.EventGetTransactionByAddr:
			go chain.processMsg(msg, reqnum, chain.getTransactionByAddr)
		case types.EventGetTransactionByHash:
			go chain.processMsg(msg, reqnum, chain.getTransactionByHashes)
		case types.EventGetBlockOverview: //blockOverview
			go chain.processMsg(msg, reqnum, chain.getBlockOverview)
		case types.EventGetAddrOverview: //addrOverview
			go chain.processMsg(msg, reqnum, chain.getAddrOverview)
		case types.EventGetBlockHash: //GetBlockHash
			go chain.processMsg(msg, reqnum, chain.getBlockHash)
		case types.EventAddBlockHeaders:
			go chain.processMsg(msg, reqnum, chain.addBlockHeaders)
		case types.EventGetLastBlock:
			go chain.processMsg(msg, reqnum, chain.getLastBlock)
		case types.EventIsSync:
			go chain.processMsg(msg, reqnum, chain.isSync)
		case types.EventIsNtpClockSync:
			go chain.processMsg(msg, reqnum, chain.isNtpClockSyncFunc)
		case types.EventGetLastBlockSequence:
			go chain.processMsg(msg, reqnum, chain.getLastBlockSequence)
		case types.EventGetBlockSequences:
			go chain.processMsg(msg, reqnum, chain.getBlockSequences)
		case types.EventGetBlockByHashes:
			go chain.processMsg(msg, reqnum, chain.getBlockByHashes)
		case types.EventGetBlockBySeq:
			go chain.processMsg(msg, reqnum, chain.getBlockBySeq)
		case types.EventDelParaChainBlockDetail:
			go chain.processMsg(msg, reqnum, chain.delParaChainBlockDetail)
		case types.EventAddParaChainBlockDetail:
			go chain.processMsg(msg, reqnum, chain.addParaChainBlockDetail)
		case types.EventGetSeqByHash:
			go chain.processMsg(msg, reqnum, chain.getSeqByHash)
		case types.EventAddBlockSeqCB:
			go chain.processMsg(msg, reqnum, chain.addBlockSeqCB)
		case types.EventListBlockSeqCB:
			go chain.processMsg(msg, reqnum, chain.listBlockSeqCB)
		case types.EventGetSeqCBLastNum:
			go chain.processMsg(msg, reqnum, chain.getSeqCBLastNum)
		case types.EventGetLastBlockMainSequence:
			go chain.processMsg(msg, reqnum, chain.GetLastBlockMainSequence)
		case types.EventGetMainSeqByHash:
			go chain.processMsg(msg, reqnum, chain.GetMainSeqByHash)
		//para共识模块操作blockchain db的事件
		case types.EventSetValueByKey:
			go chain.processMsg(msg, reqnum, chain.setValueByKey)
		case types.EventGetValueByKey:
			go chain.processMsg(msg, reqnum, chain.getValueByKey)
		//通过平行链title获取平行链的交易
		case types.EventGetParaTxByTitle:
			go chain.processMsg(msg, reqnum, chain.getParaTxByTitle)

			//获取拥有此title交易的区块高度
		case types.EventGetHeightByTitle:
			go chain.processMsg(msg, reqnum, chain.getHeightByTitle)

			//通过区块高度列表+title获取平行链交易
		case types.EventGetParaTxByTitleAndHeight:
			go chain.processMsg(msg, reqnum, chain.getParaTxByTitleAndHeight)
			// 获取chunk record
		case types.EventGetChunkRecord:
			go chain.processMsg(msg, reqnum, chain.getChunkRecord)
			// 获取chunk record
		case types.EventAddChunkRecord:
			go chain.processMsg(msg, reqnum, chain.addChunkRecord)
			// 从localdb中获取Chunk BlockBody
		case types.EventGetChunkBlockBody:
			go chain.processMsg(msg, reqnum, chain.getChunkBlockBody)
			// 通知blockchain 保存Chunk BlockBody到p2pstore
		case types.EventNotifyStoreChunk:
			go chain.processMsg(msg, reqnum, chain.storeChunkBlockBody)

		default:
			go chain.processMsg(msg, reqnum, chain.unknowMsg)
		}
	}
}

func (chain *BlockChain) unknowMsg(msg *queue.Message) {
	chainlog.Warn("ProcRecvMsg unknow msg", "msgtype", msg.Ty)
}

func (chain *BlockChain) addBlockSeqCB(msg *queue.Message) {
	reply := &types.ReplyAddSeqCallback{
		IsOk: true,
	}
	cb := (msg.Data).(*types.BlockSeqCB)
	sequences, err := chain.ProcAddBlockSeqCB(cb)
	if err != nil {
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
		reply.Seqs = sequences
		msg.Reply(chain.client.NewMessage("rpc", types.EventAddBlockSeqCB, reply))
		return
	}

	msg.Reply(chain.client.NewMessage("rpc", types.EventAddBlockSeqCB, reply))
}

func (chain *BlockChain) listBlockSeqCB(msg *queue.Message) {
	cbs, err := chain.ProcListBlockSeqCB()
	if err != nil {
		chainlog.Error("listBlockSeqCB", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventListBlockSeqCB, err))
		return
	}
	msg.Reply(chain.client.NewMessage("rpc", types.EventListBlockSeqCB, cbs))
}
func (chain *BlockChain) getSeqCBLastNum(msg *queue.Message) {
	data := (msg.Data).(*types.ReqString)

	num := chain.ProcGetSeqCBLastNum(data.Data)
	lastNum := &types.Int64{Data: num}
	msg.Reply(chain.client.NewMessage("rpc", types.EventGetSeqCBLastNum, lastNum))
}

func (chain *BlockChain) queryTx(msg *queue.Message) {
	txhash := (msg.Data).(*types.ReqHash)
	txDetail, err := chain.ProcQueryTxMsg(txhash.Hash)
	if err != nil {
		chainlog.Debug("ProcQueryTxMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetail, err))
	} else {
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetail, txDetail))
	}
}

func (chain *BlockChain) getBlocks(msg *queue.Message) {
	requestblocks := (msg.Data).(*types.ReqBlocks)
	blocks, err := chain.ProcGetBlockDetailsMsg(requestblocks)
	if err != nil {
		chainlog.Error("ProcGetBlockDetailsMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlocks, err))
	} else {
		//chainlog.Debug("ProcGetBlockDetailsMsg", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlocks, blocks))
	}
}

func (chain *BlockChain) addBlock(msg *queue.Message) {
	var reply types.Reply
	reply.IsOk = true
	blockpid := msg.Data.(*types.BlockPid)
	//chainlog.Error("addBlock", "height", blockpid.Block.Height, "pid", blockpid.Pid)
	if chain.GetDownloadSyncStatus() {
		err := chain.WriteBlockToDbTemp(blockpid.Block, true)
		if err != nil {
			chainlog.Error("WriteBlockToDbTemp", "height", blockpid.Block.Height, "err", err.Error())
			reply.IsOk = false
			reply.Msg = []byte(err.Error())
		}
		//downLoadTask 运行时设置对应的blockdone
		if chain.downLoadTask.InProgress() {
			chain.downLoadTask.Done(blockpid.Block.GetHeight())
		}
	} else {
		_, err := chain.ProcAddBlockMsg(false, &types.BlockDetail{Block: blockpid.Block}, blockpid.Pid)
		if err != nil {
			chainlog.Error("ProcAddBlockMsg", "height", blockpid.Block.Height, "err", err.Error())
			reply.IsOk = false
			reply.Msg = []byte(err.Error())
		}
		chainlog.Debug("EventAddBlock", "height", blockpid.Block.Height, "pid", blockpid.Pid, "success", "ok")
	}
	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, &reply))
}

func (chain *BlockChain) getBlockHeight(msg *queue.Message) {
	var replyBlockHeight types.ReplyBlockHeight
	replyBlockHeight.Height = chain.GetBlockHeight()
	//chainlog.Debug("EventGetBlockHeight", "success", "ok")
	msg.Reply(chain.client.NewMessage("consensus", types.EventReplyBlockHeight, &replyBlockHeight))
}

func (chain *BlockChain) txHashList(msg *queue.Message) {
	txhashlist := (msg.Data).(*types.TxHashList)
	duptxhashlist, err := chain.GetDuplicateTxHashList(txhashlist)
	if err != nil {
		chainlog.Error("txHashList", "err", err.Error())
		msg.Reply(chain.client.NewMessage("consensus", types.EventTxHashListReply, err))
		return
	}
	//chainlog.Debug("EventTxHashList", "success", "ok")
	msg.Reply(chain.client.NewMessage("consensus", types.EventTxHashListReply, duptxhashlist))
}

func (chain *BlockChain) getHeaders(msg *queue.Message) {
	requestblocks := (msg.Data).(*types.ReqBlocks)
	headers, err := chain.ProcGetHeadersMsg(requestblocks)
	if err != nil {
		chainlog.Error("ProcGetHeadersMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventHeaders, err))
	} else {
		//chainlog.Debug("EventGetHeaders", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventHeaders, headers))
	}
}

func (chain *BlockChain) isSync(msg *queue.Message) {
	ok := chain.IsCaughtUp()
	msg.Reply(chain.client.NewMessage("", types.EventReplyIsSync, &types.IsCaughtUp{Iscaughtup: ok}))
}

func (chain *BlockChain) getLastHeader(msg *queue.Message) {
	header, err := chain.ProcGetLastHeaderMsg()
	if err != nil {
		chainlog.Error("ProcGetLastHeaderMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("account", types.EventHeader, err))
	} else {
		//chainlog.Debug("EventGetLastHeader", "success", "ok")
		msg.Reply(chain.client.NewMessage("account", types.EventHeader, header))
	}
}

//共识过来的block是没有被执行的，首先判断此block的parent block是否是当前best链的tip
//在blockchain执行时需要做tx的去重处理，所以在执行成功之后需要将最新区块详情返回给共识模块
func (chain *BlockChain) addBlockDetail(msg *queue.Message) {
	blockDetail := msg.Data.(*types.BlockDetail)
	Height := blockDetail.Block.Height
	chainlog.Info("EventAddBlockDetail", "height", blockDetail.Block.Height, "parent", common.ToHex(blockDetail.Block.ParentHash))
	blockDetail, err := chain.ProcAddBlockMsg(true, blockDetail, "self")
	if err != nil {
		chainlog.Error("addBlockDetail", "err", err.Error())
		msg.Reply(chain.client.NewMessage("consensus", types.EventAddBlockDetail, err))
		return
	}
	if blockDetail == nil {
		err = types.ErrExecBlockNil
		chainlog.Error("addBlockDetail", "err", err.Error())
		msg.Reply(chain.client.NewMessage("consensus", types.EventAddBlockDetail, err))
		return
	}
	chainlog.Debug("addBlockDetail success ", "Height", Height, "hash", common.HashHex(blockDetail.Block.Hash(chain.client.GetConfig())))
	msg.Reply(chain.client.NewMessage("consensus", types.EventAddBlockDetail, blockDetail))
}

//超前太多或者落后太多的广播区块都不做处理：
//当本节点在同步阶段并且远远落后主网最新高度时不处理广播block,暂定落后128个区块
//以免广播区块占用go goroutine资源
//目前回滚只支持10000个区块，所以收到落后10000高度之外的广播区块也不做处理
func (chain *BlockChain) broadcastAddBlock(msg *queue.Message) {
	var reply types.Reply
	reply.IsOk = true
	blockwithpid := msg.Data.(*types.BlockPid)

	castheight := blockwithpid.Block.Height
	curheight := chain.GetBlockHeight()

	futureMaximum := castheight > curheight+BackBlockNum
	backWardMaximum := curheight > MaxRollBlockNum && castheight < curheight-MaxRollBlockNum

	if futureMaximum || backWardMaximum {
		chainlog.Debug("EventBroadcastAddBlock", "curheight", curheight, "castheight", castheight, "hash", common.ToHex(blockwithpid.Block.Hash(chain.client.GetConfig())), "pid", blockwithpid.Pid, "result", "Do not handle broad cast Block in sync")
		msg.Reply(chain.client.NewMessage("", types.EventReply, &reply))
		return
	}
	_, err := chain.ProcAddBlockMsg(true, &types.BlockDetail{Block: blockwithpid.Block}, blockwithpid.Pid)
	if err != nil {
		chainlog.Error("ProcAddBlockMsg", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	chainlog.Debug("EventBroadcastAddBlock", "height", blockwithpid.Block.Height, "hash", common.ToHex(blockwithpid.Block.Hash(chain.client.GetConfig())), "pid", blockwithpid.Pid, "success", "ok")

	msg.Reply(chain.client.NewMessage("", types.EventReply, &reply))
}

func (chain *BlockChain) getTransactionByAddr(msg *queue.Message) {
	addr := (msg.Data).(*types.ReqAddr)
	//chainlog.Warn("EventGetTransactionByAddr", "req", addr)
	replyTxInfos, err := chain.ProcGetTransactionByAddr(addr)
	if err != nil {
		chainlog.Error("ProcGetTransactionByAddr", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyTxInfo, err))
	} else {
		//chainlog.Debug("EventGetTransactionByAddr", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyTxInfo, replyTxInfos))
	}
}

func (chain *BlockChain) getTransactionByHashes(msg *queue.Message) {
	txhashs := (msg.Data).(*types.ReqHashes)
	//chainlog.Info("EventGetTransactionByHash", "hash", txhashs)
	TransactionDetails, err := chain.ProcGetTransactionByHashes(txhashs.Hashes)
	if err != nil {
		chainlog.Error("ProcGetTransactionByHashes", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetails, err))
	} else {
		//chainlog.Debug("EventGetTransactionByHash", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetails, TransactionDetails))
	}
}

func (chain *BlockChain) getBlockOverview(msg *queue.Message) {
	ReqHash := (msg.Data).(*types.ReqHash)
	BlockOverview, err := chain.ProcGetBlockOverview(ReqHash)
	if err != nil {
		chainlog.Error("ProcGetBlockOverview", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyBlockOverview, err))
	} else {
		//chainlog.Debug("ProcGetBlockOverview", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyBlockOverview, BlockOverview))
	}
}

func (chain *BlockChain) getAddrOverview(msg *queue.Message) {
	addr := (msg.Data).(*types.ReqAddr)
	AddrOverview, err := chain.ProcGetAddrOverview(addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyAddrOverview, err))
	} else {
		//chainlog.Debug("ProcGetAddrOverview", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyAddrOverview, AddrOverview))
	}
}

func (chain *BlockChain) getBlockHash(msg *queue.Message) {
	height := (msg.Data).(*types.ReqInt)
	replyhash, err := chain.ProcGetBlockHash(height)
	if err != nil {
		chainlog.Error("ProcGetBlockHash", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlockHash, err))
	} else {
		//chainlog.Debug("ProcGetBlockHash", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlockHash, replyhash))
	}
}

func (chain *BlockChain) addBlockHeaders(msg *queue.Message) {
	var reply types.Reply
	reply.IsOk = true
	headerspid := msg.Data.(*types.HeadersPid)
	err := chain.ProcAddBlockHeadersMsg(headerspid.Headers, headerspid.Pid)
	if err != nil {
		chainlog.Error("addBlockHeaders", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
	}
	chainlog.Debug("addBlockHeaders", "pid", headerspid.Pid, "success", "ok")
	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, &reply))
}

func (chain *BlockChain) getLastBlock(msg *queue.Message) {
	block, err := chain.ProcGetLastBlockMsg()
	if err != nil {
		chainlog.Error("ProcGetLastBlockMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("consensus", types.EventBlock, err))
	} else {
		//chainlog.Debug("ProcGetLastBlockMsg", "success", "ok")
		msg.Reply(chain.client.NewMessage("consensus", types.EventBlock, block))
	}
}

func (chain *BlockChain) isNtpClockSyncFunc(msg *queue.Message) {
	ok := chain.GetNtpClockSyncStatus()
	msg.Reply(chain.client.NewMessage("", types.EventReplyIsNtpClockSync, &types.IsNtpClockSync{Isntpclocksync: ok}))
}

type funcProcess func(msg *queue.Message)

func (chain *BlockChain) processMsg(msg *queue.Message, reqnum chan struct{}, cb funcProcess) {
	beg := types.Now()
	defer func() {
		<-reqnum
		atomic.AddInt32(&chain.runcount, -1)
		chainlog.Debug("process", "cost", types.Since(beg), "msg", types.GetEventName(int(msg.Ty)))
		if r := recover(); r != nil {
			chainlog.Error("panic error", "err", r)
			msg.Reply(chain.client.NewMessage("", msg.Ty, fmt.Errorf("%s:%v", types.ErrExecPanic.Error(), r)))
			return
		}
	}()
	cb(msg)
}

//获取最新的block执行序列号
func (chain *BlockChain) getLastBlockSequence(msg *queue.Message) {
	var lastSequence types.Int64
	var err error
	lastSequence.Data, err = chain.blockStore.LoadBlockLastSequence()
	if err != nil {
		chainlog.Debug("getLastBlockSequence", "err", err)
	}
	msg.Reply(chain.client.NewMessage("rpc", types.EventReplyLastBlockSequence, &lastSequence))
}

//获取指定区间的block执行序列信息，包含blockhash和操作类型：add/del
func (chain *BlockChain) getBlockSequences(msg *queue.Message) {
	requestSequences := (msg.Data).(*types.ReqBlocks)
	BlockSequences, err := chain.GetBlockSequences(requestSequences)
	if err != nil {
		chainlog.Error("GetBlockSequences", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyBlockSequences, err))
	} else {
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyBlockSequences, BlockSequences))
	}
}

func (chain *BlockChain) getBlockByHashes(msg *queue.Message) {
	blockhashes := (msg.Data).(*types.ReqHashes)
	BlockDetails, err := chain.GetBlockByHashes(blockhashes.Hashes)
	if err != nil {
		chainlog.Error("GetBlockByHashes", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlocks, err))
	} else {
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlocks, BlockDetails))
	}
}

func (chain *BlockChain) getBlockBySeq(msg *queue.Message) {
	seq := (msg.Data).(*types.Int64)
	req := &types.ReqBlocks{Start: seq.Data, End: seq.Data, IsDetail: false, Pid: []string{}}
	sequences, err := chain.GetBlockSequences(req)
	if err != nil {
		chainlog.Error("getBlockBySeq", "seq err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventGetBlockBySeq, err))
		return
	}
	reqHashes := &types.ReqHashes{Hashes: [][]byte{sequences.Items[0].Hash}}
	blocks, err := chain.GetBlockByHashes(reqHashes.Hashes)
	if err != nil {
		chainlog.Error("getBlockBySeq", "hash err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventGetBlockBySeq, err))
		return
	}

	blockSeq := &types.BlockSeq{
		Num:    seq.Data,
		Seq:    sequences.Items[0],
		Detail: blocks.Items[0],
	}
	msg.Reply(chain.client.NewMessage("rpc", types.EventGetBlockBySeq, blockSeq))

}

//平行链del block的处理
func (chain *BlockChain) delParaChainBlockDetail(msg *queue.Message) {
	var parablockDetail *types.ParaChainBlockDetail
	var reply types.Reply
	reply.IsOk = true
	parablockDetail = msg.Data.(*types.ParaChainBlockDetail)

	chainlog.Debug("delParaChainBlockDetail", "height", parablockDetail.Blockdetail.Block.Height, "hash", common.HashHex(parablockDetail.Blockdetail.Block.Hash(chain.client.GetConfig())))

	// 平行链上P2P模块关闭，不用广播区块
	err := chain.ProcDelParaChainBlockMsg(false, parablockDetail, "self")
	if err != nil {
		chainlog.Error("ProcDelParaChainBlockMsg", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	chainlog.Debug("delParaChainBlockDetail", "success", "ok")
	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, &reply))
}

//平行链add block的处理
func (chain *BlockChain) addParaChainBlockDetail(msg *queue.Message) {
	parablockDetail := msg.Data.(*types.ParaChainBlockDetail)

	//根据配置chain.cfgBatchSync和parablockDetail.IsSync
	//来决定写数据库时是否需要刷盘,主要是为了同步阶段提高执行区块的效率
	if !parablockDetail.IsSync && !chain.cfgBatchSync {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 1, 0)
	} else {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 0, 1)
	}

	chainlog.Debug("EventAddParaChainBlockDetail", "height", parablockDetail.Blockdetail.Block.Height, "hash", common.HashHex(parablockDetail.Blockdetail.Block.Hash(chain.client.GetConfig())))
	// 平行链上P2P模块关闭，不用广播区块
	blockDetail, err := chain.ProcAddParaChainBlockMsg(false, parablockDetail, "self")
	if err != nil {
		chainlog.Error("ProcAddParaChainBlockMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("p2p", types.EventReply, err))
	}
	chainlog.Debug("EventAddParaChainBlockDetail", "success", "ok")
	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, blockDetail))
}

//parachian 通过blockhash获取对应的seq，只记录了addblock时的seq
func (chain *BlockChain) getSeqByHash(msg *queue.Message) {
	blockhash := (msg.Data).(*types.ReqHash)
	seq, err := chain.ProcGetSeqByHash(blockhash.Hash)
	if err != nil {
		chainlog.Error("getSeqByHash", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReply, err))
	}
	msg.Reply(chain.client.NewMessage("rpc", types.EventGetSeqByHash, &types.Int64{Data: seq}))
}

//获取指定地址参与的tx交易计数
func (chain *BlockChain) localAddrTxCount(msg *queue.Message) {
	reqkey := (msg.Data).(*types.ReqKey)

	count := types.Int64{}
	var counts int64

	TxsCount, err := chain.blockStore.db.Get(reqkey.Key)
	if err != nil && err != types.ErrNotFound {
		counts = 0
		chainlog.Error("localAddrTxCount", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, &types.Int64{Data: counts}))
		return
	}
	if len(TxsCount) == 0 {
		counts = 0
		chainlog.Error("localAddrTxCount", "TxsCount", "0")
		msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, &types.Int64{Data: counts}))
		return
	}
	err = types.Decode(TxsCount, &count)
	if err != nil {
		counts = 0
		chainlog.Error("localAddrTxCount", "types.Decode", err)
		msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, &types.Int64{Data: counts}))
		return
	}
	counts = count.Data
	msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, &types.Int64{Data: counts}))
}

//GetLastBlockMainSequence 获取最新的block执行序列号
func (chain *BlockChain) GetLastBlockMainSequence(msg *queue.Message) {
	var lastSequence types.Int64
	var err error
	lastSequence.Data, err = chain.blockStore.LoadBlockLastMainSequence()
	if err != nil {
		chainlog.Debug("GetLastBlockMainSequence", "err", err)
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyLastBlockMainSequence, err))
		return
	}
	msg.Reply(chain.client.NewMessage("rpc", types.EventReplyLastBlockMainSequence, &lastSequence))
}

//GetMainSeqByHash parachian 通过blockhash获取对应的seq，只记录了addblock时的seq
func (chain *BlockChain) GetMainSeqByHash(msg *queue.Message) {
	blockhash := (msg.Data).(*types.ReqHash)
	seq, err := chain.ProcGetMainSeqByHash(blockhash.Hash)
	if err != nil {
		chainlog.Error("GetMainSeqByHash", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyMainSeqByHash, err))
		return
	}
	msg.Reply(chain.client.NewMessage("rpc", types.EventReplyMainSeqByHash, &types.Int64{Data: seq}))
}

//setValueByKey 设置kv对到blockchain db中
func (chain *BlockChain) setValueByKey(msg *queue.Message) {
	var reply types.Reply
	reply.IsOk = true
	if !chain.isParaChain {
		reply.IsOk = false
		reply.Msg = []byte("Must Para Chain Support!")
		msg.Reply(chain.client.NewMessage("", types.EventReply, &reply))
		return
	}
	kvs := (msg.Data).(*types.LocalDBSet)
	err := chain.SetValueByKey(kvs)
	if err != nil {
		chainlog.Error("setValueByKey", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	}
	msg.Reply(chain.client.NewMessage("", types.EventReply, &reply))
}

//GetValueByKey 获取value通过key从blockchain db中
func (chain *BlockChain) getValueByKey(msg *queue.Message) {
	if !chain.isParaChain {
		msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, nil))
		return
	}
	keys := (msg.Data).(*types.LocalDBGet)
	values := chain.GetValueByKey(keys)
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, values))
}

//getParaTxByTitle //通过平行链title获取平行链的交易
func (chain *BlockChain) getParaTxByTitle(msg *queue.Message) {
	req := (msg.Data).(*types.ReqParaTxByTitle)
	reply, err := chain.GetParaTxByTitle(req)
	if err != nil {
		chainlog.Error("getParaTxByTitle", "req", req, "err", err.Error())
		msg.Reply(chain.client.NewMessage("", types.EventReplyParaTxByTitle, err))
		return
	}
	msg.Reply(chain.client.NewMessage("", types.EventReplyParaTxByTitle, reply))
}

//getHeightByTitle //获取拥有此title交易的区块高度
func (chain *BlockChain) getHeightByTitle(msg *queue.Message) {
	req := (msg.Data).(*types.ReqHeightByTitle)
	reply, err := chain.LoadParaTxByTitle(req)
	if err != nil {
		chainlog.Error("getHeightByTitle", "req", req, "err", err.Error())
		msg.Reply(chain.client.NewMessage("", types.EventReplyHeightByTitle, err))
		return
	}
	msg.Reply(chain.client.NewMessage("", types.EventReplyHeightByTitle, reply))
}

//getParaTxByTitleAndHeight //通过区块高度列表+title获取平行链交易
func (chain *BlockChain) getParaTxByTitleAndHeight(msg *queue.Message) {
	req := (msg.Data).(*types.ReqParaTxByHeight)
	reply, err := chain.GetParaTxByHeight(req)
	if err != nil {
		chainlog.Error("getParaTxByTitleAndHeight", "req", req, "err", err.Error())
		msg.Reply(chain.client.NewMessage("", types.EventReplyParaTxByTitle, err))
		return
	}
	msg.Reply(chain.client.NewMessage("", types.EventReplyParaTxByTitle, reply))
}

// getChunkRecord // 获取当前chunk record
func (chain *BlockChain) getChunkRecord(msg *queue.Message) {
	req := (msg.Data).(*types.ReqChunkRecords)
	reply, err := chain.GetChunkRecord(req)
	if err != nil {
		chainlog.Error("GetChunkRecord", "req", req, "err", err.Error())
		msg.Reply(chain.client.NewMessage("", types.EventGetChunkRecord, err))
		return
	}
	msg.Reply(chain.client.NewMessage("", types.EventGetChunkRecord, reply))
}

// addChunkRecord // 添加chunk record
func (chain *BlockChain) addChunkRecord(msg *queue.Message) {
	req := (msg.Data).(*types.ChunkRecords)
	chain.AddChunkRecord(req)
	msg.Reply(chain.client.NewMessage("", types.EventAddChunkRecord, &types.Reply{IsOk: true}))
}

// getChunkBlockBody // 获取chunk BlockBody
func (chain *BlockChain) getChunkBlockBody(msg *queue.Message) {
	req := (msg.Data).(*types.ReqChunkBlockBody)
	reply, err := chain.GetChunkBlockBody(req)
	if err != nil {
		chainlog.Error("GenChunkBlockBody", "req", req, "err", err.Error())
		msg.Reply(chain.client.NewMessage("", types.EventGetChunkBlockBody, err))
		return
	}
	msg.Reply(chain.client.NewMessage("", types.EventGetChunkBlockBody, reply))
}

// storeChunkBlockBody // 获取chunk BlockBody
func (chain *BlockChain) storeChunkBlockBody(msg *queue.Message) {
	req := (msg.Data).(*types.ChunkInfo)
	reply, err := chain.StoreChunkBlockBody(req)
	if err != nil {
		chainlog.Error("StoreChunkBlockBody", "req", req, "err", err.Error())
		msg.Reply(chain.client.NewMessage("", types.EventNotifyStoreChunk, err))
		return
	}
	msg.Reply(chain.client.NewMessage("", types.EventNotifyStoreChunk, reply))
}