package blockchain

import (
	dbm "common/db"
	"common/merkle"
	"container/list"
	"errors"
	"fmt"
	"queue"
	"time"
	"types"

	log "github.com/inconshreveable/log15"
)

var (
	//cache 存贮的block个数
	DefCacheSize int = 500

	//一次最多申请获取block个数
	MaxFetchBlockNum  int64  = 100
	TimeoutSeconds    int64  = 15
	BatchBlockNum     int64  = 100
	reqTimeoutSeconds        = time.Duration(TimeoutSeconds)
	Datadir           string = "datadir"
)

var chainlog = log.New("module", "blockchain")

const trySyncIntervalMS = 1000

type BlockChain struct {
	qclient queue.IClient

	// 永久存储数据到db中
	blockStore *BlockStore

	//cache  缓存block方便快速查询
	cache      map[string]*list.Element
	cacheSize  int
	cacheQueue *list.List

	//Block 同步阶段用于缓存block信息，
	blockPool *BlockPool

	//用于请求block超时的处理
	reqBlk map[int64]*bpRequestBlk
}

func New() *BlockChain {

	//初始化blockstore 和txindex  db
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", Datadir)
	blockStore := NewBlockStore(blockStoreDB)

	pool := NewBlockPool()
	reqblk := make(map[int64]*bpRequestBlk)

	return &BlockChain{
		blockStore: blockStore,
		cache:      make(map[string]*list.Element),
		cacheSize:  DefCacheSize,
		cacheQueue: list.New(),
		blockPool:  pool,
		reqBlk:     reqblk,
	}
}

func (chain *BlockChain) SetQueue(q *queue.Queue) {
	chain.qclient = q.GetClient()
	chain.qclient.Sub("blockchain")

	//recv 消息的处理
	go chain.ProcRecvMsg()

	// 定时同步缓存中的block to db
	go chain.poolRoutine()
}
func (chain *BlockChain) ProcRecvMsg() {
	for msg := range chain.qclient.Recv() {
		msgtype := msg.Ty
		switch msgtype {
		case types.EventQueryTx:
			txhash := (msg.Data).(types.RequestHash)
			merkleproof, err := chain.ProcQueryTxMsg(txhash.Hash)
			if err != nil {
				chainlog.Error("ProcQueryTxMsg", "err", err.Error())
				var reply types.Reply
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))
			} else {
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventMerkleProof, merkleproof))
			}

		case types.EventGetBlocks:
			requestblocks := (msg.Data).(*types.RequestBlocks)
			blocks, err := chain.ProcGetBlocksMsg(requestblocks)
			if err != nil {
				chainlog.Error("ProcGetBlocksMsg", "err", err.Error())
				var reply types.Reply
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))
			} else {
				msg.Reply(chain.qclient.NewMessage("blockchain", types.EventBlocks, blocks))
			}

		case types.EventAddBlock:
			var block *types.Block
			var reply types.Reply
			reply.IsOk = true
			block = msg.Data.(*types.Block)
			err := chain.ProcAddBlockMsg(block)
			if err != nil {
				chainlog.Error("ProcAddBlockMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))

		case types.EventAddBlocks:
			var blocks *types.Blocks
			var reply types.Reply
			reply.IsOk = true
			blocks = msg.Data.(*types.Blocks)
			err := chain.ProcAddBlocksMsg(blocks)
			if err != nil {
				chainlog.Error("ProcAddBlocksMsg", "err", err.Error())
				reply.IsOk = false
				reply.Msg = []byte(err.Error())
			}
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReply, &reply))

		case types.EventGetBlockHeight:
			var replyBlockHeight types.ReplyBlockHeight
			replyBlockHeight.Height = chain.GetBlockHeight()
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventReplyBlockHeight, &replyBlockHeight))

		case types.EventTxHashList:
			txhashlist := (msg.Data).(*types.TxHashList)
			duptxhashlist := chain.GetDuplicateTxHashList(txhashlist)
			msg.Reply(chain.qclient.NewMessage("blockchain", types.EventTxHashListReply, duptxhashlist))

		default:
			chainlog.Info("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
	}
}

func (chain *BlockChain) poolRoutine() {
	trySyncTicker := time.NewTicker(trySyncIntervalMS * time.Millisecond)
FOR_LOOP:
	for {
		select {
		case <-trySyncTicker.C:
			// 定时同步缓存中的block信息到db数据库中
			newbatch := chain.blockStore.NewBatch(true)
			var stratblockheight int64 = 0
			var endblockheight int64 = 0
			var i int64
			// 可以批量处理BatchBlockNum个block到db中
			currentheight := chain.blockStore.Height()
			for i = 1; i <= BatchBlockNum; i++ {

				block := chain.blockPool.GetBlock(currentheight + i)
				if block == nil && i == 1 { //需要加载的第一个nextblock不存在，退出for循环进入下一个超时
					continue FOR_LOOP
				}
				// 缓存中连续的block数小于BatchBlockNum时，同步现有的block到db中
				if block == nil {
					break
				}
				//用于记录同步开始的第一个block高度，用于同步完成之后删除缓存中的block记录
				if i == 1 {
					stratblockheight = block.Height
				}
				//保存tx信息到db中
				err := chain.blockStore.indexTxs(newbatch, block)
				if err != nil {
					break
				}
				//保存block信息到db中
				err = chain.blockStore.SaveBlock(newbatch, block)
				if err != nil {
					break
				}
				//记录同步结束时最后一个block的高度，用于同步完成之后删除缓存中的block记录
				endblockheight = block.Height
			}
			newbatch.Write()

			//更新db中的blockheight到blockStore.Height
			chain.blockStore.UpdateHeight()

			//删除缓存中的block
			for j := stratblockheight; j <= endblockheight; j++ {
				//将已经存储的blocks添加到list缓存中便于查找
				block := chain.blockPool.GetBlock(j)
				if block != nil {
					chain.cacheBlock(block)
				}
				//通知mempool和consense模块
				chain.SendAddBlockEvent(block)

				chain.blockPool.DelBlock(j)
			}
			continue FOR_LOOP
		}
	}
}

/*
函数功能：
EventQueryTx(types.RequestHash) : rpc模块会向 blockchain 模块 发送 EventQueryTx(types.RequestHash) 消息 ，
查询交易的默克尔树，回复消息 EventMerkleProof(types.MerkleProof)
结构体：
type RequestHash struct {Hash []byte `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`}
type MerkleProof struct {Hashs [][]byte `protobuf:"bytes,1,rep,name=hashs,proto3" json:"hashs,omitempty"}
*/
func (chain *BlockChain) ProcQueryTxMsg(txhash []byte) (proof *types.MerkleProof, err error) {
	txresult, err := chain.GetTxResultFromDb(txhash)
	if err != nil {
		return nil, err
	}
	block, err := chain.GetBlock(txresult.Height)
	if err != nil {
		return nil, err
	}
	merkleproof, err := GetMerkleProof(block.Txs, txresult.Index)
	if err != nil {
		return nil, err
	}
	return merkleproof, nil
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
type RequestBlocks struct {
	Start int64 `protobuf:"varint,1,opt,name=start" json:"start,omitempty"`
	End   int64 `protobuf:"varint,2,opt,name=end" json:"end,omitempty"`}
type Blocks struct {Items []*Block `protobuf:"bytes,1,rep,name=items" json:"items,omitempty"`}
*/
func (chain *BlockChain) ProcGetBlocksMsg(requestblock *types.RequestBlocks) (respblocks *types.Blocks, err error) {
	blockhight := chain.GetBlockHeight()
	if requestblock.Start >= blockhight {
		outstr := fmt.Sprintf("input Start height :%d  but current height:%d", requestblock.Start, blockhight)
		err = errors.New(outstr)
		return nil, err
	}
	end := requestblock.End
	if requestblock.End >= blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetBlocksMsg", "blockscount", count)

	var blocks types.Blocks
	blocks.Items = make([]*types.Block, count)
	j := 0
	for i := start; i <= end; i++ {
		block, err := chain.GetBlock(i)
		if err == nil && block != nil {
			blocks.Items[j] = block
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
func (chain *BlockChain) ProcAddBlockMsg(block *types.Block) (err error) {
	currentheight := chain.GetBlockHeight()

	//不是我们需要的高度直接返回
	if currentheight >= block.Height {
		outstr := fmt.Sprintf("input add height :%d ,current store height:%d", block.Height, currentheight)
		err = errors.New(outstr)
		return err
	} else if block.Height == currentheight+1 { //我们需要的高度，直接存储到db中
		newbatch := chain.blockStore.NewBatch(true)

		//保存tx信息到db中
		chain.blockStore.indexTxs(newbatch, block)
		if err != nil {
			return err
		}
		//保存block信息到db中
		err := chain.blockStore.SaveBlock(newbatch, block)
		if err != nil {
			return err
		}
		newbatch.Write()

		//更新db中的blockheight到blockStore.Height
		chain.blockStore.UpdateHeight()

		//将此block添加到缓存中便于查找
		chain.cacheBlock(block)

		//删除此block的超时机制
		chain.RemoveReqBlk(block.Height, block.Height)

		//通知mempool和consense模块
		chain.SendAddBlockEvent(block)

		return nil
	} else {
		// 将此block 先添加到缓存中blockpool中。
		//启动一个超时定时器，如果在规定时间内没有收到就发送一个FetchBlock消息给p2p模块
		//请求currentheight+1 到 block.Height-1之间的blocks
		chain.blockPool.AddBlock(block)
		chain.WaitReqBlk(currentheight+1, block.Height-1)
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
func (chain *BlockChain) FetchBlock(reqblk *types.RequestBlocks) (err error) {
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
	var requestblock types.RequestBlocks

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
	chain.qclient.Send(msg, true)
	_, err = chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("SendAddBlockEvent", "send to mempool err:", err)
	}

	msg = chain.qclient.NewMessage("consense", types.EventAddBlock, block)
	chain.qclient.Send(msg, true)
	_, err = chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("SendAddBlockEvent", "send to consense err:", err)
	}

	return nil
}

func (chain *BlockChain) ProcAddBlocksMsg(blocks *types.Blocks) (err error) {

	//首先缓存到pool中,由poolRoutine定时同步到db中,blocks太多此时写入db会耗时很长
	blocklen := len(blocks.Items)
	startblockheight := blocks.Items[0].Height
	endblockheight := blocks.Items[blocklen-1].Height

	for _, block := range blocks.Items {
		chain.blockPool.AddBlock(block)
	}
	//删除此blocks的超时机制
	chain.RemoveReqBlk(startblockheight, endblockheight)

	return nil
}

func (chain *BlockChain) GetBlockHeight() int64 {
	return chain.blockStore.height
}

//用于获取指定高度的block，首先在缓存中获取，如果不存在就从db中获取

func (chain *BlockChain) GetBlock(height int64) (block *types.Block, err error) {

	// Check the cache.
	elem, ok := chain.cache[string(height)]
	if ok {
		// Already exists. Move to back of cacheQueue.
		chain.cacheQueue.MoveToBack(elem)
		return elem.Value.(*types.Block), nil
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
func (chain *BlockChain) cacheBlock(block *types.Block) {

	// Create entry in cache and append to cacheQueue.
	elem := chain.cacheQueue.PushBack(block)
	chain.cache[string(block.Height)] = elem

	// Maybe expire an item.
	if chain.cacheQueue.Len() > chain.cacheSize {
		height := chain.cacheQueue.Remove(chain.cacheQueue.Front()).(*types.Block).Height
		delete(chain.cache, string(height))
	}
}

/*
通过txhash 从txindex db中获取tx信息
type TxResult struct {
	Height int64                 `json:"height"`
	Index  int32                 `json:"index"`
	Tx     *types.Transaction    `json:"tx"`
}
*/
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

	var reqBlock types.RequestBlocks
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
		bpReqBlk.timeout = time.AfterFunc(time.Second*reqTimeoutSeconds, bpReqBlk.onTimeout)
	} else {
		bpReqBlk.timeout.Reset(time.Second * reqTimeoutSeconds)
	}
}

func (bpReqBlk *bpRequestBlk) onTimeout() {
	bpReqBlk.blockchain.sendTimeout(bpReqBlk.startheight, bpReqBlk.endheight)
	bpReqBlk.didTimeout = true
}

//  获取指定txindex  在txs中的merkleproof ，注释：index从0开始
func GetMerkleProof(Txs []*types.Transaction, index int32) (*types.MerkleProof, error) {

	var txproof types.MerkleProof
	txlen := len(Txs)

	//计算tx的hash值
	leaves := make([][]byte, txlen)
	for index, tx := range Txs {
		leaves[index] = tx.Hash()
		//chainlog.Info("GetMerkleProof txhash", "index", index, "txhash", tx.Hash())
	}

	merkleproof, _ := merkle.ComputeMerkleBranch(leaves, int(index))
	txproof.Hashs = merkleproof
	return &txproof, nil
}
