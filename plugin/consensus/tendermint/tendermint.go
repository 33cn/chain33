package tendermint

import (
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	ttypes "gitlab.33.cn/chain33/chain33/plugin/consensus/tendermint/types"
	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/consensus"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

const tendermint_version = "0.1.0"

var (
	tendermintlog               = log15.New("module", "tendermint")
	genesis                     string
	genesisBlockTime            int64
	timeoutTxAvail              int32 = 1000
	timeoutPropose              int32 = 3000 // millisecond
	timeoutProposeDelta         int32 = 500
	timeoutPrevote              int32 = 1000
	timeoutPrevoteDelta         int32 = 500
	timeoutPrecommit            int32 = 1000
	timeoutPrecommitDelta       int32 = 500
	timeoutCommit               int32 = 1000
	skipTimeoutCommit           bool  = false
	createEmptyBlocks           bool  = false
	createEmptyBlocksInterval   int32 = 0 // second
	validatorNodes                    = []string{"127.0.0.1:46656"}
	peerGossipSleepDuration     int32 = 100
	peerQueryMaj23SleepDuration int32 = 2000
	zeroHash                    [32]byte
)

func init() {
	drivers.Reg("tendermint", New)
	drivers.QueryData.Register("tendermint", &TendermintClient{})
}

type TendermintClient struct {
	//config
	*drivers.BaseClient
	genesisDoc    *ttypes.GenesisDoc // initial validator set
	privValidator ttypes.PrivValidator
	privKey       crypto.PrivKey // local node's p2p key
	pubKey        string
	csState       *ConsensusState
	evidenceDB    dbm.DB
	crypto        crypto.Crypto
	node          *Node
	txsAvailable  chan int64
	stopC         chan struct{}
}

type subConfig struct {
	Genesis                   string   `json:"genesis"`
	GenesisBlockTime          int64    `json:"genesisBlockTime"`
	TimeoutTxAvail            int32    `json:"timeoutTxAvail"`
	TimeoutPropose            int32    `json:"timeoutPropose"`
	TimeoutProposeDelta       int32    `json:"timeoutProposeDelta"`
	TimeoutPrevote            int32    `json:"timeoutPrevote"`
	TimeoutPrevoteDelta       int32    `json:"timeoutPrevoteDelta"`
	TimeoutPrecommit          int32    `json:"timeoutPrecommit"`
	TimeoutPrecommitDelta     int32    `json:"timeoutPrecommitDelta"`
	TimeoutCommit             int32    `json:"timeoutCommit"`
	SkipTimeoutCommit         bool     `json:"skipTimeoutCommit"`
	CreateEmptyBlocks         bool     `json:"createEmptyBlocks"`
	CreateEmptyBlocksInterval int32    `json:"createEmptyBlocksInterval"`
	ValidatorNodes            []string `json:"validatorNodes"`
}

func (client *TendermintClient) applyConfig(sub []byte) {
	var subcfg subConfig
	if sub != nil {
		types.MustDecode(sub, &subcfg)
	}
	if subcfg.Genesis != "" {
		genesis = subcfg.Genesis
	}
	if subcfg.GenesisBlockTime > 0 {
		genesisBlockTime = subcfg.GenesisBlockTime
	}
	if subcfg.TimeoutTxAvail > 0 {
		timeoutTxAvail = subcfg.TimeoutTxAvail
	}
	if subcfg.TimeoutPropose > 0 {
		timeoutPropose = subcfg.TimeoutPropose
	}
	if subcfg.TimeoutProposeDelta > 0 {
		timeoutProposeDelta = subcfg.TimeoutProposeDelta
	}
	if subcfg.TimeoutPrevote > 0 {
		timeoutPrevote = subcfg.TimeoutPrevote
	}
	if subcfg.TimeoutPrevoteDelta > 0 {
		timeoutPrevoteDelta = subcfg.TimeoutPrevoteDelta
	}
	if subcfg.TimeoutPrecommit > 0 {
		timeoutPrecommit = subcfg.TimeoutPrecommit
	}
	if subcfg.TimeoutPrecommitDelta > 0 {
		timeoutPrecommitDelta = subcfg.TimeoutPrecommitDelta
	}
	if subcfg.TimeoutCommit > 0 {
		timeoutCommit = subcfg.TimeoutCommit
	}
	skipTimeoutCommit = subcfg.SkipTimeoutCommit
	createEmptyBlocks = subcfg.CreateEmptyBlocks
	if subcfg.CreateEmptyBlocksInterval > 0 {
		createEmptyBlocksInterval = subcfg.CreateEmptyBlocksInterval
	}
	if len(subcfg.ValidatorNodes) > 0 {
		validatorNodes = subcfg.ValidatorNodes
	}
}

// DefaultDBProvider returns a database using the DBBackend and DBDir
// specified in the ctx.Config.
func DefaultDBProvider(ID string) (dbm.DB, error) {
	return dbm.NewDB(ID, "leveldb", "./datadir", 0), nil
}

func New(cfg *types.Consensus, sub []byte) queue.Module {
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

	cr, err := crypto.New(types.GetSignName("", types.ED25519))
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
	client := &TendermintClient{
		BaseClient:    c,
		genesisDoc:    genDoc,
		privValidator: privValidator,
		privKey:       priv,
		pubKey:        pubkey,
		evidenceDB:    evidenceDB,
		crypto:        cr,
		txsAvailable:  make(chan int64, 1),
		stopC:         make(chan struct{}, 1),
	}
	c.SetChild(client)

	client.applyConfig(sub)
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
	client.stopC <- struct{}{}
	tendermintlog.Info("consensus tendermint closed")
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
			time.Sleep(time.Second)
		}
	}
	hint.Stop()

	curHeight := client.GetCurrentHeight()
	blockInfo, err := client.QueryBlockInfoByHeight(curHeight)
	if curHeight != 0 && err != nil {
		tendermintlog.Error("StartConsensus GetBlockInfo failed", "error", err)
		panic(fmt.Sprintf("StartConsensus GetBlockInfo failed:%v", err))
	}

	var state State
	if blockInfo == nil {
		if curHeight != 0 {
			tendermintlog.Error("StartConsensus", "msg", "block height is not 0 but blockinfo is nil")
			panic(fmt.Sprintf("StartConsensus block height is %v but block info is nil", curHeight))
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
				BlockID: tmtypes.BlockID{
					Hash: seenCommit.BlockID.GetHash(),
				},
			}
		}
	}

	tendermintlog.Info("load state finish", "state", state, "validators", state.Validators)
	valNodes, err := client.QueryValidatorsByHeight(curHeight)
	if err == nil && valNodes != nil {
		if len(valNodes.Nodes) > 0 {
			prevValSet := state.LastValidators.Copy()
			nextValSet := prevValSet.Copy()
			err := updateValidators(nextValSet, valNodes.Nodes)
			if err != nil {
				tendermintlog.Error("Error changing validator set", "error", err)
				//return s, fmt.Errorf("Error changing validator set: %v", err)
			}
			// change results from this height but only applies to the next height
			state.LastHeightValidatorsChanged = curHeight + 1
			nextValSet.IncrementAccum(1)
			state.Validators = nextValSet
			tendermintlog.Info("StartConsensus validators updated", "update-valnodes", valNodes)
		}
	}
	tendermintlog.Info("StartConsensus", "real validators", state.Validators)
	// Log whether this node is a validator or an observer
	if state.Validators.HasAddress(client.privValidator.GetAddress()) {
		tendermintlog.Info("This node is a validator")
	} else {
		tendermintlog.Info("This node is not a validator")
	}

	stateDB := NewStateDB(client, state)

	//make evidenceReactor
	evidenceStore := NewEvidenceStore(client.evidenceDB)
	evidencePool := NewEvidencePool(stateDB, state, evidenceStore)

	// make block executor for consensus and blockchain reactors to execute blocks
	blockExec := NewBlockExecutor(stateDB, evidencePool)

	// Make ConsensusReactor
	csState := NewConsensusState(client, state, blockExec, evidencePool)
	// reset height, round, state begin at newheigt,0,0
	client.privValidator.ResetLastHeight(state.LastBlockHeight)
	csState.SetPrivValidator(client.privValidator)

	client.csState = csState

	// Create & add listener
	protocol, listeningAddress := "tcp", "0.0.0.0:46656"
	node := NewNode(validatorNodes, protocol, listeningAddress, client.privKey, state.ChainID, tendermint_version, csState, evidencePool)

	client.node = node
	node.Start()

	go client.CreateBlock()
}

func (client *TendermintClient) GetGenesisBlockTime() int64 {
	return genesisBlockTime
}

func (client *TendermintClient) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = genesis
	//gen payload
	g := &cty.CoinsAction_Genesis{}
	g.Genesis = &types.AssetsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&cty.CoinsAction{Value: g, Ty: cty.CoinsActionGenesis})
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
		if !client.CheckTxsAvailable() {
			issleep = true
			continue
		}
		issleep = false

		client.txsAvailable <- client.GetCurrentHeight() + 1
		time.Sleep(time.Duration(timeoutTxAvail) * time.Millisecond)
	}
}

func (client *TendermintClient) TxsAvailable() <-chan int64 {
	return client.txsAvailable
}

func (client *TendermintClient) StopC() <-chan struct{} {
	return client.stopC
}

func (client *TendermintClient) CheckTxsAvailable() bool {
	txs := client.RequestTx(10, nil)
	txs = client.CheckTxDup(txs, client.GetCurrentHeight())
	return len(txs) != 0
}

func (client *TendermintClient) CheckTxDup(txs []*types.Transaction, height int64) (transactions []*types.Transaction) {
	cacheTxs := types.TxsToCache(txs)
	var err error
	cacheTxs, err = util.CheckTxDup(client.GetQueueClient(), cacheTxs, height)
	if err != nil {
		return txs
	}
	return types.CacheToTxs(cacheTxs)
}

func (client *TendermintClient) BuildBlock() *types.Block {
	lastHeight := client.GetCurrentHeight()
	txs := client.RequestTx(int(types.GetP(lastHeight+1).MaxTxNumber)-1, nil)
	newblock := &types.Block{}
	newblock.Height = lastHeight + 1
	client.AddTxsToBlock(newblock, txs)
	return newblock
}

func (client *TendermintClient) CommitBlock(propBlock *types.Block) error {
	newblock := *propBlock
	lastBlock, err := client.RequestBlock(newblock.Height - 1)
	if err != nil {
		tendermintlog.Error("RequestBlock fail", "err", err)
		return err
	}
	newblock.ParentHash = lastBlock.Hash()
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	newblock.BlockTime = time.Now().Unix()
	if lastBlock.BlockTime >= newblock.BlockTime {
		newblock.BlockTime = lastBlock.BlockTime + 1
	}
	newblock.Difficulty = types.GetP(0).PowLimitBits

	err = client.WriteBlock(lastBlock.StateHash, &newblock)
	if err != nil {
		tendermintlog.Error(fmt.Sprintf("********************CommitBlock err:%v", err.Error()))
		return err
	}
	tendermintlog.Info("Commit block success", "height", newblock.Height, "CurrentHeight", client.GetCurrentHeight())
	if client.GetCurrentHeight() != newblock.Height {
		tendermintlog.Warn("Commit block fail", "height", newblock.Height, "CurrentHeight", client.GetCurrentHeight())
	}
	return nil
}

func (client *TendermintClient) CheckCommit(height int64) bool {
	retry := 0
	newHeight := int64(1)
	for {
		newHeight = client.GetCurrentHeight()
		if newHeight >= height {
			tendermintlog.Info("Sync block success", "height", height, "CurrentHeight", newHeight)
			return true
		}
		retry++
		time.Sleep(100 * time.Millisecond)
		if retry >= 600 {
			tendermintlog.Warn("Sync block fail", "height", height, "CurrentHeight", newHeight)
			return false
		}
	}
}

func (client *TendermintClient) QueryValidatorsByHeight(height int64) (*tmtypes.ValNodes, error) {
	if height <= 0 {
		return nil, types.ErrInvalidParam
	}
	req := &tmtypes.ReqNodeInfo{Height: height}
	param, err := proto.Marshal(req)
	if err != nil {
		tendermintlog.Error("QueryValidatorsByHeight", "err", err)
		return nil, types.ErrInvalidParam
	}
	msg := client.GetQueueClient().NewMessage("execs", types.EventBlockChainQuery, &types.ChainExecutor{"valnode", "GetValNodeByHeight", zeroHash[:], param, nil})
	client.GetQueueClient().Send(msg, true)
	msg, err = client.GetQueueClient().Wait(msg)
	if err != nil {
		return nil, err
	}
	return msg.GetData().(types.Message).(*tmtypes.ValNodes), nil
}

func (client *TendermintClient) QueryBlockInfoByHeight(height int64) (*tmtypes.TendermintBlockInfo, error) {
	if height <= 0 {
		return nil, types.ErrInvalidParam
	}
	req := &tmtypes.ReqBlockInfo{Height: height}
	param, err := proto.Marshal(req)
	if err != nil {
		tendermintlog.Error("QueryBlockInfoByHeight", "err", err)
		return nil, types.ErrInvalidParam
	}
	msg := client.GetQueueClient().NewMessage("execs", types.EventBlockChainQuery, &types.ChainExecutor{"valnode", "GetBlockInfoByHeight", zeroHash[:], param, nil})
	client.GetQueueClient().Send(msg, true)
	msg, err = client.GetQueueClient().Wait(msg)
	if err != nil {
		return nil, err
	}
	return msg.GetData().(types.Message).(*tmtypes.TendermintBlockInfo), nil
}

func (client *TendermintClient) LoadSeenCommit(height int64) *tmtypes.TendermintCommit {
	blockInfo, err := client.QueryBlockInfoByHeight(height)
	if err != nil {
		panic(fmt.Sprintf("LoadSeenCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		tendermintlog.Error("LoadSeenCommit get nil block info")
		return nil
	}
	return blockInfo.GetSeenCommit()
}

func (client *TendermintClient) LoadBlockCommit(height int64) *tmtypes.TendermintCommit {
	blockInfo, err := client.QueryBlockInfoByHeight(height)
	if err != nil {
		panic(fmt.Sprintf("LoadBlockCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		tendermintlog.Error("LoadBlockCommit get nil block info")
		return nil
	}
	return blockInfo.GetLastCommit()
}

func (client *TendermintClient) LoadProposalBlock(height int64) *tmtypes.TendermintBlock {
	block, err := client.RequestBlock(height)
	if err != nil {
		tendermintlog.Error("LoadProposal by height failed", "curHeight", client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := client.QueryBlockInfoByHeight(height)
	if err != nil {
		panic(fmt.Sprintf("LoadProposal GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		tendermintlog.Error("LoadProposal get nil block info")
		return nil
	}

	proposalBlock := blockInfo.GetBlock()
	if proposalBlock != nil {
		proposalBlock.Txs = append(proposalBlock.Txs, block.Txs[1:]...)
		txHash := merkle.CalcMerkleRoot(proposalBlock.Txs)
		tendermintlog.Debug("LoadProposalBlock txs hash", "height", proposalBlock.Header.Height, "tx-hash", fmt.Sprintf("%X", txHash))
	}
	return proposalBlock
}
