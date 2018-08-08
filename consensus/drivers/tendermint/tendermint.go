package tendermint

import (
	"errors"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	ttypes "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

var (
	tendermintlog = log15.New("module", "tendermint")
	genesisDocKey = []byte("genesisDoc")
)

const tendermint_version = "0.1.0"

type TendermintClient struct {
	//config
	*drivers.BaseClient
	genesisDoc    *ttypes.GenesisDoc // initial validator set
	privValidator ttypes.PrivValidator
	privKey       crypto.PrivKey // local node's p2p key
	csState       *ConsensusState
	blockStore    *ttypes.BlockStore
	evidenceDB    dbm.DB
	crypto        crypto.Crypto
	node          *Node
	txsAvailable  chan int64
	consResult    chan int64
}

// DefaultDBProvider returns a database using the DBBackend and DBDir
// specified in the ctx.Config.
func DefaultDBProvider(ID string) (dbm.DB, error) {
	return dbm.NewDB(ID, "leveldb", "./datadir", 0), nil
}

func New(cfg *types.Consensus) *TendermintClient {
	tendermintlog.Info("Start to create tendermint client")

	//init rand
	ttypes.Init()

	genDoc, err := ttypes.GenesisDocFromFile("./genesis.json")
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "msg", "GenesisDocFromFile failded", "error", err)
		return nil
	}

	// Make Evidence Reactor
	evidenceDB, err := DefaultDBProvider("CSevidence")
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "msg", "DefaultDBProvider evidenceDB failded", "error", err)
		return nil
	}

	cr, err := crypto.New(types.GetSignatureTypeName(types.ED25519))
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "err", err)
		return nil
	}

	ttypes.ConsensusCrypto = cr

	priv, err := cr.GenKey()
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "GenKey err", err)
		return nil
	}

	privValidator := ttypes.LoadOrGenPrivValidatorFS("./priv_validator.json")
	if privValidator == nil {
		tendermintlog.Error("NewTendermintClient create priv_validator file failed")
		return nil
	}

	ttypes.InitMessageMap()

	pubkey := privValidator.GetPubKey().KeyString()

	c := drivers.NewBaseClient(cfg)

	blockStore := ttypes.NewBlockStore(c, pubkey)

	client := &TendermintClient{
		BaseClient:    c,
		genesisDoc:    genDoc,
		privValidator: privValidator,
		privKey:       priv,
		blockStore:    blockStore,
		evidenceDB:    evidenceDB,
		crypto:        cr,
		txsAvailable:  make(chan int64, 1),
		consResult:    make(chan int64, 1),
	}

	c.SetChild(client)
	return client
}

// PrivValidator returns the Node's PrivValidator.
// XXX: for convenience only!
func (client *TendermintClient) PrivValidator() ttypes.PrivValidator {
	return client.privValidator
}

// GenesisDoc returns the Node's GenesisDoc.
func (client *TendermintClient) GenesisDoc() *ttypes.GenesisDoc {
	return client.genesisDoc
}

func (client *TendermintClient) Close() {
	tendermintlog.Info("TendermintClientClose", "consensus tendermint closed")
}

func (client *TendermintClient) SetQueueClient(q queue.Client) {
	client.InitClient(q, func() {
		//call init block
		client.InitBlock()
	})

	go client.EventLoop()
	go client.StartConsensus()
}

const DEBUG_CATCHUP = false

func (client *TendermintClient) StartConsensus() {
	//进入共识前先同步到最大高度
	hint := time.NewTicker(5 * time.Second)
	beg := time.Now()
OuterLoop:
	for !DEBUG_CATCHUP {
		select {
		case <-hint.C:
			tendermintlog.Info("Still catching up max height......", "Height", client.GetCurrentHeight(), "cost", time.Since(beg))
		default:
			if client.IsCaughtUp() {
				tendermintlog.Info("This node has caught up max height")
				break OuterLoop
			}
		}
	}
	hint.Stop()

	block := client.GetCurrentBlock()
	if block == nil {
		tendermintlog.Error("StartConsensus failed for current block is nil")
		panic("StartConsensus failed for current block is nil")
	}

	blockInfo, err := ttypes.GetBlockInfo(block)
	if err != nil {
		tendermintlog.Error("StartConsensus GetBlockInfo failed", "error", err)
		panic(fmt.Sprintf("StartConsensus GetBlockInfo failed:%v", err))
	}

	var state State
	if blockInfo == nil {
		if block.Height != 0 {
			tendermintlog.Error("StartConsensus", "msg", "block height is not 0 but blockinfo is nil")
			panic(fmt.Sprintf("StartConsensus block height is %v but block info is nil", block.Height))
		}
		statetmp, err := MakeGenesisState(client.genesisDoc)
		if err != nil {
			tendermintlog.Error("StartConsensus", "msg", "MakeGenesisState failded", "error", err)
			return
		}
		state = statetmp.Copy()
	} else {
		tendermintlog.Info("StartConsensus", "blockinfo", blockInfo)
		csState := blockInfo.GetState()
		if csState == nil {
			tendermintlog.Error("StartConsensus", "msg", "blockInfo.GetState is nil")
			return
		}
		state = LoadState(csState)
		if seenCommit := blockInfo.SeenCommit; seenCommit != nil {
			state.LastBlockID = ttypes.BlockID{
				Hash: seenCommit.BlockID.GetHash(),
			}
		}
	}

	tendermintlog.Info("load state finish", "state", state)
	// Log whether this node is a validator or an observer
	if state.Validators.HasAddress(client.privValidator.GetAddress()) {
		tendermintlog.Info("This node is a validator")
	} else {
		tendermintlog.Info("This node is not a validator")
	}

	stateDB := NewStateDB(client.BaseClient, state)

	//make evidenceReactor
	evidenceStore := NewEvidenceStore(client.evidenceDB)
	evidencePool := NewEvidencePool(stateDB, state, evidenceStore)

	// make block executor for consensus and blockchain reactors to execute blocks
	blockExec := NewBlockExecutor(stateDB, evidencePool)

	// Make ConsensusReactor
	csState := NewConsensusState(client, client.blockStore, state, blockExec, evidencePool)
	// reset height, round, state begin at newheigt,0,0
	client.privValidator.ResetLastHeight(state.LastBlockHeight)
	csState.SetPrivValidator(client.privValidator)

	client.csState = csState

	// Create & add listener
	protocol, listeningAddress := "tcp", "0.0.0.0:46656"
	node := NewNode(client.Cfg.Seeds, protocol, listeningAddress, client.privKey, state.ChainID, tendermint_version, csState, evidencePool)

	client.node = node
	node.Start()

	go client.CreateBlock()
}

func (client *TendermintClient) CreateGenesisTx() (ret []*types.Transaction) {
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

//暂不检查任何的交易
func (client *TendermintClient) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	return nil
}

func (client *TendermintClient) ProcEvent(msg queue.Message) bool {
	return false
}

func (client *TendermintClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error) {
	//exec block
	if block.Height == 0 {
		block.Difficulty = types.GetP(0).PowLimitBits
	}
	blockdetail, deltx, err := util.ExecBlock(client.GetQueueClient(), prevHash, block, false, false)
	if err != nil { //never happen
		return nil, deltx, err
	}
	if len(blockdetail.Block.Txs) == 0 {
		return nil, deltx, types.ErrNoTx
	}
	return blockdetail, deltx, nil
}

func (client *TendermintClient) CreateBlock() {
	issleep := true

	for {
		if !client.csState.IsRunning() {
			tendermintlog.Error("consensus not running now")
			time.Sleep(time.Second)
			continue
		}

		if issleep {
			time.Sleep(time.Second)
		}

		txs := client.RequestTx(10, nil)
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false

		client.txsAvailable <- client.GetCurrentHeight() + 1
		time.Sleep(time.Second)
		//select {
		//case height := <-client.consResult:
		//	tendermintlog.Info("Tendermint consensus reached at", "height", height)
		//}
	}
}

func (client *TendermintClient) TxsAvailable() <-chan int64 {
	return client.txsAvailable
}

func (client *TendermintClient) ConsResult() chan<- int64 {
	return client.consResult
}

func (client *TendermintClient) BuildBlock() *types.Block {
	lastBlock, err := client.RequestLastBlock()
	if err != nil {
		tendermintlog.Error("RequestLastBlock fail", "err", err.Error())
		return nil
	}
	txs := client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber)-1, nil)
	//check dup
	txs = client.CheckTxDup(txs)
	if len(txs) == 0 {
		tendermintlog.Error("No new txs")
		return nil
	}

	newblock := &types.Block{}
	newblock.ParentHash = lastBlock.Hash()
	newblock.Height = lastBlock.Height + 1
	client.AddTxsToBlock(newblock, txs)
	newblock.TxHash = merkle.CalcMerkleRoot(txs)
	//挖矿固定难度
	newblock.Difficulty = types.GetP(0).PowLimitBits
	return newblock
}

func (client *TendermintClient) CommitBlock(propBlock *types.Block) error {
	newblock := *propBlock
	lastBlock, err := client.RequestBlock(newblock.Height - 1)
	if err != nil {
		tendermintlog.Error("RequestBlock fail", "err", err.Error())
		return err
	}
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	newblock.BlockTime = time.Now().Unix()
	if lastBlock.BlockTime >= newblock.BlockTime {
		newblock.BlockTime = lastBlock.BlockTime + 1
	}
	err = client.WriteBlock(lastBlock.StateHash, &newblock)
	if err != nil {
		tendermintlog.Error(fmt.Sprintf("********************CommitBlock err:%v", err.Error()))
		return err
	}
	tendermintlog.Info("Commit block complete", "height", newblock.Height, "CurrentHeight", client.GetCurrentHeight())
	if client.GetCurrentHeight() != newblock.Height {
		tendermintlog.Warn("Encounter problem in commit block")
	}
	return nil
}

func (client *TendermintClient) CheckCommit(height int64) (bool, error) {
	retry := 0
	newHeight := int64(1)
	for {
		newHeight = client.GetCurrentHeight()
		if newHeight >= height {
			tendermintlog.Info("Sync block success", "height", height, "CurrentHeight", newHeight)
			return true, nil
		}
		retry++
		time.Sleep(100 * time.Millisecond)
		if retry >= 600 {
			tendermintlog.Error("Sync block fail", "height", height, "CurrentHeight", newHeight)
			break
		}
	}
	if client.IsCaughtUp() {
		tendermintlog.Info("Tendermint consensus not reached at", "height", height)
		return false, nil
	}
	return false, errors.New("sync block fail")
}
