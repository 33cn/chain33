package blockchain

//message callback
import (
	"time"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

func (chain *BlockChain) queryTx(msg queue.Message) {
	txhash := (msg.Data).(*types.ReqHash)
	TransactionDetail, err := chain.ProcQueryTxMsg(txhash.Hash)
	if err != nil {
		chainlog.Error("ProcQueryTxMsg", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetail, err))
	} else {
		chainlog.Info("ProcQueryTxMsg", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetail, TransactionDetail))
	}
}

func (chain *BlockChain) getBlocks(msg queue.Message) {
	requestblocks := (msg.Data).(*types.ReqBlocks)
	blocks, err := chain.ProcGetBlockDetailsMsg(requestblocks)
	if err != nil {
		chainlog.Error("ProcGetBlockDetailsMsg", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlocks, err))
	} else {
		chainlog.Info("ProcGetBlockDetailsMsg", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlocks, blocks))
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
		chain.notifySync()
	}
	chainlog.Info("EventAddBlock", "height", block.Height, "success", "ok")
	msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))
}

func (chain *BlockChain) getBlockHeight(msg queue.Message) {
	var replyBlockHeight types.ReplyBlockHeight
	replyBlockHeight.Height = chain.GetBlockHeight()
	chainlog.Info("EventGetBlockHeight", "success", "ok")
	msg.Reply(chain.qclient.NewMessage("consensus", types.EventReplyBlockHeight, &replyBlockHeight))
}

func (chain *BlockChain) txHashList(msg queue.Message) {
	txhashlist := (msg.Data).(*types.TxHashList)
	duptxhashlist := chain.GetDuplicateTxHashList(txhashlist)
	chainlog.Info("EventTxHashList", "success", "ok")
	msg.Reply(chain.qclient.NewMessage("consensus", types.EventTxHashListReply, duptxhashlist))
}

func (chain *BlockChain) getHeaders(msg queue.Message) {
	requestblocks := (msg.Data).(*types.ReqBlocks)
	headers, err := chain.ProcGetHeadersMsg(requestblocks)
	if err != nil {
		chainlog.Error("ProcGetHeadersMsg", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventHeaders, err))
	} else {
		chainlog.Info("EventGetHeaders", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventHeaders, headers))
	}
}

func (chain *BlockChain) getLastHeader(msg queue.Message) {
	header, err := chain.ProcGetLastHeaderMsg()
	if err != nil {
		chainlog.Error("ProcGetLastHeaderMsg", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("account", types.EventHeader, err))
	} else {
		chainlog.Info("EventGetLastHeader", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("account", types.EventHeader, header))
	}
	//本节点共识模块发送过来的blockdetail，需要广播到全网
}

func (chain *BlockChain) addBlockDetail(msg queue.Message) {
	var blockDetail *types.BlockDetail
	var reply types.Reply
	reply.IsOk = true
	blockDetail = msg.Data.(*types.BlockDetail)
	currentheight := chain.GetBlockHeight()
	//我们需要的高度，直接存储到db中
	if blockDetail.Block.Height != currentheight+1 {
		errmsg := "EventAddBlockDetail.Height not currentheight+1"
		chainlog.Error("EventAddBlockDetail", "err", errmsg)
		reply.IsOk = false
		reply.Msg = []byte(errmsg)
		msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))
		return
	}
	err := chain.ProcAddBlockMsg(true, blockDetail)
	if err != nil {
		chainlog.Error("ProcAddBlockMsg", "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
		chain.wg.Add(1)
		chain.SynBlockToDbOneByOne()
	}
	chainlog.Info("EventAddBlockDetail", "success", "ok")
	msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))

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
		chain.notifySync()
	}
	chainlog.Info("EventBroadcastAddBlock", "success", "ok")
	msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))
}

func (chain *BlockChain) getTransactionByAddr(msg queue.Message) {
	addr := (msg.Data).(*types.ReqAddr)
	chainlog.Warn("EventGetTransactionByAddr", "req", addr)
	replyTxInfos, err := chain.ProcGetTransactionByAddr(addr)
	if err != nil {
		chainlog.Error("ProcGetTransactionByAddr", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyTxInfo, err))
	} else {
		chainlog.Info("EventGetTransactionByAddr", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyTxInfo, replyTxInfos))
	}
}

func (chain *BlockChain) getTransactionByHashes(msg queue.Message) {
	txhashs := (msg.Data).(*types.ReqHashes)
	chainlog.Info("EventGetTransactionByHash", "hash", txhashs)
	TransactionDetails, err := chain.ProcGetTransactionByHashes(txhashs.Hashes)
	if err != nil {
		chainlog.Error("ProcGetTransactionByHashes", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetails, err))
	} else {
		chainlog.Info("EventGetTransactionByHash", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetails, TransactionDetails))
	}
}

func (chain *BlockChain) getBlockOverview(msg queue.Message) {
	ReqHash := (msg.Data).(*types.ReqHash)
	BlockOverview, err := chain.ProcGetBlockOverview(ReqHash)
	if err != nil {
		chainlog.Error("ProcGetBlockOverview", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyBlockOverview, err))
	} else {
		chainlog.Info("ProcGetBlockOverview", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyBlockOverview, BlockOverview))
	}
}

func (chain *BlockChain) getAddrOverview(msg queue.Message) {
	addr := (msg.Data).(*types.ReqAddr)
	AddrOverview, err := chain.ProcGetAddrOverview(addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyAddrOverview, err))
	} else {
		chainlog.Info("ProcGetAddrOverview", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyAddrOverview, AddrOverview))
	}
}

func (chain *BlockChain) getBlockHash(msg queue.Message) {
	height := (msg.Data).(*types.ReqInt)
	replyhash, err := chain.ProcGetBlockHash(height)
	if err != nil {
		chainlog.Error("ProcGetBlockHash", "err", err.Error())
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlockHash, err))
	} else {
		chainlog.Info("ProcGetBlockHash", "success", "ok")
		msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlockHash, replyhash))
	}
}

func (chain *BlockChain) localGet(msg queue.Message) {
	keys := (msg.Data).(*types.LocalDBGet)
	values := chain.blockStore.Get(keys)
	chainlog.Debug("localget", "success", "ok")
	msg.Reply(chain.qclient.NewMessage("rpc", types.EventLocalReplyValue, values))
}

type funcProcess func(msg queue.Message)

func (chain *BlockChain) processMsg(msg queue.Message, reqnum chan struct{}, cb funcProcess) {
	beg := time.Now()
	defer func() {
		<-reqnum
		chain.recvwg.Done()
		chainlog.Info("process", "cost", time.Now().Sub(beg), "msg", types.GetEventName(int(msg.Ty)))
	}()
	cb(msg)
}
