package tendermint

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	ttypes "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	sm "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/state"

	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/core"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/p2p"
	crypto "github.com/tendermint/go-crypto"
	"bytes"
	"time"
	"encoding/json"
	"fmt"
	"errors"
	"net"
	"strings"
	"math/rand"
	wire "github.com/tendermint/go-wire"
	"sync"
	"gitlab.33.cn/chain33/chain33/util"
	bc "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/blockchain"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/evidence"
)

var (
	tendermintlog = log.New("module", "tendermint")
	genesisDocKey = []byte("genesisDoc")

	listSize     int = 10000
	zeroHash     [32]byte
	currentBlock *types.Block
	mulock       sync.Mutex

)

type TendermintClient struct{
	//config
	*drivers.BaseClient
	genesisDoc    *ttypes.GenesisDoc   // initial validator set
	privValidator ttypes.PrivValidator
	csState       *core.ConsensusState
	eventBus      *ttypes.EventBus
	privKey       crypto.PrivKeyEd25519   // local node's p2p key
	sw            *p2p.Switch
	state         sm.State
	blockStore    *core.BlockStore
}

// DefaultDBProvider returns a database using the DBBackend and DBDir
// specified in the ctx.Config.
func DefaultDBProvider(ID string) (dbm.DB, error) {
	return dbm.NewDB(ID, "leveldb", "./datadir", 0), nil
}

// panics if failed to unmarshal bytes
func loadGenesisDoc(db dbm.DB) (*ttypes.GenesisDoc, error) {
	bytes := db.Get(genesisDocKey)
	if len(bytes) == 0 {
		return nil, errors.New("Genesis doc not found")
	} else {
		var genDoc *ttypes.GenesisDoc
		err := json.Unmarshal(bytes, &genDoc)
		if err != nil {
			panic(fmt.Sprintf("Panicked on a Crisis: %v",fmt.Sprintf("Failed to load genesis doc due to unmarshaling error: %v (bytes: %X)", err, bytes)))
		}
		return genDoc, nil
	}
}

// panics if failed to marshal the given genesis document
func saveGenesisDoc(db dbm.DB, genDoc *ttypes.GenesisDoc) {
	bytes, err := json.Marshal(genDoc)
	if err != nil {
		panic(fmt.Sprintf("Panicked on a Crisis: %v",fmt.Sprintf("Failed to save genesis doc due to marshaling error: %v", err)))
	}
	db.SetSync(genesisDocKey, bytes)
}

func New(cfg *types.Consensus) *TendermintClient {
	tendermintlog.Info("Start to create tendermint client")

	//store block
	blockStoreDB, err := DefaultDBProvider("CSblockstore")
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "msg", "DefaultDBProvider blockstore failded", "error", err)
		return nil
	}
	blockStore := core.NewBlockStore(blockStoreDB)

	//store State
	stateDB, err := DefaultDBProvider("CSstate")
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "msg", "DefaultDBProvider state failded", "error", err)
		return nil
	}

	genDoc, err := loadGenesisDoc(stateDB)
	if err != nil {
		genDoc, err = ttypes.GenesisDocFromFile("./genesis.json")
		if err != nil {
			tendermintlog.Error("NewTendermintClient", "msg", "GenesisDocFromFile failded", "error", err)
			return nil
		}
		// save genesis doc to prevent a certain class of user errors (e.g. when it
		// was changed, accidentally or not). Also good for audit trail.
		saveGenesisDoc(stateDB, genDoc)
	}

	state, err := sm.LoadStateFromDBOrGenesisDoc(stateDB, genDoc)
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "msg", "LoadStateFromDBOrGenesisDoc failded", "error", err)
		return nil
	}

	// Generate node PrivKey
	privKey := crypto.GenPrivKeyEd25519()

	privValidator := ttypes.LoadOrGenPrivValidatorFS("./priv_validator.json")
	if privValidator == nil{
		tendermintlog.Info("NewTendermintClient","msg", "priv_validator file missing, create new one")
		var privVal *ttypes.PrivValidatorFS
		privVal = ttypes.GenPrivValidatorFS(".")
		privVal.Save()
		privValidator = privVal
		//return nil
	}

	fastSync := true
	if state.Validators.Size() == 1 {
		addr, _ := state.Validators.GetByIndex(0)
		if bytes.Equal(privValidator.GetAddress(), addr) {
			fastSync = false
		}
	}

	// Log whether this node is a validator or an observer
	if state.Validators.HasAddress(privValidator.GetAddress()) {
		tendermintlog.Info("This node is a validator")
	} else {
		tendermintlog.Info("This node is not a validator")
	}

	// Make Evidence Reactor
	evidenceDB, err := DefaultDBProvider("CSevidence")
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "msg", "DefaultDBProvider evidenceDB failded", "error", err)
		return nil
	}

	//make evidenceReactor
	evidenceLogger := log.New("module", "tendermint-evidence")
	evidenceStore := evidence.NewEvidenceStore(evidenceDB)
	evidencePool := evidence.NewEvidencePool(stateDB, evidenceStore)
	evidenceReactor := evidence.NewEvidenceReactor(evidencePool)
	evidenceReactor.SetLogger(evidenceLogger)

	blockExecLogger := log.New("module", "tendermint-state")
	// make block executor for consensus and blockchain reactors to execute blocks
	blockExec := sm.NewBlockExecutor(stateDB, blockExecLogger, evidencePool)

	// Make BlockchainReactor
	bcReactor := bc.NewBlockchainReactor(state.Copy(), blockExec, blockStore, fastSync)
	bcReactor.SetLogger(log.New("module", "tendermint-blockchain"))

	c := drivers.NewBaseClient(cfg)

	// Make ConsensusReactor
	csState := core.NewConsensusState(c, state.Copy(), blockStore, blockExec, evidencePool)
	csState.SetPrivValidator(privValidator)

	consensusReactor := core.NewConsensusReactor(csState, fastSync)
	consensusReactor.SetLogger(tendermintlog)

	p2pLogger := log.New("module", "tendermint-p2p")
	sw := p2p.NewSwitch(p2p.DefaultP2PConfig())
	sw.SetLogger(p2pLogger)
	sw.AddReactor("CONSENSUS", consensusReactor)
	sw.AddReactor("BLOCKCHAIN", bcReactor)
	sw.AddReactor("EVIDENCE", evidenceReactor)

	eventBus := ttypes.NewEventBus()
	// services which will be publishing and/or subscribing for messages (events)
	// consensusReactor will set it on consensusState and blockExecutor
	consensusReactor.SetEventBus(eventBus)

	client := &TendermintClient{
		BaseClient:       c,
		genesisDoc :      genDoc,
		privValidator :   privValidator,
		privKey:          privKey,
		state:            state,
		blockStore:       blockStore,
		sw:               sw,
		csState:          csState,
		eventBus:         eventBus,
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
	tendermintlog.Info("TendermintClientClose","consensus tendermint closed")
}

func (client *TendermintClient) SetQueueClient(q queue.Client) {
	client.InitClient(q, func() {
		//call init block
		client.InitBlock()
	})
	//event start
	err := client.eventBus.Start()
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

	go client.EventLoop()
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

func (client *TendermintClient) ProcEvent(msg queue.Message) {

}

func (client *TendermintClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error) {
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

func (client *TendermintClient) CreateBlock() {
	issleep := true
	for {

		if issleep {
			time.Sleep(time.Second)
		}

		lastBlock := client.GetCurrentBlock()
		txs := client.RequestTx(int(types.GetP(lastBlock.Height + 1).MaxTxNumber)-1, nil)
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
		tendermintlog.Info("get mempool txs not empty")
		client.csState.NewTxsAvailable(lastBlock.Height)
		tendermintlog.Info("waiting NewTxsFinished")
		select {
		case finish := <- client.csState.NewTxsFinished :
			tendermintlog.Info("TendermintClientSetQueue", "msg", "new txs finish dealing", "result", finish)
			continue
		}
	}
}

func (client *TendermintClient) checkValidators() bool {
	if client.sw.Peers().Size() > 0{
		return true
	} else {
		return false
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
		PubKey:  client.privKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker: "test_"+fmt.Sprintf("%v",rand.Intn(100)),
		Network: client.state.ChainID,
		Version: "v0.1.0",
		Other: []string{
			fmt.Sprintf("wire_version=%v", wire.Version),
			fmt.Sprintf("p2p_version=%v", p2p.Version),
		},
	}

	nodeInfo.ListenAddr = GetPulicIPInUse() + ":36656"

	return nodeInfo
}