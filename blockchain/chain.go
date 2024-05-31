// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package blockchain 实现区块链模块，包含区块链存储
*/
package blockchain

import (
	"fmt"
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

// var
var (
	//cache 存贮的block个数
	zeroHash             [32]byte
	InitBlockNum         int64 = 10240 //节点刚启动时从db向index和bestchain缓存中添加的blocknode数，和blockNodeCacheLimit保持一致
	chainlog                   = log.New("module", "blockchain")
	FutureBlockDelayTime int64 = 1
)

const (
	maxFutureBlocks = 256

	maxActiveBlocks = 1024

	// 默认轻广播组装临时区块缓存， 100M
	maxActiveBlocksCacheSize = 100

	defaultChunkBlockNum = 1
)

// BlockChain 区块链结构体
type BlockChain struct {
	client        queue.Client
	blockCache    *BlockCache
	txHeightCache txHeightCacheType

	// 永久存储数据到db中
	blockStore *BlockStore
	push       *Push
	//cache  缓存block方便快速查询
	cfg             *types.BlockChain
	syncTask        *Task
	downLoadTask    *Task
	chunkRecordTask *Task

	query *Query

	//记录收到的最新广播的block高度,用于节点追赶active链
	rcvLastBlockHeight int64

	//记录本节点已经同步的block高度,用于节点追赶active链,处理节点分叉不同步的场景
	synBlockHeight int64

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
	downLoadInfo *DownLoadInfo
	downloadMode int //当本节点落后很多时，可以先下载区块到db，启动单独的goroutine去执行block

	isRecordBlockSequence bool //是否记录add或者del block的序列，方便blcokchain的恢复通过记录的序列表
	enablePushSubscribe   bool //是否允许推送订阅
	isParaChain           bool //是否是平行链。平行链需要记录Sequence信息
	isStrongConsistency   bool
	//lock
	synBlocklock     sync.Mutex
	peerMaxBlklock   sync.RWMutex
	castlock         sync.Mutex
	ntpClockSynclock sync.Mutex
	faultpeerlock    sync.Mutex
	bestpeerlock     sync.Mutex
	downLoadlock     sync.Mutex
	downLoadModeLock sync.Mutex
	isNtpClockSync   bool //ntp时间是否同步

	//cfg
	MaxFetchBlockNum int64 //一次最多申请获取block个数
	TimeoutSeconds   int64
	blockSynInterVal time.Duration

	failed int32

	blockOnChain   *BlockOnChain
	onChainTimeout int64

	//记录当前已经连续的最高高度
	maxSerialChunkNum     int64
	processingGenChunk    int32
	processingDeleteChunk int32
	deleteChunkCount      int64

	// TODO
	lastHeight             int64
	heightNotIncreaseTimes int32

	// 标识区块是否回滚
	neverRollback bool

	// 是否正在下载chunk
	chunkDownloading int32
	forkPointChan    chan int64
	finalizer        *finalizer
}

// New new
func New(cfg *types.Chain33Config) *BlockChain {
	mcfg := cfg.GetModuleConfig().BlockChain
	futureBlocks, err := lru.New(maxFutureBlocks)
	if err != nil {
		panic("when New BlockChain lru.New return err")
	}

	if atomic.LoadInt64(&mcfg.ChunkblockNum) == 0 {
		atomic.StoreInt64(&mcfg.ChunkblockNum, defaultChunkBlockNum)
	}

	blockchain := &BlockChain{
		rcvLastBlockHeight: -1,
		synBlockHeight:     -1,
		maxSerialChunkNum:  -1,
		peerList:           nil,
		cfg:                mcfg,
		recvwg:             &sync.WaitGroup{},
		tickerwg:           &sync.WaitGroup{},
		reducewg:           &sync.WaitGroup{},

		syncTask:        newTask(300 * time.Second), //考虑到区块交易多时执行耗时，需要延长task任务的超时时间
		downLoadTask:    newTask(180 * time.Second),
		chunkRecordTask: newTask(120 * time.Second),

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
		downloadMode:        fastDownLoadMode,
		blockOnChain:        &BlockOnChain{},
		onChainTimeout:      0,
		forkPointChan:       make(chan int64, 1),
	}
	blockchain.initConfig(cfg)
	blockchain.blockCache = newBlockCache(cfg, defaultBlockHashCacheSize)
	blockchain.neverRollback = cfg.GetModuleConfig().Consensus.NoneRollback
	return blockchain
}

func (chain *BlockChain) initConfig(cfg *types.Chain33Config) {
	mcfg := cfg.GetModuleConfig().BlockChain

	if mcfg.MaxFetchBlockNum > 0 {
		chain.MaxFetchBlockNum = mcfg.MaxFetchBlockNum
	}

	if mcfg.TimeoutSeconds > 0 {
		chain.TimeoutSeconds = mcfg.TimeoutSeconds
	}

	if mcfg.DefCacheSize <= 0 {
		mcfg.DefCacheSize = 128
	}

	chain.blockSynInterVal = time.Duration(chain.TimeoutSeconds)
	chain.isStrongConsistency = mcfg.IsStrongConsistency
	chain.isRecordBlockSequence = mcfg.IsRecordBlockSequence
	chain.enablePushSubscribe = mcfg.EnablePushSubscribe
	chain.isParaChain = mcfg.IsParaChain
	cfg.S("quickIndex", mcfg.EnableTxQuickIndex)
	cfg.S("reduceLocaldb", mcfg.EnableReduceLocaldb)

	if mcfg.OnChainTimeout > 0 {
		chain.onChainTimeout = mcfg.OnChainTimeout
	}
	chain.initOnChainTimeout()
	// 	初始化AllowPackHeight
	initAllowPackHeight(chain.cfg)
	if mcfg.BlockFinalizeEnableHeight < 0 {
		mcfg.BlockFinalizeEnableHeight = 0
	}
	if mcfg.BlockFinalizeGapHeight <= 0 {
		mcfg.BlockFinalizeGapHeight = defaultFinalizeGapHeight
	}
}

// Close 关闭区块链
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

	if chain.push != nil {
		chainlog.Info("blockchain wait for push quit")
		chain.push.Close()
	}

	//关闭数据库
	chain.blockStore.db.Close()
	chainlog.Info("blockchain module closed")
}

// SetQueueClient 设置队列
func (chain *BlockChain) SetQueueClient(client queue.Client) {
	chain.client = client
	chain.client.Sub("blockchain")

	blockStoreDB := dbm.NewDB("blockchain", chain.cfg.Driver, chain.cfg.DbPath, chain.cfg.DbCache)
	chain.blockStore = NewBlockStore(chain, blockStoreDB, client)
	stateHash := chain.getStateHash()
	chain.query = NewQuery(blockStoreDB, chain.client, stateHash)
	if chain.enablePushSubscribe && chain.isRecordBlockSequence {
		chain.push = newpush(chain.blockStore, chain.blockStore, chain.client)
		chainlog.Info("chain push is setup")
	}

	//startTime
	chain.startTime = types.Now()
	// 获取当前最大chunk连续高度
	chain.maxSerialChunkNum = chain.blockStore.GetMaxSerialChunkNum()

	chain.finalizer = &finalizer{}
	chain.finalizer.Init(chain)
	//recv 消息的处理，共识模块需要获取lastblock从数据库中
	chain.recvwg.Add(1)
	//初始化blockchian模块
	chain.InitBlockChain()
	go chain.ProcRecvMsg()
}

// Wait for ready
func (chain *BlockChain) Wait() {}

// GetStore only used for test
func (chain *BlockChain) GetStore() *BlockStore {
	return chain.blockStore
}

// GetOrphanPool 获取孤儿链
func (chain *BlockChain) GetOrphanPool() *OrphanPool {
	return chain.orphanPool
}

// InitBlockChain 区块链初始化
func (chain *BlockChain) InitBlockChain() {
	//isRecordBlockSequence配置的合法性检测
	seqStatus := chain.blockStore.CheckSequenceStatus(chain.isRecordBlockSequence)
	if seqStatus == seqStatusNeedCreate {
		chain.blockStore.CreateSequences(100000)
	}

	//先缓存最新的128个block信息到cache中
	curheight := chain.GetBlockHeight()
	chain.InitCache(curheight)

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

	if !chain.cfg.DisableShard {
		chain.tickerwg.Add(2)
		go chain.chunkDeleteRoutine()
		go chain.chunkGenerateRoutine()
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

// SendAddBlockEvent blockchain 模块add block到db之后通知mempool 和consense模块做相应的更新
func (chain *BlockChain) SendAddBlockEvent(block *types.BlockDetail) (err error) {
	if chain.client == nil {
		chainlog.Error("SendAddBlockEvent: chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}
	if block == nil {
		chainlog.Error("SendAddBlockEvent block is null")
		return types.ErrInvalidParam
	}
	height := block.GetBlock().GetHeight()
	chainlog.Debug("SendAddBlockEvent", "Height", height)

	// 首先向加密模块推送, 保证交易验签依赖最新的区块信息
	header := &types.Header{Height: height, BlockTime: block.GetBlock().GetBlockTime()}
	chain.sendAddBlockEvent("crypto", header, height)

	chain.sendAddBlockEvent("mempool", block, height)
	chain.sendAddBlockEvent("consensus", block, height)
	chain.sendAddBlockEvent("p2p", block.GetBlock(), height)
	chain.sendAddBlockEvent("wallet", block, height)

	return nil
}

func (chain *BlockChain) sendAddBlockEvent(topic string, data interface{}, height int64) {
	chainlog.Debug("SendAddBlockEvent", "topic", topic)
	msg := chain.client.NewMessage(topic, types.EventAddBlock, data)
	//此处采用同步发送模式，主要是为了消息在消息队列内部走高速通道，使得消息能尽快被处理
	if err := chain.client.Send(msg, true); err != nil {
		chainlog.Error("SendAddBlockEvent", "topic", topic, "height", height, "err", err)
	}
}

// SendBlockBroadcast blockchain模块广播此block到网络中
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
	// wait reply情况消息队列走高速模式
	err := chain.client.Send(msg, true)
	if err != nil {
		chainlog.Error("SendBlockBroadcast", "Height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)), "err", err)
	}
}

// GetBlockHeight 获取区块高度
func (chain *BlockChain) GetBlockHeight() int64 {
	return chain.blockStore.Height()
}

// GetBlock 用于获取指定高度的block，首先在缓存中获取，如果不存在就从db中获取
func (chain *BlockChain) GetBlock(height int64) (detail *types.BlockDetail, err error) {

	cfg := chain.client.GetConfig()

	var hash []byte
	if hash, err = chain.blockStore.GetBlockHashByHeight(height); err != nil {
		return nil, err
	}

	//从缓存的最新区块中尝试获取，最新区块的add是在执行block流程中处理
	if detail = chain.blockCache.GetBlockByHash(hash); detail != nil {
		if len(detail.Receipts) == 0 && len(detail.Block.Txs) != 0 {
			chainlog.Debug("GetBlockByHash  GetBlockByHeight Receipts ==0", "height", height)
		}
		return detail, nil
	}

	if detail, exist := chain.blockStore.GetActiveBlock(string(hash)); exist {
		return detail, nil
	}

	//从blockstore db中通过block height获取block
	detail, err = chain.blockStore.LoadBlock(height, hash)
	if err != nil {
		chainlog.Error("GetBlock", "height", height, "LoadBlock err", err)
		return nil, err
	}
	if len(detail.Receipts) == 0 && len(detail.Block.Txs) != 0 {
		chainlog.Debug("GetBlock  LoadBlock Receipts ==0", "height", height)
	}
	//缓存到活跃区块中
	chain.blockStore.AddActiveBlock(string(detail.Block.Hash(cfg)), detail)
	return detail, nil

}

// SendDelBlockEvent blockchain 模块 del block从db之后通知mempool 和consense以及wallet模块做相应的更新
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

// GetDB 获取DB
func (chain *BlockChain) GetDB() dbm.DB {
	return chain.blockStore.db
}

// InitCache 初始化缓存
func (chain *BlockChain) InitCache(currHeight int64) {

	if !chain.client.GetConfig().IsEnable("TxHeight") {
		chain.txHeightCache = &noneCache{}
		return
	}
	chain.txHeightCache = newTxHashCache(chain, types.HighAllowPackHeight, types.LowAllowPackHeight)
	// cache history block if exist
	if currHeight < 0 {
		return
	}

	for i := currHeight - chain.cfg.DefCacheSize; i <= currHeight; i++ {
		if i < 0 {
			i = 0
		}
		block, err := chain.GetBlock(i)
		if err != nil {
			panic(fmt.Sprintf("getBlock err=%s, height=%d", err.Error(), i))
		}
		chain.blockCache.AddBlock(block)
	}

	for i := currHeight - types.HighAllowPackHeight - types.LowAllowPackHeight + 1; i <= currHeight; i++ {
		if i < 0 {
			i = 0
		}
		block, err := chain.GetBlock(i)
		if err != nil {
			panic(err)
		}
		chain.txHeightCache.Add(block.GetBlock())
	}
}

// InitIndexAndBestView 第一次启动之后需要将数据库中最新的128个block的node添加到index和bestchain中
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
			chainlog.Error("InitIndexAndBestView GetBlockHeaderByHeight", "height", height, "err", err)
			//开始升级localdb到2.0.0版本时需要兼容旧的存储方式
			header, err = chain.blockStore.getBlockHeaderByHeightOld(height)
			if header == nil {
				chainlog.Error("InitIndexAndBestView getBlockHeaderByHeightOld", "height", height, "err", err)
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

// UpdateRoutine 定时延时广播futureblock
func (chain *BlockChain) UpdateRoutine() {
	//1秒尝试检测一次futureblock，futureblock的time小于当前系统时间就广播此block
	futureblockTicker := time.NewTicker(1 * time.Second)
	defer futureblockTicker.Stop()

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

// ProcFutureBlocks 循环遍历所有futureblocks，当futureblock的block生成time小于当前系统时间就将此block广播出去
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

// SetValueByKey 设置kv对到blockchain数据库
func (chain *BlockChain) SetValueByKey(kvs *types.LocalDBSet) error {
	return chain.blockStore.SetConsensusPara(kvs)
}

// GetValueByKey 通过key值从blockchain数据库中获取value值
func (chain *BlockChain) GetValueByKey(keys *types.LocalDBGet) *types.LocalReplyValue {
	return chain.blockStore.Get(keys)
}

// AddCacheBlock 添加区块相关缓存
func (chain *BlockChain) AddCacheBlock(detail *types.BlockDetail) {
	//txHeight缓存先增加
	chain.txHeightCache.Add(detail.Block)
	chain.blockCache.AddBlock(detail)
}

// DelCacheBlock 删除缓存的中对应的区块
func (chain *BlockChain) DelCacheBlock(height int64, hash []byte) {
	//txHeight缓存先删除
	chain.txHeightCache.Del(height)
	chain.blockCache.DelBlock(height)
	chain.blockStore.RemoveActiveBlock(string(hash))
}

// initAllowPackHeight 根据配置修改LowAllowPackHeight和值HighAllowPackHeight
func initAllowPackHeight(mcfg *types.BlockChain) {
	if mcfg.HighAllowPackHeight > 0 && mcfg.LowAllowPackHeight > 0 {
		if mcfg.HighAllowPackHeight+mcfg.LowAllowPackHeight > types.MaxAllowPackInterval {
			panic("when Enable TxHeight HighAllowPackHeight + LowAllowPackHeight must less than types.MaxAllowPackInterval")
		}
		types.HighAllowPackHeight = mcfg.HighAllowPackHeight
		types.LowAllowPackHeight = mcfg.LowAllowPackHeight
	}
	chainlog.Debug("initAllowPackHeight", "types.HighAllowPackHeight", types.HighAllowPackHeight, "types.LowAllowPackHeight", types.LowAllowPackHeight)
}
