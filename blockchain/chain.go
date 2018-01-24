package blockchain

import (
	"bytes"
	"container/list"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
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
	TimeoutSeconds   int64 = 2
	BatchBlockNum    int64 = 100
	blockSynSeconds        = time.Duration(TimeoutSeconds)
)

var chainlog = log.New("module", "blockchain")

var cachelock sync.Mutex
var castlock sync.Mutex
var synBlocklock sync.Mutex
var peerMaxBlklock sync.Mutex

const poolToDbMS = 500         //从blockpool写block到db的定时器
const fetchPeerListSeconds = 5 //获取一次peerlist最新block高度

type BlockChain struct {
	qclient queue.IClient
	q       *queue.Queue
	// 永久存储数据到db中
	blockStore *BlockStore
	//cache  缓存block方便快速查询
	cache      map[int64]*list.Element
	cacheSize  int64
	cacheQueue *list.List

	task *Task
	//Block 同步阶段用于缓存block信息，
	blockPool *BlockPool

	//记录收到的最新广播的block高度,用于节点追赶active链
	rcvLastBlockHeight int64

	//记录本节点已经同步的block高度,用于节点追赶active链
	synBlockHeight int64

	//记录peer的最新block高度,用于节点追赶active链
	peerMaxBlkHeight int64
	wg               *sync.WaitGroup
	recvwg           *sync.WaitGroup
	synblock         chan struct{}
	quit             chan struct{}
	isclosed         int32
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
		wg:                 &sync.WaitGroup{},
		recvwg:             &sync.WaitGroup{},
		task:               newTask(60 * time.Second),
		quit:               make(chan struct{}, 0),
		synblock:           make(chan struct{}, 1),
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

	chainlog.Error("begin close")
	//等待所有的写线程退出，防止数据库写到一半被暂停
	atomic.StoreInt32(&chain.isclosed, 1)
	chain.wg.Wait()
	//退出线程
	//退出接受数据
	chain.qclient.Close()
	close(chain.quit)
	//wait for recvwg quit:
	chain.recvwg.Wait()

	//关闭数据库
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
	reqnum := make(chan struct{}, 1000)
	for msg := range chain.qclient.Recv() {
		chainlog.Info("blockchain recv", "msg", types.GetEventName(int(msg.Ty)), "cap", len(reqnum))
		msgtype := msg.Ty
		reqnum <- struct{}{}
		chain.recvwg.Add(1)
		switch msgtype {
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
		default:
			<-reqnum
			chain.wg.Done()
			chainlog.Warn("ProcRecvMsg unknow msg", "msgtype", msgtype)
		}
	}
}

func (chain *BlockChain) poolRoutine() {
	//获取peerlist的定时器，默认1分钟
	fetchPeerListTicker := time.NewTicker(fetchPeerListSeconds * time.Second)
	//向peer请求同步block的定时器，默认5s
	blockSynTicker := time.NewTicker(time.Duration(blockSynSeconds) * time.Second)

	for {
		select {
		case <-chain.quit:
			//chainlog.Info("quit poolRoutine!")
			return
		case _ = <-blockSynTicker.C:
			//chainlog.Info("blockSynTicker")
			chain.SynBlocksFromPeers()

		case _ = <-fetchPeerListTicker.C:
			//chainlog.Info("blockUpdateTicker")
			chain.FetchPeerList()
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
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		gen := action.GetGenesis()
		TransactionDetail.Amount = gen.Amount
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
	if requestblock.GetStart() > requestblock.GetEnd() {
		outstr := fmt.Sprintf("input must Start <= End: Startheight :%d  Endheight:%d", requestblock.Start, requestblock.End)
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
func (chain *BlockChain) ProcAddBlockMsg(broadcast bool, blockdetail *types.BlockDetail) (err error) {
	block := blockdetail.Block
	if block == nil {
		err = errors.New("ProcAddBlockMsg input block is null")
		return err
	}
	currentheight := chain.GetBlockHeight()
	//不是我们需要的高度直接返回
	if currentheight >= block.Height {
		outstr := fmt.Sprintf("input add height :%d ,current store height:%d, isbroadcast:%v", block.Height, currentheight, broadcast)
		err = errors.New(outstr)
		return err
	} else {
		chain.blockPool.AddBlock(blockdetail, broadcast)
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
func (chain *BlockChain) FetchBlock(start int64, end int64) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}

	chainlog.Debug("FetchBlock input", "StartHeight", start, "EndHeight", end)
	blockcount := end - start
	if blockcount < 0 {
		return types.ErrStartBigThanEnd
	}
	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.Isdetail = false

	if blockcount >= MaxFetchBlockNum {
		requestblock.End = start + MaxFetchBlockNum - 1
	} else {
		requestblock.End = end
	}

	err = chain.task.Start(requestblock.Start, requestblock.End, func() {
		chain.SynBlocksFromPeers()
	})
	if err != nil {
		return err
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

	chainlog.Debug("SendAddBlockEvent -->>wallet", "height", block.GetBlock().GetHeight())
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

func (chain *BlockChain) notifySync() {
	chain.wg.Add(1)
	go chain.SynBlockToDbOneByOne()
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

//添加block到cache中，方便快速查询
func (chain *BlockChain) DelBlockFromCache(height int64) {
	cachelock.Lock()
	defer cachelock.Unlock()
	elem, ok := chain.cache[height]
	if ok {
		delheight := chain.cacheQueue.Remove(elem).(*types.BlockDetail).Block.Height
		if delheight != height {
			chainlog.Error("DelBlockFromCache height err ", "height", height, "delheight", delheight)
		}
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
			head.Hash = blockdetail.Block.Hash()
			head.TxCount = int64(len(blockdetail.Block.Txs))
			headers.Items[j] = head
		} else {
			return nil, err
		}
		j++
	}
	chainlog.Error("getHeaders", "len", len(headers.Items), "start", start, "end", end)
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
	chain.fetchPeerList()
}

func (chain *BlockChain) fetchPeerList() error {
	if chain.qclient == nil {
		chainlog.Error("fetchPeerList chain client not bind message queue.")
		return nil
	}
	msg := chain.qclient.NewMessage("p2p", types.EventPeerInfo, nil)
	chain.qclient.Send(msg, true)
	resp, err := chain.qclient.Wait(msg)
	if err != nil {
		chainlog.Error("fetchPeerList", "qclient.Wait err:", err)
		return err
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
	if maxPeerHeight == -1 {
		return types.ErrNoPeer
	}
	return nil
}

//获取地址对应的所有交易信息
//存储格式key:addr:flag:height ,value:txhash
//key=addr :获取本地参与的所有交易
//key=addr:0 :获取本地作为from方的所有交易
//key=addr:1 :获取本地作为to方的所有交易
func (chain *BlockChain) ProcGetTransactionByAddr(addr *types.ReqAddr) (*types.ReplyTxInfos, error) {
	if addr == nil || len(addr.Addr) == 0 {
		err := errors.New("ProcGetTransactionByAddr addr is nil")
		return nil, err
	}
	//入参数校验
	curheigt := chain.GetBlockHeight()
	if addr.GetHeight() > curheigt || addr.GetHeight() < -1 {
		err := errors.New("ProcGetTransactionByAddr Height err")
		return nil, err
	}
	if addr.GetDirection() != 0 && addr.GetDirection() != 1 {
		err := errors.New("ProcGetTransactionByAddr Direction err")
		return nil, err
	}
	if addr.GetIndex() < 0 || addr.GetIndex() > MaxTxsPerBlock {
		err := errors.New("ProcGetTransactionByAddr Index err")
		return nil, err
	}

	txinfos, err := chain.blockStore.GetTxsByAddr(addr)
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
			} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
				gen := action.GetGenesis()
				txDetail.Amount = gen.Amount
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
	//如果任务正常，那么不重复启动任务
	if chain.task.InProgress() {
		chainlog.Error("chain task InProgress")
		return
	}

	curheight := chain.GetBlockHeight()
	RcvLastCastBlkHeight := chain.GetRcvLastCastBlkHeight()
	peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()

	chainlog.Info("SynBlocksFromPeers", "curheight", curheight, "LastCastBlkHeight", RcvLastCastBlkHeight, "peerMaxBlkHeight", peerMaxBlkHeight)
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
	chainlog.Info("ProcGetAddrOverview", "Addr", addr.GetAddr())

	var addrOverview types.AddrOverview

	//获取地址的reciver
	amount, err := chain.blockStore.GetAddrReciver(addr.Addr)
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrReciver err", err)
		return nil, err
	}
	addrOverview.Reciver = amount

	//获取地址对应的交易count
	addr.Flag = 0
	addr.Count = 0x7fffffff
	addr.Height = -1
	addr.Index = 0
	txinfos, err := chain.blockStore.GetTxsByAddr(addr)
	if err != nil {
		chainlog.Info("ProcGetAddrOverview", "GetTxsByAddr err", err)
		return nil, err
	}
	addrOverview.TxCount = int64(len(txinfos.GetTxInfos()))

	chainlog.Info("ProcGetAddrOverview", "addr", addr.Addr, "addrOverview", addrOverview.String())

	return &addrOverview, nil
}

//通过blockheight 获取blockhash
func (chain *BlockChain) ProcGetBlockHash(height *types.ReqInt) (*types.ReplyHash, error) {
	if height == nil || 0 > height.GetHeight() {
		err := errors.New("ProcGetBlockHash input err!")
		return nil, err
	}
	CurHeight := chain.GetBlockHeight()
	if height.GetHeight() > CurHeight {
		err := errors.New("ProcGetBlockHash input height err!")
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

// 定时同步缓存中连续的block信息到db数据库中OneByOne
func (chain *BlockChain) SynBlockToDbOneByOne() {
	chain.synblock <- struct{}{}
	defer func() {
		chain.wg.Done()
		<-chain.synblock
	}()
	for {
		if atomic.LoadInt32(&chain.isclosed) == 1 {
			return
		}
		//获取当前block的statehash和blockhash用于nextblock的校验
		var prevStateHash []byte
		var prevblkHash []byte
		currentheight := chain.blockStore.Height()
		if currentheight == -1 {
			prevStateHash = common.Hash{}.Bytes()
			prevblkHash = common.Hash{}.Bytes()
		} else {
			curblock, err := chain.GetBlock(currentheight)
			if err != nil {
				chainlog.Error("SynBlockToDbOneByOne GetBlock:", "height", currentheight, "err", err)
				return
			}
			prevStateHash = curblock.Block.GetStateHash()
			prevblkHash = curblock.Block.Hash()
		}
		//从pool缓存池中获取当前block的nextblock
		blockdetail, broadcast := chain.blockPool.GetBlock(currentheight + 1)
		if blockdetail == nil || blockdetail.Block == nil {
			return
		}
		block := blockdetail.Block
		//校验ParentHash 不过需要从blockpool中删除，重新发起请求
		if !bytes.Equal(prevblkHash, block.GetParentHash()) {
			chainlog.Error("SynBlockToDbOneByOne ParentHash err!", "height", block.Height)
			chain.blockPool.DelBlock(block.GetHeight())
			return
		}
		var err error
		if blockdetail.Receipts == nil {
			//block执行不过需要从blockpool中删除，重新发起请求
			blockdetail, err = util.ExecBlock(chain.q, prevStateHash, block, true)
			if err != nil {
				chainlog.Error("SynBlockToDbOneByOne ExecBlock is err!", "height", block.Height, "err", err)
				chain.blockPool.DelBlock(block.GetHeight())
				return
			}
		}

		//批量将block信息写入磁盘
		newbatch := chain.blockStore.NewBatch(true)
		cacheDB := NewCacheDB()
		//保存tx信息到db中
		err = chain.blockStore.AddTxs(newbatch, cacheDB, blockdetail)
		if err != nil {
			chainlog.Error("SynBlockToDbOneByOne indexTxs:", "height", block.Height, "err", err)
			return
		}

		//保存block信息到db中
		err = chain.blockStore.SaveBlock(newbatch, blockdetail)
		if err != nil {
			chainlog.Error("SynBlockToDbOneByOne SaveBlock:", "height", block.Height, "err", err)
			return
		}
		cacheDB.SetBatch(newbatch)
		newbatch.Write()

		chain.blockStore.UpdateHeight()
		chain.task.Done(blockdetail.Block.GetHeight())
		chain.cacheBlock(blockdetail)
		chain.SendAddBlockEvent(blockdetail)
		chain.blockPool.DelBlock(blockdetail.Block.Height)

		if broadcast {
			chain.SendBlockBroadcast(blockdetail)
			//更新广播block的高度
			castblockheight := chain.GetRcvLastCastBlkHeight()
			if castblockheight < blockdetail.Block.Height {
				chain.UpdateRcvCastBlkHeight(blockdetail.Block.Height)
			}
		}
	}
}

//删除指定高度的block从数据库中，只能删除当前高度的block
func (chain *BlockChain) DelBlock(height int64) (bool, error) {
	currentheight := chain.blockStore.Height()
	if currentheight == height { //只删除当前高度的block
		blockdetail, err := chain.GetBlock(currentheight)
		if err != nil {
			chainlog.Error("DelBlock chainGetBlock:", "height", currentheight, "err", err)
			return false, err
		}
		//批量将删除block的信息从磁盘中删除
		newbatch := chain.blockStore.NewBatch(true)
		cacheDB := NewCacheDB()

		//从db中删除tx相关的信息
		err = chain.blockStore.DelTxs(newbatch, cacheDB, blockdetail)
		if err != nil {
			chainlog.Error("DelBlock DelTxs:", "height", currentheight, "err", err)
			return false, err
		}

		//从db中删除block相关的信息
		err = chain.blockStore.DelBlock(newbatch, blockdetail)
		if err != nil {
			chainlog.Error("DelBlock blockStoreDelBlock:", "height", currentheight, "err", err)
			return false, err
		}
		cacheDB.SetBatch(newbatch)
		newbatch.Write()

		chain.blockStore.UpdateHeight()

		//删除缓存中的block信息
		chain.DelBlockFromCache(blockdetail.Block.Height)
		//通知共识，mempool和钱包删除block
		chain.SendDelBlockEvent(blockdetail)
	}
	return true, nil
}

//blockchain 模块 del block从db之后通知mempool 和consense以及wallet模块做相应的更新
func (chain *BlockChain) SendDelBlockEvent(block *types.BlockDetail) (err error) {
	if chain.qclient == nil {
		fmt.Println("chain client not bind message queue.")
		err := errors.New("chain client not bind message queue")
		return err
	}
	if block == nil {
		chainlog.Error("SendDelBlockEvent block is null")
		return nil
	}
	chainlog.Error("SendDelBlockEvent", "Height", block.Block.Height)
	chainlog.Debug("SendDelBlockEvent -->>wallet", "height", block.GetBlock().GetHeight())
	msg := chain.qclient.NewMessage("wallet", types.EventDelBlock, block)
	chain.qclient.Send(msg, false)

	return nil
}
