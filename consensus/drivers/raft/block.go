package raft

import (
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/util"
	"github.com/coreos/etcd/snap"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"

	//"sync/atomic"
	//"golang.org/x/tools/go/gcimporter15/testdata"
	"fmt"
)

var (
	listSize     int = 10000
	zeroHash     [32]byte
	currentBlock *types.Block
	mulock       sync.Mutex
)

//type Miner interface {
//	CreateGenesisTx() []*types.Transaction
//	CreateBlock()
//}
type RaftClient struct {
	//TODO: BaseClient中有些参数访问不了，所以暂时不用baseClient
	*drivers.BaseClient
	proposeC    chan<- *types.Block
	commitC     <-chan *types.Block
	errorC      <-chan error
	snapshotter *snap.Snapshotter
	validatorC  <-chan map[string]bool
	once        sync.Once
}

func NewBlockstore(cfg *types.Consensus, snapshotter *snap.Snapshotter, proposeC chan<- *types.Block, commitC <-chan *types.Block, errorC <-chan error, validatorC <-chan map[string]bool) *RaftClient {
	c := drivers.NewBaseClient(cfg)
	client := &RaftClient{BaseClient: c, proposeC: proposeC, snapshotter: snapshotter, validatorC: validatorC, commitC: commitC, errorC: errorC}
	c.SetChild(client)
	return client
}
func (client *RaftClient) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = client.Cfg.Genesis
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

func (client *RaftClient) ProcEvent(msg queue.Message) {

}

func (client *RaftClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error) {
	//exec block
	if block.Height == 0 {
		block.Difficulty = types.GetP(0).PowLimitBits
	}
	blockdetail, deltx, err := util.ExecBlock(client.GetQueueClient(), prevHash, block, false)
	if err != nil { //never happen
		return nil, deltx, err
	}
	if len(blockdetail.Block.Txs) == 0 {
		return nil, deltx, types.ErrNoTx
	}
	return blockdetail, deltx, nil
}
func (client *RaftClient) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	return nil
}

func (client *RaftClient) getSnapshot() ([]byte, error) {
	//这里可能导致死锁
	return proto.Marshal(client.GetCurrentBlock())
}

func (client *RaftClient) recoverFromSnapshot(snapshot []byte) error {
	var block types.Block
	if err := proto.Unmarshal(snapshot, &block); err != nil {
		return err
	}
	client.SetCurrentBlock(&block)
	return nil
}

func (client *RaftClient) SetQueueClient(c queue.Client) {
	log.Info("Enter SetQueue method of raft consensus")
	client.InitClient(c, func() {
		//初始化应该等到leader选举成功之后在pollingTask中执行
		//client.InitBlock()
	})
	go client.EventLoop()
	go client.readCommits(client.commitC, client.errorC)
	go client.pollingTask(c)
}

func (client *RaftClient) Close() {
	rlog.Info("consensus raft closed")
}

func (client *RaftClient) InitBlock() {
	block, err := client.RequestLastBlock()
	if err != nil {
		panic(err)
	}
	if block == nil {
		// 创世区块
		newblock := &types.Block{}
		newblock.Height = 0
		newblock.BlockTime = client.Cfg.GenesisBlockTime
		// TODO: 下面这些值在创世区块中赋值nil，是否合理？
		newblock.ParentHash = zeroHash[:]
		tx := client.CreateGenesisTx()
		newblock.Txs = tx
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		// 初始块不需要参与共识
		//client.propose(newblock)
		// 把区块放在内存中
		//TODO:这里要等确认后才能把当前的块设置为新块
		client.SetCurrentBlock(newblock)
		err := client.WriteBlock(zeroHash[:], newblock)
		if err != nil {
			log.Error("chain33 init block failed!", err)
		}
	} else {
		block, err := client.RequestBlock(block.GetHeight())
		if err != nil {
			panic(err)
		}
		client.SetCurrentBlock(block)
	}
}

func (client *RaftClient) CreateBlock() {
	issleep := true
	for {
		//如果leader节点突然挂了，不是打包节点，需要退出
		if !isLeader {
			log.Warn("I'm not the validator node anymore,exit.=============================")
			break
		}
		rlog.Info("==================This is Leader node=====================")
		if issleep {
			time.Sleep(10 * time.Second)
		}
		lastBlock := client.GetCurrentBlock()
		txs := client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber), nil)
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		rlog.Info("==================start create new block!=====================")
		//check dup
		txs = client.CheckTxDup(txs)
		fmt.Println(len(txs))
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		newblock.Txs = txs
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = time.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		client.propose(&newblock)
		time.Sleep(time.Second)

		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		if err != nil {
			issleep = true
			log.Error("********************err:", err)
			continue
		}

	}
}

// 向raft底层发送block
func (client *RaftClient) propose(block *types.Block) {
	client.proposeC <- block
}

// 从receive channel中读leader发来的block
func (client *RaftClient) readCommits(commitC <-chan *types.Block, errorC <-chan error) {
	//var prevHash []byte
	for {
		select {
		case data := <-commitC:
			if data == nil {
				//此处不需要从snapshot中加载，当前内存中block信息可从blockchain模块获取
				//snapshot, err := client.snapshotter.Load()
				//if err == snap.ErrNoSnapshot {
				//	return
				//}
				//
				//log.Info("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
				//if err := client.recoverFromSnapshot(snapshot.Data); err != nil {
				//	panic(err)
				//}
				continue
			}
			rlog.Info("===============Get block from commit channel===========")
			// 在程序刚开始启动的时候有可能存在丢失数据的问题
			client.SetCurrentBlock(data)

		case err, ok := <-errorC:
			if ok {
				panic(err)
			}

		}
	}
}

//轮询任务，去检测本机器是否为validator节点，如果是，则执行打包任务
func (client *RaftClient) pollingTask(c queue.Client) {
	for {
		select {
		case validator := <-client.validatorC:
			if value, ok := validator[LeaderIsOK]; ok && value {
				//各个节点Block只初始化一次
				client.once.Do(func() {
					client.InitBlock()
				})
				if value, ok := validator[IsLeader]; ok && !value {
					rlog.Warn("================I'm not the validator node!=============")
					isLeader = false
				} else if !isLeader && value {
					client.once.Do(
						func() {
							client.InitMiner()
						})
					//TODO：当raft集群中的leader节点突然发生故障，此时另外的节点已经选举出新的leader，
					// 老的leader中运行的打包程此刻应该被终止？
					isLeader = true
					//go client.EventLoop()
					go client.CreateBlock()
				}
			}
		}
	}
}
