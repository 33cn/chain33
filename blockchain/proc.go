package blockchain

//message callback
import (
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

//blockchain模块的消息接收处理
func (chain *BlockChain) ProcRecvMsg() {
	reqnum := make(chan struct{}, 1000)
	for msg := range chain.client.Recv() {
		chainlog.Debug("blockchain recv", "msg", types.GetEventName(int(msg.Ty)), "id", msg.Id, "cap", len(reqnum))
		msgtype := msg.Ty
		reqnum <- struct{}{}
		chain.recvwg.Add(1)
		switch msgtype {
		case types.EventLocalGet:
			go chain.processMsg(msg, reqnum, chain.localGet)
		case types.EventLocalList:
			go chain.processMsg(msg, reqnum, chain.localList)
		case types.EventQueryTx:
			go chain.processMsg(msg, reqnum, chain.queryTx)
		case types.EventGetBlocks:
			go chain.processMsg(msg, reqnum, chain.getBlocks)
		case types.EventAddBlock: // block
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
		case types.EventQuery:
			go chain.processMsg(msg, reqnum, chain.getQuery)
		case types.EventAddBlockHeaders:
			go chain.processMsg(msg, reqnum, chain.addBlockHeaders)
		case types.EventGetLastBlock:
			go chain.processMsg(msg, reqnum, chain.getLastBlock)
		case types.EventIsSync:
			go chain.processMsg(msg, reqnum, chain.isSync)
		case types.EventIsNtpClockSync:
			go chain.processMsg(msg, reqnum, chain.isNtpClockSync)
		default:
			<-reqnum
			chainlog.Warn("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
	}
}

func (chain *BlockChain) queryTx(msg queue.Message) {
	txhash := (msg.Data).(*types.ReqHash)
	TransactionDetail, err := chain.ProcQueryTxMsg(txhash.Hash)
	if err != nil {
		chainlog.Error("ProcQueryTxMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetail, err))
	} else {
		chainlog.Debug("ProcQueryTxMsg", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetail, TransactionDetail))
	}
}

func (chain *BlockChain) getBlocks(msg queue.Message) {
	requestblocks := (msg.Data).(*types.ReqBlocks)
	blocks, err := chain.ProcGetBlockDetailsMsg(requestblocks)
	if err != nil {
		chainlog.Error("ProcGetBlockDetailsMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlocks, err))
	} else {
		chainlog.Debug("ProcGetBlockDetailsMsg", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlocks, blocks))
	}
}

func (chain *BlockChain) addBlock(msg queue.Message) {
	var block *types.Block
	var reply types.Reply
	reply.IsOk = true
	block = msg.Data.(*types.Block)
	err := chain.ProcAddBlockMsg(false, &types.BlockDetail{Block: block})
	if err != nil {
		chainlog.Error("ProcAddBlockMsg", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
		//chain.notifySync()
	}
	chainlog.Debug("EventAddBlock", "height", block.Height, "success", "ok")
	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, &reply))
}

func (chain *BlockChain) getBlockHeight(msg queue.Message) {
	var replyBlockHeight types.ReplyBlockHeight
	replyBlockHeight.Height = chain.GetBlockHeight()
	chainlog.Debug("EventGetBlockHeight", "success", "ok")
	msg.Reply(chain.client.NewMessage("consensus", types.EventReplyBlockHeight, &replyBlockHeight))
}

func (chain *BlockChain) txHashList(msg queue.Message) {
	txhashlist := (msg.Data).(*types.TxHashList)
	duptxhashlist := chain.GetDuplicateTxHashList(txhashlist)
	chainlog.Debug("EventTxHashList", "success", "ok")
	msg.Reply(chain.client.NewMessage("consensus", types.EventTxHashListReply, duptxhashlist))
}

func (chain *BlockChain) getHeaders(msg queue.Message) {
	requestblocks := (msg.Data).(*types.ReqBlocks)
	headers, err := chain.ProcGetHeadersMsg(requestblocks)
	if err != nil {
		chainlog.Error("ProcGetHeadersMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventHeaders, err))
	} else {
		chainlog.Debug("EventGetHeaders", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventHeaders, headers))
	}
}

func (chain *BlockChain) isSync(msg queue.Message) {
	ok := chain.IsCaughtUp()
	msg.Reply(chain.client.NewMessage("", types.EventReplyIsSync, &types.IsCaughtUp{ok}))
}

func (chain *BlockChain) getLastHeader(msg queue.Message) {
	header, err := chain.ProcGetLastHeaderMsg()
	if err != nil {
		chainlog.Error("ProcGetLastHeaderMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("account", types.EventHeader, err))
	} else {
		chainlog.Debug("EventGetLastHeader", "success", "ok")
		msg.Reply(chain.client.NewMessage("account", types.EventHeader, header))
	}
	//本节点共识模块发送过来的blockdetail，需要广播到全网
}

func (chain *BlockChain) addBlockDetail(msg queue.Message) {
	var blockDetail *types.BlockDetail
	var reply types.Reply
	reply.IsOk = true
	blockDetail = msg.Data.(*types.BlockDetail)

	chainlog.Info("EventAddBlockDetail", "height", blockDetail.Block.Height, "hash", common.ToHex(blockDetail.Block.Hash()))

	err := chain.ProcAddBlockMsg(true, blockDetail)
	if err != nil {
		chainlog.Error("ProcAddBlockMsg", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
		//chain.wg.Add(1)
		//chain.SynBlockToDbOneByOne()
	}
	chainlog.Debug("EventAddBlockDetail", "success", "ok")
	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, &reply))

	//收到p2p广播过来的block，如果刚好是我们期望的就添加到db并广播到全网
}

func (chain *BlockChain) broadcastAddBlock(msg queue.Message) {
	var block *types.Block
	var reply types.Reply
	reply.IsOk = true
	block = msg.Data.(*types.Block)

	err := chain.ProcAddBlockMsg(true, &types.BlockDetail{Block: block})
	if err != nil {
		chainlog.Error("ProcAddBlockMsg", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
		//chain.notifySync()
	}
	chainlog.Debug("EventBroadcastAddBlock", "height", block.Height, "hash", common.ToHex(block.Hash()), "success", "ok")

	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, &reply))
}

func (chain *BlockChain) getTransactionByAddr(msg queue.Message) {
	addr := (msg.Data).(*types.ReqAddr)
	chainlog.Warn("EventGetTransactionByAddr", "req", addr)
	replyTxInfos, err := chain.ProcGetTransactionByAddr(addr)
	if err != nil {
		chainlog.Error("ProcGetTransactionByAddr", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyTxInfo, err))
	} else {
		chainlog.Debug("EventGetTransactionByAddr", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyTxInfo, replyTxInfos))
	}
}

func (chain *BlockChain) getTransactionByHashes(msg queue.Message) {
	txhashs := (msg.Data).(*types.ReqHashes)
	chainlog.Info("EventGetTransactionByHash", "hash", txhashs)
	TransactionDetails, err := chain.ProcGetTransactionByHashes(txhashs.Hashes)
	if err != nil {
		chainlog.Error("ProcGetTransactionByHashes", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetails, err))
	} else {
		chainlog.Debug("EventGetTransactionByHash", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventTransactionDetails, TransactionDetails))
	}
}

func (chain *BlockChain) getBlockOverview(msg queue.Message) {
	ReqHash := (msg.Data).(*types.ReqHash)
	BlockOverview, err := chain.ProcGetBlockOverview(ReqHash)
	if err != nil {
		chainlog.Error("ProcGetBlockOverview", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyBlockOverview, err))
	} else {
		chainlog.Debug("ProcGetBlockOverview", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyBlockOverview, BlockOverview))
	}
}

func (chain *BlockChain) getAddrOverview(msg queue.Message) {
	addr := (msg.Data).(*types.ReqAddr)
	AddrOverview, err := chain.ProcGetAddrOverview(addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyAddrOverview, err))
	} else {
		chainlog.Debug("ProcGetAddrOverview", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyAddrOverview, AddrOverview))
	}
}

func (chain *BlockChain) getBlockHash(msg queue.Message) {
	height := (msg.Data).(*types.ReqInt)
	replyhash, err := chain.ProcGetBlockHash(height)
	if err != nil {
		chainlog.Error("ProcGetBlockHash", "err", err.Error())
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlockHash, err))
	} else {
		chainlog.Debug("ProcGetBlockHash", "success", "ok")
		msg.Reply(chain.client.NewMessage("rpc", types.EventBlockHash, replyhash))
	}
}

func (chain *BlockChain) localGet(msg queue.Message) {
	keys := (msg.Data).(*types.LocalDBGet)
	values := chain.blockStore.Get(keys)
	msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, values))
}

func (chain *BlockChain) localList(msg queue.Message) {
	q := (msg.Data).(*types.LocalDBList)
	values := db.NewListHelper(chain.blockStore.db).List(q.Prefix, q.Key, q.Count, q.Direction)
	msg.Reply(chain.client.NewMessage("rpc", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))
}

func (chain *BlockChain) getQuery(msg queue.Message) {
	query := (msg.Data).(*types.Query)
	reply, err := chain.query.Query(string(query.Execer), query.FuncName, query.Payload)
	if err != nil {
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyQuery, err))
	} else {
		msg.Reply(chain.client.NewMessage("rpc", types.EventReplyQuery, reply))
	}
}

func (chain *BlockChain) addBlockHeaders(msg queue.Message) {
	var reply types.Reply
	reply.IsOk = true
	headers := msg.Data.(*types.Headers)
	err := chain.ProcAddBlockHeadersMsg(headers)
	if err != nil {
		chainlog.Error("addBlockHeaders", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
	}
	chainlog.Debug("addBlockHeaders", "success", "ok")
	msg.Reply(chain.client.NewMessage("p2p", types.EventReply, &reply))
}

func (chain *BlockChain) getLastBlock(msg queue.Message) {
	block, err := chain.ProcGetLastBlockMsg()
	if err != nil {
		chainlog.Error("ProcGetLastBlockMsg", "err", err.Error())
		msg.Reply(chain.client.NewMessage("consensus", types.EventBlock, err))
	} else {
		chainlog.Debug("ProcGetLastBlockMsg", "success", "ok")
		msg.Reply(chain.client.NewMessage("consensus", types.EventBlock, block))
	}
}

func (chain *BlockChain) isNtpClockSync(msg queue.Message) {
	ok := GetNtpClockSyncStatus()
	msg.Reply(chain.client.NewMessage("", types.EventReplyIsNtpClockSync, &types.IsNtpClockSync{ok}))
}

type funcProcess func(msg queue.Message)

func (chain *BlockChain) processMsg(msg queue.Message, reqnum chan struct{}, cb funcProcess) {
	beg := time.Now()
	defer func() {
		<-reqnum
		chain.recvwg.Done()
		chainlog.Debug("process", "cost", time.Since(beg), "msg", types.GetEventName(int(msg.Ty)))
	}()
	cb(msg)
}
