package tendermint

import (
	log "github.com/inconshreveable/log15"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	ttypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	sm "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/state"

	"code.aliyun.com/chain33/chain33/queue"
	"sync"
	"net"
	"code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/p2p"
	"runtime/debug"
	core "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/core"
)

var tendermintlog = log.New("module", "tendermint")

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


// ProposalMessage is sent when a new block is proposed.
type ProposalMessage struct {
	Proposal *ttypes.Proposal
}

type TendermintClient struct{
	//config
	*drivers.BaseClient
	core core.ConsensusState
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

	// for tests where we want to limit the number of transitions the state makes
	nSteps int

	// state changes may be triggered by msgs from peers,
	// msgs from ourself, or by timeouts
	peerMsgQueue     chan msgInfo
	internalMsgQueue chan msgInfo
	timeoutTicker    sm.TimeoutTicker

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
		ChainID:          genesisDoc.ChainID,
		BaseClient:       c,
		genesisDoc :      genesisDoc,
		privValidator :   privValidator,

		peerMsgQueue:     make(chan msgInfo, msgQueueSize),
		internalMsgQueue: make(chan msgInfo, msgQueueSize),
		timeoutTicker:    sm.NewTimeoutTicker(),

		done:             make(chan struct{}),

		state:            state.Copy(),
	}
	client.decideProposal = client.defaultDecideProposal
	client.doPrevote = client.defaultDoPrevote
	client.setProposal = client.defaultSetProposal
	//client.updateToState(state)
	//client.reconstructLastCommit(state)
	c.SetChild(client)
	return client
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
	err := client.timeoutTicker.Start()
	if err != nil {
		tendermintlog.Error("TendermintClientSetQueue", "msg", "TimeoutTicker start failed", "error", err.Error())
		return
	}
	// we may have lost some votes if the process crashed
	// reload from consensus log to catchup
	/*20180224 hg first test, will add later
	if client.doWALCatchup {
		if err := tc.catchupReplay(tc.Height); err != nil {
			tc.Logger.Error("Error on catchup replay. Proceeding to start ConsensusState anyway", "err", err.Error())
			// NOTE: if we ever do return an error here,
			// make sure to stop the timeoutTicker
		}
	}
	*/
	// now start the receiveRoutine
	go client.receiveRoutine(0)

	// schedule the first round!
	// use GetRoundState so we don't race the receiveRoutine for access
	client.scheduleRound0(client.GetRoundState())

	go client.EventLoop()
	//go client.child.CreateBlock()
}

//-----------------------------------------
// the main go routines

// receiveRoutine handles messages which may cause state transitions.
// it's argument (n) is the number of messages to process before exiting - use 0 to run forever
// It keeps the RoundState and is the only thing that updates it.
// Updates (state transitions) happen on timeouts, complete proposals, and 2/3 majorities.
// ConsensusState must be locked before any internal state is updated.
func (tc *TendermintClient) receiveRoutine(maxSteps int) {
	defer func() {
		if r := recover(); r != nil {
			tendermintlog.Error("TendermintClient-receiveRoutine", "msg", "CONSENSUS FAILURE!!!", "err", r, "stack", string(debug.Stack()))
		}
	}()

	for {
		if maxSteps > 0 {
			if tc.nSteps >= maxSteps {
				tendermintlog.Info("TendermintClient-receiveRoutine", "msg", "reached max steps. exiting receive routine")
				tc.nSteps = 0
				return
			}
		}
		rs := tc.RoundState
		var mi msgInfo

		select {
		case height := tc.GetCurrentHeight():
			tc.handleTxsAvailable(height)
		case mi = <-tc.peerMsgQueue:
			tc.wal.Save(mi)
			// handles proposals, block parts, votes
			// may generate internal events (votes, complete proposals, 2/3 majorities)
			tc.handleMsg(mi)
		case mi = <-tc.internalMsgQueue:
			tc.wal.Save(mi)
			// handles proposals, block parts, votes
			tc.handleMsg(mi)
		case ti := <-tc.timeoutTicker.Chan(): // tockChan:
			tc.wal.Save(ti)
			// if the timeout is relevant to the rs
			// go to the next step
			tc.handleTimeout(ti, rs)
		case <-tc.Quit:

			// NOTE: the internalMsgQueue may have signed messages from our
			// priv_val that haven't hit the WAL, but its ok because
			// priv_val tracks LastSig

			// close wal now that we're done writing to it
			tc.wal.Stop()

			close(tc.done)
			return
		}
	}
}

func (tc *TendermintClient) handleTxsAvailable(height int64) {
	tc.mtx.Lock()
	defer tc.mtx.Unlock()
	// we only need to do this for round 0
	tc.enterPropose(height, 0)
}

// Enter (CreateEmptyBlocks): from enterNewRound(height,round)
// Enter (CreateEmptyBlocks, CreateEmptyBlocksInterval > 0 ): after enterNewRound(height,round), after timeout of CreateEmptyBlocksInterval
// Enter (!CreateEmptyBlocks) : after enterNewRound(height,round), once txs are in the mempool
func (tc *TendermintClient) enterPropose(height int64, round int) {
	if tc.Height != height || round < tc.Round || (tc.Round == round && ttypes.RoundStepPropose <= tc.Step) {
		tendermintlog.Debug("enterPropose", "msg", "Invalid args.", "height", height, "round", round, "Current step height", tc.Height, "Current step round", tc.Round, "Current step step", tc.Step)
		return
	}
	tendermintlog.Info("enterPropose", "height", height, "round", round, "step", tc.Step)

	defer func() {
		// Done enterPropose:
		tc.updateRoundStep(round, ttypes.RoundStepPropose)
		tc.newStep()

		// If we have the whole proposal + POL, then goto Prevote now.
		// else, we'll enterPrevote when the rest of the proposal is received (in AddProposalBlockPart),
		// or else after timeoutPropose
		if tc.isProposalComplete() {
			tc.enterPrevote(height, tc.Round)
		}
	}()

	// If we don't get the proposal and all block parts quick enough, enterPrevote
	tc.scheduleTimeout(tc.config.Propose(round), height, round, cstypes.RoundStepPropose)

	// Nothing more to do if we're not a validator
	if tc.privValidator == nil {
		tc.Logger.Debug("This node is not a validator")
		return
	}

	if !tc.isProposer() {
		tc.Logger.Info("enterPropose: Not our turn to propose", "proposer", tc.Validators.GetProposer().Address, "privValidator", tc.privValidator)
		if tc.Validators.HasAddress(tc.privValidator.GetAddress()) {
			tc.Logger.Debug("This node is a validator")
		} else {
			tc.Logger.Debug("This node is not a validator")
		}
	} else {
		tc.Logger.Info("enterPropose: Our turn to propose", "proposer", tc.Validators.GetProposer().Address, "privValidator", tc.privValidator)
		tc.Logger.Debug("This node is a validator")
		tc.decideProposal(height, round)
	}
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

// send a msg into the receiveRoutine regarding our own proposal, block part, or vote
func (tc *TendermintClient) sendInternalMessage(mi msgInfo) {
	select {
	case tc.internalMsgQueue <- mi:
	default:
		// NOTE: using the go-routine means our votes can
		// be processed out of order.
		// TODO: use CList here for strict determinism and
		// attempt push to internalMsgQueue in receiveRoutine
		tendermintlog.Info("Internal msg queue is full. Using a go-routine")
		go func() { tc.internalMsgQueue <- mi }()
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
		tendermintlog.Info("Signed proposal", "height", height, "round", round, "proposal", proposal)
		//tendermintlog.Debug(fmt.Printf("Signed proposal block: %v", block))
	} else {
		if !tc.replayMode {
			tendermintlog.Error("enterPropose: Error signing proposal", "height", height, "round", round, "err", err)
		}
	}
}

// Create the next block to propose and return it.
// Returns nil block upon error.
// NOTE: keep it side-effect free for clarity.
func (tc *TendermintClient) createProposalBlock() (block *ttypes.Block, blockParts *ttypes.PartSet) {
	var commit *ttypes.Commit
	if tc.Height == 1 {
		// We're creating a proposal for the first block.
		// The commit is empty, but not nil.
		commit = &ttypes.Commit{}
	} else if tc.LastCommit.HasTwoThirdsMajority() {
		// Make the commit from LastCommit
		commit = tc.LastCommit.MakeCommit()
	} else {
		// This shouldn't happen.
		tendermintlog.Error("enterPropose: Cannot propose anything: No commit for the previous block.")
		return
	}
/*
	// Mempool validated transactions
	txs := tc.mempool.Reap(tc.config.MaxBlockSizeTxs)
	block, parts := tc.state.MakeBlock(tc.Height, txs, commit)
	evidence := tc.evpool.PendingEvidence()
	block.AddEvidence(evidence)

	return block, parts
*/
}

func (tc *TendermintClient) defaultDoPrevote(height int64, round int) {
	//logger := tc.Logger.With("height", height, "round", round)
	// If a block is locked, prevote that.
	if tc.LockedBlock != nil {
		tendermintlog.Info("enterPrevote: Block was locked")
		tc.signAddVote(types.VoteTypePrevote, tc.LockedBlock.Hash(), tc.LockedBlockParts.Header())
		return
	}

	// If ProposalBlock is nil, prevote nil.
	if tc.ProposalBlock == nil {
		tendermintlog.Info("enterPrevote: ProposalBlock is nil")
		tc.signAddVote(types.VoteTypePrevote, nil, types.PartSetHeader{})
		return
	}

	// Validate proposal block
	err := tc.blockExec.ValidateBlock(tc.state, tc.ProposalBlock)
	if err != nil {
		// ProposalBlock is invalid, prevote nil.
		tendermintlog.Error("enterPrevote: ProposalBlock is invalid", "err", err)
		tc.signAddVote(types.VoteTypePrevote, nil, types.PartSetHeader{})
		return
	}

	// Prevote tc.ProposalBlock
	// NOTE: the proposal signature is validated when it is received,
	// and the proposal block parts are validated as they are received (against the merkle hash in the proposal)
	tendermintlog.Info("enterPrevote: ProposalBlock is valid")
	tc.signAddVote(types.VoteTypePrevote, tc.ProposalBlock.Hash(), tc.ProposalBlockParts.Header())
}

func (tc *TendermintClient) defaultSetProposal(proposal *ttypes.Proposal) error {
	// Already have one
	// TODO: possibly catch double proposals
	if tc.Proposal != nil {
		return nil
	}

	// Does not apply
	if proposal.Height != tc.Height || proposal.Round != tc.Round {
		return nil
	}

	// We don't care about the proposal if we're already in cstypes.RoundStepCommit.
	if ttypes.RoundStepCommit <= tc.Step {
		return nil
	}

	// Verify POLRound, which must be -1 or between 0 and proposal.Round exclusive.
	if proposal.POLRound != -1 &&
		(proposal.POLRound < 0 || proposal.Round <= proposal.POLRound) {
		return ErrInvalidProposalPOLRound
	}

	// Verify signature
	if !tc.Validators.GetProposer().PubKey.VerifyBytes(types.SignBytes(tc.state.ChainID, proposal), proposal.Signature) {
		return ErrInvalidProposalSignature
	}

	tc.Proposal = proposal
	tc.ProposalBlockParts = types.NewPartSetFromHeader(proposal.BlockPartsHeader)
	return nil
}

func (v *TendermintClient) ListenAndServe() {
	_, port, err := net.SplitHostPort(v.ListenAddr)
	if err != nil {
		tendermintlog.Crit(err.Error())
	}
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		tendermintlog.Crit("Listen error:", err)
	}

	tendermintlog.Crit("Validator %s run on %s\n", myAddress, v.ListenAddr)

	for {
		c, err := l.Accept()
		if err != nil {
			tendermintlog.Crit("Accept error:", err)
		}
		p := p2p.NewPeer(c)
		p.RegService(connectSvc, v)
		go p.Serve()
	}
}

func (v *TendermintClient) dialPeer(addr string) error {
	tendermintlog.Info("dailPeer: ", addr)
	c, err := net.Dial("tcp", addr)
	if err != nil {
		tendermintlog.Info(err.Error())
		return err
	}
	p := p2p.NewPeer(c)
	p.RegService(connectSvc, v)

	err = v.sendMsg(p, &ConnectMsg{Addr: v.ListenAddr, Height: v.lastBlock.Height, Ha: ConnectMsg_Hello}, connectSvc, true)
	if err != nil {
		tendermintlog.Info(err.Error())
		return err
	}
	go p.Serve()
	return nil
}