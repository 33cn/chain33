package tendermint

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	sm "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/state"
	ttypes "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"gitlab.33.cn/chain33/chain33/types"

	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/core"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/evidence"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/util"
)

var (
	tendermintlog = log.New("module", "tendermint")
	genesisDocKey = []byte("genesisDoc")
)

type TendermintClient struct {
	//config
	*drivers.BaseClient
	genesisDoc    *ttypes.GenesisDoc // initial validator set
	privValidator ttypes.PrivValidator
	eventBus     *ttypes.EventBus
	privKey      crypto.PrivKey // local node's p2p key
	sw           *p2p.Switch
	state        sm.State
	csState      *core.ConsensusState
	blockStore   *ttypes.BlockStore
	evidenceDB   dbm.DB
	crypto       crypto.Crypto
}

// DefaultDBProvider returns a database using the DBBackend and DBDir
// specified in the ctx.Config.
func DefaultDBProvider(ID string) (dbm.DB, error) {
	return dbm.NewDB(ID, "leveldb", "./datadir", 0), nil
}

func New(cfg *types.Consensus) *TendermintClient {
	tendermintlog.Info("Start to create tendermint client")

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

	p2pLogger := log.New("module", "tendermint-p2p")
	sw := p2p.NewSwitch(p2p.DefaultP2PConfig())
	sw.SetLogger(p2pLogger)

	eventBus := ttypes.NewEventBus()

	c := drivers.NewBaseClient(cfg)

	blockStore := ttypes.NewBlockStore(c, pubkey)

	client := &TendermintClient{
		BaseClient:    c,
		genesisDoc:    genDoc,
		privValidator: privValidator,
		privKey:       priv,
		sw:            sw,
		eventBus:      eventBus,
		blockStore:    blockStore,
		evidenceDB:    evidenceDB,
		crypto:        cr,
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

func (client *TendermintClient) StartConsensus() {
	//caught up
	for {
		if !client.IsCaughtUp() {
			time.Sleep(time.Second)
			continue
		}
		break
	}

	block := client.GetCurrentBlock()
	if block == nil {
		tendermintlog.Error("StartConsensus failed for current block is nil")
		panic("StartConsensus failed for current block is nil")
	}

	blockInfo, err := ttypes.GetBlockInfo(block)
	if err != nil {
		tendermintlog.Error("StartConsensus GetBlockInfo failed", "error", err)
		panic(fmt.Sprintf("StartConsensus GetBlockInfo failed:%v",err))
	}

	var state sm.State
	if blockInfo == nil {
		if block.Height != 0 {
			tendermintlog.Error("StartConsensus", "msg", "block height is not 0 but blockinfo is nil")
			panic(fmt.Sprintf("StartConsensus block height is %v but block info is nil", block.Height))
		}
		statetmp, err := sm.MakeGenesisState(client.genesisDoc)
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
		state = sm.LoadState(csState)
		if seenCommit := blockInfo.SeenCommit; seenCommit != nil {
			state.LastBlockID = ttypes.BlockID{
				Hash: seenCommit.BlockID.GetHash(),
				PartsHeader: ttypes.PartSetHeader{
					Total: int(seenCommit.BlockID.GetPartsHeader().Total),
					Hash:  seenCommit.BlockID.GetPartsHeader().Hash,
				},
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

	stateDB := sm.NewStateDB(client.BaseClient, state)

	//make evidenceReactor
	evidenceLogger := log.New("module", "tendermint-evidence")
	evidenceStore := evidence.NewEvidenceStore(client.evidenceDB)
	evidencePool := evidence.NewEvidencePool(stateDB, state, evidenceStore)
	evidenceReactor := evidence.NewEvidenceReactor(evidencePool)
	evidenceReactor.SetLogger(evidenceLogger)

	blockExecLogger := log.New("module", "tendermint-state")
	// make block executor for consensus and blockchain reactors to execute blocks
	blockExec := sm.NewBlockExecutor(stateDB, blockExecLogger, evidencePool)

	// Make ConsensusReactor
	csState := core.NewConsensusState(client.BaseClient, client.blockStore, state, blockExec, evidencePool)
	csState.SetPrivValidator(client.privValidator)

	consensusReactor := core.NewConsensusReactor(csState, false)
	consensusReactor.SetLogger(tendermintlog)
	// services which will be publishing and/or subscribing for messages (events)
	// consensusReactor will set it on consensusState and blockExecutor
	consensusReactor.SetEventBus(client.eventBus)

	client.sw.AddReactor("EVIDENCE", evidenceReactor)

	client.sw.AddReactor("CONSENSUS", consensusReactor)

	client.csState = csState

	//event start
	err = client.eventBus.Start()
	if err != nil {
		tendermintlog.Error("TendermintClientSetQueue", "msg", "EventBus start failed", "error", err)
		return
	}
	// Create & add listener
	protocol, address := "tcp", "0.0.0.0:46656"
	l := p2p.NewDefaultListener(protocol, address, false, log.New("module", "tendermint-p2p"))
	client.sw.AddListener(l)

	// Start the switch
	client.sw.SetNodeInfo(client.MakeDefaultNodeInfo())
	client.sw.SetNodePrivKey(client.privKey)
	err = client.sw.Start()
	if err != nil {
		tendermintlog.Error("TendermintClientSetQueue", "msg", "switch start failed", "error", err)
		return
	}

	// If seeds exist, add them to the address book and dial out
	if len(client.Cfg.Seeds) != 0 {
		// dial out
		client.sw.DialSeeds(nil, client.Cfg.Seeds)
	}

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
			tendermintlog.Info("consensus not running now")
			time.Sleep(time.Second)
			continue
		}

		if issleep {
			time.Sleep(time.Second)
		}

		lastBlock := client.GetCurrentBlock()
		//tendermintlog.Info("get last block","height", lastBlock.Height, "time", lastBlock.BlockTime,"txhash",lastBlock.TxHash)
		txs := client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber)-1, nil)
		//check dup
		txs = client.CheckTxDup(txs)

		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		/*
			var newblock types.Block
			newblock.ParentHash = lastBlock.Hash()
			newblock.Height = lastBlock.Height + 1
			newblock.Txs = txs
			newblock.StateHash = lastBlock.StateHash
			newblock.BlockTime = time.Now().Unix()
			if lastBlock.BlockTime >= newblock.BlockTime {
				newblock.BlockTime = lastBlock.BlockTime + 1
			}
		*/
		tendermintlog.Debug("get mempool txs not empty", "txslen", len(txs), "tx[0]", txs[0])
		client.csState.NewTxsAvailable(lastBlock.Height)
		tendermintlog.Debug("waiting NewTxsFinished")
		select {
		case finish := <-client.csState.NewTxsFinished:
			tendermintlog.Debug("TendermintClientSetQueue", "msg", "new txs finish dealing", "result", finish)
			continue
		}
	}
}

func GetPulicIPInUse() string {
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	localAddr := conn.LocalAddr().String()
	idx := strings.LastIndex(localAddr, ":")
	return localAddr[0:idx]
}

func (client *TendermintClient) MakeDefaultNodeInfo() *p2p.NodeInfo {
	nodeInfo := &p2p.NodeInfo{
		PubKey:  client.privKey.PubKey(),
		Moniker: "test_" + fmt.Sprintf("%v", rand.Intn(100)),
		Network: client.state.ChainID,
		Version: "v0.1.0",
		Other: []string{
			fmt.Sprintf("p2p_version=%v", p2p.Version),
		},
	}

	nodeInfo.ListenAddr = GetPulicIPInUse() + ":36656"

	return nodeInfo
}
