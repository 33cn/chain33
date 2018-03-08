package tendermint

import (
	log "github.com/inconshreveable/log15"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	ttypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	sm "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/state"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/core"
	dbm "github.com/tendermint/tmlibs/db"
	"code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/p2p"
	crypto "github.com/tendermint/go-crypto"
	tlog "github.com/tendermint/tmlibs/log"
	"bytes"
	"os"
	"time"
	"encoding/json"
	"fmt"
	"errors"
)

var (
	tendermintlog = log.New("module", "tendermint")
	genesisDocKey = []byte("genesisDoc")
)

type TendermintClient struct{
	//config
	*drivers.BaseClient
	genesisDoc    *ttypes.GenesisDoc   // initial validator set
	privValidator ttypes.PrivValidator
	csState       *core.ConsensusState
	csReactor     *core.ConsensusReactor
	eventBus      *ttypes.EventBus
	privKey       crypto.PrivKeyEd25519   // local node's p2p key
	sw            *p2p.Switch
	Logger        tlog.Logger
	state         sm.State
	blockStore    *core.BlockStore
	//ListenPort    string
	//Moniker       string  //node name
}

// DefaultDBProvider returns a database using the DBBackend and DBDir
// specified in the ctx.Config.
func DefaultDBProvider(ID string) (dbm.DB, error) {
	return dbm.NewDB(ID, "leveldb", "./datadir"), nil
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
	tendermintlog.Info("Start to create raft cluster")

	logger := tlog.NewTMLogger(tlog.NewSyncWriter(os.Stdout)).With("module", "consensus")

	//store block
	blockStoreDB, err := DefaultDBProvider("blockstore")
	if err != nil {
		tendermintlog.Error("NewTendermintClient", "msg", "DefaultDBProvider blockstore failded", "error", err)
		return nil
	}
	blockStore := core.NewBlockStore(blockStoreDB)

	//store State
	stateDB, err := DefaultDBProvider("state")
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

	privValidator := ttypes.LoadOrGenPrivValidatorFS("./priv_validator.json")
	if privValidator == nil{
		//return nil
		tendermintlog.Info("NewTendermintClient","msg", "priv_validator file missing")
	}

	// Generate node PrivKey
	privKey := crypto.GenPrivKeyEd25519()

	c := drivers.NewBaseClient(cfg)

	csState := core.NewConsensusState(c, state, blockStore)
	csState.SetPrivValidator(privValidator)

	fastSync := true
	if state.Validators.Size() == 1 {
		addr, _ := state.Validators.GetByIndex(0)
		if bytes.Equal(privValidator.GetAddress(), addr) {
			fastSync = false
		}
	}
	consensusReactor := core.NewConsensusReactor(csState, fastSync)

	sw := p2p.NewSwitch(p2p.DefaultP2PConfig())
	sw.AddReactor("CONSENSUS", consensusReactor)

	eventBus := ttypes.NewEventBus()
	// services which will be publishing and/or subscribing for messages (events)
	// consensusReactor will set it on consensusState and blockExecutor
	consensusReactor.SetEventBus(eventBus)


	client := &TendermintClient{
		BaseClient:       c,
		genesisDoc :      genDoc,
		privValidator :   privValidator,
		privKey:          privKey,
		Logger:           logger,
		state:            state,
		blockStore:       blockStore,
		sw:               sw,
		csState:          csState,
		csReactor:        consensusReactor,
		eventBus:         eventBus,
		//ListenPort:       "36656",
		//Moniker:          "test_"+fmt.Sprintf("%v",rand.Intn(100)),
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

func (client *TendermintClient) SetQueue(q *queue.Queue) {
	client.InitClient(q, func() {
		//call init block
		client.InitBlock()
	})

	go func() {
		for {
			txs, err:=client.GetMempoolTxs()
			if err != nil {
				tendermintlog.Error("TendermintClientSetQueue", "msg", "GetMempoolTxs failed", "error", err)
			} else if len(txs) != 0{
				txs = client.CheckTxDup(txs)
				lastBlock := client.GetCurrentBlock()
				if len(txs) != 0{
					//our chain index init -1, tendermint index init 0
					client.csState.NewTxsAvailable(lastBlock.Height + 1)
					select {
					case finish := <- client.csState.NewTxsFinished :
						if finish {
							continue
						} else {
							tendermintlog.Error("TendermintClientSetQueue", "msg", "GetMempoolTxs msg not finiish", "error", err)
						}

					}
				}
			}
			time.Sleep(1*time.Second)
		}
	}()
	//event start
	err := client.eventBus.Start()
	if err != nil {
		tendermintlog.Error("TendermintClientSetQueue", "msg", "EventBus start failed", "error", err)
		return
	}
	// Create & add listener
	protocol, address := "tcp", "0.0.0.0:46656"
	l := p2p.NewDefaultListener(protocol, address, false, client.Logger.With("module", "p2p"))
	client.sw.AddListener(l)

	// Start the switch
	client.sw.SetNodeInfo(client.csReactor.MakeDefaultNodeInfo())
	client.sw.SetNodePrivKey(client.privKey)
	err = client.sw.Start()
	if err != nil {
		tendermintlog.Error("TendermintClientSetQueue", "msg", "switch start failed", "error", err)
		return
	}

	go client.EventLoop()
	//go client.child.CreateBlock()
}

func (client *TendermintClient) InitBlock(){
	height := client.GetInitHeight()
	if height == -1 {
		// 创世区块
		/* do nothing, will use tendermint to write genesis block
		newblock := &types.Block{}
		newblock.Height = 0
		newblock.BlockTime = client.Cfg.GenesisBlockTime
		// TODO: 下面这些值在创世区块中赋值nil，是否合理？
		newblock.ParentHash = zeroHash[:]
		tx := client.child.CreateGenesisTx()
		newblock.Txs = tx
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		client.WriteBlock(zeroHash[:], newblock)
		*/
	} else {
		block, err := client.RequestBlock(height)
		if err != nil {
			panic(err)
		}
		client.SetCurrentBlock(block)
		client.state.LastBlockHeight = height
	}
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

func (client *TendermintClient) CreateBlock() {
	return
	/*
	issleep := true
	for {
		if !client.IsMining() {
			time.Sleep(time.Second)
			continue
		}
		if issleep {
			time.Sleep(time.Second)
		}
		txs := client.RequestTx()
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		//check dup
		txs = client.CheckTxDup(txs)
		lastBlock := client.GetCurrentBlock()
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		newblock.Txs = txs
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = time.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		if err != nil {
			issleep = true
			continue
		}
	}
	*/
}

