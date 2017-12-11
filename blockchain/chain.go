package blockchain

import (
	"container/list"
	"errors"
	"fmt"
	"time"

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
	MaxFetchBlockNum  int64 = 100
	TimeoutSeconds    int64 = 5
	BatchBlockNum     int64 = 100
	reqTimeoutSeconds       = time.Duration(TimeoutSeconds)
)

var chainlog = log.New("module", "blockchain")

const trySyncIntervalMS = 5000
const blockUpdateIntervalSeconds = 60

type BlockChain struct {
	qclient queue.IClient
	q       *queue.Queue
	// 永久存储数据到db中
	blockStore *BlockStore

	//cache  缓存block方便快速查询
	cache      map[string]*list.Element
	cacheSize  int64
	cacheQueue *list.List

	//Block 同步阶段用于缓存block信息，
	blockPool *BlockPool

	//用于请求block超时的处理
	reqBlk map[int64]*bpRequestBlk

	//记录收到的最新block高度,用于节点追赶active链
	recvLastBlockHeight int64
}

func New(cfg *types.BlockChain) *BlockChain {

	//初始化blockstore 和txindex  db
	blockStoreDB := dbm.NewDB("blockchain", cfg.Driver, cfg.DbPath)
	blockStore := NewBlockStore(blockStoreDB)
	initConfig(cfg)
	pool := NewBlockPool()
	reqblk := make(map[int64]*bpRequestBlk)

	return &BlockChain{
		blockStore:          blockStore,
		cache:               make(map[string]*list.Element),
		cacheSize:           DefCacheSize,
		cacheQueue:          list.New(),
		blockPool:           pool,
		reqBlk:              reqblk,
		recvLastBlockHeight: 0,
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
	chain.blockStore.db.Close()
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
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventTransactionDetail, TransactionDetail))
			}

		case types.EventGetBlocks:
			requestblocks := (msg.Data).(*types.ReqBlocks)
			blocks, err := chain.ProcGetBlockDetailsMsg(requestblocks)
			if err != nil {
				chainlog.Error("ProcGetBlockDetailsMsg", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlocks, err))
			} else {
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventBlocks, blocks))
			}

		case types.EventAddBlock:
			var block *types.BlockDetail
			var reply types.Reply
			reply.IsOk = true
			block = msg.Data.(*types.BlockDetail)
			err := chain.ProcAddBlockMsg(block)
			if err != nil {
				chainlog.Error("ProcAddBlockMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))

		case types.EventAddBlocks:
			var blocks *types.BlockDetails
			var reply types.Reply
			reply.IsOk = true
			blocks = msg.Data.(*types.BlockDetails)
			err := chain.ProcAddBlocksMsg(blocks)
			if err != nil {
				chainlog.Error("ProcAddBlocksMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(chain.qclient.NewMessage("p2p", types.EventReply, &reply))

		case types.EventGetBlockHeight:
			var replyBlockHeight types.ReplyBlockHeight
			replyBlockHeight.Height = chain.GetBlockHeight()
			msg.Reply(chain.qclient.NewMessage("consensus", types.EventReplyBlockHeight, &replyBlockHeight))

		case types.EventTxHashList:
			txhashlist := (msg.Data).(*types.TxHashList)
			duptxhashlist := chain.GetDuplicateTxHashList(txhashlist)
			msg.Reply(chain.qclient.NewMessage("consensus", types.EventTxHashListReply, duptxhashlist))

		case types.EventGetHeaders:
			requestblocks := (msg.Data).(*types.ReqBlocks)
			headers, err := chain.ProcGetHeadersMsg(requestblocks)
			if err != nil {
				chainlog.Error("ProcGetHeadersMsg", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventHeaders, err))
			} else {
				msg.Reply(chain.qclient.NewMessage("rpc", types.EventHeaders, headers))
			}

		case types.EventGetLastHeader:
			header, err := chain.ProcGetLastHeaderMsg()
			if err != nil {
				chainlog.Error("ProcGetLastHeaderMsg", "err", err.Error())
				msg.Reply(chain.qclient.NewMessage("account", types.EventHeader, err))
			} else {
				msg.Reply(chain.qclient.NewMessage("account", types.EventHeader, header))
			}

		case types.EventAddBlockDetail:
			var blockDetail *types.BlockDetail
			var reply types.Reply
			reply.IsOk = true
			blockDetail = msg.Data.(*types.BlockDetail)
			err := chain.ProcAddBlockDetailMsg(blockDetail)
			if err != nil {
				chainlog.Error("ProcAddBlockDetailMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(chain.qclient.NewMessage("consensus", types.EventReply, &reply))

		default:
			chainlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
	}
}

func (chain *BlockChain) poolRoutine() {
	trySyncTicker := time.NewTicker(trySyncIntervalMS * time.Millisecond)
	blockUpdateTicker := time.NewTicker(blockUpdateIntervalSeconds * time.Second)

FOR_LOOP:
	for {
		select {
		case _ = <-blockUpdateTicker.C:
			//chainlog.Info("blockUpdateTicker")
			go chain.FetchPeerList()
		case _ = <-trySyncTicker.C:
			//chainlog.Info("trySyncTicker")
			// 定时同步缓存中的block信息到db数据库中
			newbatch := chain.blockStore.NewBatch(true)
			var stratblockheight int64 = 0
			var endblockheight int64 = 0
			var i int64
			// 可以批量处理BatchBlockNum个block到db中
			currentheight := chain.blockStore.Height()
			curblock, err := chain.GetBlock(currentheight)
			if err != nil {
				continue FOR_LOOP
			}
			prevHash := curblock.Block.StateHash
			for i = 1; i <= BatchBlockNum; i++ {

				block := chain.blockPool.GetBlock(currentheight + i)

				//需要加载的第一个nextblock不存在，退出for循环进入下一个超时
				if block == nil && i == 1 {
					//chainlog.Info("trySyncTicker continue FOR_LOOP")
					continue FOR_LOOP
				}
				// 缓存中连续的block数小于BatchBlockNum时，同步现有的block到db中
				if block == nil {
					//chainlog.Info("trySyncTicker block is nil")
					break
				}
				//用于记录同步开始的第一个block高度，用于同步完成之后删除缓存中的block记录
				if i == 1 {
					stratblockheight = block.Height
				}
				blockdetail, err := util.ExecBlock(chain.q, prevHash, block, true)
				if err != nil {
					chainlog.Info("trySyncTicker ExecBlock is err!", "height", block.Height, "err", err)
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

				//记录同步结束时最后一个block的高度，用于同步完成之后删除缓存中的block记录
				endblockheight = blockdetail.Block.Height
			}
			newbatch.Write()

			//更新db中的blockheight到blockStore.Height
			chain.blockStore.UpdateHeight()

			//删除缓存中的block
			for j := stratblockheight; j <= endblockheight; j++ {
				block := chain.blockPool.GetBlock(j)
				if block != nil {
					//通知mempool和consense模块
					chain.SendAddBlockEvent(block)
					chain.blockPool.DelBlock(j)
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
	TransactionDetail, err := GetTransactionDetail(block.Block.Txs, txresult.Index)
	if err != nil {
		return nil, err
	}
	return TransactionDetail, nil
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
				block.Receipts = nil
				blocks.Items[j] = block
			}
		} else {
			return nil, err
		}
		j++
	}
	return &blocks, nil
}

/*
EventAddBlock(types.Block), P2P模块会向系统发送 EventAddBlock(types.Block) 的请求，表示添加一个区块。
有时候，广播过来的区块不是当前高度+1，在等待一个超时时间以后。可以主动请求区块。
type Block struct {
	ParentHash []byte         `protobuf:"bytes,1,opt,name=parentHash,proto3" json:"parentHash,omitempty"`
	TxHash     []byte         `protobuf:"bytes,2,opt,name=txHash,proto3" json:"txHash,omitempty"`
	BlockTime  int64          `protobuf:"varint,3,opt,name=blockTime" json:"blockTime,omitempty"`
	Txs        []*Transaction `protobuf:"bytes,4,rep,name=txs" json:"txs,omitempty"`
}
*/
func (chain *BlockChain) ProcAddBlockMsg(blockdetail *types.BlockDetail) (err error) {
	currentheight := chain.GetBlockHeight()
	var prevstateHash []byte
	if blockdetail.Block.Height == currentheight+1 {
		//获取前一个block的statehash
		if currentheight == -1 {
			prevstateHash = common.Hash{}.Bytes()
		} else {
			block, err := chain.GetBlock(currentheight)
			if err != nil {
				chainlog.Error("ProcAddBlockMsg", "err", err)
				return err
			}
			prevstateHash = block.Block.StateHash
		}
		blockdetail, err = util.ExecBlock(chain.q, prevstateHash, blockdetail.Block, true)
		if err != nil {
			chainlog.Error("ProcAddBlockMsg ExecBlock err!", "err", err)
			return err
		}
	}

	err = chain.ProcAddBlockDetailMsg(blockdetail)
	return err
}

func (chain *BlockChain) ProcAddBlockDetailMsg(blockDetail *types.BlockDetail) (err error) {
	currentheight := chain.GetBlockHeight()

	//不是我们需要的高度直接返回
	if currentheight >= blockDetail.Block.Height {
		outstr := fmt.Sprintf("input add height :%d ,current store height:%d", blockDetail.Block.Height, currentheight)
		err = errors.New(outstr)
		return err
	} else if blockDetail.Block.Height == currentheight+1 { //我们需要的高度，直接存储到db中
		newbatch := chain.blockStore.NewBatch(true)

		//保存tx交易结果信息到db中
		chain.blockStore.indexTxs(newbatch, blockDetail)
		if err != nil {
			return err
		}

		//保存block信息到db中
		err := chain.blockStore.SaveBlock(newbatch, blockDetail)
		if err != nil {
			return err
		}
		newbatch.Write()

		//更新db中的blockheight到blockStore.Height
		chain.blockStore.UpdateHeight()

		//将此block添加到缓存中便于查找
		chain.cacheBlock(blockDetail)

		//删除此block的超时机制
		chain.RemoveReqBlk(blockDetail.Block.Height, blockDetail.Block.Height)

		//通知mempool和consense模块
		chain.SendAddBlockEvent(blockDetail.Block)

		return nil
	} else {
		// 首先将此block缓存到blockpool中。
		chain.blockPool.AddBlock(blockDetail.Block)

		// block.Height之前的block已经被请求了，此时不需要再次请求,避免在追赶的时候多次请求同一个block
		if chain.recvLastBlockHeight+1 == blockDetail.Block.Height && chain.recvLastBlockHeight != 0 {
			chain.recvLastBlockHeight = blockDetail.Block.Height
			return
		}

		// 记录收到peer广播的最新block高度，用于block的同步
		if chain.recvLastBlockHeight < blockDetail.Block.Height {
			chain.recvLastBlockHeight = blockDetail.Block.Height

			//启动一个超时定时器，如果在规定时间内没有收到就发送一个FetchBlock消息给p2p模块
			//请求currentheight+1 到 block.Height-1之间的blocks
			chain.WaitReqBlk(currentheight+1, blockDetail.Block.Height-1)
		}
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
func (chain *BlockChain) FetchBlock(reqblk *types.ReqBlocks) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}

	chainlog.Debug("FetchBlock", "StartHeight", reqblk.Start, "EndHeight", reqblk.End)
	blockcount := reqblk.End - reqblk.Start
	if blockcount > MaxFetchBlockNum {
		chainlog.Error("FetchBlock", "blockscount", blockcount, "MaxFetchBlockNum", MaxFetchBlockNum)
		err := errors.New("FetchBlock blockcount > MaxFetchBlockNum")
		return err
	}
	var requestblock types.ReqBlocks

	requestblock.Start = reqblk.Start
	requestblock.End = reqblk.End

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
func (chain *BlockChain) SendAddBlockEvent(block *types.Block) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}
	if block == nil {
		chainlog.Error("SendAddBlockEvent block is null")
		return nil
	}
	chainlog.Debug("SendAddBlockEvent", "Height", block.Height)

	msg := chain.qclient.NewMessage("mempool", types.EventAddBlock, block)
	chain.qclient.Send(msg, false)

	msg = chain.qclient.NewMessage("consensus", types.EventAddBlock, block)
	chain.qclient.Send(msg, false)

	return nil
}

func (chain *BlockChain) ProcAddBlocksMsg(blocks *types.BlockDetails) (err error) {

	//首先缓存到pool中,由poolRoutine定时同步到db中,blocks太多此时写入db会耗时很长
	blocklen := len(blocks.Items)
	startblockheight := blocks.Items[0].Block.Height
	endblockheight := blocks.Items[blocklen-1].Block.Height

	for _, block := range blocks.Items {
		chain.blockPool.AddBlock(block.Block)
	}
	//删除此blocks的超时机制
	chain.RemoveReqBlk(startblockheight, endblockheight)

	return nil
}

func (chain *BlockChain) GetBlockHeight() int64 {
	return chain.blockStore.height
}

//用于获取指定高度的block，首先在缓存中获取，如果不存在就从db中获取
func (chain *BlockChain) GetBlock(height int64) (block *types.BlockDetail, err error) {

	// Check the cache.
	elem, ok := chain.cache[string(height)]
	if ok {
		// Already exists. Move to back of cacheQueue.
		chain.cacheQueue.MoveToBack(elem)
		return elem.Value.(*types.BlockDetail), nil
	} else {
		//从blockstore db中通过block height获取block
		blockinfo := chain.blockStore.LoadBlock(height)
		if blockinfo != nil {
			chain.cacheBlock(blockinfo)
			return blockinfo, nil
		}
	}
	err = errors.New("GetBlock error")
	return nil, err
}

//添加block到cache中，方便快速查询
func (chain *BlockChain) cacheBlock(blockdetail *types.BlockDetail) {

	// Create entry in cache and append to cacheQueue.
	elem := chain.cacheQueue.PushBack(blockdetail)
	chain.cache[string(blockdetail.Block.Height)] = elem

	// Maybe expire an item.
	if int64(chain.cacheQueue.Len()) > chain.cacheSize {
		height := chain.cacheQueue.Remove(chain.cacheQueue.Front()).(*types.BlockDetail).Block.Height
		delete(chain.cache, string(height))
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

//删除对应block的超时请求机制
func (chain *BlockChain) RemoveReqBlk(startheight int64, endheight int64) {
	chainlog.Debug("RemoveReqBlk", "startheight", startheight, "endheight", endheight)

	delete(chain.reqBlk, startheight)
}

//启动对应block的超时请求机制
func (chain *BlockChain) WaitReqBlk(start int64, end int64) {
	chainlog.Debug("WaitReqBlk", "startheight", start, "endheight", end)

	reqblk := chain.reqBlk[start]
	if reqblk != nil {
		(chain.reqBlk[start]).resetTimeout()
	} else {
		blockcount := end - start + 1

		if blockcount <= MaxFetchBlockNum {
			requestBlk := newBPRequestBlk(chain, start, end)
			chain.reqBlk[start] = requestBlk
			(chain.reqBlk[start]).resetTimeout()

		} else { //需要多次发送请求
			multiple := blockcount / MaxFetchBlockNum
			remainder := blockcount % MaxFetchBlockNum
			var count int64
			for count = 0; count < multiple; count++ {
				startheight := count*MaxFetchBlockNum + start
				endheight := startheight + MaxFetchBlockNum - 1

				requestBlk := newBPRequestBlk(chain, startheight, endheight)
				chain.reqBlk[startheight] = requestBlk
				(chain.reqBlk[startheight]).resetTimeout()
			}
			//  需要处理有余数的情况
			if count == multiple && remainder != 0 {
				startheight := count*MaxFetchBlockNum + start
				endheight := startheight + remainder - 1

				requestBlk := newBPRequestBlk(chain, startheight, endheight)
				chain.reqBlk[startheight] = requestBlk
				(chain.reqBlk[startheight]).resetTimeout()
			}
		}
	}
}

//对应block请求超时发起FetchBlock
func (chain *BlockChain) sendTimeout(startheight int64, endheight int64) {
	//chainlog.Debug("sendTimeout", "startheight", startheight, "endheight", endheight)
	//此时判断需要请求的block高度是否大于本节点已经存储的高度，小于就直接删除对应的请求超时机制，直接返回
	curheight := chain.GetBlockHeight()
	if curheight > endheight {
		chain.RemoveReqBlk(startheight, endheight)
		return
	}
	var reqBlock types.ReqBlocks
	reqBlock.Start = startheight
	reqBlock.End = endheight
	chain.FetchBlock(&reqBlock)
}

//-------------------------------------

type bpRequestBlk struct {
	blockchain  *BlockChain
	startheight int64
	endheight   int64
	timeout     *time.Timer
	didTimeout  bool //用于判断发送一次后，超时了还要不要再次发送
}

func newBPRequestBlk(chain *BlockChain, startheight int64, endheight int64) *bpRequestBlk {
	bprequestBlk := &bpRequestBlk{
		blockchain:  chain,
		startheight: startheight,
		endheight:   endheight,
	}
	return bprequestBlk
}

func (bpReqBlk *bpRequestBlk) resetTimeout() {
	if bpReqBlk.timeout == nil {
		//chainlog.Debug("resetTimeout", "startheight", bpReqBlk.startheight, "endheight", bpReqBlk.endheight)
		bpReqBlk.timeout = time.AfterFunc(time.Second*reqTimeoutSeconds, bpReqBlk.onTimeout)
	} else {
		bpReqBlk.timeout.Reset(time.Second * reqTimeoutSeconds)
	}
}

func (bpReqBlk *bpRequestBlk) onTimeout() {
	//chainlog.Debug("onTimeout", "startheight", bpReqBlk.startheight, "endheight", bpReqBlk.endheight)
	bpReqBlk.blockchain.sendTimeout(bpReqBlk.startheight, bpReqBlk.endheight)
	bpReqBlk.didTimeout = true
}

//  获取指定txindex  在txs中的TransactionDetail ，注释：index从0开始
func GetTransactionDetail(Txs []*types.Transaction, index int32) (*types.TransactionDetail, error) {

	var txdetail types.TransactionDetail
	txlen := len(Txs)

	//计算tx的hash值
	leaves := make([][]byte, txlen)
	for index, tx := range Txs {
		leaves[index] = tx.Hash()
		//chainlog.Info("GetTransactionDetail txhash", "index", index, "txhash", tx.Hash())
	}

	proofs := merkle.GetMerkleBranch(leaves, uint32(index))
	txdetail.Proofs = proofs
	return &txdetail, nil
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

	currentheight := chain.GetBlockHeight()

	msg := chain.qclient.NewMessage("p2p", types.EventPeerInfo, nil)
	chain.qclient.Send(msg, true)
	resp, err := chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("FetchPeerList", "qclient.Wait err:", err)
		return
	}
	peerlist := resp.GetData().(*types.PeerList)

	//获取peerlist中最新的高度
	var maxPeerHeight int64 = 0
	for _, peer := range peerlist.Peers {
		if peer != nil && maxPeerHeight < peer.Header.Height {
			maxPeerHeight = peer.Header.Height
		}
	}
	//如果没有收到peer广播的block 并且本节点高度落后于peer ，主动发起同步blokc的请求
	if chain.recvLastBlockHeight == 0 && maxPeerHeight > currentheight {
		chainlog.Info("FetchPeerList WaitReqBlk ", "startheight", currentheight+1, "endheight", maxPeerHeight)
		chain.WaitReqBlk(currentheight+1, maxPeerHeight)
	}
	//本节点已经追赶上active链，此时可以考虑退出block同步的poolRoutine
	if currentheight != 0 && currentheight >= maxPeerHeight {
		chainlog.Info("FetchPeerList CaughtUp ok!")
	}
	//异常情况打印日志记录
	if chain.recvLastBlockHeight != 0 && maxPeerHeight > currentheight+5 {
		chainlog.Info("FetchPeerList", "recvLastBlockHeight", chain.recvLastBlockHeight, "maxPeerHeight", maxPeerHeight, "storecurheight", currentheight)
	}
	return
}
