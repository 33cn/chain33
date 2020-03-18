// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package blockchain 实现区块链模块，包含区块链存储
*/
package blockchain

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	lru "github.com/hashicorp/golang-lru"
)

//var
var (
	//cache 存贮的block个数
	MaxSeqCB             int64 = 20
	zeroHash             [32]byte
	InitBlockNum         int64 = 10240 //节点刚启动时从db向index和bestchain缓存中添加的blocknode数，和blockNodeCacheLimit保持一致
	chainlog                   = log.New("module", "blockchain")
	FutureBlockDelayTime int64 = 1
)

const maxFutureBlocks = 256

//BlockChain 区块链结构体
type BlockChain struct {
	client queue.Client
	cache  *BlockCache
	// 永久存储数据到db中
	blockStore  *BlockStore
	pushseq     *pushseq
	pushservice *PushService1
	//cache  缓存block方便快速查询
	cfg          *types.BlockChain
	syncTask     *Task
	downLoadTask *Task

	query *Query

	//记录收到的最新广播的block高度,用于节点追赶active链
	rcvLastBlockHeight int64

	//记录本节点已经同步的block高度,用于节点追赶active链,处理节点分叉不同步的场景
	synBlockHeight int64

	//记录当前已经chunk的高度
	curChunkNum  int64

	//记录peer的最新block高度,用于节点追赶active链
	peerList PeerInfoList
	recvwg   *sync.WaitGroup
	tickerwg *sync.WaitGroup
	reducewg *sync.WaitGroup

	synblock            chan struct{}
	quit                chan struct{}
	isclosed            int32
	runcount            int32
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

	//downLoad block info
	downLoadInfo       *DownLoadInfo
	isFastDownloadSync bool //当本节点落后很多时，可以先下载区块到db，启动单独的goroutine去执行block

	isRecordBlockSequence bool //是否记录add或者del block的序列，方便blcokchain的恢复通过记录的序列表
	isParaChain           bool //是否是平行链。平行链需要记录Sequence信息
	isStrongConsistency   bool
	//lock
	synBlocklock        sync.Mutex
	peerMaxBlklock      sync.Mutex
	castlock            sync.Mutex
	ntpClockSynclock    sync.Mutex
	faultpeerlock       sync.Mutex
	bestpeerlock        sync.Mutex
	downLoadlock        sync.Mutex
	fastDownLoadSynLock sync.Mutex
	isNtpClockSync      bool //ntp时间是否同步

	//cfg
	MaxFetchBlockNum int64 //一次最多申请获取block个数
	TimeoutSeconds   int64
	blockSynInterVal time.Duration
	DefCacheSize     int64
	failed           int32

	blockOnChain   *BlockOnChain
	onChainTimeout int64
}

//New new
func New(cfg *types.Chain33Config) *BlockChain {
	mcfg := cfg.GetModuleConfig().BlockChain
	futureBlocks, err := lru.New(maxFutureBlocks)
	if err != nil {
		panic("when New BlockChain lru.New return err")
	}
	defCacheSize := int64(128)
	if mcfg.DefCacheSize > 0 {
		defCacheSize = mcfg.DefCacheSize
	}
	blockchain := &BlockChain{
		cache:              NewBlockCache(cfg, defCacheSize),
		DefCacheSize:       defCacheSize,
		rcvLastBlockHeight: -1,
		synBlockHeight:     -1,
		curChunkNum:        -1,
		peerList:           nil,
		cfg:                mcfg,
		recvwg:             &sync.WaitGroup{},
		tickerwg:           &sync.WaitGroup{},
		reducewg:           &sync.WaitGroup{},

		syncTask:     newTask(300 * time.Second), //考虑到区块交易多时执行耗时，需要延长task任务的超时时间
		downLoadTask: newTask(300 * time.Second),

		quit:                make(chan struct{}),
		synblock:            make(chan struct{}, 1),
		orphanPool:          NewOrphanPool(cfg),
		index:               newBlockIndex(),
		isCaughtUp:          false,
		isbatchsync:         1,
		firstcheckbestchain: 0,
		cfgBatchSync:        mcfg.Batchsync,
		faultPeerList:       make(map[string]*FaultPeerInfo),
		bestChainPeerList:   make(map[string]*BestPeerInfo),
		futureBlocks:        futureBlocks,
		downLoadInfo:        &DownLoadInfo{},
		isNtpClockSync:      true,
		MaxFetchBlockNum:    128 * 6, //一次最多申请获取block个数
		TimeoutSeconds:      2,
		isFastDownloadSync:  true,
		blockOnChain:        &BlockOnChain{},
		onChainTimeout:      0,
	}
	blockchain.initConfig(cfg)
	return blockchain
}

func (chain *BlockChain) initConfig(cfg *types.Chain33Config) {
	mcfg := cfg.GetModuleConfig().BlockChain
	if cfg.IsEnable("TxHeight") && chain.DefCacheSize <= (types.LowAllowPackHeight+types.HighAllowPackHeight+1) {
		panic("when Enable TxHeight DefCacheSize must big than types.LowAllowPackHeight")
	}

	if mcfg.MaxFetchBlockNum > 0 {
		chain.MaxFetchBlockNum = mcfg.MaxFetchBlockNum
	}

	if mcfg.TimeoutSeconds > 0 {
		chain.TimeoutSeconds = mcfg.TimeoutSeconds
	}
	chain.blockSynInterVal = time.Duration(chain.TimeoutSeconds)
	chain.isStrongConsistency = mcfg.IsStrongConsistency
	chain.isRecordBlockSequence = mcfg.IsRecordBlockSequence
	chain.isParaChain = mcfg.IsParaChain
	cfg.S("quickIndex", mcfg.EnableTxQuickIndex)
	cfg.S("reduceLocaldb", mcfg.EnableReduceLocaldb)

	if mcfg.OnChainTimeout > 0 {
		chain.onChainTimeout = mcfg.OnChainTimeout
	}
	chain.initOnChainTimeout()
}

//Close 关闭区块链
func (chain *BlockChain) Close() {
	//等待所有的写线程退出，防止数据库写到一半被暂停
	atomic.StoreInt32(&chain.isclosed, 1)

	//退出线程
	close(chain.quit)

	//等待执行完成
	for atomic.LoadInt32(&chain.runcount) > 0 {
		time.Sleep(time.Microsecond)
	}
	chain.client.Close()
	//wait for recvwg quit:
	chainlog.Info("blockchain wait for recvwg quit")
	chain.recvwg.Wait()

	//wait for tickerwg quit:
	chainlog.Info("blockchain wait for tickerwg quit")
	chain.tickerwg.Wait()

	//wait for reducewg quit:
	chainlog.Info("blockchain wait for reducewg quit")
	chain.reducewg.Wait()

	//关闭数据库
	chain.blockStore.db.Close()
	chainlog.Info("blockchain module closed")
}

//SetQueueClient 设置队列
func (chain *BlockChain) SetQueueClient(client queue.Client) {
	chain.client = client
	chain.client.Sub("blockchain")

	blockStoreDB := dbm.NewDB("blockchain", chain.cfg.Driver, chain.cfg.DbPath, chain.cfg.DbCache)
	blockStore := NewBlockStore(chain, blockStoreDB, client)
	chain.blockStore = blockStore
	stateHash := chain.getStateHash()
	chain.query = NewQuery(blockStoreDB, chain.client, stateHash)

	chain.pushservice = newPushService(chain.blockStore, chain.blockStore)
	chain.pushseq = newpushseq(chain.blockStore, chain.pushservice.pushStore)
	//startTime
	chain.startTime = types.Now()
	// 获取当前chunk高度
	chain.curChunkNum = chain.GetCurChunkNum()

	//recv 消息的处理，共识模块需要获取lastblock从数据库中
	chain.recvwg.Add(1)
	//初始化blockchian模块
	chain.pushseq.init()
	chain.InitBlockChain()
	go chain.ProcRecvMsg()
}

//Wait for ready
func (chain *BlockChain) Wait() {}

//GetStore only used for test
func (chain *BlockChain) GetStore() *BlockStore {
	return chain.blockStore
}

//GetOrphanPool 获取孤儿链
func (chain *BlockChain) GetOrphanPool() *OrphanPool {
	return chain.orphanPool
}

//InitBlockChain 区块链初始化
func (chain *BlockChain) InitBlockChain() {
	//isRecordBlockSequence配置的合法性检测
	seqStatus := chain.blockStore.CheckSequenceStatus(chain.isRecordBlockSequence)
	if seqStatus == seqStatusNeedCreate {
		chain.blockStore.CreateSequences(100000)
	}

	//先缓存最新的128个block信息到cache中
	curheight := chain.GetBlockHeight()
	if chain.client.GetConfig().IsEnable("TxHeight") {
		chain.InitCache(curheight)
	}
	//获取数据库中最新的10240个区块加载到index和bestview链中
	beg := types.Now()
	chain.InitIndexAndBestView()
	chainlog.Info("InitIndexAndBestView", "cost", types.Since(beg))

	//获取数据库中最新的区块高度，以及blockchain的数据库版本号
	curdbver := chain.blockStore.GetDbVersion()
	if curdbver == 0 && curheight == -1 {
		curdbver = 1
		err := chain.blockStore.SetDbVersion(curdbver)
		//设置失败后恢复成原来的值保持和types.S("dbversion", curdbver)设置的版本一致
		if err != nil {
			curdbver = 0
			chainlog.Error("InitIndexAndBestView SetDbVersion ", "err", err)
		}
	}
	cfg := chain.client.GetConfig()
	cfg.S("dbversion", curdbver)
	if !chain.cfg.IsParaChain && chain.cfg.RollbackBlock <= 0 {
		// 定时检测/同步block
		go chain.SynRoutine()

		// 定时处理futureblock
		go chain.UpdateRoutine()
	}
	if chain.cfg.EnableShard {
		chain.tickerwg.Add(1)
		go chain.DeleteHaveChunkData()
	}
	//初始化默认DownLoadInfo
	chain.DefaultDownLoadInfo()
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

//SendAddBlockEvent blockchain 模块add block到db之后通知mempool 和consense模块做相应的更新
func (chain *BlockChain) SendAddBlockEvent(block *types.BlockDetail) (err error) {
	if chain.client == nil {
		chainlog.Error("SendAddBlockEvent: chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}
	if block == nil {
		chainlog.Error("SendAddBlockEvent block is null")
		return types.ErrInvalidParam
	}
	chainlog.Debug("SendAddBlockEvent", "Height", block.Block.Height)

	chainlog.Debug("SendAddBlockEvent -->>mempool")
	msg := chain.client.NewMessage("mempool", types.EventAddBlock, block)
	Err := chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Error("SendAddBlockEvent -->>mempool", "err", Err)
	}
	chainlog.Debug("SendAddBlockEvent -->>consensus")

	msg = chain.client.NewMessage("consensus", types.EventAddBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Error("SendAddBlockEvent -->>consensus", "err", Err)
	}
	chainlog.Debug("SendAddBlockEvent -->>wallet", "height", block.GetBlock().GetHeight())
	msg = chain.client.NewMessage("wallet", types.EventAddBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Error("SendAddBlockEvent -->>wallet", "err", Err)
	}
	return nil
}

//SendBlockBroadcast blockchain模块广播此block到网络中
func (chain *BlockChain) SendBlockBroadcast(block *types.BlockDetail) {
	cfg := chain.client.GetConfig()
	if chain.client == nil {
		chainlog.Error("SendBlockBroadcast: chain client not bind message queue.")
		return
	}
	if block == nil {
		chainlog.Error("SendBlockBroadcast block is null")
		return
	}
	chainlog.Debug("SendBlockBroadcast", "Height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))

	msg := chain.client.NewMessage("p2p", types.EventBlockBroadcast, block.Block)
	err := chain.client.Send(msg, false)
	if err != nil {
		chainlog.Error("SendBlockBroadcast", "Height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)), "err", err)
	}
}

//GetBlockHeight 获取区块高度
func (chain *BlockChain) GetBlockHeight() int64 {
	return chain.blockStore.Height()
}

//GetBlock 用于获取指定高度的block，首先在缓存中获取，如果不存在就从db中获取
func (chain *BlockChain) GetBlock(height int64) (block *types.BlockDetail, err error) {
	blockdetail := chain.cache.CheckcacheBlock(height)
	if blockdetail != nil {
		if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
			chainlog.Debug("GetBlock  CheckcacheBlock Receipts ==0", "height", height)
		}
		return blockdetail, nil
	}
	//从blockstore db中通过block height获取block
	blockinfo, err := chain.blockStore.LoadBlockByHeight(height)
	if blockinfo != nil {
		if len(blockinfo.Receipts) == 0 && len(blockinfo.Block.Txs) != 0 {
			chainlog.Debug("GetBlock  LoadBlock Receipts ==0", "height", height)
		}
		return blockinfo, nil
	}
	return nil, err

}

//SendDelBlockEvent blockchain 模块 del block从db之后通知mempool 和consense以及wallet模块做相应的更新
func (chain *BlockChain) SendDelBlockEvent(block *types.BlockDetail) (err error) {
	if chain.client == nil {
		chainlog.Error("SendDelBlockEvent: chain client not bind message queue.")
		err := types.ErrClientNotBindQueue
		return err
	}
	if block == nil {
		chainlog.Error("SendDelBlockEvent block is null")
		return nil
	}

	chainlog.Debug("SendDelBlockEvent -->>mempool&consensus&wallet", "height", block.GetBlock().GetHeight())

	msg := chain.client.NewMessage("consensus", types.EventDelBlock, block)
	Err := chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Debug("SendDelBlockEvent -->>consensus", "err", err)
	}
	msg = chain.client.NewMessage("mempool", types.EventDelBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Debug("SendDelBlockEvent -->>mempool", "err", err)
	}
	msg = chain.client.NewMessage("wallet", types.EventDelBlock, block)
	Err = chain.client.Send(msg, false)
	if Err != nil {
		chainlog.Debug("SendDelBlockEvent -->>wallet", "err", err)
	}
	return nil
}

//GetDB 获取DB
func (chain *BlockChain) GetDB() dbm.DB {
	return chain.blockStore.db
}

//InitCache 初始化缓存
func (chain *BlockChain) InitCache(height int64) {
	if height < 0 {
		return
	}
	for i := height - chain.DefCacheSize; i <= height; i++ {
		if i < 0 {
			i = 0
		}
		blockdetail, err := chain.GetBlock(i)
		if err != nil {
			panic(err)
		}
		chain.cache.cacheBlock(blockdetail)
	}
}

//InitIndexAndBestView 第一次启动之后需要将数据库中最新的128个block的node添加到index和bestchain中
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
	}
	if curheight >= InitBlockNum {
		height = curheight - InitBlockNum
	} else {
		height = 0
	}
	for ; height <= curheight; height++ {
		header, err := chain.blockStore.GetBlockHeaderByHeight(height)
		if header == nil {
			//开始升级localdb到2.0.0版本时需要兼容旧的存储方式
			header, err = chain.blockStore.getBlockHeaderByHeightOld(height)
			if header == nil {
				chainlog.Error("InitIndexAndBestView GetBlockHeaderByHeight", "height", height, "err", err)
				panic("InitIndexAndBestView fail!")
			}
		}

		newNode := newBlockNodeByHeader(false, header, "self", -1)
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

//UpdateRoutine 定时延时广播futureblock
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

//ProcFutureBlocks 循环遍历所有futureblocks，当futureblock的block生成time小于当前系统时间就将此block广播出去
func (chain *BlockChain) ProcFutureBlocks() {
	cfg := chain.client.GetConfig()
	for _, hash := range chain.futureBlocks.Keys() {
		if block, exist := chain.futureBlocks.Peek(hash); exist {
			if block != nil {
				blockdetail := block.(*types.BlockDetail)
				//block产生的时间小于当前时间，广播此block，然后将此block从futureblocks中移除
				if types.Now().Unix() > blockdetail.Block.BlockTime {
					chain.SendBlockBroadcast(blockdetail)
					chain.futureBlocks.Remove(hash)
					chainlog.Debug("ProcFutureBlocks Remove", "height", blockdetail.Block.Height, "hash", common.ToHex(blockdetail.Block.Hash(cfg)), "blocktime", blockdetail.Block.BlockTime, "curtime", types.Now().Unix())
				}
			}
		}
	}
}

//SetValueByKey 设置kv对到blockchain数据库
func (chain *BlockChain) SetValueByKey(kvs *types.LocalDBSet) error {
	return chain.blockStore.SetConsensusPara(kvs)
}

//GetValueByKey 通过key值从blockchain数据库中获取value值
func (chain *BlockChain) GetValueByKey(keys *types.LocalDBGet) *types.LocalReplyValue {
	return chain.blockStore.Get(keys)
}
