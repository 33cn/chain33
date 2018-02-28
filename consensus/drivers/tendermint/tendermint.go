package tendermint

import (
	log "github.com/inconshreveable/log15"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	ttypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	sm "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/state"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/core"
	"math/rand"
	"fmt"
)

var tendermintlog = log.New("module", "tendermint")

type TendermintClient struct{
	//config
	*drivers.BaseClient
	genesisDoc    *ttypes.GenesisDoc   // initial validator set
	privValidator ttypes.PrivValidator
	csState       *core.ConsensusState
	csReactor     *core.ConsensusReactor
	ListenAddr    string
	Moniker       string  //node name
}

func New(cfg *types.Consensus) *TendermintClient {
	tendermintlog.Info("Start to create raft cluster")

	genesisDoc, err := ttypes.GenesisDocFromFile("./genesis.json")
	if err != nil{
		tendermintlog.Info(err.Error())
		return nil
	}

	privValidator := ttypes.LoadOrGenPrivValidatorFS("./priv_validator.json")
	if privValidator == nil{
		//return nil
		tendermintlog.Info("NewTendermintClient","msg", "priv_validator file missing")
	}

	state, err := sm.MakeGenesisState(genesisDoc)
	if err != nil {
		tendermintlog.Error("NewTendermintClient","msg", "MakeGenesisState failed ")
		return nil
	}

	c := drivers.NewBaseClient(cfg)

	state.LastBlockHeight = c.GetInitHeight()



	client := &TendermintClient{
		BaseClient:       c,
		genesisDoc :      genesisDoc,
		privValidator :   privValidator,
		ListenAddr:       "0.0.0.0:36656",
		Moniker:          "test_"+fmt.Sprintf("%v",rand.Intn(100)),
	}
	csState := core.NewConsensusState(client, state)

	fastSync := true
	consensusReactor := core.NewConsensusReactor(csState, fastSync)

	eventBus := ttypes.NewEventBus()
	// services which will be publishing and/or subscribing for messages (events)
	// consensusReactor will set it on consensusState and blockExecutor
	consensusReactor.SetEventBus(eventBus)
	client.csState = csState
	client.csReactor = consensusReactor
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
	// we need the timeoutRoutine for replay so
	// we don't block on the tick chan.
	// NOTE: we will get a build up of garbage go routines
	// firing on the tockChan until the receiveRoutine is started
	// to deal with them (by that point, at most one will be valid)
	err := client.csState.Start()
	if err != nil {
		tendermintlog.Error("TendermintClientSetQueue", "msg", "TimeoutTicker start failed", "error", err.Error())
		return
	}

	err = client.csReactor.Start()
	if err != nil {
		tendermintlog.Error("TendermintClientSetQueue", "msg", "TimeoutTicker start failed", "error", err.Error())
		return
	}

	go client.EventLoop()
	//go client.child.CreateBlock()
}

func (client *TendermintClient) InitBlock(){
	height := client.GetInitHeight()
	block, err := client.RequestBlock(height)
	if err != nil {
		panic(err)
	}
	client.SetCurrentBlock(block)
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

