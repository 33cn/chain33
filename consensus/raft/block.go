package raft

import (
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/util"
	"github.com/coreos/etcd/snap"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	//"github.com/robfig/cron"
)

var (
	listSize     int = 10000
	zeroHash     [32]byte
	currentBlock *types.Block
	mulock       sync.Mutex
)

type RaftClient struct {
	proposeC    chan<- *types.Block
	mu          sync.RWMutex
	blockstore  *types.Block
	qclient     queue.IClient
	q           *queue.Queue
	snapshotter *snap.Snapshotter
	validatorC   chan bool
}

func NewBlockstore(snapshotter *snap.Snapshotter, proposeC chan<- *types.Block, commitC <-chan *types.Block, errorC <-chan error, leaderC chan int,validatorC chan bool) *RaftClient {
	b := &RaftClient{proposeC: proposeC, snapshotter: snapshotter,validatorC: validatorC}
	go b.readCommits(commitC, errorC, leaderC)
	return b
}

func (client *RaftClient) getSnapshot() ([]byte, error) {
	client.mu.Lock()
	defer client.mu.Unlock()
	return proto.Marshal(client.blockstore)
}

func (client *RaftClient) SetQueue(q *queue.Queue) {
	log.Info("Enter SetQueue method of consensus")
	// 只有主节点打包区块，其余节点接受p2p广播过来的区块
	//TODO:这里应该要改成轮询执行，当本台节点为Validator节点时则执行
	client.qclient = q.GetClient()
	client.q = q
	go client.pollingTask(q)
	//client.qclient = q.GetClient()
	//client.q = q
	//
	//// 程序初始化时，先从blockchain取区块链高度
	//height := client.getInitHeight()
	//
	//if height == -1 {
	//	// 创世区块
	//	newblock := &types.Block{}
	//	newblock.Height = 0
	//	newblock.ParentHash = zeroHash[:]
	//	tx := createGenesisTx()
	//	newblock.Txs = append(newblock.Txs, tx)
	//	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	//	// 通过propose channel把block传到raft核心
	//	client.propose(newblock)
	//	// 把区块放在内存中
	//	setCurrentBlock(newblock)
	//	//client.writeBlock(zeroHash[:], newblock)
	//} else {
	//	block := client.RequestBlock(height)
	//	setCurrentBlock(block)
	//}
	//
	//go client.eventLoop()
	//go client.createBlock()
}

func (client *RaftClient) Close() {
	rlog.Info("consensus raft closed")
}

func (client *RaftClient) checkTxDup(txs []*types.Transaction) (transactions []*types.Transaction) {
	var checkHashList types.TxHashList
	txMap := make(map[string]*types.Transaction)
	for _, tx := range txs {
		hash := tx.Hash()
		txMap[string(hash)] = tx
		checkHashList.Hashes = append(checkHashList.Hashes, hash)
	}

	// 发送Hash过后的交易列表给blockchain模块
	hashList := client.qclient.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
	client.qclient.Send(hashList, true)
	dupTxList, _ := client.qclient.Wait(hashList)

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

func (client *RaftClient) createBlock() {
	issleep := true
	for {
		//如果leader节点突然挂了，不是打包节点，需要退出
		if !isValidator {
			log.Warn("I'm not the validator node anymore,exit.=============================")
			break
		}
		log.Info("==================start create new block!=====================")
		if issleep {
			time.Sleep(time.Second)
		}
		txs := client.RequestTx()
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		//check dup
		txs = client.checkTxDup(txs)
		lastBlock := getCurrentBlock()
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		newblock.Txs = txs
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		rlog.Info("Send block to raft core")
		client.propose(&newblock)
		//client.writeBlock(lastBlock.StateHash, &newblock)
	}
}

// 准备新区块
func (client *RaftClient) eventLoop() {
	// 监听blockchain模块，获取当前最高区块
	client.qclient.Sub("consensus")
	go func() {
		for msg := range client.qclient.Recv() {
            //当本节点不再时leader节点时，需要退出。
			if !isValidator {
				log.Warn("I'm not the validator node anymore,exit.=============================")
				break
			}
			rlog.Info("consensus recv", "msg", msg)
			if msg.Ty == types.EventAddBlock {
				block := msg.GetData().(*types.BlockDetail).Block
				setCurrentBlock(block)
			}
		}
	}()
}

// Mempool中取交易列表
func (client *RaftClient) RequestTx() []*types.Transaction {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTxList, listSize)
	client.qclient.Send(msg, true)
	resp, _ := client.qclient.Wait(msg)
	return resp.GetData().(*types.ReplyTxList).GetTxs()
}

func (client *RaftClient) RequestBlock(start int64) *types.Block {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, start, false})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		panic(err)
	}
	blocks := resp.GetData().(*types.BlockDetails)
	return blocks.Items[0].Block
}

func (client *RaftClient) getInitHeight() int64 {

	msg := client.qclient.NewMessage("blockchain", types.EventGetBlockHeight, nil)

	client.qclient.Send(msg, true)
	replyHeight, err := client.qclient.Wait(msg)
	h := replyHeight.GetData().(*types.ReplyBlockHeight).Height
	rlog.Info("init = ", "height", h)
	if err != nil {
		panic("error happens when get height from blockchain")
	}
	return h
}

// 向blockchain写区块
func (client *RaftClient) writeBlock(prevHash []byte, block *types.Block) {

	blockdetail, err := util.ExecBlock(client.q, prevHash, block, false)
	if err != nil { //never happen
		panic(err)
	}
	for {
		msg := client.qclient.NewMessage("blockchain", types.EventAddBlockDetail, blockdetail)
		client.qclient.Send(msg, true)
		resp, _ := client.qclient.Wait(msg)

		if resp.GetData().(*types.Reply).IsOk {
			setCurrentBlock(block)
			break
		} else {
			log.Info("Send block to blockchian return fail, retry!")
		}
	}
}

func setCurrentBlock(b *types.Block) {
	mulock.Lock()
	if currentBlock == nil || currentBlock.Height <= b.Height {
		currentBlock = b
	}
	mulock.Unlock()
}

func getCurrentBlock() (b *types.Block) {
	mulock.Lock()
	defer mulock.Unlock()
	return currentBlock
}

func getCurrentHeight() int64 {
	mulock.Lock()
	start := currentBlock.Height
	mulock.Unlock()
	return start
}

func createGenesisTx() *types.Transaction {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = genesisAddr
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{1e8 * types.Coin}
	tx.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	return &tx
}

// 向raft底层发送block
func (client *RaftClient) propose(block *types.Block) {
	client.proposeC <- block
}

// 从receive channel中读leader发来的block
func (b *RaftClient) readCommits(commitC <-chan *types.Block, errorC <-chan error, leaderC chan int) {
	var prevHash []byte
	for data := range commitC {
		rlog.Info("Get block from commit channel")
		if data == nil {
			rlog.Info("data is nil")
			//			snapshot, err := b.snapshotter.Load()
			//			if err == snap.ErrNoSnapshot {
			//				return
			//			}

			//			log.Info("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
			//			if err := b.recoverFromSnapshot(snapshot.Data); err != nil {
			//				panic(err)
			//			}
			continue
		}
		if !isValidator{
			log.Warn("I'm not the validator node, I don't need to writeBlock!==========================")
			return
		}
		log.Warn("I'm the validator node, I need to writeBlock!==========================")
		lastBlock := getCurrentBlock()
		if lastBlock == nil {
			prevHash = zeroHash[:]
		} else {
			prevHash = lastBlock.StateHash
		}
		b.mu.Lock()
		b.writeBlock(prevHash, data)
		b.mu.Unlock()
	}

	if err, ok := <-errorC; ok {
		panic(err)
	}
}

//轮询任务，去检测本机器是否为validator节点，如果是，则执行打包任务
func (client *RaftClient) pollingTask(q *queue.Queue) {

	for {
		select {
		case value := <-client.validatorC:
			if !value{
				log.Warn("I'm not the validator node!=====")
				isValidator = false
			}else if !isValidator&&value{
				log.Info("==================start init block========================")
				height := client.getInitHeight()

				if height == -1 {
					// 创世区块
					newblock := &types.Block{}
					newblock.Height = 0
					newblock.ParentHash = zeroHash[:]
					tx := createGenesisTx()
					newblock.Txs = append(newblock.Txs, tx)
					newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
					// 通过propose channel把block传到raft核心
					client.propose(newblock)
					// 把区块放在内存中
					setCurrentBlock(newblock)
					//client.writeBlock(zeroHash[:], newblock)
				} else {
					block := client.RequestBlock(height)
					setCurrentBlock(block)
				}
				//TODO：当raft集群中的leader节点突然发生故障，此时另外的节点已经选举出新的leader，
				// 老的leader中运行的打包程此刻应该被终止？
				isValidator=true
				go client.eventLoop()
				go client.createBlock()

			}

		}
	}
}
