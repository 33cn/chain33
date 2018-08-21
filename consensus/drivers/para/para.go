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
	"gitlab.33.cn/chain33/chain33/common"
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
	genesisBlockTime   int64 = 1514533390
	startHeight        int64 = 0 //parachain sync from startHeight in mainchain
	searchSeq          int64 = 0 //start sequence to search  startHeight in mainchain
	blockSec           int64 = 5 //write block interval, second
	emptyBlockInterval int64 = 4 //write empty block every interval blocks in mainchain
	zeroHash           [32]byte
	grpcRecSize        int = 30 * 1024 * 1024 //the size should be limited in server
	paraAccount            = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
)

type ParaClient struct {
	*drivers.BaseClient
	conn            *grpc.ClientConn
	grpcClient      types.GrpcserviceClient
	lock            sync.RWMutex
	isCatchingUp    bool
	commitMsgClient *CommitMsgClient
}

func New(cfg *types.Consensus) *ParaClient {
	c := drivers.NewBaseClient(cfg)
	if cfg.ParaRemoteGrpcClient != "" {
		grpcSite = cfg.ParaRemoteGrpcClient
	}
	if cfg.StartHeight > 0 {
		startHeight = cfg.StartHeight
		searchSeq = calcSearchseq(cfg.StartHeight)
	}
	if cfg.WriteBlockSeconds > 0 {
		blockSec = cfg.WriteBlockSeconds
	}
	if cfg.EmptyBlockInterval > 0 {
		emptyBlockInterval = cfg.EmptyBlockInterval
	}
	if cfg.ParaAccount != "" {
		paraAccount = cfg.ParaAccount
	}

	plog.Debug("New Para consensus client")

	msgRecvOp := grpc.WithMaxMsgSize(grpcRecSize)
	conn, err := grpc.Dial(grpcSite, grpc.WithInsecure(), msgRecvOp)

	if err != nil {
		panic(err)
	}
	grpcClient := types.NewGrpcserviceClient(conn)

	para := &ParaClient{c, conn, grpcClient, sync.RWMutex{}, false, nil}
	para.commitMsgClient = &CommitMsgClient{
		paraClient:      para,
		commitMsgNotify: make(chan *CommitMsg, 1),
		delMsgNotify:    make(chan *CommitMsg, 1),
		mainBlockAdd:    make(chan *types.BlockDetail, 1),
		quit:            make(chan struct{}),
	}

	c.SetChild(para)

	return para
}

func calcSearchseq(height int64) (seq int64) {
	if height < 1000 {
		return 0
	} else {
		seq = height - 1000
	}
	return seq
}

//para 不检查任何的交易
func (client *ParaClient) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	return nil
}

func (client *ParaClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error) {
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

func (client *ParaClient) Close() {
	client.BaseClient.Close()
	close(client.commitMsgClient.quit)
	client.conn.Close()
	plog.Info("consensus para closed")
}

func (client *ParaClient) SetQueueClient(c queue.Client) {
	plog.Info("Enter SetQueueClient method of Para consensus")
	client.InitClient(c, func() {
		client.InitBlock()
	})
	go client.EventLoop()
	go client.CreateBlock()
	go client.commitMsgClient.handler()
}

func (client *ParaClient) InitBlock() {
	block, err := client.RequestLastBlock()
	if err != nil {
		panic(err)
	}

	if block == nil {
		startSeq := int64(0)
		if searchSeq > 0 {
			startSeq = client.GetSeqByHeightOnMain(startHeight, searchSeq)
		}
		// 创世区块
		newblock := &types.Block{}
		newblock.Height = 0
		newblock.BlockTime = genesisBlockTime
		newblock.ParentHash = zeroHash[:]
		tx := client.CreateGenesisTx()
		newblock.Txs = tx
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		client.WriteBlock(zeroHash[:], newblock, nil, startSeq-int64(1))
	} else {
		client.SetCurrentBlock(block)
	}
}

func (client *ParaClient) GetSeqByHeightOnMain(height int64, originSeq int64) int64 {
	lastSeq, err := client.GetLastSeqOnMainChain()
	plog.Info("Searching for the sequence", "heightOnMain", height, "searchSeq", searchSeq, "lastSeq", lastSeq)
	if err != nil {
		panic(err)
	}
	hint := time.NewTicker(10 * time.Second)
	defer hint.Stop()
	for originSeq <= lastSeq {
		select {
		case <-hint.C:
			plog.Info("Still Searching......", "searchAtSeq", originSeq, "lastSeq", lastSeq)
		default:
			blockDetail, seqTy, err := client.GetBlockOnMainBySeq(originSeq)
			if err != nil {
				panic(err)
			}
			if blockDetail.Block.Height == height && seqTy == AddAct {
				plog.Info("the target sequence in mainchain", "heightOnMain", height, "targetSeq", originSeq)
				return originSeq
			}
			originSeq++
		}
	}
	panic("Main chain has not reached the height currently")
}

func (client *ParaClient) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	tx.Execer = []byte(types.ExecName("coins"))
	tx.To = client.Cfg.Genesis
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

func (client *ParaClient) ProcEvent(msg queue.Message) bool {
	return false
}

func (client *ParaClient) FilterTxsForPara(main *types.BlockDetail) []*types.Transaction {
	var txs []*types.Transaction
	for i, tx := range main.Block.Txs {
		if bytes.Contains(tx.Execer, []byte(types.ExecNamePrefix)) && main.Receipts[i].Ty == types.ExecOk {
			txs = append(txs, tx)
		}
	}
	return txs
}

//get the last sequence in parachain
func (client *ParaClient) GetLastSeq() (int64, error) {
	msg := client.GetQueueClient().NewMessage("blockchain", types.EventGetLastBlockSequence, "")
	client.GetQueueClient().Send(msg, true)
	resp, err := client.GetQueueClient().Wait(msg)
	if err != nil {
		return -2, err
	}
	if lastSeq, ok := resp.GetData().(*types.Int64); ok {
		return lastSeq.Data, nil
	}
	return -2, errors.New("Not an int64 data")
}

func (client *ParaClient) GetBlockedSeq(hash []byte) (int64, error) {
	//from blockchain db
	msg := client.GetQueueClient().NewMessage("blockchain", types.EventGetSeqByHash, &types.ReqHash{hash})
	client.GetQueueClient().Send(msg, true)
	resp, _ := client.GetQueueClient().Wait(msg)
	if blockedSeq, ok := resp.GetData().(*types.Int64); ok {
		return blockedSeq.Data, nil
	}
	return -2, errors.New("Not an int64 data")
}

func (client *ParaClient) GetLastSeqOnMainChain() (int64, error) {
	seq, err := client.grpcClient.GetLastBlockSequence(context.Background(), &types.ReqNil{})
	if err != nil {
		plog.Error("GetLastSeqOnMainChain", "Error", err.Error())
		return -1, err
	}
	//the reflect checked in grpcHandle
	return seq.Data, nil
}

func (client *ParaClient) GetBlocksByHashesFromMainChain(hashes [][]byte) (*types.BlockDetails, error) {
	req := &types.ReqHashes{hashes}
	blocks, err := client.grpcClient.GetBlockByHashes(context.Background(), req)
	if err != nil {
		plog.Error("GetBlocksByHashesFromMainChain", "Error", err.Error())
		return nil, err
	}
	return blocks, nil
}

func (client *ParaClient) GetBlockHashFromMainChain(start int64, end int64) (*types.BlockSequences, error) {
	req := &types.ReqBlocks{start, end, true, []string{}}
	blockSeqs, err := client.grpcClient.GetBlockSequences(context.Background(), req)
	if err != nil {
		plog.Error("GetBlockHashFromMainChain", "Error", err.Error())
		return nil, err
	}
	return blockSeqs, nil
}

func (client *ParaClient) GetBlockOnMainBySeq(seq int64) (*types.BlockDetail, int64, error) {
	blockSeqs, err := client.GetBlockHashFromMainChain(seq, seq)
	if err != nil {
		plog.Error("Not found block hash on seq", "start", seq, "end", seq)
		return nil, -1, err
	}

	var hashes [][]byte
	for _, item := range blockSeqs.Items {
		hashes = append(hashes, item.Hash)
	}

	blockDetails, err := client.GetBlocksByHashesFromMainChain(hashes)
	if err != nil {
		return nil, -1, err
	}

	//protect the boundary
	if len(blockSeqs.Items) != len(blockDetails.Items) {
		panic("Inconsistency between GetBlockSequences and GetBlockByHashes")
	}

	return blockDetails.Items[0], blockSeqs.Items[0].Type, nil
}

func (client *ParaClient) RequestTx(currSeq int64) ([]*types.Transaction, *types.Block, int64, error) {
	plog.Debug("Para consensus RequestTx")

	lastSeq, err := client.GetLastSeqOnMainChain()
	if err != nil {
		return nil, nil, -1, err
	}
	plog.Info("RequestTx", "LastSeq", lastSeq, "CurrSeq", currSeq)
	if lastSeq >= currSeq {
		if lastSeq-currSeq > emptyBlockInterval {
			client.isCatchingUp = true
		} else {
			client.isCatchingUp = false
		}
		blockDetail, seqTy, err := client.GetBlockOnMainBySeq(currSeq)
		if err != nil {
			return nil, nil, -1, err
		}
		txs := client.FilterTxsForPara(blockDetail)
		plog.Info("GetCurrentSeq", "Len of txs", len(txs), "seqTy", seqTy)

		client.commitMsgClient.onMainBlockAdded(blockDetail)

		return txs, blockDetail.Block, seqTy, nil
	}
	plog.Debug("Waiting new sequence from main chain")
	time.Sleep(time.Second * time.Duration(blockSec*2))
	return nil, nil, -1, errors.New("Waiting new sequence")
}

//正常情况下，打包交易
func (client *ParaClient) CreateBlock() {
	incSeqFlag := true
	currSeq, err := client.GetLastSeq()
	if err != nil {
		plog.Error("Parachain GetLastSeq fail", "err", err)
		return
	}
	for {
		lastSeq, err := client.GetLastSeq()
		if err != nil {
			plog.Error("Parachain GetLastSeq fail", "err", err)
			time.Sleep(time.Second)
			continue
		}

		if incSeqFlag || currSeq == lastSeq {
			currSeq++
		}

		txs, blockOnMain, seqTy, err := client.RequestTx(currSeq)
		if err != nil {
			incSeqFlag = false
			time.Sleep(time.Second)
			continue
		}

		lastBlock, err := client.RequestLastBlock()
		if err != nil {
			plog.Error("Parachain RequestLastBlock fail", "err", err)
			incSeqFlag = false
			time.Sleep(time.Second)
			continue
		}
		blockedSeq, err := client.GetBlockedSeq(lastBlock.Hash())
		if err != nil {
			plog.Error("Parachain GetBlockedSeq fail", "err", err)
			incSeqFlag = false
			time.Sleep(time.Second)
			continue
		}
		//sequence in main chain start from 0
		if blockedSeq == -1 {
			blockedSeq = 0
		}
		savedBlockOnMain, _, err := client.GetBlockOnMainBySeq(blockedSeq)
		if err != nil {
			incSeqFlag = false
			time.Sleep(time.Second)
			continue
		}
		plog.Info("Parachain process block", "blockedSeq", blockedSeq, "blockOnMain.Height", blockOnMain.Height, "savedBlockOnMain.Height", savedBlockOnMain.Block.Height)

		if seqTy == DelAct {
			if len(txs) == 0 {
				if blockOnMain.Height > savedBlockOnMain.Block.Height {
					incSeqFlag = true
					continue
				}
				plog.Info("Delete empty block")
			}
			err := client.DelBlock(lastBlock, currSeq)
			incSeqFlag = false
			if err != nil {
				plog.Error(fmt.Sprintf("********************err:%v", err.Error()))
			}
		} else if seqTy == AddAct {
			if len(txs) == 0 {
				if blockOnMain.Height-savedBlockOnMain.Block.Height < emptyBlockInterval {
					incSeqFlag = true
					continue
				}
				plog.Info("Create empty block")
			}
			err := client.createBlock(lastBlock, txs, currSeq, blockOnMain)
			incSeqFlag = false
			if err != nil {
				plog.Error(fmt.Sprintf("********************err:%v", err.Error()))
			}
		} else {
			plog.Error("Incorrect sequence type")
			incSeqFlag = false
		}
		if !client.isCatchingUp {
			time.Sleep(time.Second * time.Duration(blockSec))
		}
	}
}

func (client *ParaClient) createBlock(lastBlock *types.Block, txs []*types.Transaction, seq int64, mainBlock *types.Block) error {
	var newblock types.Block
	plog.Debug(fmt.Sprintf("the len txs is: %v", len(txs)))
	newblock.ParentHash = lastBlock.Hash()
	newblock.Height = lastBlock.Height + 1
	newblock.Txs = txs
	//挖矿固定难度
	newblock.Difficulty = types.GetP(0).PowLimitBits
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	newblock.BlockTime = mainBlock.BlockTime
	err := client.WriteBlock(lastBlock.StateHash, &newblock, mainBlock, seq)
	plog.Debug("para create new Block", "newblock.ParentHash", common.ToHex(newblock.ParentHash),
		"newblock.Height", newblock.Height,
		"newblock.TxHash", common.ToHex(newblock.TxHash),
		"newblock.BlockTime", newblock.BlockTime,
		"sequence", seq)
	return err
}

// 向blockchain写区块
func (client *ParaClient) WriteBlock(prev []byte, paraBlock *types.Block, mainBlock *types.Block, seq int64) error {
	plog.Debug("write block in parachain")
	var oriTxHashs [][]byte
	for _, tx := range paraBlock.Txs {
		oriTxHashs = append(oriTxHashs, tx.Hash())
	}

	blockDetail, deltx, err := client.ExecBlock(prev, paraBlock)
	if len(deltx) > 0 {
		plog.Warn("parachain receive invalid txs")
	}
	if err != nil {
		return err
	}
	parablockDetail := &types.ParaChainBlockDetail{blockDetail, seq}
	msg := client.GetQueueClient().NewMessage("blockchain", types.EventAddParaChainBlockDetail, parablockDetail)
	client.GetQueueClient().Send(msg, true)
	resp, err := client.GetQueueClient().Wait(msg)
	if err != nil {
		return err
	}

	if resp.GetData().(*types.Reply).IsOk {
		client.SetCurrentBlock(paraBlock)
		if mainBlock != nil {
			commitMsg := &CommitMsg{
				initTxHashs:   oriTxHashs,
				mainBlockHash: mainBlock.Hash(),
				blockDetail:   blockDetail,
			}
			client.commitMsgClient.onBlockAdded(commitMsg)

		}

	} else {
		reply := resp.GetData().(*types.Reply)
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

// 向blockchain删区块
func (client *ParaClient) DelBlock(block *types.Block, seq int64) error {
	plog.Debug("delete block in parachain")
	start := block.Height
	if start == 0 {
		panic("Parachain attempt to Delete GenesisBlock !")
	}
	msg := client.GetQueueClient().NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, start, true, []string{""}})
	client.GetQueueClient().Send(msg, true)
	resp, err := client.GetQueueClient().Wait(msg)
	if err != nil {
		return err
	}
	blocks := resp.GetData().(*types.BlockDetails)

	parablockDetail := &types.ParaChainBlockDetail{blocks.Items[0], seq}
	msg = client.GetQueueClient().NewMessage("blockchain", types.EventDelParaChainBlockDetail, parablockDetail)
	client.GetQueueClient().Send(msg, true)
	resp, err = client.GetQueueClient().Wait(msg)
	if err != nil {
		return err
	}

	if resp.GetData().(*types.Reply).IsOk {
		commitMsg := &CommitMsg{
			blockDetail: blocks.Items[0],
		}
		client.commitMsgClient.onBlockDeleted(commitMsg)
	} else {
		reply := resp.GetData().(*types.Reply)
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}
