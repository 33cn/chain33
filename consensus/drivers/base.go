package drivers

import (
	"errors"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var tlog = log.New("module", "consensus")

var (
	zeroHash [32]byte
)

var randgen *rand.Rand

func init() {
	randgen = rand.New(rand.NewSource(time.Now().UnixNano()))
}

type Miner interface {
	CreateGenesisTx() []*types.Transaction
	CreateBlock()
	CheckBlock(parent *types.Block, current *types.BlockDetail) error
	ProcEvent(msg queue.Message)
	ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error)
}

type BaseClient struct {
	client       queue.Client
	minerStart   int32
	once         sync.Once
	Cfg          *types.Consensus
	currentBlock *types.Block
	mulock       sync.Mutex
	child        Miner
	minerstartCB func()
	isCaughtUp   int32
}

func NewBaseClient(cfg *types.Consensus) *BaseClient {
	cfg.Genesis = types.GenesisAddr
	cfg.HotkeyAddr = types.HotkeyAddr
	cfg.GenesisBlockTime = types.GenesisBlockTime
	var flag int32
	if cfg.Minerstart {
		flag = 1
	}
	client := &BaseClient{minerStart: flag, isCaughtUp: 0}
	client.Cfg = cfg
	log.Info("Enter consensus " + cfg.GetName())
	return client
}

func (bc *BaseClient) SetChild(c Miner) {
	bc.child = c
}

func (bc *BaseClient) InitClient(c queue.Client, minerstartCB func()) {
	log.Info("Enter SetQueueClient method of consensus")
	bc.client = c
	bc.minerstartCB = minerstartCB
	bc.InitMiner()
}

func (bc *BaseClient) GetQueueClient() queue.Client {
	return bc.client
}

func (bc *BaseClient) RandInt64() int64 {
	return randgen.Int63()
}

func (bc *BaseClient) InitMiner() {
	bc.once.Do(bc.minerstartCB)
}

func (bc *BaseClient) SetQueueClient(c queue.Client) {
	bc.InitClient(c, func() {
		//call init block
		bc.InitBlock()
	})
	go bc.EventLoop()
	go bc.child.CreateBlock()
}

//change init block
func (bc *BaseClient) InitBlock() {
	block, err := bc.RequestLastBlock()
	if err != nil {
		panic(err)
	}
	if block == nil {
		// 创世区块
		newblock := &types.Block{}
		newblock.Height = 0
		newblock.BlockTime = bc.Cfg.GenesisBlockTime
		// TODO: 下面这些值在创世区块中赋值nil，是否合理？
		newblock.ParentHash = zeroHash[:]
		tx := bc.child.CreateGenesisTx()
		newblock.Txs = tx
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		bc.WriteBlock(zeroHash[:], newblock)
	} else {
		bc.SetCurrentBlock(block)
	}
}

func (bc *BaseClient) Close() {
	atomic.StoreInt32(&bc.minerStart, 0)
	bc.client.Close()
	log.Info("consensus base closed")
}

func (bc *BaseClient) CheckTxDup(txs []*types.Transaction) (transactions []*types.Transaction) {
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
	hashList := bc.client.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
	bc.client.Send(hashList, true)
	dupTxList, _ := bc.client.Wait(hashList)
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

func (bc *BaseClient) IsMining() bool {
	return atomic.LoadInt32(&bc.minerStart) == 1
}

func (bc *BaseClient) IsCaughtUp() bool {
	if bc.client == nil {
		panic("bc not bind message queue.")
	}
	msg := bc.client.NewMessage("blockchain", types.EventIsSync, nil)
	bc.client.Send(msg, true)
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return false
	}
	return resp.GetData().(*types.IsCaughtUp).GetIscaughtup()
}

// 准备新区块
func (bc *BaseClient) EventLoop() {
	// 监听blockchain模块，获取当前最高区块
	bc.client.Sub("consensus")
	go func() {
		for msg := range bc.client.Recv() {
			tlog.Debug("consensus recv", "msg", msg)
			if msg.Ty == types.EventAddBlock {
				block := msg.GetData().(*types.BlockDetail).Block
				bc.SetCurrentBlock(block)
			} else if msg.Ty == types.EventCheckBlock {
				block := msg.GetData().(*types.BlockDetail)
				err := bc.CheckBlock(block)
				msg.ReplyErr("EventCheckBlock", err)
			} else if msg.Ty == types.EventMinerStart {
				if !atomic.CompareAndSwapInt32(&bc.minerStart, 0, 1) {
					msg.ReplyErr("EventMinerStart", types.ErrMinerIsStared)
				} else {
					bc.InitMiner()
					msg.ReplyErr("EventMinerStart", nil)
				}
			} else if msg.Ty == types.EventMinerStop {
				if !atomic.CompareAndSwapInt32(&bc.minerStart, 1, 0) {
					msg.ReplyErr("EventMinerStop", types.ErrMinerNotStared)
				} else {
					msg.ReplyErr("EventMinerStop", nil)
				}
			} else if msg.Ty == types.EventDelBlock {
				block := msg.GetData().(*types.BlockDetail).Block
				bc.UpdateCurrentBlock(block)
			} else {
				bc.child.ProcEvent(msg)
			}
		}
	}()
}

func (bc *BaseClient) CheckBlock(block *types.BlockDetail) error {
	//check parent
	if block.Block.Height <= 0 { //genesis block not check
		return nil
	}
	parent, err := bc.RequestBlock(block.Block.Height - 1)
	if err != nil {
		return err
	}
	//check base info
	if parent.Height+1 != block.Block.Height {
		return types.ErrBlockHeight
	}
	//check parent hash
	if string(block.Block.GetParentHash()) != string(parent.Hash()) {
		return types.ErrParentHash
	}
	//check by drivers
	err = bc.child.CheckBlock(parent, block)
	return err
}

// Mempool中取交易列表
func (bc *BaseClient) RequestTx(listSize int, txHashList [][]byte) []*types.Transaction {
	if bc.client == nil {
		panic("bc not bind message queue.")
	}
	msg := bc.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{txHashList, int64(listSize)})
	bc.client.Send(msg, true)
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return nil
	}
	return resp.GetData().(*types.ReplyTxList).GetTxs()
}

func (bc *BaseClient) RequestBlock(start int64) (*types.Block, error) {
	if bc.client == nil {
		panic("bc not bind message queue.")
	}
	msg := bc.client.NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, start, false, []string{""}})
	bc.client.Send(msg, true)
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	blocks := resp.GetData().(*types.BlockDetails)
	return blocks.Items[0].Block, nil
}

//获取最新的block从blockchain模块
func (bc *BaseClient) RequestLastBlock() (*types.Block, error) {
	if bc.client == nil {
		panic("client not bind message queue.")
	}
	msg := bc.client.NewMessage("blockchain", types.EventGetLastBlock, nil)
	bc.client.Send(msg, true)
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	block := resp.GetData().(*types.Block)
	return block, nil
}

//del mempool
func (bc *BaseClient) delMempoolTx(deltx []*types.Transaction) error {
	hashList := buildHashList(deltx)
	msg := bc.client.NewMessage("mempool", types.EventDelTxList, hashList)
	bc.client.Send(msg, true)
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return err
	}
	if resp.GetData().(*types.Reply).GetIsOk() {
		return nil
	}
	return errors.New(string(resp.GetData().(*types.Reply).GetMsg()))
}

func buildHashList(deltx []*types.Transaction) *types.TxHashList {
	list := &types.TxHashList{}
	for i := 0; i < len(deltx); i++ {
		list.Hashes = append(list.Hashes, deltx[i].Hash())
	}
	return list
}

// 向blockchain写区块
func (bc *BaseClient) WriteBlock(prev []byte, block *types.Block) error {
	blockdetail, deltx, err := bc.child.ExecBlock(prev, block)
	if len(deltx) > 0 {
		bc.delMempoolTx(deltx)
	}
	if err != nil {
		return err
	}
	msg := bc.client.NewMessage("blockchain", types.EventAddBlockDetail, blockdetail)
	bc.client.Send(msg, true)
	resp, err := bc.client.Wait(msg)
	if err != nil {
		return err
	}
	//从mempool 中删除错误的交易

	if resp.GetData().(*types.Reply).IsOk {
		bc.SetCurrentBlock(block)
	} else {
		reply := resp.GetData().(*types.Reply)
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func (bc *BaseClient) SetCurrentBlock(b *types.Block) {
	bc.mulock.Lock()
	bc.currentBlock = b
	bc.mulock.Unlock()
}

func (bc *BaseClient) UpdateCurrentBlock(b *types.Block) {
	bc.mulock.Lock()
	defer bc.mulock.Unlock()
	block, err := bc.RequestLastBlock()
	if err != nil {
		log.Error("UpdateCurrentBlock", "RequestLastBlock", err)
		return
	}
	bc.currentBlock = block
}

func (bc *BaseClient) GetCurrentBlock() (b *types.Block) {
	bc.mulock.Lock()
	defer bc.mulock.Unlock()
	return bc.currentBlock
}

func (bc *BaseClient) GetCurrentHeight() int64 {
	bc.mulock.Lock()
	start := bc.currentBlock.Height
	bc.mulock.Unlock()
	return start
}

func (bc *BaseClient) Lock() {
	bc.mulock.Lock()
}

func (bc *BaseClient) Unlock() {
	bc.mulock.Unlock()
}

func (bc *BaseClient) ConsensusTicketMiner(iscaughtup *types.IsCaughtUp) {
	if !atomic.CompareAndSwapInt32(&bc.isCaughtUp, 0, 1) {
		log.Info("ConsensusTicketMiner", "isCaughtUp", bc.isCaughtUp)
	} else {
		log.Info("ConsensusTicketMiner", "isCaughtUp", bc.isCaughtUp)
	}
}
