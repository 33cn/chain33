package raft

import (
	"fmt"
	"sync"
	"time"

	"github.com/coreos/etcd/snap"
	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

var (
	zeroHash [32]byte
)

type RaftClient struct {
	*drivers.BaseClient
	proposeC    chan<- *types.Block
	commitC     <-chan *types.Block
	errorC      <-chan error
	snapshotter *snap.Snapshotter
	validatorC  <-chan bool
	once        sync.Once
}

func NewBlockstore(cfg *types.Consensus, snapshotter *snap.Snapshotter, proposeC chan<- *types.Block, commitC <-chan *types.Block, errorC <-chan error, validatorC <-chan bool) *RaftClient {
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
	blockdetail, deltx, err := util.ExecBlock(client.GetQueueClient(), prevHash, block, false, true)
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
	rlog.Info("Enter SetQueue method of raft consensus")
	client.InitClient(c, func() {
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
		newblock.ParentHash = zeroHash[:]
		tx := client.CreateGenesisTx()
		newblock.Txs = tx
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		err := client.WriteBlock(zeroHash[:], newblock)
		if err != nil {
			rlog.Error(fmt.Sprintf("chain33 init block failed!", err.Error()))
			return
		}
		client.SetCurrentBlock(newblock)
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
	var lastBlock *types.Block
	var newblock types.Block
	var txs []*types.Transaction
	for {
		//如果leader节点突然挂了，不是打包节点，需要退出
		if !isLeader {
			rlog.Warn("I'm not the validator node anymore,exit.=============================")
			break
		}
		rlog.Info("==================This is Leader node=====================")
		if issleep {
			time.Sleep(10 * time.Second)
		}
		lastBlock = client.GetCurrentBlock()
		txs = client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber), nil)
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		rlog.Debug("==================start create new block!=====================")
		//check dup
		//txs = client.CheckTxDup(txs)
		rlog.Debug(fmt.Sprintf("the len txs is: %v", len(txs)))
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		newblock.Txs = txs
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = time.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		client.propose(&newblock)
		time.Sleep(100 * time.Millisecond)

		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		if err != nil {
			issleep = true
			rlog.Error(fmt.Sprintf("********************err:%v", err.Error()))
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
	var data *types.Block
	var ok bool
	for {
		select {
		case data, ok = <-commitC:
			if !ok || data == nil {
				continue
			}
			rlog.Debug("===============Get block from commit channel===========")
			// 在程序刚开始启动的时候有可能存在丢失数据的问题
			//区块高度统一由base中的相关代码进行变更，防止错误区块出现
			//client.SetCurrentBlock(data)

		case err, ok := <-errorC:
			if ok {
				panic(err)
			}

		}
	}
}

//轮询任务，去检测本机器是否为validator节点，如果是，则执行打包任务
func (client *RaftClient) pollingTask(c queue.Client) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case value, ok := <-client.validatorC:
			//各个节点Block只初始化一次
			client.once.Do(func() {
				client.InitBlock()
			})
			if ok && !value {
				rlog.Debug("================I'm not the validator node!=============")
				isLeader = false
			} else if ok && !isLeader && value {
				client.once.Do(
					func() {
						client.InitMiner()
					})
				isLeader = true
				go client.CreateBlock()
			} else if !ok {
				break
			}
		case <-ticker.C:
			rlog.Debug("Gets the leader node information timeout and triggers the ticker.")
		}
	}
}
