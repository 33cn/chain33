package tendermint

import (
	log "github.com/inconshreveable/log15"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	ttypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	sm "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/state"
	"time"
	"fmt"
	//ttypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	"code.aliyun.com/chain33/chain33/queue"
)

var tlog = log.New("module", "tendermint")

// ConsensusMessage is a message that can be sent and received on the ConsensusReactor
type ConsensusMessage interface{}

var (
	msgQueueSize = 1000
)

// msgs from the reactor which may update the state
type msgInfo struct {
	Msg     ConsensusMessage `json:"msg"`
	PeerKey string           `json:"peer_key"`
}

// internally generated messages which may update the state
type timeoutInfo struct {
	Duration time.Duration         `json:"duration"`
	Height   int64                 `json:"height"`
	Round    int                   `json:"round"`
	Step     ttypes.RoundStepType `json:"step"`
}

func (ti *timeoutInfo) String() string {
	return fmt.Sprintf("%v ; %d/%d %v", ti.Duration, ti.Height, ti.Round/*, ti.Step*/)
}

type TendermintClient struct{
	//config
	baseConfig *drivers.BaseClient
	/*
	// All timeouts are in ms
	TimeoutPropose int
	TimeoutProposeDelta int
	TimeoutPrevote int
	TimeoutPrevoteDelta int
	TimeoutPrecommit int
	TimeoutPrecommitDelta int
	TimeoutCommit int
	// Make progress as soon as we have all the precommits (as if TimeoutCommit = 0)
	SkipTimeoutCommit bool

	// EmptyBlocks mode and possible interval between empty blocks in seconds
	// whether the consensus should wait for transactions before entering the propose step
	CreateEmptyBlocks         bool
	CreateEmptyBlocksInterval int
*/
	genesisDoc    *ttypes.GenesisDoc   // initial validator set
	privValidator ttypes.PrivValidator

	// state changes may be triggered by msgs from peers,
	// msgs from ourself, or by timeouts
	peerMsgQueue     chan msgInfo
	internalMsgQueue chan msgInfo
	timeoutTicker    TimeoutTicker

	// some functions can be overwritten for testing
	decideProposal func(height int64, round int)
	doPrevote      func(height int64, round int)
	setProposal    func(proposal *ttypes.Proposal) error

	// closed when we finish shutting down
	done chan struct{}

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight  int64
	LastBlockTotalTx int64
	LastBlockID      ttypes.BlockID
	LastBlockTime    time.Time

	// LastValidators is used to validate block.LastCommit.
	// Validators are persisted to the database separately every time they change,
	// so we can query for historical validator sets.
	// Note that if s.LastBlockHeight causes a valset change,
	// we set s.LastHeightValidatorsChanged = s.LastBlockHeight + 1
	Validators                  *ttypes.ValidatorSet
	LastValidators              *ttypes.ValidatorSet
	LastHeightValidatorsChanged int64
}

func New(cfg *types.Consensus) *TendermintClient {
	tlog.Info("Start to create raft cluster")

	genesisDoc, err := ttypes.GenesisDocFromFile("./genesis.json")
	if err != nil{
		tlog.Info(err.Error())
		return nil
	}

	privValidator := ttypes.LoadOrGenPrivValidatorFS("./priv_validator.json")
	if privValidator == nil{
		//return nil
		tlog.Info("priv_validator file missing")
	}

	state, err := sm.MakeGenesisState(genesisDoc)
	if err != nil {
		tlog.Info("MakeGenesisState failed ")
		return nil
	}

	c := drivers.NewBaseClient(cfg)

	client := &TendermintClient{
		baseConfig :      c,
		genesisDoc :      genesisDoc,
		privValidator :   privValidator,
		peerMsgQueue:     make(chan msgInfo, msgQueueSize),
		internalMsgQueue: make(chan msgInfo, msgQueueSize),
		timeoutTicker:    NewTimeoutTicker(),
		done:             make(chan struct{}),
		Validators:       state.Validators.Copy(),

	}
	c.SetChild(client)
	return client
}

func (client *TendermintClient) Close() {
	log.Info("consensus tendermint closed")
}

func (client *TendermintClient) SetQueue(q *queue.Queue) {
	client.baseConfig.InitClient(q, func() {
		//call init block
		client.baseConfig.InitBlock()
	})
	go client.baseConfig.EventLoop()
	//go client.child.CreateBlock()
}


func (client *TendermintClient) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = client.baseConfig.Cfg.Genesis
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
		if !client.baseConfig.IsMining() {
			time.Sleep(time.Second)
			continue
		}
		if issleep {
			time.Sleep(time.Second)
		}
		txs := client.baseConfig.RequestTx()
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