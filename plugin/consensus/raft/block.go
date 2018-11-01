package raft

import (
	"fmt"
	"sync"
	"time"

	"github.com/coreos/etcd/snap"
	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/consensus"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	zeroHash [32]byte
)

func init() {
	drivers.Reg("raft", NewRaftCluster)
	drivers.QueryData.Register("raft", &RaftClient{})
}

type RaftClient struct {
	*drivers.BaseClient
	proposeC    chan<- *types.Block
	commitC     <-chan *types.Block
	errorC      <-chan error
	snapshotter *snap.Snapshotter
	validatorC  <-chan bool
	stopC       chan<- struct{}
	once        sync.Once
}

func NewBlockstore(cfg *types.Consensus, snapshotter *snap.Snapshotter, proposeC chan<- *types.Block, commitC <-chan *types.Block, errorC <-chan error, validatorC <-chan bool, stopC chan<- struct{}) *RaftClient {
	c := drivers.NewBaseClient(cfg)
	client := &RaftClient{BaseClient: c, proposeC: proposeC, snapshotter: snapshotter, validatorC: validatorC, commitC: commitC, errorC: errorC, stopC: stopC}
	c.SetChild(client)
	return client
}

func (client *RaftClient) GetGenesisBlockTime() int64 {
	return genesisBlockTime
}

func (client *RaftClient) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	tx.Execer = []byte(cty.CoinsX)
	tx.To = genesis
	//gen payload
	g := &cty.CoinsAction_Genesis{}
	g.Genesis = &types.AssetsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&cty.CoinsAction{Value: g, Ty: cty.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

func (client *RaftClient) ProcEvent(msg queue.Message) bool {
	return false
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
	client.stopC <- struct{}{}
	rlog.Info("consensus raft closed")
}

func (client *RaftClient) CreateBlock() {
	issleep := true
	retry := 0
	infoflag := 0
	count := 0

	//打包区块前先同步到最大高度
	for {
		if client.IsCaughtUp() {
			rlog.Info("Leader has caught up the max height")
			break
		}
		time.Sleep(time.Second)
		retry++
		if retry >= 600 {
			panic("This node encounter problem, exit.")
		}
	}

	for {
		//如果leader节点突然挂了，不是打包节点，需要退出
		if !isLeader {
			rlog.Warn("I'm not the validator node anymore, exit.=============================")
			break
		}
		infoflag++
		if infoflag >= 3 {
			rlog.Info("==================This is Leader node=====================")
			infoflag = 0
		}
		if issleep {
			time.Sleep(10 * time.Second)
			count++
		}

		if count >= 12 {
			rlog.Info("Create an empty block")
			block := client.GetCurrentBlock()
			emptyBlock := &types.Block{}
			emptyBlock.StateHash = block.StateHash
			emptyBlock.ParentHash = block.Hash()
			emptyBlock.Height = block.Height + 1
			emptyBlock.Txs = nil
			emptyBlock.TxHash = zeroHash[:]
			emptyBlock.BlockTime = types.Now().Unix()

			entry := emptyBlock
			client.propose(entry)

			er := client.WriteBlock(block.StateHash, emptyBlock)
			if er != nil {
				rlog.Error(fmt.Sprintf("********************err:%v", er.Error()))
				continue
			}
			client.SetCurrentBlock(emptyBlock)
			count = 0
		}

		lastBlock := client.GetCurrentBlock()
		txs := client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber), nil)
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		count = 0
		rlog.Debug("==================start create new block!=====================")
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		client.AddTxsToBlock(&newblock, txs)
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = types.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		blockEntry := newblock
		client.propose(&blockEntry)
		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		if err != nil {
			issleep = true
			rlog.Error(fmt.Sprintf("********************err:%v", err.Error()))
			continue
		}
		time.Sleep(time.Second * time.Duration(writeBlockSeconds))
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
