package drivers

import (
	"errors"
	"sync"
	"sync/atomic"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/util"
	log "github.com/inconshreveable/log15"
)

var tlog = log.New("module", "consensus")

var (
	listSize int = 10000
	zeroHash [32]byte
)

type Miner interface {
	CreateGenesisTx() []*types.Transaction
	CreateBlock()
	CheckBlock(parent *types.Block, current *types.BlockDetail) error
}

type BaseClient struct {
	qclient      queue.IClient
	q            *queue.Queue
	minerStart   int32
	once         sync.Once
	Cfg          *types.Consensus
	currentBlock *types.Block
	mulock       sync.Mutex
	child        Miner
	minerstartCB func()
}

func NewBaseClient(cfg *types.Consensus) *BaseClient {
	var flag int32
	if cfg.Minerstart {
		flag = 1
	}
	client := &BaseClient{minerStart: flag}
	client.Cfg = cfg
	log.Info("Enter consensus solo")
	return client
}

func (client *BaseClient) SetChild(c Miner) {
	client.child = c
}

func (client *BaseClient) InitClient(q *queue.Queue, minerstartCB func()) {
	log.Info("Enter SetQueue method of consensus")
	client.qclient = q.GetClient()
	client.q = q
	client.minerstartCB = minerstartCB
}

func (client *BaseClient) InitMiner() {
	client.once.Do(client.minerstartCB)
}

func (client *BaseClient) SetQueue(q *queue.Queue) {
	client.InitClient(q, func() {
		client.InitBlock()
	})
	go client.EventLoop()
	go client.child.CreateBlock()
}

func (client *BaseClient) InitBlock() {
	height := client.GetInitHeight()
	if height == -1 {
		// 创世区块
		newblock := &types.Block{}
		newblock.Height = 0
		newblock.BlockTime = client.Cfg.GenesisBlockTime
		// TODO: 下面这些值在创世区块中赋值nil，是否合理？
		newblock.ParentHash = zeroHash[:]
		tx := client.child.CreateGenesisTx()
		newblock.Txs = tx
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		client.WriteBlock(zeroHash[:], newblock)
	} else {
		block, err := client.RequestBlock(height)
		if err != nil {
			panic(err)
		}
		client.SetCurrentBlock(block)
	}
}

func (client *BaseClient) Close() {
	log.Info("consensus solo closed")
}

func (client *BaseClient) CheckTxDup(txs []*types.Transaction) (transactions []*types.Transaction) {
	var checkHashList types.TxHashList
	txMap := make(map[string]*types.Transaction)
	for _, tx := range txs {
		hash := tx.Hash()
		txMap[string(hash)] = tx
		checkHashList.Hashes = append(checkHashList.Hashes, hash)
	}
	// 发送Hash过后的交易列表给blockchain模块
	//beg := time.Now()
	//log.Error("----EventTxHashList----->[beg]", "time", beg)
	hashList := client.qclient.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
	client.qclient.Send(hashList, true)
	dupTxList, _ := client.qclient.Wait(hashList)
	//log.Error("----EventTxHashList----->[end]", "time", time.Now().Sub(beg))
	// 取出blockchain返回的重复交易列表
	dupTxs := dupTxList.GetData().(*types.TxHashList).Hashes

	for _, hash := range dupTxs {
		delete(txMap, string(hash))
	}

	for _, tx := range txMap {
		transactions = append(transactions, tx)
	}
	return transactions
}

func (client *BaseClient) IsMining() bool {
	return atomic.LoadInt32(&client.minerStart) == 1
}

// 准备新区块
func (client *BaseClient) EventLoop() {
	// 监听blockchain模块，获取当前最高区块
	client.qclient.Sub("consensus")
	go func() {
		for msg := range client.qclient.Recv() {
			tlog.Info("consensus recv", "msg", msg)
			if msg.Ty == types.EventAddBlock {
				block := msg.GetData().(*types.BlockDetail).Block
				client.SetCurrentBlock(block)
			} else if msg.Ty == types.EventCheckBlock {
				block := msg.GetData().(*types.BlockDetail)
				err := client.CheckBlock(block)
				msg.ReplyErr("EventCheckBlock", err)
			} else if msg.Ty == types.EventMinerStart {
				if !atomic.CompareAndSwapInt32(&client.minerStart, 0, 1) {
					msg.ReplyErr("EventMinerStart", types.ErrMinerIsStared)
				} else {
					client.InitMiner()
					msg.ReplyErr("EventMinerStart", nil)
				}
			} else if msg.Ty == types.EventMinerStop {
				if !atomic.CompareAndSwapInt32(&client.minerStart, 1, 0) {
					msg.ReplyErr("EventMinerStop", types.ErrMinerNotStared)
				} else {
					msg.ReplyErr("EventMinerStop", nil)
				}
			}
		}
	}()
}

func (client *BaseClient) CheckBlock(block *types.BlockDetail) error {
	//check parent
	if block.Block.Height == 0 { //genesis block not check
		return nil
	}
	log.Warn("query parent beg")
	parent, err := client.RequestBlock(block.Block.Height - 1)
	if err != nil {
		return err
	}
	log.Warn("query parent end")
	//check base info
	if parent.Height+1 != block.Block.Height {
		return types.ErrBlockHeight
	}
	//check by drivers
	err = client.child.CheckBlock(parent, block)
	return err
}

// Mempool中取交易列表
func (client *BaseClient) RequestTx() []*types.Transaction {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	//debug.PrintStack()
	//tlog.Error("requestTx", "time", time.Now().Format(time.RFC3339Nano))
	msg := client.qclient.NewMessage("mempool", types.EventTxList, listSize)
	client.qclient.Send(msg, true)
	resp, _ := client.qclient.Wait(msg)
	return resp.GetData().(*types.ReplyTxList).GetTxs()
}

func (client *BaseClient) RequestBlock(start int64) (*types.Block, error) {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, start, false})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	blocks := resp.GetData().(*types.BlockDetails)
	return blocks.Items[0].Block, nil
}

// solo初始化时，取一次区块高度放在内存中，后面自增长，不用再重复去blockchain取
func (client *BaseClient) GetInitHeight() int64 {

	msg := client.qclient.NewMessage("blockchain", types.EventGetBlockHeight, nil)

	client.qclient.Send(msg, true)
	replyHeight, err := client.qclient.Wait(msg)
	h := replyHeight.GetData().(*types.ReplyBlockHeight).Height
	tlog.Info("init = ", "height", h)
	if err != nil {
		panic("error happens when get height from blockchain")
	}
	return h
}

// 向blockchain写区块
func (client *BaseClient) WriteBlock(prevHash []byte, block *types.Block) error {
	blockdetail, err := util.ExecBlock(client.q, prevHash, block, false)
	if err != nil { //never happen
		panic(err)
	}
	if len(blockdetail.Block.Txs) == 0 {
		return errors.New("ErrNoTxs")
	}
	msg := client.qclient.NewMessage("blockchain", types.EventAddBlockDetail, blockdetail)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return err
	}
	if resp.GetData().(*types.Reply).IsOk {
		client.SetCurrentBlock(block)
	} else {
		//TODO:
		//把txs写回mempool
		reply := resp.GetData().(*types.Reply)
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func (client *BaseClient) SetCurrentBlock(b *types.Block) {
	client.mulock.Lock()
	if client.currentBlock == nil || client.currentBlock.Height <= b.Height {
		client.currentBlock = b
	}
	client.mulock.Unlock()
}

func (client *BaseClient) GetCurrentBlock() (b *types.Block) {
	client.mulock.Lock()
	defer client.mulock.Unlock()
	return client.currentBlock
}

func (client *BaseClient) GetCurrentHeight() int64 {
	client.mulock.Lock()
	start := client.currentBlock.Height
	client.mulock.Unlock()
	return start
}

func (client *BaseClient) Lock() {
	client.mulock.Lock()
}

func (client *BaseClient) Unlock() {
	client.mulock.Unlock()
}
