package tendermint

import (
	log "github.com/inconshreveable/log15"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	ttypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	sm "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/state"
	"time"
	//ttypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	"code.aliyun.com/chain33/chain33/queue"
	"fmt"
	"sync"
	"net"
	"code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/p2p"
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

// ProposalMessage is sent when a new block is proposed.
type ProposalMessage struct {
	Proposal *ttypes.Proposal
}

type TendermintClient struct{
	//config
	*drivers.BaseClient
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
	ChainID string
	genesisDoc    *ttypes.GenesisDoc   // initial validator set
	privValidator ttypes.PrivValidator

	// internal state
	mtx sync.Mutex
	ttypes.RoundState
	state sm.State

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
/*
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
*/
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

	state.LastBlockHeight = c.GetInitHeight()

	client := &TendermintClient{
		ChainID:          genesisDoc.ChainID,
		BaseClient:       c,
		genesisDoc :      genesisDoc,
		privValidator :   privValidator,

		peerMsgQueue:     make(chan msgInfo, msgQueueSize),
		internalMsgQueue: make(chan msgInfo, msgQueueSize),
		timeoutTicker:    NewTimeoutTicker(),

		done:             make(chan struct{}),

		state:            state.Copy(),
	}
	client.decideProposal = client.defaultDecideProposal
	client.doPrevote = client.defaultDoPrevote
	client.setProposal = client.defaultSetProposal
	c.SetChild(client)
	return client
}

func (client *TendermintClient) Close() {
	log.Info("consensus tendermint closed")
}

func (client *TendermintClient) SetQueue(q *queue.Queue) {
	client.InitClient(q, func() {
		//call init block
		client.InitBlock()
	})
	go client.EventLoop()
	//go client.child.CreateBlock()
}

func (client *TendermintClient) InitBlock(){
	height := client.GetInitHeight();
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

// send a msg into the receiveRoutine regarding our own proposal, block part, or vote
func (cs *TendermintClient) sendInternalMessage(mi msgInfo) {
	select {
	case cs.internalMsgQueue <- mi:
	default:
		// NOTE: using the go-routine means our votes can
		// be processed out of order.
		// TODO: use CList here for strict determinism and
		// attempt push to internalMsgQueue in receiveRoutine
		log.Info("Internal msg queue is full. Using a go-routine")
		go func() { cs.internalMsgQueue <- mi }()
	}
}

func (tc *TendermintClient) defaultDecideProposal(height int64, round int) {
	var block *ttypes.Block
	var blockParts *ttypes.PartSet

	// Decide on block
	if tc.LockedBlock != nil {
		// If we're locked onto a block, just choose that.
		block, blockParts = tc.LockedBlock, tc.LockedBlockParts
	} else {
		// Create a new proposal block from state/txs from the mempool.
		block, blockParts = tc.createProposalBlock()
		if block == nil { // on error
			return
		}
	}

	// Make proposal
	polRound, polBlockID := tc.Votes.POLInfo()
	proposal := ttypes.NewProposal(height, round, blockParts.Header(), polRound, polBlockID)
	if err := tc.privValidator.SignProposal(tc.ChainID, proposal); err == nil {
		// Set fields
		/*  fields set by setProposal and addBlockPart
		tc.Proposal = proposal
		tc.ProposalBlock = block
		tc.ProposalBlockParts = blockParts
		*/

		// send proposal and block parts on internal msg queue
		tc.sendInternalMessage(msgInfo{&ProposalMessage{proposal}, ""})
		for i := 0; i < blockParts.Total(); i++ {
			part := blockParts.GetPart(i)
			tc.sendInternalMessage(msgInfo{&BlockPartMessage{tc.Height, tc.Round, part}, ""})
		}
		log.Info("Signed proposal", "height", height, "round", round, "proposal", proposal)
		log.Debug(cmn.Fmt("Signed proposal block: %v", block))
	} else {
		if !tc.replayMode {
			log.Error("enterPropose: Error signing proposal", "height", height, "round", round, "err", err)
		}
	}
}

// Create the next block to propose and return it.
// Returns nil block upon error.
// NOTE: keep it side-effect free for clarity.
func (cs *TendermintClient) createProposalBlock() (block *ttypes.Block, blockParts *ttypes.PartSet) {
	var commit *ttypes.Commit
	if cs.Height == 1 {
		// We're creating a proposal for the first block.
		// The commit is empty, but not nil.
		commit = &ttypes.Commit{}
	} else if cs.LastCommit.HasTwoThirdsMajority() {
		// Make the commit from LastCommit
		commit = cs.LastCommit.MakeCommit()
	} else {
		// This shouldn't happen.
		log.Error("enterPropose: Cannot propose anything: No commit for the previous block.")
		return
	}
/*
	// Mempool validated transactions
	txs := cs.mempool.Reap(cs.config.MaxBlockSizeTxs)
	block, parts := cs.state.MakeBlock(cs.Height, txs, commit)
	evidence := cs.evpool.PendingEvidence()
	block.AddEvidence(evidence)

	return block, parts
*/
}

func (cs *TendermintClient) defaultDoPrevote(height int64, round int) {
	//logger := cs.Logger.With("height", height, "round", round)
	// If a block is locked, prevote that.
	if cs.LockedBlock != nil {
		log.Info("enterPrevote: Block was locked")
		cs.signAddVote(types.VoteTypePrevote, cs.LockedBlock.Hash(), cs.LockedBlockParts.Header())
		return
	}

	// If ProposalBlock is nil, prevote nil.
	if cs.ProposalBlock == nil {
		log.Info("enterPrevote: ProposalBlock is nil")
		cs.signAddVote(types.VoteTypePrevote, nil, types.PartSetHeader{})
		return
	}

	// Validate proposal block
	err := cs.blockExec.ValidateBlock(cs.state, cs.ProposalBlock)
	if err != nil {
		// ProposalBlock is invalid, prevote nil.
		log.Error("enterPrevote: ProposalBlock is invalid", "err", err)
		cs.signAddVote(types.VoteTypePrevote, nil, types.PartSetHeader{})
		return
	}

	// Prevote cs.ProposalBlock
	// NOTE: the proposal signature is validated when it is received,
	// and the proposal block parts are validated as they are received (against the merkle hash in the proposal)
	log.Info("enterPrevote: ProposalBlock is valid")
	cs.signAddVote(types.VoteTypePrevote, cs.ProposalBlock.Hash(), cs.ProposalBlockParts.Header())
}

func (cs *TendermintClient) defaultSetProposal(proposal *ttypes.Proposal) error {
	// Already have one
	// TODO: possibly catch double proposals
	if cs.Proposal != nil {
		return nil
	}

	// Does not apply
	if proposal.Height != cs.Height || proposal.Round != cs.Round {
		return nil
	}

	// We don't care about the proposal if we're already in cstypes.RoundStepCommit.
	if ttypes.RoundStepCommit <= cs.Step {
		return nil
	}

	// Verify POLRound, which must be -1 or between 0 and proposal.Round exclusive.
	if proposal.POLRound != -1 &&
		(proposal.POLRound < 0 || proposal.Round <= proposal.POLRound) {
		return ErrInvalidProposalPOLRound
	}

	// Verify signature
	if !cs.Validators.GetProposer().PubKey.VerifyBytes(types.SignBytes(cs.state.ChainID, proposal), proposal.Signature) {
		return ErrInvalidProposalSignature
	}

	cs.Proposal = proposal
	cs.ProposalBlockParts = types.NewPartSetFromHeader(proposal.BlockPartsHeader)
	return nil
}

func (v *TendermintClient) ListenAndServe() {
	_, port, err := net.SplitHostPort(v.ListenAddr)
	if err != nil {
		log.Crit(err)
	}
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Crit("Listen error:", err)
	}

	log.Crit("Validator %s run on %s\n", myAddress, v.ListenAddr)

	for {
		c, err := l.Accept()
		if err != nil {
			log.Crit("Accept error:", err)
		}
		p := p2p.NewPeer(c)
		p.RegService(connectSvc, v)
		go p.Serve()
	}
}

func (v *TendermintClient) dialPeer(addr string) error {
	log.Info("dailPeer: ", addr)
	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Info(err.Error())
		return err
	}
	p := p2p.NewPeer(c)
	p.RegService(connectSvc, v)

	err = v.sendMsg(p, &ConnectMsg{Addr: v.ListenAddr, Height: v.lastBlock.Height, Ha: ConnectMsg_Hello}, connectSvc, true)
	if err != nil {
		log.Info(err.Error())
		return err
	}
	go p.Serve()
	return nil
}