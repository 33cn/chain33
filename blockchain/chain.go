package blockchain

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	//cache 存贮的block个数
	DefCacheSize        int64 = 512
	cachelock           sync.Mutex
	zeroHash            [32]byte
	InitBlockNum        int64 = 1024 //节点刚启动时从db向index和bestchain缓存中添加的blocknode数
	isStrongConsistency       = false

	chainlog                    = log.New("module", "blockchain")
	FutureBlockDelayTime  int64 = 1
	isRecordBlockSequence       = false //是否记录add或者del block的序列，方便blcokchain的恢复通过记录的序列表
	isParaChain                 = false //是否是平行链。平行链需要记录Sequence信息

)

const maxFutureBlocks = 256

type BlockChain struct {
	client queue.Client
	// 永久存储数据到db中
	blockStore *BlockStore
	//cache  缓存block方便快速查询
	cache      map[int64]*list.Element
	cacheSize  int64
	cacheQueue *list.List
	cfg        *types.BlockChain
	task       *Task
	query      *Query

	//记录收到的最新广播的block高度,用于节点追赶active链
	rcvLastBlockHeight int64

	//记录本节点已经同步的block高度,用于节点追赶active链,处理节点分叉不同步的场景
	synBlockHeight int64

	//记录peer的最新block高度,用于节点追赶active链
	peerList            PeerInfoList
	recvwg              *sync.WaitGroup
	synblock            chan struct{}
	quit                chan struct{}
	isclosed            int32
	isbatchsync         int32
	firstcheckbestchain int32 //节点启动之后首次检测最优链的标志

	// 孤儿链
	orphanPool *OrphanPool
	// 主链或者侧链的blocknode信息
	index *blockIndex
	//当前主链
	bestChain *chainView

	chainLock sync.RWMutex
	//blockchain的启动时间
	startTime time.Time

	//标记本节点是否已经追赶上主链
	isCaughtUp bool

	//同步block批量写数据库时，是否需要刷盘的标志。
	//非固态硬盘的电脑可以关闭刷盘，提高同步性能.
	cfgBatchSync bool

	//记录可疑故障节点peer信息
	//在ExecBlock执行失败时记录对应的peerid以及故障区块的高度和hash
	faultPeerList map[string]*FaultPeerInfo

	bestChainPeerList map[string]*BestPeerInfo

	//记录futureblocks
	futureBlocks *lru.Cache // future blocks are broadcast later processing

}

func New(cfg *types.BlockChain) *BlockChain {
	initConfig(cfg)
	futureBlocks, _ := lru.New(maxFutureBlocks)

	blockchain := &BlockChain{
		cache:               make(map[int64]*list.Element),
		cacheSize:           DefCacheSize,
		cacheQueue:          list.New(),
		rcvLastBlockHeight:  -1,
		synBlockHeight:      -1,
		peerList:            nil,
		cfg:                 cfg,
		recvwg:              &sync.WaitGroup{},
		task:                newTask(160 * time.Second),
		quit:                make(chan struct{}),
		synblock:            make(chan struct{}, 1),
		orphanPool:          NewOrphanPool(),
		index:               newBlockIndex(),
		isCaughtUp:          false,
		isbatchsync:         1,
		firstcheckbestchain: 0,
		cfgBatchSync:        cfg.Batchsync,
		faultPeerList:       make(map[string]*FaultPeerInfo),
		bestChainPeerList:   make(map[string]*BestPeerInfo),
		futureBlocks:        futureBlocks,
	}
	return blockchain
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
	isStrongConsistency = cfg.IsStrongConsistency
	isRecordBlockSequence = cfg.IsRecordBlockSequence
	isParaChain = cfg.IsParaChain
}

func (chain *BlockChain) Close() {
	//等待所有的写线程退出，防止数据库写到一半被暂停
	atomic.StoreInt32(&chain.isclosed, 1)

	//退出线程
	close(chain.quit)

	//wait for recvwg quit:
	chain.recvwg.Wait()

	//退出接受数据, 在最后一个block写磁盘时addtx还需要接受数据
	chain.client.Close()

	//关闭数据库
	chain.blockStore.db.Close()
	chainlog.Info("blockchain module closed")
}

func (chain *BlockChain) SetQueueClient(client queue.Client) {
	chain.client = client
	chain.client.Sub("blockchain")

	blockStoreDB := dbm.NewDB("blockchain", chain.cfg.Driver, chain.cfg.DbPath, chain.cfg.DbCache)
	blockStore := NewBlockStore(blockStoreDB, client)
	chain.blockStore = blockStore
	stateHash := chain.getStateHash()
	chain.query = NewQuery(blockStoreDB, chain.client, stateHash)

	//获取lastblock从数据库,创建bestviewtip节点
	chain.InitIndexAndBestView()

	//startTime
	chain.startTime = types.Now()

	//获取数据库中最新的区块高度，以及blockchain的数据库版本号
	curheight := chain.GetBlockHeight()
	curdbver := chain.blockStore.GetDbVersion()
	if curdbver == 0 && curheight == -1 {
		chain.blockStore.SetDbVersion(1)
	}
	//recv 消息的处理
	go chain.ProcRecvMsg()
	if !chain.cfg.IsParaChain {
		// 定时检测/同步block
		go chain.SynRoutine()

		// 定时处理futureblock
		go chain.UpdateRoutine()
	}

}

func (chain *BlockChain) getStateHash() []byte {
	blockhight := chain.GetBlockHeight()
	blockdetail, err := chain.GetBlock(blockhight)
	if err != nil {
		return zeroHash[:]
	}
	if blockdetail != nil {
		return blockdetail.GetBlock().GetStateHash()
	}
	return zeroHash[:]
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
	amount, err := txresult.GetTx().Amount()
	if err != nil {
		// return nil, err
		amount = 0
	}
	TransactionDetail.Amount = amount
	TransactionDetail.ActionName = txresult.GetTx().ActionName()

	//获取from地址
	addr := txresult.GetTx().From()
	TransactionDetail.Fromaddr = addr
	if TransactionDetail.GetTx().IsWithdraw() {
		//swap from and to
		TransactionDetail.Fromaddr, TransactionDetail.Tx.To = TransactionDetail.Tx.To, TransactionDetail.Fromaddr
	}
	chainlog.Debug("ProcQueryTxMsg", "TransactionDetail", TransactionDetail.String())

	return &TransactionDetail, nil
}

func (chain *BlockChain) GetDuplicateTxHashList(txhashlist *types.TxHashList) (duptxhashlist *types.TxHashList) {

	var dupTxHashList types.TxHashList

	for _, txhash := range txhashlist.Hashes {
		txresult, err := chain.GetTxResultFromDb(txhash)
		if err == nil && txresult != nil {
			dupTxHashList.Hashes = append(dupTxHashList.Hashes, txhash)
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
		chainlog.Error("ProcGetBlockDetailsMsg Startheight err", "startheight", requestblock.Start, "curheight", blockhight)
		return nil, types.ErrStartHeight
	}
	if requestblock.GetStart() > requestblock.GetEnd() {
		chainlog.Error("ProcGetBlockDetailsMsg input must Start <= End:", "Startheight", requestblock.Start, "Endheight", requestblock.End)
		return nil, types.ErrEndLessThanStartHeight
	}

	chainlog.Debug("ProcGetBlockDetailsMsg", "Start", requestblock.Start, "End", requestblock.End, "Isdetail", requestblock.IsDetail)

	end := requestblock.End
	if requestblock.End > blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetBlockDetailsMsg", "blockscount", count)

	var blocks types.BlockDetails
	blocks.Items = make([]*types.BlockDetail, count)
	j := 0
	for i := start; i <= end; i++ {
		block, err := chain.GetBlock(i)
		if err == nil && block != nil {
			if requestblock.IsDetail {
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
	if requestblock.IsDetail {
		for _, blockinfo := range blocks.Items {
			chainlog.Debug("ProcGetBlocksMsg", "blockinfo", blockinfo.String())
		}
	}
	return &blocks, nil
}

//处理从peer对端同步过来的block消息
func (chain *BlockChain) ProcAddBlockMsg(broadcast bool, blockdetail *types.BlockDetail, pid string) (err error) {
	block := blockdetail.Block
	if block == nil {
		chainlog.Error("ProcAddBlockMsg input block is null")
		return types.ErrInputPara
	}
	ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, true, -1)
	//非孤儿block或者已经存在的block
	if (!isorphan && err == nil) || (err == types.ErrBlockExist) {
		chain.task.Done(blockdetail.Block.GetHeight())
	}
	//此处只更新广播block的高度
	if broadcast {
		chain.UpdateRcvCastBlkHeight(blockdetail.Block.Height)
	}

	chainlog.Debug("ProcAddBlockMsg result:", "height", blockdetail.Block.Height, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(blockdetail.Block.Hash()), "err", err)

	return err
}

//blockchain 模块add block到db之后通知mempool 和consense模块做相应的更新
func (chain *BlockChain) SendAddBlockEvent(block *types.BlockDetail) (err error) {
	if chain.client == nil {
		fmt.Println("chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}
	if block == nil {
		chainlog.Error("SendAddBlockEvent block is null")
		return types.ErrInputPara
	}
	chainlog.Debug("SendAddBlockEvent", "Height", block.Block.Height)

	chainlog.Debug("SendAddBlockEvent -->>mempool")
	msg := chain.client.NewMessage("mempool", types.EventAddBlock, block)
	chain.client.Send(msg, false)

	chainlog.Debug("SendAddBlockEvent -->>consensus")

	msg = chain.client.NewMessage("consensus", types.EventAddBlock, block)
	chain.client.Send(msg, false)

	chainlog.Debug("SendAddBlockEvent -->>wallet", "height", block.GetBlock().GetHeight())
	msg = chain.client.NewMessage("wallet", types.EventAddBlock, block)
	chain.client.Send(msg, false)

	return nil
}

//blockchain模块广播此block到网络中
func (chain *BlockChain) SendBlockBroadcast(block *types.BlockDetail) {
	if chain.client == nil {
		fmt.Println("chain client not bind message queue.")
		return
	}
	if block == nil {
		chainlog.Error("SendBlockBroadcast block is null")
		return
	}
	chainlog.Debug("SendBlockBroadcast", "Height", block.Block.Height, "hash", common.ToHex(block.Block.Hash()))

	msg := chain.client.NewMessage("p2p", types.EventBlockBroadcast, block.Block)
	chain.client.Send(msg, false)
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
		blockinfo, err := chain.blockStore.LoadBlockByHeight(height)
		if blockinfo != nil {
			if len(blockinfo.Receipts) == 0 && len(blockinfo.Block.Txs) != 0 {
				chainlog.Debug("GetBlock  LoadBlock Receipts ==0", "height", height)
			}
			chain.cacheBlock(blockinfo)
			return blockinfo, nil
		}
		return nil, err
	}
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

	if requestblock.GetStart() > requestblock.GetEnd() {
		chainlog.Error("ProcGetHeadersMsg input must Start <= End:", "Startheight", requestblock.Start, "Endheight", requestblock.End)
		return nil, types.ErrEndLessThanStartHeight
	}

	if requestblock.Start > blockhight {
		chainlog.Error("ProcGetHeadersMsg Startheight err", "startheight", requestblock.Start, "curheight", blockhight)
		return nil, types.ErrStartHeight
	}
	end := requestblock.End
	if requestblock.End > blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetHeadersMsg", "headerscount", count)
	if count < 1 {
		chainlog.Error("ProcGetHeadersMsg count err", "startheight", requestblock.Start, "endheight", requestblock.End, "curheight", blockhight)
		return nil, types.ErrEndLessThanStartHeight
	}
	var headers types.Headers
	headers.Items = make([]*types.Header, count)
	j := 0
	for i := start; i <= end; i++ {
		head, err := chain.blockStore.GetBlockHeaderByHeight(i)
		if err == nil && head != nil {
			headers.Items[j] = head
		} else {
			return nil, err
		}
		j++
	}
	chainlog.Debug("getHeaders", "len", len(headers.Items), "start", start, "end", end)
	return &headers, nil
}

func (chain *BlockChain) ProcGetLastHeaderMsg() (*types.Header, error) {
	//首先从缓存中获取最新的blockheader
	head := chain.blockStore.LastHeader()
	if head == nil {
		blockhight := chain.GetBlockHeight()
		tmpHead, err := chain.blockStore.GetBlockHeaderByHeight(blockhight)
		if err == nil && tmpHead != nil {
			chainlog.Error("ProcGetLastHeaderMsg from cache is nil.", "blockhight", blockhight, "hash", common.ToHex(tmpHead.Hash))
			return tmpHead, nil
		} else {
			return nil, err
		}
	}
	return head, nil
}

func (chain *BlockChain) ProcGetLastBlockMsg() (respblock *types.Block, err error) {
	block := chain.blockStore.LastBlock()
	return block, nil
}

func (chain *BlockChain) ProcGetBlockByHashMsg(hash []byte) (respblock *types.BlockDetail, err error) {
	blockhight, err := chain.blockStore.GetHeightByBlockHash(hash)
	if err != nil {
		return nil, err
	}
	blockdetail, err := chain.GetBlock(blockhight)
	if err != nil {
		return nil, err
	}
	return blockdetail, nil
}

//获取地址对应的所有交易信息
//存储格式key:addr:flag:height ,value:txhash
//key=addr :获取本地参与的所有交易
//key=addr:1 :获取本地作为from方的所有交易
//key=addr:2 :获取本地作为to方的所有交易
func (chain *BlockChain) ProcGetTransactionByAddr(addr *types.ReqAddr) (*types.ReplyTxInfos, error) {
	if addr == nil || len(addr.Addr) == 0 {
		return nil, types.ErrInputPara
	}
	//入参数校验
	curheigt := chain.GetBlockHeight()
	if addr.GetHeight() > curheigt || addr.GetHeight() < -1 {
		chainlog.Error("ProcGetTransactionByAddr Height err")
		return nil, types.ErrInputPara
	}
	if addr.GetDirection() != 0 && addr.GetDirection() != 1 {
		chainlog.Error("ProcGetTransactionByAddr Direction err")
		return nil, types.ErrInputPara
	}
	if addr.GetIndex() < 0 || addr.GetIndex() > types.MaxTxsPerBlock {
		chainlog.Error("ProcGetTransactionByAddr Index err")
		return nil, types.ErrInputPara
	}
	//查询的drivers--> main 驱动的名称
	//查询的方法：  --> GetTxsByAddr
	//查询的参数：  --> interface{} 类型
	txinfos, err := chain.query.Query(types.ExecName("coins"), "GetTxsByAddr", types.Encode(addr))
	if err != nil {
		chainlog.Info("ProcGetTransactionByAddr does not exist tx!", "addr", addr, "err", err)
		return nil, err
	}
	return txinfos.(*types.ReplyTxInfos), nil
}

//type TransactionDetails struct {
//	Txs []*Transaction
//}
//通过hashs获取交易详情
func (chain *BlockChain) ProcGetTransactionByHashes(hashs [][]byte) (TxDetails *types.TransactionDetails, err error) {

	//chainlog.Info("ProcGetTransactionByHashes", "txhash len:", len(hashs))
	var txDetails types.TransactionDetails
	for _, txhash := range hashs {
		txresult, err := chain.GetTxResultFromDb(txhash)
		if err == nil && txresult != nil {
			var txDetail types.TransactionDetail
			txDetail.Receipt = txresult.Receiptdate
			txDetail.Tx = txresult.GetTx()
			txDetail.Blocktime = txresult.GetBlocktime()
			txDetail.Height = txresult.GetHeight()
			txDetail.Index = int64(txresult.GetIndex())

			//获取Amount
			amount, err := txresult.GetTx().Amount()
			if err != nil {
				txDetail.Amount = 0
			} else {
				txDetail.Amount = amount
			}

			txDetail.ActionName = txresult.GetTx().ActionName()

			//获取from地址
			txDetail.Fromaddr = txresult.GetTx().From()
			if txDetail.GetTx().IsWithdraw() {
				//swap from and to
				txDetail.Fromaddr, txDetail.Tx.To = txDetail.Tx.To, txDetail.Fromaddr
			}
			//chainlog.Debug("ProcGetTransactionByHashes", "txDetail", txDetail.String())
			txDetails.Txs = append(txDetails.Txs, &txDetail)
		} else {
			txDetails.Txs = append(txDetails.Txs, nil)
			chainlog.Debug("ProcGetTransactionByHashes hash no exit", "txhash", common.ToHex(txhash))
		}
	}
	return &txDetails, nil
}

//type  BlockOverview {
//	Header head = 1;
//	int64  txCount = 2;
//	repeated bytes txHashes = 3;}
//获取BlockOverview
func (chain *BlockChain) ProcGetBlockOverview(ReqHash *types.ReqHash) (*types.BlockOverview, error) {

	if ReqHash == nil {
		chainlog.Error("ProcGetBlockOverview input err!")
		return nil, types.ErrInputPara
	}
	//通过blockhash获取blockheight
	height, err := chain.blockStore.GetHeightByBlockHash(ReqHash.Hash)
	if err != nil {
		chainlog.Error("ProcGetBlockOverview:GetHeightByBlockHash err")
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
	header.Hash = block.Block.Hash()
	header.TxCount = int64(len(block.Block.GetTxs()))
	header.Difficulty = block.Block.Difficulty
	header.Signature = block.Block.Signature

	blockOverview.Head = &header

	blockOverview.TxCount = int64(len(block.Block.GetTxs()))

	txhashs := make([][]byte, blockOverview.TxCount)
	for index, tx := range block.Block.Txs {
		txhashs[index] = tx.Hash()
	}
	blockOverview.TxHashes = txhashs
	chainlog.Debug("ProcGetBlockOverview", "blockOverview:", blockOverview.String())
	return &blockOverview, nil
}

//type  AddrOverview {
//	int64 reciver = 1;
//	int64 balance = 2;
//	int64 txCount = 3;}
//获取addrOverview
func (chain *BlockChain) ProcGetAddrOverview(addr *types.ReqAddr) (*types.AddrOverview, error) {

	if addr == nil || len(addr.Addr) == 0 {
		chainlog.Error("ProcGetAddrOverview input err!")
		return nil, types.ErrInputPara
	}
	chainlog.Debug("ProcGetAddrOverview", "Addr", addr.GetAddr())

	var addrOverview types.AddrOverview

	//获取地址的reciver
	amount, err := chain.query.Query(types.ExecName("coins"), "GetAddrReciver", types.Encode(addr))
	if err != nil {
		chainlog.Error("ProcGetAddrOverview", "GetAddrReciver err", err)
		addrOverview.Reciver = 0
	} else {
		addrOverview.Reciver = amount.(*types.Int64).GetData()
	}
	beg := types.Now()
	curdbver := chain.blockStore.GetDbVersion()
	var reqkey types.ReqKey

	if curdbver == 0 {
		//旧的数据库获取地址对应的交易count，使用前缀查找的方式获取
		//前缀和util.go 文件中的CalcTxAddrHashKey保持一致
		reqkey.Key = []byte(fmt.Sprintf("TxAddrHash:%s:%s", addr.Addr, ""))
		count, err := chain.query.Query(types.ExecName("coins"), "GetPrefixCount", types.Encode(&reqkey))
		if err != nil {
			chainlog.Error("ProcGetAddrOverview", "GetPrefixCount err", err)
			addrOverview.TxCount = 0
		} else {
			addrOverview.TxCount = count.(*types.Int64).GetData()
		}
		chainlog.Debug("GetPrefixCount", "cost ", types.Since(beg))
	} else {
		//新的数据库直接使用key值查找就可以
		//前缀和util.go 文件中的calcAddrTxsCountKey保持一致
		reqkey.Key = []byte(fmt.Sprintf("AddrTxsCount:%s", addr.Addr))
		count, err := chain.query.Query(types.ExecName("coins"), "GetAddrTxsCount", types.Encode(&reqkey))
		if err != nil {
			chainlog.Error("ProcGetAddrOverview", "GetAddrTxsCount err", err)
			addrOverview.TxCount = 0
		} else {
			addrOverview.TxCount = count.(*types.Int64).GetData()
		}
		chainlog.Debug("GetAddrTxsCount", "cost ", types.Since(beg))
	}
	return &addrOverview, nil
}

//通过blockheight 获取blockhash
func (chain *BlockChain) ProcGetBlockHash(height *types.ReqInt) (*types.ReplyHash, error) {
	if height == nil || 0 > height.GetHeight() {
		chainlog.Error("ProcGetBlockHash input err!")
		return nil, types.ErrInputPara
	}
	CurHeight := chain.GetBlockHeight()
	if height.GetHeight() > CurHeight {
		chainlog.Error("ProcGetBlockHash input height err!")
		return nil, types.ErrInputPara
	}
	var ReplyHash types.ReplyHash
	block, err := chain.GetBlock(height.GetHeight())
	if err != nil {
		return nil, err
	}
	ReplyHash.Hash = block.Block.Hash()
	return &ReplyHash, nil
}

//blockchain 模块 del block从db之后通知mempool 和consense以及wallet模块做相应的更新
func (chain *BlockChain) SendDelBlockEvent(block *types.BlockDetail) (err error) {
	if chain.client == nil {
		fmt.Println("chain client not bind message queue.")
		err := types.ErrClientNotBindQueue
		return err
	}
	if block == nil {
		chainlog.Error("SendDelBlockEvent block is null")
		return nil
	}

	chainlog.Debug("SendDelBlockEvent -->>mempool&consensus&wallet", "height", block.GetBlock().GetHeight())

	msg := chain.client.NewMessage("consensus", types.EventDelBlock, block)
	chain.client.Send(msg, false)

	msg = chain.client.NewMessage("mempool", types.EventDelBlock, block)
	chain.client.Send(msg, false)

	msg = chain.client.NewMessage("wallet", types.EventDelBlock, block)
	chain.client.Send(msg, false)

	return nil
}

// 第一次启动之后需要将数据库中最新的128个block的node添加到index和bestchain中
// 主要是为了接下来分叉时的block处理，.........todo
func (chain *BlockChain) InitIndexAndBestView() {
	//获取lastblocks从数据库,创建bestviewtip节点
	var node *blockNode
	var prevNode *blockNode
	var height int64
	var initflag = false
	curheight := chain.blockStore.height
	if curheight == -1 {
		node = newPreGenBlockNode()
		node.parent = nil
		chain.bestChain = newChainView(node)
		chain.index.AddNode(node)
		return
	} else {
		if curheight >= InitBlockNum {
			height = curheight - InitBlockNum
		} else {
			height = 0
		}
		for ; height <= curheight; height++ {
			block, _ := chain.blockStore.LoadBlockByHeight(height)
			if block == nil {
				return
			}
			newNode := newBlockNode(false, block.Block, "self", -1)
			newNode.parent = prevNode
			prevNode = newNode

			chain.index.AddNode(newNode)
			if !initflag {
				chain.bestChain = newChainView(newNode)
				initflag = true
			} else {
				chain.bestChain.SetTip(newNode)

			}

		}
	}
}

//定时延时广播futureblock
func (chain *BlockChain) UpdateRoutine() {

	//1秒尝试检测一次futureblock，futureblock的time小于当前系统时间就广播此block
	futureblockTicker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-chain.quit:
			//chainlog.Info("UpdateRoutine quit")
			return
		case <-futureblockTicker.C:
			chain.ProcFutureBlocks()
		}
	}
}

//循环遍历所有futureblocks，当futureblock的block生成time小于当前系统时间就将此block广播出去
func (chain *BlockChain) ProcFutureBlocks() {
	for _, hash := range chain.futureBlocks.Keys() {
		if block, exist := chain.futureBlocks.Peek(hash); exist {
			if block != nil {
				blockdetail := block.(*types.BlockDetail)
				//block产生的时间小于当前时间，广播此block，然后将此block从futureblocks中移除
				if types.Now().Unix() > blockdetail.Block.BlockTime {
					chain.SendBlockBroadcast(blockdetail)
					chain.futureBlocks.Remove(hash)
					chainlog.Debug("ProcFutureBlocks Remove", "height", blockdetail.Block.Height, "hash", common.ToHex(blockdetail.Block.Hash()), "blocktime", blockdetail.Block.BlockTime, "curtime", types.Now().Unix())
				}
			}
		}
	}
}

//通过blockhash 获取对应的block信息
func (chain *BlockChain) GetBlockByHashes(hashes [][]byte) (respblocks *types.BlockDetails, err error) {
	var blocks types.BlockDetails
	for _, hash := range hashes {
		block, err := chain.blockStore.LoadBlockByHash(hash)
		if err == nil && block != nil {
			blocks.Items = append(blocks.Items, block)
		} else {
			blocks.Items = append(blocks.Items, nil)
		}
	}
	return &blocks, nil
}

//通过记录的block序列号获取blockd序列存储的信息
func (chain *BlockChain) GetBlockSequences(requestblock *types.ReqBlocks) (*types.BlockSequences, error) {
	blockLastSeq, _ := chain.blockStore.LoadBlockLastSequence()
	if requestblock.Start > blockLastSeq {
		chainlog.Error("GetBlockSequences StartSeq err", "startSeq", requestblock.Start, "lastSeq", blockLastSeq)
		return nil, types.ErrStartHeight
	}
	if requestblock.GetStart() > requestblock.GetEnd() {
		chainlog.Error("GetBlockSequences input must Start <= End:", "startSeq", requestblock.Start, "endSeq", requestblock.End)
		return nil, types.ErrEndLessThanStartHeight
	}

	end := requestblock.End
	if requestblock.End > blockLastSeq {
		end = blockLastSeq
	}
	start := requestblock.Start
	count := end - start + 1

	chainlog.Debug("GetBlockSequences", "Start", requestblock.Start, "End", requestblock.End, "lastSeq", blockLastSeq, "counts", count)

	var blockSequences types.BlockSequences

	for i := start; i <= end; i++ {
		blockSequence, err := chain.blockStore.GetBlockSequence(i)
		if err == nil && blockSequence != nil {
			blockSequences.Items = append(blockSequences.Items, blockSequence)
		} else {
			blockSequences.Items = append(blockSequences.Items, nil)
		}
	}
	return &blockSequences, nil
}

//处理共识过来的删除block的消息，目前只提供给平行链使用
func (chain *BlockChain) ProcDelParaChainBlockMsg(broadcast bool, ParaChainblockdetail *types.ParaChainBlockDetail, pid string) (err error) {
	if ParaChainblockdetail == nil || ParaChainblockdetail.GetBlockdetail() == nil || ParaChainblockdetail.GetBlockdetail().GetBlock() == nil {
		chainlog.Error("ProcDelParaChainBlockMsg input block is null")
		return types.ErrInputPara
	}
	blockdetail := ParaChainblockdetail.GetBlockdetail()
	block := ParaChainblockdetail.GetBlockdetail().GetBlock()
	sequence := ParaChainblockdetail.GetSequence()

	ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, false, sequence)
	chainlog.Debug("ProcDelParaChainBlockMsg result:", "height", block.Height, "sequence", sequence, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash()), "err", err)

	return err
}

//处理共识过来的add block的消息，目前只提供给平行链使用
func (chain *BlockChain) ProcAddParaChainBlockMsg(broadcast bool, ParaChainblockdetail *types.ParaChainBlockDetail, pid string) (err error) {
	if ParaChainblockdetail == nil || ParaChainblockdetail.GetBlockdetail() == nil || ParaChainblockdetail.GetBlockdetail().GetBlock() == nil {
		chainlog.Error("ProcAddParaChainBlockMsg input block is null")
		return types.ErrInputPara
	}
	blockdetail := ParaChainblockdetail.GetBlockdetail()
	block := ParaChainblockdetail.GetBlockdetail().GetBlock()
	sequence := ParaChainblockdetail.GetSequence()

	ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, true, sequence)
	chainlog.Debug("ProcAddParaChainBlockMsg result:", "height", block.Height, "sequence", sequence, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash()), "err", err)

	return err
}

//处理共识过来的通过blockhash获取seq的消息，只提供add block时的seq，用于平行链block回退
func (chain *BlockChain) ProcGetSeqByHash(hash []byte) (int64, error) {
	if len(hash) == 0 {
		chainlog.Error("ProcGetSeqByHash input hash is null")
		return -1, types.ErrInputPara
	}
	seq, err := chain.blockStore.GetSequenceByHash(hash)
	chainlog.Debug("ProcGetSeqByHash", "blockhash", common.ToHex(hash), "seq", seq, "err", err)

	return seq, err
}
