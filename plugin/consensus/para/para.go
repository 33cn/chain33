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
	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	paracross "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/consensus"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

const (
	AddAct int64 = 1
	DelAct int64 = 2 //reference blockstore.go

	ParaCrossTxCount = 2 //current only support 2 txs for cross
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
	//current miner tx take any privatekey for unify all nodes sign purpose, and para chain is free
	minerPrivateKey string = "6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b"
)

func init() {
	drivers.Reg("para", New)
	drivers.QueryData.Register("para", &ParaClient{})
}

type ParaClient struct {
	*drivers.BaseClient
	conn            *grpc.ClientConn
	grpcClient      types.Chain33Client
	paraClient      paracross.ParacrossClient
	isCatchingUp    bool
	commitMsgClient *CommitMsgClient
	authAccount     string
	privateKey      crypto.PrivKey
	wg              sync.WaitGroup
}

func New(cfg *types.Consensus, sub []byte) queue.Module {
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

	pk, err := hex.DecodeString(minerPrivateKey)
	if err != nil {
		panic(err)
	}
	secp, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	priKey, err := secp.PrivKeyFromBytes(pk)
	if err != nil {
		panic(err)
	}

	plog.Debug("New Para consensus client")

	msgRecvOp := grpc.WithMaxMsgSize(grpcRecSize)
	conn, err := grpc.Dial(grpcSite, grpc.WithInsecure(), msgRecvOp)

	if err != nil {
		panic(err)
	}
	grpcClient := types.NewChain33Client(conn)
	paraCli := paracross.NewParacrossClient(conn)

	para := &ParaClient{
		BaseClient:  c,
		conn:        conn,
		grpcClient:  grpcClient,
		paraClient:  paraCli,
		authAccount: cfg.AuthAccount,
		privateKey:  priKey,
	}

	if cfg.WaitBlocks4CommitMsg < 2 {
		panic("config WaitBlocks4CommitMsg should not less 2")
	}
	para.commitMsgClient = &CommitMsgClient{
		paraClient:      para,
		waitMainBlocks:  cfg.WaitBlocks4CommitMsg,
		commitMsgNotify: make(chan int64, 1),
		delMsgNotify:    make(chan int64, 1),
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
	err := CheckMinerTx(current)
	return err
}

func (client *ParaClient) Close() {
	client.BaseClient.Close()
	close(client.commitMsgClient.quit)
	client.wg.Wait()
	client.conn.Close()
	plog.Info("consensus para closed")
}

func (client *ParaClient) SetQueueClient(c queue.Client) {
	plog.Info("Enter SetQueueClient method of Para consensus")
	client.InitClient(c, func() {
		client.InitBlock()
	})
	go client.EventLoop()

	client.wg.Add(1)
	go client.commitMsgClient.handler()
	go client.CreateBlock()
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
		client.WriteBlock(zeroHash[:], newblock, startSeq-int64(1))
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
	tx.Execer = []byte(types.ExecName(cty.CoinsX))
	tx.To = client.Cfg.Genesis
	//gen payload
	g := &cty.CoinsAction_Genesis{}
	g.Genesis = &types.AssetsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&cty.CoinsAction{Value: g, Ty: cty.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

func (client *ParaClient) ProcEvent(msg queue.Message) bool {
	return false
}

//1. 如果涉及跨链合约，如果有超过两条平行链的交易被判定为失败，交易组会执行不成功,也不PACK。（这样的情况下，主链交易一定会执行不成功）
//2. 如果不涉及跨链合约，那么交易组没有任何规定，可以是20比，10条链。 如果主链交易有失败，平行链也不会执行
//3. 如果交易组有一个ExecOk,主链上的交易都是ok的，可以全部打包
//4. 如果全部是ExecPack，有两种情况，一是交易组所有交易都是平行链交易，另一是主链有交易失败而打包了的交易，需要检查LogErr，如果有错，全部不打包
func calcParaCrossTxGroup(tx *types.Transaction, main *types.BlockDetail, index int) ([]*types.Transaction, int) {
	var headIdx int

	for i := index; i >= 0; i-- {
		if bytes.Equal(tx.Header, main.Block.Txs[i].Hash()) {
			headIdx = i
			break
		}
	}

	endIdx := headIdx + int(tx.GroupCount)
	for i := headIdx; i < endIdx; i++ {
		if types.IsParaExecName(string(main.Block.Txs[i].Execer)) {
			continue
		}
		if main.Receipts[i].Ty == types.ExecOk {
			return main.Block.Txs[headIdx:endIdx], endIdx
		} else {
			for _, log := range main.Receipts[i].Logs {
				if log.Ty == types.TyLogErr {
					return nil, endIdx
				}
			}
		}
	}
	//全部是平行链交易 或主链执行非失败的tx
	return main.Block.Txs[headIdx:endIdx], endIdx
}

func (client *ParaClient) FilterTxsForPara(main *types.BlockDetail) []*types.Transaction {
	var txs []*types.Transaction
	for i := 0; i < len(main.Block.Txs); i++ {
		tx := main.Block.Txs[i]
		if types.IsParaExecName(string(tx.Execer)) {
			if tx.GroupCount >= ParaCrossTxCount {
				mainTxs, endIdx := calcParaCrossTxGroup(tx, main, i)
				txs = append(txs, mainTxs...)
				i = endIdx - 1
				continue
			}
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

		if client.authAccount != "" {
			client.commitMsgClient.onMainBlockAdded(blockDetail)
		}

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

// miner tx need all para node create, but not all node has auth account, here just not sign to keep align
func (client *ParaClient) addMinerTx(preStateHash []byte, block *types.Block, main *types.Block) error {
	status := &pt.ParacrossNodeStatus{
		Title:           types.GetTitle(),
		Height:          block.Height,
		PreBlockHash:    block.ParentHash,
		PreStateHash:    preStateHash,
		MainBlockHash:   main.Hash(),
		MainBlockHeight: main.Height,
	}

	tx, err := paracross.CreateRawMinerTx(status)
	if err != nil {
		return err
	}
	tx.Sign(types.SECP256K1, client.privateKey)

	block.Txs = append([]*types.Transaction{tx}, block.Txs...)
	return nil

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

	err := client.addMinerTx(lastBlock.StateHash, &newblock, mainBlock)
	if err != nil {
		return err
	}

	err = client.WriteBlock(lastBlock.StateHash, &newblock, seq)

	plog.Debug("para create new Block", "newblock.ParentHash", common.ToHex(newblock.ParentHash),
		"newblock.Height", newblock.Height, "newblock.TxHash", common.ToHex(newblock.TxHash),
		"newblock.BlockTime", newblock.BlockTime, "sequence", seq)
	return err
}

// 向blockchain写区块
func (client *ParaClient) WriteBlock(prev []byte, paraBlock *types.Block, seq int64) error {
	//共识模块不执行block，统一由blockchain模块执行block并做去重的处理，返回执行后的blockdetail
	blockDetail := &types.BlockDetail{Block: paraBlock}

	parablockDetail := &types.ParaChainBlockDetail{blockDetail, seq}
	msg := client.GetQueueClient().NewMessage("blockchain", types.EventAddParaChainBlockDetail, parablockDetail)
	client.GetQueueClient().Send(msg, true)
	resp, err := client.GetQueueClient().Wait(msg)
	if err != nil {
		return err
	}
	blkdetail := resp.GetData().(*types.BlockDetail)
	if blkdetail == nil {
		return errors.New("block detail is nil")
	}

	client.SetCurrentBlock(blkdetail.Block)

	if client.authAccount != "" {
		client.commitMsgClient.onBlockAdded(blkdetail.Block.Height)
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
		if client.authAccount != "" {
			client.commitMsgClient.onBlockDeleted(blocks.Items[0].Block.Height)
		}
	} else {
		reply := resp.GetData().(*types.Reply)
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}
