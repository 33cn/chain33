package para

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/inconshreveable/log15"
	//"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
	"google.golang.org/grpc"
)

const (
	AddAct int64 = 1
	DelAct int64 = 2 //reference blockstore.go
)

var (
	plog                     = log.New("module", "para")
	grpcSite                 = "localhost:8802"
	filterExec               = "user.ticket" //execName not decided
	txCacheSize        int64 = 10240
	blockSec           int64 = 2 //write block interval, second
	emptyBlockInterval int64 = 5 //write empty block every interval blocks in main chain
	zeroHash           [32]byte
	grpcRecSize        int = 30 * 1024 * 1024 //the size should be limited in server
)

type Client struct {
	*drivers.BaseClient
	conn       *grpc.ClientConn
	grpcClient types.GrpcserviceClient
	lock       sync.RWMutex
}

func New(cfg *types.Consensus) *Client {
	c := drivers.NewBaseClient(cfg)
	grpcSite = cfg.ParaRemoteGrpcClient

	plog.Debug("New Para consensus client")

	msgRecvOp := grpc.WithMaxMsgSize(grpcRecSize)
	conn, err := grpc.Dial(grpcSite, grpc.WithInsecure(), msgRecvOp)

	if err != nil {
		panic(err)
	}
	grpcClient := types.NewGrpcserviceClient(conn)

	para := &Client{c, conn, grpcClient, sync.RWMutex{}}

	c.SetChild(para)

	return para
}

//para 不检查任何的交易
func (client *Client) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	return nil
}

func (client *Client) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error) {
	//exec block
	if block.Height == 0 {
		block.Difficulty = types.GetP(0).PowLimitBits
	}
	blockdetail, deltx, err := util.ExecBlock(client.GetQueueClient(), prevHash, block, false, true)
	if err != nil { //never happen
		return nil, deltx, err
	}
	//if len(blockdetail.Block.Txs) == 0 {
	//	return nil, deltx, types.ErrNoTx
	//}
	return blockdetail, deltx, nil
}

func (client *Client) Close() {
	plog.Info("consensus para closed")
}

func (client *Client) CreateGenesisTx() (ret []*types.Transaction) {
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

func (client *Client) ProcEvent(msg queue.Message) bool {
	return false
}

func (client *Client) FilterTxsForPara(Txs []*types.Transaction) []*types.Transaction {
	var txs []*types.Transaction
	for _, tx := range Txs {
		if bytes.Contains(tx.Execer, []byte(types.ExecNamePrefix)) {
			txs = append(txs, tx)
		}
	}
	return txs
}

//get the last sequence in parachain
func (client *Client) GetBlockedSeq() int64 {
	lastBlock := client.GetCurrentBlock()
	if lastBlock.Height == 0 {
		return -1
	}
	return client.GetSeqByBlockhash(lastBlock.Hash())
}

func (client *Client) GetSeqByBlockhash(hash []byte) int64 {
	//from blockchain db
	blockedSeq := seqMap[string(hash)]
	return blockedSeq
}

func (client *Client) GetLastSeqOnMainChain() (int64, error) {
	seq, err := client.grpcClient.GetLastBlockSequence(context.Background(), &types.ReqNil{})
	if err != nil {
		plog.Error("GetLastSeqOnMainChain", "Error", err.Error())
		return -1, err
	}
	//the reflect checked in grpcHandle
	return seq.Data, nil
}

func (client *Client) GetBlocksByHashesFromMainChain(hashes [][]byte) (*types.BlockDetails, error) {
	req := &types.ReqHashes{hashes}
	blocks, err := client.grpcClient.GetBlockByHashes(context.Background(), req)
	if err != nil {
		plog.Error("GetBlocksByHashesFromMainChain", "Error", err.Error())
		return nil, err
	}
	return blocks, nil
}

func (client *Client) GetBlockHashFromMainChain(start int64, end int64) (*types.BlockSequences, error) {
	req := &types.ReqBlocks{start, end, true, []string{}}
	blockSeq, err := client.grpcClient.GetBlockSequences(context.Background(), req)
	if err != nil {
		plog.Error("GetBlockHashFromMainChain", "Error", err.Error())
		return nil, err
	}
	return blockSeq, nil
}

func (client *Client) GetBlockOnMainBySeq(seq int64) (*types.Block, int64, error) {
	blockSeq, err := client.GetBlockHashFromMainChain(seq, seq)
	if err != nil {
		plog.Error("Not found block hash on seq", "start", seq, "end", seq)
		return nil, -1, err
	}

	var hashes [][]byte
	for _, item := range blockSeq.Items {
		hashes = append(hashes, item.Hash)
	}

	blockDetails, err := client.GetBlocksByHashesFromMainChain(hashes)
	if err != nil {
		return nil, -1, err
	}

	//protect the boundary
	if len(blockSeq.Items) != len(blockDetails.Items) {
		panic("")
	}

	return blockDetails.Items[0].Block, blockSeq.Items[0].Type, nil
}

func (client *Client) RequestTx(currSeq int64) ([]*types.Transaction, int64, int64, error) {
	plog.Debug("Para consensus RequestTx")

	lastSeq, err := client.GetLastSeqOnMainChain()
	if err != nil {
		return nil, -1, -1, err
	}
	plog.Info("RequestTx", "LastSeq", lastSeq, "CurrSeq", currSeq)
	if lastSeq >= currSeq {
		//debug phase
		//if currSeq > 10 {
		//	return nil, -1, -1, errors.New("Just for debug")
		//}

		block, seqTy, err := client.GetBlockOnMainBySeq(currSeq)
		if err != nil {
			return nil, -1, -1, err
		}
		height := block.Height
		txs := client.FilterTxsForPara(block.Txs)
		plog.Info("GetCurrentSeq", "Len of txs", len(txs), "seqTy", seqTy)

		return txs, height, seqTy, nil
	}
	plog.Info("Waiting new sequence from main chain")
	time.Sleep(time.Second * time.Duration(blockSec*2))
	return nil, -1, -1, errors.New("Waiting new sequence")
}

var seqMap map[string]int64

//正常情况下，打包交易
func (client *Client) CreateBlock() {
	incSeqFlag := true
	currSeq := client.GetBlockedSeq()
	seqMap = make(map[string]int64)
	for {
		//don't check condition for block caughtup
		if !client.IsMining() {
			time.Sleep(time.Second)
			continue
		}
		if incSeqFlag || client.GetBlockedSeq() == currSeq {
			currSeq++
		}

		txs, heightOnMain, seqTy, err := client.RequestTx(currSeq)
		if err != nil {
			incSeqFlag = false
			time.Sleep(time.Second)
			continue
		}
		lastBlock := client.GetCurrentBlock()
		blockedSeq := client.GetBlockedSeq()
		//sequence in main chain start from 0
		if blockedSeq == -1 {
			blockedSeq = 0
		}
		blockOnMain, _, err := client.GetBlockOnMainBySeq(blockedSeq)
		if err != nil {
			incSeqFlag = false
			time.Sleep(time.Second)
			continue
		}
		plog.Info("Parachain CreateBlock", "blockedSeq", blockedSeq, "heightOnMain", heightOnMain, "blockOnMain.Height", blockOnMain.Height)

		if seqTy == DelAct {
			if len(txs) == 0 {
				if heightOnMain > blockOnMain.Height {
					incSeqFlag = true
					time.Sleep(time.Second)
					continue
				}
				plog.Info("Delete empty block")
			}
			err := client.DelBlock(lastBlock, currSeq)
			if err != nil {
				incSeqFlag = false
				continue
			}
			incSeqFlag = true
			time.Sleep(time.Second * time.Duration(blockSec))
		} else if seqTy == AddAct {
			if len(txs) == 0 {
				if heightOnMain-blockOnMain.Height < emptyBlockInterval {
					incSeqFlag = true
					time.Sleep(time.Second)
					continue
				}
				plog.Info("Create empty block")
			}

			//check dup
			//txs = client.CheckTxDup(txs)
			err := client.createBlock(lastBlock, txs, currSeq)
			incSeqFlag = false
			if err != nil {
				continue
			}
			time.Sleep(time.Second * time.Duration(blockSec))
		} else {
			incSeqFlag = false
			plog.Error("Incorrect sequence type")
			time.Sleep(time.Second)
		}
	}
}

func (client *Client) createBlock(lastBlock *types.Block, txs []*types.Transaction, seq int64) error {
	var newblock types.Block
	plog.Debug(fmt.Sprintf("the len txs is: %v", len(txs)))
	newblock.ParentHash = lastBlock.Hash()
	newblock.Height = lastBlock.Height + 1
	client.AddTxsToBlock(&newblock, txs)
	//挖矿固定难度
	newblock.Difficulty = types.GetP(0).PowLimitBits
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	newblock.BlockTime = time.Now().Unix()
	if lastBlock.BlockTime >= newblock.BlockTime {
		newblock.BlockTime = lastBlock.BlockTime + 1
	}
	err := client.WriteBlock(lastBlock.StateHash, &newblock)
	if err != nil {
		plog.Error(fmt.Sprintf("********************err:%v", err.Error()))
		return err
	}
	seqMap[string(newblock.Hash())] = seq
	return nil
}

// 向blockchain写区块
func (client *Client) WriteBlock(prev []byte, block *types.Block) error {
	plog.Debug("write block in parachain")
	blockdetail, deltx, err := client.ExecBlock(prev, block)
	if len(deltx) > 0 {
		plog.Warn("parachain receive invalid txs")
	}
	if err != nil {
		return err
	}
	msg := client.GetQueueClient().NewMessage("blockchain", types.EventAddBlockDetail, blockdetail)
	client.GetQueueClient().Send(msg, true)
	resp, err := client.GetQueueClient().Wait(msg)
	if err != nil {
		return err
	}

	if resp.GetData().(*types.Reply).IsOk {
		client.SetCurrentBlock(block)
	} else {
		reply := resp.GetData().(*types.Reply)
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

// 向blockchain删区块
func (client *Client) DelBlock(block *types.Block, seq int64) error {
	delete(seqMap, string(block.Hash()))
	return nil
}
