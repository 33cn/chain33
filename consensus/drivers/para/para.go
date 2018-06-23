package para

import (
	"bytes"
	"context"
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

var (
	plog             = log.New("module", "para")
	grpcSite         = "localhost:8802"
	currSeq    int64 = 0
	lastSeq    int64 = 0
	seqStep    int64 = 10 //experience needed
	txOps      []txOperation
	filterExec       = "ticket" //execName not decided
	AddAct     int64 = 1
	DelAct     int64 = 2 //reference blockstore.go
)

type txOperation struct {
	tx *types.Transaction
	ty int64 //AddAct or DelAct
}

type Client struct {
	*drivers.BaseClient
	conn       *grpc.ClientConn
	grpcClient types.GrpcserviceClient
	Txsmu      sync.Mutex
}

func New(cfg *types.Consensus) *Client {
	c := drivers.NewBaseClient(cfg)
	grpcSite = cfg.ParaRemoteGrpcClient
	plog.Debug("New Para consensus client")
	msgRecvOp := grpc.WithMaxMsgSize(11 * 1024 * 1024)
	conn, err := grpc.Dial(grpcSite, grpc.WithInsecure(), msgRecvOp)
	if err != nil {
		panic(err)
	}
	grpcClient := types.NewGrpcserviceClient(conn)

	para := &Client{c, conn, grpcClient, sync.Mutex{}}

	c.SetChild(para)

	go para.ManageTxs()

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
	if len(blockdetail.Block.Txs) == 0 {
		return nil, deltx, types.ErrNoTx
	}
	return blockdetail, deltx, nil
}

func (client *Client) FilterTxsForPara(Txs []*types.Transaction) []*types.Transaction {
	var txs []*types.Transaction
	for _, tx := range Txs {
		if string(tx.Execer) == filterExec {
			txs = append(txs, tx)
		}
	}
	return txs
}

//sequence start from 0 in blockchain
func (client *Client) GetCurrentSeq() int64 {
	//database or from txhash
	return 0
}

func (client *Client) SetTxs() {
	plog.Debug("Para consensus SetTxs")
	client.Txsmu.Lock()
	defer client.Txsmu.Unlock()

	lastSeq, err := client.GetLastSeqOnMainChain()
	if err != nil {
		return
	}
	plog.Error("SetTxs", "LastSeq", lastSeq, "currSeq", currSeq)
	if lastSeq > currSeq {
		blockSeq, _ := client.GetBlockHashFromMainChain(currSeq, currSeq+1)
		if blockSeq == nil {
			plog.Debug("Not found block hash on seq", "start", currSeq, "end", currSeq+1)
			return
		}
		currSeq += 1
		var hashes [][]byte
		for _, item := range blockSeq.Items {
			hashes = append(hashes, item.Hash)
			//break
		}

		blockDetails, _ := client.GetBlocksByHashesFromMainChain(hashes)
		if blockDetails == nil {
			plog.Error("GetBlockDetailerr")
			return
		}

		for i, _ := range blockSeq.Items {

			opTy := blockSeq.Items[i].Type
			txs := blockDetails.Items[i].Block.Txs
			//对每一个block进行操作，保留相关TX
			//为TX置标志位
			txs = client.FilterTxsForPara(txs)
			client.SetOpTxs(txs, opTy)
		}
	}

}

func (client *Client) SetOpTxs(txs []*types.Transaction, ty int64) {

	if len(txOps) != 0 {
		for i, _ := range txOps {
			for j, _ := range txs {
				if bytes.Equal(txOps[i].tx.Hash(), txs[j].Hash()) {
					txOps[i].ty = ty
					//modify:生成两个新切片，记录下来，后续再处理
					txs = append(txs[:j], txs[j+1:]...)
				}
			}
		}
	}

	for i, _ := range txs {
		temp := txOperation{txs[i], ty}
		txOps = append(txOps, temp)
	}
}

func (client *Client) MonitorTxs() {
	plog.Debug("MonitorTxs", "len for txs", len(txOps))
}

func (client *Client) ManageTxs() {
	//during start
	currSeq = client.GetCurrentSeq()
	plog.Debug("Para consensus ManageTxs")
	for {
		time.Sleep(time.Second)
		client.SetTxs()
		client.MonitorTxs()
	}

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

func (client *Client) Close() {
	//清空交易
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

//从txOps拿交易
//正常情况下，打包交易
//如果有del标识，先删除原来区块，重新打包
//需要更新txOps
func (client *Client) CreateBlock() {

	issleep := true
	for {
		//don't check condition for block coughtup
		if !client.IsMining() {
			time.Sleep(time.Second)
			continue
		}
		if issleep {
			time.Sleep(time.Second)
		}
		lastBlock := client.GetCurrentBlock()
		txs := client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber), nil)
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		//check dup
		//txs = client.CheckTxDup(txs)
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		client.AddTxsToBlock(&newblock, txs)
		//solo 挖矿固定难度
		newblock.Difficulty = types.GetP(0).PowLimitBits
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = time.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		//判断有没有交易是被删除的，这类交易要从mempool 中删除
		if err != nil {
			issleep = true
			continue
		}
	}
}
