package blockchain

import (
	"container/list"
	"errors"
	"fmt"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/util"
	log "github.com/inconshreveable/log15"
)

var (
	//cache 存贮的block个数
	DefCacheSize int64 = 500

	//一次最多申请获取block个数
	MaxFetchBlockNum int64 = 100
	TimeoutSeconds   int64 = 5
	BatchBlockNum    int64 = 100
	blockSynSeconds        = time.Duration(TimeoutSeconds)
)

var chainlog = log.New("module", "blockchain")

var cachelock sync.Mutex
var castlock sync.Mutex
var synBlocklock sync.Mutex
var peerMaxBlklock sync.Mutex

const poolToDbMS = 500          //从blockpool写block到db的定时器
const fetchPeerListSeconds = 30 //获取一次peerlist最新block高度

type BlockChain struct {
	qclient queue.IClient
	q       *queue.Queue
	// 永久存储数据到db中
	blockStore *BlockStore

	//cache  缓存block方便快速查询
	cache      map[int64]*list.Element
	cacheSize  int64
	cacheQueue *list.List

	//Block 同步阶段用于缓存block信息，
	blockPool *BlockPool

	//记录收到的最新广播的block高度,用于节点追赶active链
	rcvLastBlockHeight int64

	//记录本节点已经同步的block高度,用于节点追赶active链
	synBlockHeight int64

	//记录peer的最新block高度,用于节点追赶active链
	peerMaxBlkHeight int64

	recvdone  chan struct{}
	poolclose chan struct{}
	quit      chan struct{}
}

func New(cfg *types.BlockChain) *BlockChain {

	//初始化blockstore 和txindex  db
	blockStoreDB := dbm.NewDB("blockchain", cfg.Driver, cfg.DbPath)
	blockStore := NewBlockStore(blockStoreDB)
	initConfig(cfg)
	pool := NewBlockPool()

	return &BlockChain{
		blockStore:         blockStore,
		cache:              make(map[int64]*list.Element),
		cacheSize:          DefCacheSize,
		cacheQueue:         list.New(),
		blockPool:          pool,
		rcvLastBlockHeight: -1,
		synBlockHeight:     -1,
		peerMaxBlkHeight:   -1,
		recvdone:           make(chan struct{}, 0),
		poolclose:          make(chan struct{}, 0),
		quit:               make(chan struct{}, 0),
	}
}

func initConfig(cfg *types.BlockChain) {
	if cfg.DefCacheSize > 0 {
		DefCacheSize = cfg.DefCacheSize
	}

	if cfg.MaxFetchBlockNum > 0 {
		MaxFetchBlockNum = cfg.MaxFetchBlockNum
	}

	if cfg.TimeoutSeconds > 0 {
		TimeoutSeconds = cfg.TimeoutSeconds
	}

	if cfg.BatchBlockNum > 0 {
		BatchBlockNum = cfg.BatchBlockNum
	}
}

func (chain *BlockChain) Close() {
	//wait recv done
	chain.qclient.Close()
	<-chain.recvdone
	//退出线程
	close(chain.quit)
	//wait
	chain.blockStore.db.Close()
	chainlog.Info("blockchain module closed")
}

func (chain *BlockChain) SetQueue(q *queue.Queue) {
	chain.qclient = q.GetClient()
	chain.qclient.Sub("blockchain")
	chain.q = q
	//recv 消息的处理
	go chain.ProcRecvMsg()

	// 定时同步缓存中的block to db
	go chain.poolRoutine()
}

func (chain *BlockChain) ProcRecvMsg() {
	defer func() {
		chain.recvdone <- struct{}{}
	}()
	for msg := range chain.qclient.Recv() {
		chainlog.Info("blockchain recv", "msg", msg)
		msgtype := msg.Ty
		switch msgtype {

		case types.EventQueryTx:
			txhash := (msg.Data).(*types.ReqHash)
			TransactionDetail, err := chain.ProcQueryTxMsg(txhash.Hash)
			if err != nil {
				chainlog.Error("ProcQueryTxMsg", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetail, err))
			} else {
				chainlog.Info("ProcQueryTxMsg", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetail, TransactionDetail))
			}

		case types.EventGetBlocks:
			requestblocks := (msg.Data).(*types.ReqBlocks)
			blocks, err := chain.ProcGetBlockDetailsMsg(requestblocks)
			if err != nil {
				chainlog.Error("ProcGetBlockDetailsMsg", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlocks, err))
			} else {
				chainlog.Info("ProcGetBlockDetailsMsg", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlocks, blocks))
			}

		case types.EventAddBlock: // block
			var block *types.Block
			var reply types.Reply
			reply.IsOk = true
			block = msg.Data.(*types.Block)
			err := chain.ProcAddBlockMsg(false, block)
			if err != nil {
				chainlog.Error("ProcAddBlockMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			chainlog.Info("EventAddBlock", "success", "ok")
			msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))

		case types.EventAddBlocks: //block
			var blocks *types.Blocks
			blocks = msg.Data.(*types.Blocks)
			err := chain.ProcAddBlocksMsg(blocks)
			if err != nil {
				chainlog.Error("ProcAddBlocksMsg", "err", err.Error())
			}
			chainlog.Info("EventAddBlocks", "success", "ok")

		case types.EventGetBlockHeight:
			var replyBlockHeight types.ReplyBlockHeight
			replyBlockHeight.Height = chain.GetBlockHeight()
			chainlog.Info("EventGetBlockHeight", "success", "ok")
			msg.Reply(chain.qclient.NewMessage("consensus", types.EventReplyBlockHeight, &replyBlockHeight))

		case types.EventTxHashList:
			txhashlist := (msg.Data).(*types.TxHashList)
			duptxhashlist := chain.GetDuplicateTxHashList(txhashlist)
			chainlog.Info("EventTxHashList", "success", "ok")
			msg.Reply(chain.qclient.NewMessage("consensus", types.EventTxHashListReply, duptxhashlist))

		case types.EventGetHeaders:
			requestblocks := (msg.Data).(*types.ReqBlocks)
			headers, err := chain.ProcGetHeadersMsg(requestblocks)
			if err != nil {
				chainlog.Error("ProcGetHeadersMsg", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventHeaders, err))
			} else {
				chainlog.Info("EventGetHeaders", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventHeaders, headers))
			}

		case types.EventGetLastHeader:
			header, err := chain.ProcGetLastHeaderMsg()
			if err != nil {
				chainlog.Error("ProcGetLastHeaderMsg", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("account", types.EventHeader, err))
			} else {
				chainlog.Info("EventGetLastHeader", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("account", types.EventHeader, header))
			}
			//本节点共识模块发送过来的blockdetail，需要广播到全网
		case types.EventAddBlockDetail:
			var blockDetail *types.BlockDetail
			var reply types.Reply
			reply.IsOk = true
			blockDetail = msg.Data.(*types.BlockDetail)
			err := chain.ProcAddBlockDetail(true, blockDetail)
			if err != nil {
				chainlog.Error("ProcAddBlockDetailMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			chainlog.Info("EventAddBlockDetail", "success", "ok")
			msg.Reply(chain.qclient.NewMessage("consensus", types.EventReply, &reply))

		case types.EventGetTransactionByAddr:
			addr := (msg.Data).(*types.ReqAddr)
			replyTxInfos, err := chain.ProcGetTransactionByAddr(addr.Addr)
			if err != nil {
				chainlog.Error("ProcGetTransactionByAddr", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyTxInfo, err))
			} else {
				chainlog.Info("EventGetTransactionByAddr", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyTxInfo, replyTxInfos))
			}

		case types.EventGetTransactionByHash:
			txhashs := (msg.Data).(*types.ReqHashes)
			TransactionDetails, err := chain.ProcGetTransactionByHashes(txhashs.Hashes)
			if err != nil {
				chainlog.Error("ProcGetTransactionByHashes", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetails, err))
			} else {
				chainlog.Info("EventGetTransactionByHash", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetails, TransactionDetails))
			}

			//收到p2p广播过来的block，如果刚好是我们期望的就添加到db并广播到全网
		case types.EventBroadcastAddBlock: //block
			var block *types.Block
			var reply types.Reply
			reply.IsOk = true
			block = msg.Data.(*types.Block)
			err := chain.ProcAddBlockMsg(true, block)
			if err != nil {
				chainlog.Error("ProcAddBlockMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			chainlog.Info("EventBroadcastAddBlock", "success", "ok")
			msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))

		case types.EventGetBlockOverview: //BlockOverview
			ReqHash := (msg.Data).(*types.ReqHash)
			BlockOverview, err := chain.ProcGetBlockOverview(ReqHash)
			if err != nil {
				chainlog.Error("ProcGetBlockOverview", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyBlockOverview, err))
			} else {
				chainlog.Info("ProcGetBlockOverview", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyBlockOverview, BlockOverview))
			}
		case types.EventGetAddrOverview: //AddrOverview
			addr := (msg.Data).(*types.ReqAddr)
			AddrOverview, err := chain.ProcGetAddrOverview(addr)
			if err != nil {
				chainlog.Error("ProcGetAddrOverview", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyAddrOverview, err))
			} else {
				chainlog.Info("ProcGetAddrOverview", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventReplyAddrOverview, AddrOverview))
			}
		case types.EventGetBlockHash: //GetBlockHash
			height := (msg.Data).(*types.ReqInt)
			replyhash, err := chain.ProcGetBlockHash(height)
			if err != nil {
				chainlog.Error("ProcGetBlockHash", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlockHash, err))
			} else {
				chainlog.Info("ProcGetBlockHash", "success", "ok")
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlockHash, replyhash))
			}
		default:
			chainlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
	}
}

func (chain *BlockChain) poolRoutine() {
	//获取peerlist的定时器，默认1分钟
	fetchPeerListTicker := time.NewTicker(fetchPeerListSeconds * time.Second)
	//向peer请求同步block的定时器，默认5s
	blockSynTicker := time.NewTicker(time.Duration(blockSynSeconds) * time.Second)

FOR_LOOP:
	for {
		select {
		case <-chain.quit:
			//chainlog.Info("quit poolRoutine!")
			return
		case _ = <-blockSynTicker.C:
			//chainlog.Info("blockSynTicker")
			go chain.SynBlocksFromPeers()

		case _ = <-fetchPeerListTicker.C:
			//chainlog.Info("blockUpdateTicker")
			go chain.FetchPeerList()

		case <-chain.blockPool.synblock:
			//chainlog.Info("trySyncTicker")
			// 定时同步缓存中的block信息到db数据库中
			newbatch := chain.blockStore.NewBatch(true)
			var stratblockheight int64 = 0
			var endblockheight int64 = 0
			var prevHash []byte
			// 可以批量处理BatchBlockNum个block到db中
			currentheight := chain.blockStore.Height()
			if currentheight == -1 {
				prevHash = common.Hash{}.Bytes()
			} else {
				curblock, err := chain.GetBlock(currentheight)
				if err != nil {
					continue FOR_LOOP
				}
				prevHash = curblock.Block.StateHash
			}
			prevheight := currentheight
			for {
				block := chain.blockPool.GetBlock(prevheight + 1)

				//需要加载的第一个nextblock不存在，退出for循环进入下一个超时
				if block == nil && prevheight == currentheight {
					//chainlog.Info("trySyncTicker continue FOR_LOOP")
					continue FOR_LOOP
				}
				// 缓存中连续的block数小于BatchBlockNum时，同步现有的block到db中
				if block == nil {
					//chainlog.Info("trySyncTicker block is nil")
					break
				}

				blockdetail, err := util.ExecBlock(chain.q, prevHash, block, true)
				if err != nil {
					chainlog.Info("trySyncTicker ExecBlock is err!", "height", block.Height, "err", err)
					//block校验不过需要从blockpool中删除，重新发起请求
					chain.blockPool.DelBlock(block.GetHeight())
					go chain.FetchBlock(block.GetHeight(), block.GetHeight())
					break
				}

				//保存tx信息到db中
				err = chain.blockStore.indexTxs(newbatch, blockdetail)
				if err != nil {
					chainlog.Info("trySyncTicker indexTxs err", "height", block.Height, "err", err)
					break
				}

				//保存block信息到db中
				err = chain.blockStore.SaveBlock(newbatch, blockdetail)
				if err != nil {
					chainlog.Info("trySyncTicker SaveBlock is err")
					break
				}

				//将已经存储的blocks添加到list缓存中便于查找
				chain.cacheBlock(blockdetail)

				prevHash = block.StateHash

				//记录同步开始的第一个block高度和最后一个block的高度，用于同步完成之后删除缓存中的block记录
				if prevheight == currentheight {
					stratblockheight = block.Height
				}
				endblockheight = blockdetail.Block.Height
				prevheight = blockdetail.Block.Height
			}
			newbatch.Write()

			//更新db中的blockheight到blockStore.Height
			chain.blockStore.UpdateHeight()

			//删除缓存中的block
			for j := stratblockheight; j <= endblockheight; j++ {
				//block := chain.blockPool.GetBlock(j)
				chain.blockPool.DelBlock(j)
				block, err := chain.GetBlock(j)
				if block != nil && err == nil {
					//通知mempool和consense模块
					chain.SendAddBlockEvent(block)
				}
			}
			continue FOR_LOOP
		}
	}
}

/*
函数功能：
EventQueryTx(types.ReqHash) : rpc模块会向 blockchain 模块 发送 EventQueryTx(types.ReqHash) 消息 ，
查询交易的默克尔树，回复消息 EventTransactionDetail(types.TransactionDetail)
结构体：
type ReqHash struct {Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`}
type TransactionDetail struct {Hashs [][]byte `protobuf:"bytes,1,rep,name=hashs,proto3" json:"hashs,omitempty"}
*/
func (chain *BlockChain) ProcQueryTxMsg(txhash []byte) (proof *types.TransactionDetail, err error) {
	txresult, err := chain.GetTxResultFromDb(txhash)
	if err != nil {
		return nil, err
	}
	block, err := chain.GetBlock(txresult.Height)
	if err != nil {
		return nil, err
	}
	var TransactionDetail types.TransactionDetail
	//获取指定tx在txlist中的proof
	proofs, err := GetTransactionProofs(block.Block.Txs, txresult.Index)
	if err != nil {
		return nil, err
	}

	TransactionDetail.Proofs = proofs
	chainlog.Debug("ProcQueryTxMsg", "proofs", TransactionDetail.Proofs)
	TransactionDetail.Receipt = txresult.Receiptdate
	TransactionDetail.Tx = txresult.GetTx()
	TransactionDetail.Height = txresult.GetHeight()
	TransactionDetail.Index = int64(txresult.GetIndex())
	TransactionDetail.Blocktime = txresult.GetBlocktime()

	//获取Amount
	var action types.CoinsAction
	err = types.Decode(txresult.GetTx().GetPayload(), &action)
	if err != nil {
		chainlog.Error("ProcQueryTxMsg Decode err!", "Height", txresult.GetHeight(), "txindex", txresult.GetIndex(), "err", err)
	}
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		TransactionDetail.Amount = transfer.Amount
	}
	//获取from地址
	pubkey := txresult.GetTx().Signature.GetPubkey()
	addr := account.PubKeyToAddress(pubkey)
	TransactionDetail.Fromaddr = addr.String()

	chainlog.Debug("ProcQueryTxMsg", "TransactionDetail", TransactionDetail.String())

	return &TransactionDetail, nil
}

func (chain *BlockChain) GetDuplicateTxHashList(txhashlist *types.TxHashList) (duptxhashlist *types.TxHashList) {

	var dupTxHashList types.TxHashList

	for _, txhash := range txhashlist.Hashes {
		txresult, err := chain.GetTxResultFromDb(txhash)
		if err == nil && txresult != nil {
			dupTxHashList.Hashes = append(dupTxHashList.Hashes, txhash)
			//chainlog.Debug("GetDuplicateTxHashList txresult", "height", txresult.Height, "index", txresult.Index)
			//chainlog.Debug("GetDuplicateTxHashList txresult  tx", "txinfo", txresult.Tx.String())
		}
	}
	return &dupTxHashList
}

/*
EventGetBlocks(types.RequestGetBlock): rpc 模块 会向 blockchain 模块发送 EventGetBlocks(types.RequestGetBlock) 消息，
功能是查询 区块的信息, 回复消息是 EventBlocks(types.Blocks)
type ReqBlocks struct {
	Start int64 `protobuf:"varint,1,opt,name=start" json:"start,omitempty"`
	End   int64 `protobuf:"varint,2,opt,name=end" json:"end,omitempty"`}
type Blocks struct {Items []*Block `protobuf:"bytes,1,rep,name=items" json:"items,omitempty"`}
*/
func (chain *BlockChain) ProcGetBlockDetailsMsg(requestblock *types.ReqBlocks) (respblocks *types.BlockDetails, err error) {
	blockhight := chain.GetBlockHeight()
	if requestblock.Start > blockhight {
		outstr := fmt.Sprintf("input Start height :%d  but current height:%d", requestblock.Start, blockhight)
		err = errors.New(outstr)
		return nil, err
	}

	chainlog.Debug("ProcGetBlocksMsg", "Start", requestblock.Start, "End", requestblock.End, "Isdetail", requestblock.Isdetail)

	end := requestblock.End
	if requestblock.End > blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetBlocksMsg", "blockscount", count)

	var blocks types.BlockDetails
	blocks.Items = make([]*types.BlockDetail, count)
	j := 0
	for i := start; i <= end; i++ {
		block, err := chain.GetBlock(i)
		if err == nil && block != nil {
			if requestblock.Isdetail {
				blocks.Items[j] = block
			} else {
				var blockdetail types.BlockDetail
				blockdetail.Block = block.Block
				blockdetail.Receipts = nil
				blocks.Items[j] = &blockdetail
			}
		} else {
			return nil, err
		}
		j++
	}
	//print
	if requestblock.Isdetail {
		for _, blockinfo := range blocks.Items {
			chainlog.Debug("ProcGetBlocksMsg", "blockinfo", blockinfo.String())
		}
	}
	return &blocks, nil
}

//处理从peer对端同步过来的block消息
func (chain *BlockChain) ProcAddBlockMsg(broadcast bool, block *types.Block) (err error) {
	if block == nil {
		err = errors.New("ProcAddBlockMsg input block is null")
		return err
	}
	currentheight := chain.GetBlockHeight()
	//不是我们需要的高度直接返回
	if currentheight >= block.Height {
		outstr := fmt.Sprintf("input add height :%d ,current store height:%d", block.Height, currentheight)
		err = errors.New(outstr)
		return err
	} else if block.Height == currentheight+1 { //我们需要的高度，直接存储到db中
		//通过前一个block的statehash来执行此block
		var prevstateHash []byte
		if currentheight == -1 {
			prevstateHash = common.Hash{}.Bytes()
		} else {
			prvblock, err := chain.GetBlock(currentheight)
			if err != nil {
				chainlog.Error("ProcAddBlockDetailMsg", "err", err)
				return err
			}
			prevstateHash = prvblock.Block.StateHash
		}
		blockDetail, err := util.ExecBlock(chain.q, prevstateHash, block, true)
		if err != nil {
			chainlog.Error("ProcAddBlockMsg ExecBlock err!", "err", err)
			return err
		}

		newbatch := chain.blockStore.NewBatch(true)
		//保存tx交易结果信息到db中
		chain.blockStore.indexTxs(newbatch, blockDetail)
		if err != nil {
			return err
		}

		//保存block信息到db中
		err = chain.blockStore.SaveBlock(newbatch, blockDetail)
		if err != nil {
			return err
		}
		newbatch.Write()

		//更新db中的blockheight到blockStore.Height
		chain.blockStore.UpdateHeight()

		//将此block添加到缓存中便于查找
		chain.cacheBlock(blockDetail)

		//通知mempool和consense模块
		chain.SendAddBlockEvent(blockDetail)

		//广播此block到网络中
		if broadcast {
			chain.SendBlockBroadcast(blockDetail)

			//更新广播block的高度
			castblockheight := chain.GetRcvLastCastBlkHeight()
			if castblockheight < blockDetail.Block.Height {
				chain.UpdateRcvCastBlkHeight(blockDetail.Block.Height)
			}
		}
		return nil
	} else {
		// 首先将此block缓存到blockpool中。
		chain.blockPool.AddBlock(block)

		defer func() {
			chain.blockPool.synblock <- struct{}{}
		}()

		//更新广播block的高度
		if broadcast {
			castblockheight := chain.GetRcvLastCastBlkHeight()
			if castblockheight < block.Height {
				chain.UpdateRcvCastBlkHeight(block.Height)
			}
		}
	}
	return nil
}

//处理共识模块发过来addblock的消息，需要广播到全网
func (chain *BlockChain) ProcAddBlockDetail(broadcast bool, blockDetail *types.BlockDetail) (err error) {
	currentheight := chain.GetBlockHeight()

	//我们需要的高度，直接存储到db中
	if blockDetail.Block.Height == currentheight+1 {
		newbatch := chain.blockStore.NewBatch(true)

		//保存tx交易结果信息到db中
		chain.blockStore.indexTxs(newbatch, blockDetail)
		if err != nil {
			chainlog.Error("ProcAddBlockDetail", "err", err)
			return err
		}

		//保存block信息到db中
		err := chain.blockStore.SaveBlock(newbatch, blockDetail)
		if err != nil {
			chainlog.Error("ProcAddBlockDetail", "err", err)
			return err
		}
		newbatch.Write()

		//更新db中的blockheight到blockStore.Height
		chain.blockStore.UpdateHeight()

		//将此block添加到缓存中便于查找
		chain.cacheBlock(blockDetail)

		//通知mempool和consense以及wallet模块
		chain.SendAddBlockEvent(blockDetail)

		//广播此block到网络中
		chain.SendBlockBroadcast(blockDetail)

		return nil
	} else {
		outstr := fmt.Sprintf("input height :%d ,current store height:%d", blockDetail.Block.Height, currentheight)
		err = errors.New(outstr)
		chainlog.Error("ProcAddBlockDetail", "err", err)
		return err
	}
	return nil
}

/*
函数功能：
通过向P2P模块送 EventFetchBlock(types.RequestGetBlock)，向其他节点主动请求区块，
P2P区块收到这个消息后，会向blockchain 模块回复， EventReply。
其他节点如果有这个范围的区块，P2P模块收到其他节点发来的数据，
会发送送EventAddBlocks(types.Blocks) 给 blockchain 模块，
blockchain 模块回复 EventReply
结构体：
*/
//func (chain *BlockChain) FetchBlock(reqblk *types.ReqBlocks) (err error) {
func (chain *BlockChain) FetchBlock(start int64, end int64) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}

	chainlog.Debug("FetchBlock input", "StartHeight", start, "EndHeight", end)
	blockcount := end - start

	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.Isdetail = false

	if blockcount > MaxFetchBlockNum {
		requestblock.End = start + MaxFetchBlockNum
	} else {
		requestblock.End = end
	}
	chainlog.Debug("FetchBlock", "Start", requestblock.Start, "End", requestblock.End)
	msg := chain.qclient.NewMessage("p2p", types.EventFetchBlocks, &requestblock)
	chain.qclient.Send(msg, true)
	resp, err := chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("FetchBlock", "qclient.Wait err:", err)
		return err
	}
	return resp.Err()
}

//blockchain 模块add block到db之后通知mempool 和consense模块做相应的更新
func (chain *BlockChain) SendAddBlockEvent(block *types.BlockDetail) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}
	if block == nil {
		chainlog.Error("SendAddBlockEvent block is null")
		return nil
	}
	chainlog.Debug("SendAddBlockEvent", "Height", block.Block.Height)

	chainlog.Debug("SendAddBlockEvent -->>mempool")
	msg := chain.qclient.NewMessage("mempool", types.EventAddBlock, block)
	chain.qclient.Send(msg, false)

	chainlog.Debug("SendAddBlockEvent -->>consensus")

	msg = chain.qclient.NewMessage("consensus", types.EventAddBlock, block)
	chain.qclient.Send(msg, false)

	chainlog.Debug("SendAddBlockEvent -->>wallet")
	msg = chain.qclient.NewMessage("wallet", types.EventAddBlock, block)
	chain.qclient.Send(msg, false)

	return nil
}

//blockchain模块广播此block到网络中
func (chain *BlockChain) SendBlockBroadcast(block *types.BlockDetail) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		return
	}
	if block == nil {
		chainlog.Error("SendBlockBroadcast block is null")
		return
	}
	chainlog.Debug("SendBlockBroadcast", "Height", block.Block.Height)

	msg := chain.qclient.NewMessage("p2p", types.EventBlockBroadcast, block.Block)
	chain.qclient.Send(msg, false)
	return
}

//处理从peer对端同步过来的blocks
//首先缓存到pool中,由poolRoutine定时同步到db中,blocks太多此时写入db会耗时很长
func (chain *BlockChain) ProcAddBlocksMsg(blocks *types.Blocks) (err error) {
	defer func() {
		chain.blockPool.synblock <- struct{}{}
	}()

	var preheight int64
	chainlog.Debug("ProcAddBlocksMsg", "blockcount", len(blocks.Items))
	//我们只处理连续的block，不连续时直接忽略掉
	for index, block := range blocks.Items {
		if index == 0 {
			preheight = block.GetHeight()
			chain.blockPool.AddBlock(block)
		} else if preheight+1 == block.GetHeight() {
			preheight = block.GetHeight()
			chain.blockPool.AddBlock(block)
		} else {
			chainlog.Error("ProcAddBlocksMsg Height is not continuous", "preheight", preheight, "nextheight", block.GetHeight())
			break
		}
	}
	return nil
}

func (chain *BlockChain) GetBlockHeight() int64 {
	return chain.blockStore.Height()
}

//用于获取指定高度的block，首先在缓存中获取，如果不存在就从db中获取
func (chain *BlockChain) GetBlock(height int64) (block *types.BlockDetail, err error) {

	blockdetail := chain.CheckcacheBlock(height)
	if blockdetail != nil {
		if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
			chainlog.Debug("GetBlock  CheckcacheBlock Receipts ==0", "height", height)
		}
		return blockdetail, nil
	} else {
		//从blockstore db中通过block height获取block
		blockinfo := chain.blockStore.LoadBlock(height)
		if blockinfo != nil {
			if len(blockinfo.Receipts) == 0 && len(blockinfo.Block.Txs) != 0 {
				chainlog.Debug("GetBlock  LoadBlock Receipts ==0", "height", height)
			}
			chain.cacheBlock(blockinfo)
			return blockinfo, nil
		}
	}
	err = errors.New("GetBlock error")
	return nil, err
}

//从cache缓存中获取block信息
func (chain *BlockChain) CheckcacheBlock(height int64) (block *types.BlockDetail) {
	cachelock.Lock()
	defer cachelock.Unlock()

	elem, ok := chain.cache[height]
	if ok {
		// Already exists. Move to back of cacheQueue.
		chain.cacheQueue.MoveToBack(elem)
		return elem.Value.(*types.BlockDetail)
	}
	return nil
}

//添加block到cache中，方便快速查询
func (chain *BlockChain) cacheBlock(blockdetail *types.BlockDetail) {
	cachelock.Lock()
	defer cachelock.Unlock()

	if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
		chainlog.Debug("cacheBlock  Receipts ==0", "height", blockdetail.Block.GetHeight())
	}

	// Create entry in cache and append to cacheQueue.
	elem := chain.cacheQueue.PushBack(blockdetail)
	chain.cache[blockdetail.Block.Height] = elem

	// Maybe expire an item.
	if int64(chain.cacheQueue.Len()) > chain.cacheSize {
		height := chain.cacheQueue.Remove(chain.cacheQueue.Front()).(*types.BlockDetail).Block.Height
		delete(chain.cache, height)
	}
}

//通过txhash 从txindex db中获取tx信息
//type TxResult struct {
//	Height int64
//	Index  int32
//	Tx     *types.Transaction
//  Receiptdate *ReceiptData
//}

func (chain *BlockChain) GetTxResultFromDb(txhash []byte) (tx *types.TxResult, err error) {

	txinfo, err := chain.blockStore.GetTx(txhash)
	if err != nil {
		return nil, err
	}
	return txinfo, nil
}

//  获取指定txindex  在txs中的TransactionDetail ，注释：index从0开始
func GetTransactionProofs(Txs []*types.Transaction, index int32) ([][]byte, error) {
	txlen := len(Txs)

	//计算tx的hash值
	leaves := make([][]byte, txlen)
	for index, tx := range Txs {
		leaves[index] = tx.Hash()
		//chainlog.Info("GetTransactionDetail txhash", "index", index, "txhash", tx.Hash())
	}

	proofs := merkle.GetMerkleBranch(leaves, uint32(index))
	chainlog.Debug("GetTransactionDetail proofs", "proofs", proofs)

	return proofs, nil
}

//type Header struct {
//	Version    int64
//	ParentHash []byte
//	TxHash     []byte
//	Height     int64
//	BlockTime  int64
//}
func (chain *BlockChain) ProcGetHeadersMsg(requestblock *types.ReqBlocks) (respheaders *types.Headers, err error) {
	blockhight := chain.GetBlockHeight()
	if requestblock.Start > blockhight {
		outstr := fmt.Sprintf("input Start height :%d  but current height:%d", requestblock.Start, blockhight)
		err = errors.New(outstr)
		return nil, err
	}
	end := requestblock.End
	if requestblock.End > blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetBlocksMsg", "blockscount", count)

	var headers types.Headers
	headers.Items = make([]*types.Header, count)
	j := 0
	for i := start; i <= end; i++ {
		blockdetail, err := chain.GetBlock(i)
		if err == nil && blockdetail != nil {
			//获取header的信息从block中
			head := &types.Header{}
			head.Version = blockdetail.Block.Version
			head.ParentHash = blockdetail.Block.ParentHash
			head.TxHash = blockdetail.Block.TxHash
			head.StateHash = blockdetail.Block.StateHash
			head.BlockTime = blockdetail.Block.BlockTime
			head.Height = blockdetail.Block.Height

			headers.Items[j] = head
		} else {
			return nil, err
		}
		j++
	}
	return &headers, nil
}

//type Header struct {
//	Version    int64
//	ParentHash []byte
//	TxHash     []byte
//	StateHash  []byte
//	Height     int64
//	BlockTime  int64
//}
func (chain *BlockChain) ProcGetLastHeaderMsg() (respheader *types.Header, err error) {
	head := &types.Header{}
	blockhight := chain.GetBlockHeight()
	blockdetail, err := chain.GetBlock(blockhight)
	if err == nil && blockdetail != nil {
		//获取header的信息从block中
		head.Version = blockdetail.Block.Version
		head.ParentHash = blockdetail.Block.ParentHash
		head.TxHash = blockdetail.Block.TxHash
		head.StateHash = blockdetail.Block.StateHash
		head.BlockTime = blockdetail.Block.BlockTime
		head.Height = blockdetail.Block.Height
	} else {
		return nil, err
	}
	return head, nil
}

func (chain *BlockChain) ProcGetBlockByHashMsg(hash []byte) (respblock *types.BlockDetail, err error) {
	blockhight := chain.blockStore.GetHeightByBlockHash(hash)
	blockdetail, err := chain.GetBlock(blockhight)
	if err != nil {
		return nil, err
	}
	return blockdetail, nil
}

//从p2p模块获取peerlist，用于获取active链上最新的高度。
//如果没有收到广播block就主动向p2p模块发送请求
func (chain *BlockChain) FetchPeerList() {
	if chain.qclient == nil {
		chainlog.Error("FetchPeerList chain client not bind message queue.")
		return
	}
	//chainlog.Info("FetchPeerList")
	msg := chain.qclient.NewMessage("p2p", types.EventPeerInfo, nil)
	chain.qclient.Send(msg, true)
	resp, err := chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("FetchPeerList", "qclient.Wait err:", err)
		return
	}
	peerlist := resp.GetData().(*types.PeerList)

	//获取peerlist中最新的高度
	var maxPeerHeight int64 = -1
	for _, peer := range peerlist.Peers {
		if peer != nil && maxPeerHeight < peer.Header.Height {
			maxPeerHeight = peer.Header.Height
		}
	}
	if maxPeerHeight != -1 {
		chain.UpdatePeerMaxBlkHeight(maxPeerHeight)
	}
}
func (chain *BlockChain) GetTxsListByAddr(addr []byte) (txs []*types.TxResult) {
	txinfos, err := chain.blockStore.GetTxsByAddr(addr)
	if err != nil {
		chainlog.Info("GetTxsListByAddr does not exist tx!", "addr", string(addr), "err", err)
		return nil
	}
	if txinfos == nil {
		return nil
	}
	for _, txinfo := range txinfos.TxInfos {
		txresult, err := chain.GetTxResultFromDb(txinfo.GetHash())
		if err == nil {
			txs = append(txs, txresult)
		}
		//chainlog.Info("GetTxsListByAddr txs", "txhash", tx)
	}
	return
}

//获取地址对应的所有交易信息
//存储格式key:addr:flag:height ,value:txhash
//key=addr :获取本地参与的所有交易
//key=addr:0 :获取本地作为from方的所有交易
//key=addr:1 :获取本地作为to方的所有交易
func (chain *BlockChain) ProcGetTransactionByAddr(addr string) (*types.ReplyTxInfos, error) {
	if len(addr) == 0 {
		err := errors.New("ProcGetTransactionByAddr addr is nil")
		return nil, err
	}
	txinfos, err := chain.blockStore.GetTxsByAddr([]byte(addr))
	if err != nil {
		chainlog.Info("ProcGetTransactionByAddr does not exist tx!", "addr", addr, "err", err)
		return nil, err
	}
	return txinfos, nil
}

//type TransactionDetails struct {
//	Txs []*Transaction
//}
//通过hashs获取交易详情
func (chain *BlockChain) ProcGetTransactionByHashes(hashs [][]byte) (TxDetails *types.TransactionDetails, err error) {

	//chainlog.Info("ProcGetTransactionByHashes", "txhash len:", len(hashs))
	var txDetails types.TransactionDetails

	txDetails.Txs = make([]*types.TransactionDetail, len(hashs))
	for index, txhash := range hashs {
		txresult, err := chain.GetTxResultFromDb(txhash)
		if err == nil && txresult != nil {
			var txDetail types.TransactionDetail
			txDetail.Receipt = txresult.Receiptdate
			txDetail.Tx = txresult.GetTx()
			txDetail.Blocktime = txresult.GetBlocktime()
			txDetail.Height = txresult.GetHeight()
			txDetail.Index = int64(txresult.GetIndex())

			//获取Amount
			var action types.CoinsAction
			err := types.Decode(txresult.GetTx().GetPayload(), &action)
			if err != nil {
				chainlog.Error("ProcGetTransactionByHashes Decode err!", "Height", txresult.GetHeight(), "txindex", txresult.GetIndex(), "err", err)
				continue
			}
			if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
				transfer := action.GetTransfer()
				txDetail.Amount = transfer.Amount
			}
			//获取from地址
			pubkey := txresult.GetTx().Signature.GetPubkey()
			addr := account.PubKeyToAddress(pubkey)
			txDetail.Fromaddr = addr.String()

			chainlog.Info("ProcGetTransactionByHashes", "txDetail", txDetail.String())
			txDetails.Txs[index] = &txDetail
		}
	}
	return &txDetails, nil
}

//存储广播的block最新高度
func (chain *BlockChain) GetRcvLastCastBlkHeight() int64 {
	castlock.Lock()
	defer castlock.Unlock()
	return chain.rcvLastBlockHeight
}
func (chain *BlockChain) UpdateRcvCastBlkHeight(height int64) {
	castlock.Lock()
	defer castlock.Unlock()
	chain.rcvLastBlockHeight = height
}

//存储已经通同步到db的block高度
func (chain *BlockChain) GetsynBlkHeight() int64 {
	synBlocklock.Lock()
	defer synBlocklock.Unlock()
	return chain.synBlockHeight
}
func (chain *BlockChain) UpdatesynBlkHeight(height int64) {
	synBlocklock.Lock()
	defer synBlocklock.Unlock()
	chain.synBlockHeight = height
}

//存储peer的最新block高度
func (chain *BlockChain) GetPeerMaxBlkHeight() int64 {
	peerMaxBlklock.Lock()
	defer peerMaxBlklock.Unlock()
	return chain.peerMaxBlkHeight
}
func (chain *BlockChain) UpdatePeerMaxBlkHeight(height int64) {
	peerMaxBlklock.Lock()
	defer peerMaxBlklock.Unlock()
	chain.peerMaxBlkHeight = height
}

//blockSynSeconds时间检测一次本节点的height是否有增长，没有增长就需要通过对端peerlist获取最新高度，发起同步
func (chain *BlockChain) SynBlocksFromPeers() {

	GetsynBlkHeight := chain.GetsynBlkHeight()
	curheight := chain.GetBlockHeight()
	RcvLastCastBlkHeight := chain.GetRcvLastCastBlkHeight()
	peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()

	chainlog.Info("SynBlocksFromPeers", "synBlkHeight", GetsynBlkHeight, "curheight", curheight)
	chainlog.Info("SynBlocksFromPeers", "LastCastBlkHeight", RcvLastCastBlkHeight, "peerMaxBlkHeight", curheight)

	//如果上个周期已经同步的block高度小于等于当前高度
	//说明本节点在这个周期没有收到block需要主动发起同步
	if curheight >= GetsynBlkHeight {
		//RcvLastCastBlkHeight := chain.GetRcvLastCastBlkHeight()
		//首先和广播的block高度做比较，小于广播的block高度就直接发送block同步的请求
		if curheight < RcvLastCastBlkHeight {
			chain.FetchBlock(curheight+1, RcvLastCastBlkHeight)
		} else {
			//大于等于广播的block高度时，需要获取peer的最新高度继续做校验
			//获取peers的最新高度.处理没有收到广播block的情况
			//peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()
			if curheight < peerMaxBlkHeight {
				chain.FetchBlock(curheight+1, peerMaxBlkHeight)
			}
		}
		chain.UpdatesynBlkHeight(curheight)
	} else {
		chainlog.Error("SynBlocksFromPeers", "synBlkHeight", GetsynBlkHeight, "curheight", curheight)
	}
	return
}

//type  BlockOverview {
//	Header head = 1;
//	int64  txCount = 2;
//	repeated bytes txHashes = 3;}
//获取BlockOverview
func (chain *BlockChain) ProcGetBlockOverview(ReqHash *types.ReqHash) (*types.BlockOverview, error) {

	if ReqHash == nil {
		err := errors.New("ProcGetBlockOverview input err!")
		return nil, err
	}
	//通过blockhash获取blockheight
	height := chain.blockStore.GetHeightByBlockHash(ReqHash.Hash)
	if height <= -1 {
		err := errors.New("ProcGetBlockOverview:GetHeightByBlockHash err")
		return nil, err
	}
	var blockOverview types.BlockOverview
	//通过height获取block
	block, err := chain.GetBlock(height)
	if err != nil || block == nil {
		chainlog.Error("ProcGetBlockOverview", "GetBlock err ", err)
		return nil, err
	}

	//获取header的信息从block中
	var header types.Header
	header.Version = block.Block.Version
	header.ParentHash = block.Block.ParentHash
	header.TxHash = block.Block.TxHash
	header.StateHash = block.Block.StateHash
	header.BlockTime = block.Block.BlockTime
	header.Height = block.Block.Height
	blockOverview.Head = &header

	blockOverview.TxCount = int64(len(block.Block.GetTxs()))

	txhashs := make([][]byte, blockOverview.TxCount)
	for index, tx := range block.Block.Txs {
		txhashs[index] = tx.Hash()
	}
	blockOverview.TxHashes = txhashs
	chainlog.Error("ProcGetBlockOverview", "blockOverview:", blockOverview.String())
	return &blockOverview, nil
}

//type  AddrOverview {
//	int64 reciver = 1;
//	int64 balance = 2;
//	int64 txCount = 3;}
//获取addrOverview
func (chain *BlockChain) ProcGetAddrOverview(addr *types.ReqAddr) (*types.AddrOverview, error) {

	if addr == nil || len(addr.Addr) == 0 {
		err := errors.New("ProcGetAddrOverview input err!")
		return nil, err
	}
	var addrOverview types.AddrOverview

	//获取地址的reciver
	amount, err := chain.blockStore.GetAddrReciver(addr.Addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrReciver err", err)
		return nil, err
	}
	addrOverview.Reciver = amount

	//获取地址的balance从account模块
	addrs := make([]string, 1)
	addrs[0] = addr.Addr
	accounts, err := account.LoadAccounts(chain.q, addrs)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "LoadAccounts err", err)
		return nil, err
	}
	if len(accounts) != 0 {
		addrOverview.Balance = accounts[0].Balance
	}

	//获取地址对应的交易count
	txinfos, err := chain.blockStore.GetTxsByAddr([]byte(addr.Addr))
	if err != nil {
		chainlog.Info("ProcGetAddrOverview", "GetTxsByAddr err", err)
		return nil, err
	}
	addrOverview.TxCount = int64(len(txinfos.GetTxInfos()))

	chainlog.Info("ProcGetAddrOverview", "addrOverview", addrOverview.String())

	return &addrOverview, nil
}

//通过blockheight 获取blockhash
func (chain *BlockChain) ProcGetBlockHash(height *types.ReqInt) (*types.ReplyHash, error) {
	if height == nil {
		err := errors.New("ProcGetBlockHash input err!")
		return nil, err
	}
	var ReplyHash types.ReplyHash
	block, err := chain.GetBlock(height.GetHeight())
	if err != nil {
		return nil, err
	}
	ReplyHash.Hash = block.Block.Hash()
	return &ReplyHash, nil
}
