package tendermint

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	ttypes "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/merkle"
)

//-----------------------------------------------------------------------------
// Config

const (
	proposalHeartbeatIntervalSeconds = 2
)

//-----------------------------------------------------------------------------
// Errors

var (
	ErrInvalidProposalSignature = errors.New("Error invalid proposal signature")
	ErrInvalidProposalPOLRound  = errors.New("Error invalid proposal POL round")
	ErrAddingVote               = errors.New("Error adding vote")
	ErrVoteHeightMismatch       = errors.New("Error vote height mismatch")
)

//-----------------------------------------------------------------------------

var (
	msgQueueSize      = 1000
	maxCommitInfoSize = 101 //can deal with max 300 validator nodes
)

// internally generated messages which may update the state
type timeoutInfo struct {
	Duration time.Duration        `json:"duration"`
	Height   int64                `json:"height"`
	Round    int                  `json:"round"`
	Step     ttypes.RoundStepType `json:"step"`
}

func (ti *timeoutInfo) String() string {
	return fmt.Sprintf("%v ; %d/%d %v", ti.Duration, ti.Height, ti.Round, ti.Step)
}

type CommitInfo struct {
	height int64
	round  int
}

// ConsensusState handles execution of the consensus algorithm.
// It processes votes and proposals, and upon reaching agreement,
// commits blocks to the chain and executes them against the application.
// The internal state machine receives input from peers, the internal validator, and from a timer.
type ConsensusState struct {
	// config details
	client        *TendermintClient
	privValidator ttypes.PrivValidator // for signing votes

	// services for creating and executing blocks
	// TODO: encapsulate all of this in one "BlockManager"
	blockExec *BlockExecutor

	evpool ttypes.EvidencePool

	// internal state
	mtx sync.Mutex
	ttypes.RoundState
	state State // State until height-1.

	// state changes may be triggered by msgs from peers,
	// msgs from ourself, or by timeouts
	peerMsgQueue     chan MsgInfo
	internalMsgQueue chan MsgInfo
	timeoutTicker    TimeoutTicker

	// a Write-Ahead Log ensures we can recover from any kind of crash
	// and helps us avoid signing conflicting votes
	//wal          WAL
	replayMode   bool // so we don't log signing errors during replay
	doWALCatchup bool // determines if we even try to do the catchup

	// for tests where we want to limit the number of transitions the state makes
	nSteps int

	// some functions can be overwritten for testing
	decideProposal func(height int64, round int)
	doPrevote      func(height int64, round int)
	setProposal    func(proposal *types.Proposal) error

	// closed when we finish shutting down
	done chan struct{}

	NewTxsHeight   chan int64
	NewTxsFinished chan bool
	blockStore     *ttypes.BlockStore
	needSetFinish  bool
	syncMutex      sync.Mutex

	LastProposals BaseQueue

	roundstateRWLock sync.RWMutex //only lock roundstate so gossip in reactor can go on

	broadcastChannel chan<- MsgInfo

	ourId ID

	started uint32 // atomic
	stopped uint32 // atomic
	Quit    chan struct{}

	needCommitByInfo chan CommitInfo
	precommitOK      bool
	txsAvailable     chan int64
	begCons          time.Time
}

// NewConsensusState returns a new ConsensusState.
func NewConsensusState(client *TendermintClient, blockStore *ttypes.BlockStore, state State, blockExec *BlockExecutor, evpool ttypes.EvidencePool) *ConsensusState {
	cs := &ConsensusState{
		client:           client,
		blockExec:        blockExec,
		peerMsgQueue:     make(chan MsgInfo, msgQueueSize),
		internalMsgQueue: make(chan MsgInfo, msgQueueSize),
		timeoutTicker:    NewTimeoutTicker(),
		done:             make(chan struct{}),
		doWALCatchup:     true,
		//wal:              nilWAL{},
		evpool: evpool,

		NewTxsHeight:     make(chan int64, 1),
		NewTxsFinished:   make(chan bool),
		blockStore:       blockStore,
		needSetFinish:    false,
		Quit:             make(chan struct{}),
		needCommitByInfo: make(chan CommitInfo, maxCommitInfoSize),
		precommitOK:      false,
		txsAvailable:     make(chan int64, 1),
		begCons:          time.Time{},
	}
	// set function defaults (may be overwritten before calling Start)
	cs.decideProposal = cs.defaultDecideProposal
	cs.doPrevote = cs.defaultDoPrevote
	cs.setProposal = cs.defaultSetProposal

	cs.updateToState(state)
	// Don't call scheduleRound0 yet.
	// We do that upon Start().
	cs.reconstructLastCommit(state)

	//old proposal cache
	cs.LastProposals = BaseQueue{}
	cs.LastProposals.InitQueue(2)

	return cs
}

func (cs *ConsensusState) SetOurID(id ID) {
	cs.ourId = id
}

func (cs *ConsensusState) SetBroadcastChannel(broadcastChannel chan<- MsgInfo) {
	cs.broadcastChannel = broadcastChannel
}

func (cs *ConsensusState) IsRunning() bool {
	return atomic.LoadUint32(&cs.started) == 1 && atomic.LoadUint32(&cs.stopped) == 0
}

//----------------------------------------
// String returns a string.
func (cs *ConsensusState) String() string {
	// better not to access shared variables
	return fmt.Sprintf("ConsensusState") //(H:%v R:%v S:%v", cs.Height, cs.Round, cs.Step)
}

// GetState returns a copy of the chain state.
func (cs *ConsensusState) GetState() State {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	return cs.state.Copy()
}

// GetRoundState returns a copy of the internal consensus state.
func (cs *ConsensusState) GetRoundState() *ttypes.RoundState {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	return cs.getRoundState()
}

func (cs *ConsensusState) getRoundState() *ttypes.RoundState {
	//cs.roundstateRWLock.RLock()
	rs := cs.RoundState // copy
	//cs.roundstateRWLock.RUnlock()
	return &rs
}

// GetValidators returns a copy of the current validators.
func (cs *ConsensusState) GetValidators() (int64, []*ttypes.Validator) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	return cs.state.LastBlockHeight, cs.state.Validators.Copy().Validators
}

// SetPrivValidator sets the private validator account for signing votes.
func (cs *ConsensusState) SetPrivValidator(priv ttypes.PrivValidator) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	cs.privValidator = priv
}

// SetTimeoutTicker sets the local timer. It may be useful to overwrite for testing.
func (cs *ConsensusState) SetTimeoutTicker(timeoutTicker TimeoutTicker) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	cs.timeoutTicker = timeoutTicker
}

// LoadCommit loads the commit for a given height.
func (cs *ConsensusState) LoadCommit(height int64) *types.TendermintCommit {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	if height == cs.blockStore.Height() {
		return cs.blockStore.LoadSeenCommit(height)
	}
	return cs.blockStore.LoadBlockCommit(height)
}

// OnStart implements cmn.Service.
// It loads the latest state via the WAL, and starts the timeout and receive routines.
func (cs *ConsensusState) Start() {
	if atomic.CompareAndSwapUint32(&cs.started, 0, 1) {
		if atomic.LoadUint32(&cs.stopped) == 1 {
			tendermintlog.Error("ConsensusState already stoped")
		}
		cs.timeoutTicker.Start()

		go cs.checkTxsAvailable()
		// now start the receiveRoutine
		go cs.receiveRoutine(0)

		// schedule the first round!
		// use GetRoundState so we don't race the receiveRoutine for access
		cs.scheduleRound0(cs.GetRoundState())

	}
}

// OnStop implements cmn.Service. It stops all routines and waits for the WAL to finish.
func (cs *ConsensusState) Stop() {
	cs.timeoutTicker.Stop()

	// Make BaseService.Wait() wait until cs.wal.Wait()
	if cs.IsRunning() {
		//cs.wal.Wait()
	}
}

// Wait waits for the the main routine to return.
// NOTE: be sure to Stop() the event switch and drain
// any event channels or this may deadlock
func (cs *ConsensusState) Wait() {
	<-cs.done
}

/* 20180224 hg
// OpenWAL opens a file to log all consensus messages and timeouts for deterministic accountability
func (cs *ConsensusState) OpenWAL(walFile string) (WAL, error) {
	wal, err := NewWAL(walFile, cs.config.WalLight)
	if err != nil {
		tendermintlog.Error("Failed to open WAL for consensus state", "wal", walFile, "err", err)
		return nil, err
	}
	wal.SetLogger(tendermintlog.With("wal", walFile))
	if err := wal.Start(); err != nil {
		return nil, err
	}
	return wal, nil
}
*/

//------------------------------------------------------------
// internal functions for managing the state

func (cs *ConsensusState) updateHeight(height int64) {
	//cs.roundstateRWLock.Lock()
	cs.Height = height
	//cs.roundstateRWLock.Unlock()
}

func (cs *ConsensusState) updateRoundStep(round int, step ttypes.RoundStepType) {
	//cs.roundstateRWLock.Lock()
	cs.Round = round
	cs.Step = step
	//cs.roundstateRWLock.Unlock()
}

// enterNewRound(height, 0) at cs.StartTime.
func (cs *ConsensusState) scheduleRound0(rs *ttypes.RoundState) {
	//tendermintlog.Info("scheduleRound0", "now", time.Now(), "startTime", cs.StartTime)
	sleepDuration := rs.StartTime.Sub(time.Now()) // nolint: gotype, gosimple
	cs.scheduleTimeout(sleepDuration, rs.Height, 0, ttypes.RoundStepNewHeight)
}

// Attempt to schedule a timeout (by sending timeoutInfo on the tickChan)
func (cs *ConsensusState) scheduleTimeout(duration time.Duration, height int64, round int, step ttypes.RoundStepType) {
	cs.timeoutTicker.ScheduleTimeout(timeoutInfo{duration, height, round, step})
}

// send a msg into the receiveRoutine regarding our own proposal, block part, or vote
func (cs *ConsensusState) sendInternalMessage(mi MsgInfo) {
	select {
	case cs.internalMsgQueue <- mi:
	default:
		// NOTE: using the go-routine means our votes can
		// be processed out of order.
		// TODO: use CList here for strict determinism and
		// attempt push to internalMsgQueue in receiveRoutine
		tendermintlog.Info("Internal msg queue is full. Using a go-routine")
		go func() { cs.internalMsgQueue <- mi }()
	}
}

// Reconstruct LastCommit from SeenCommit, which we saved along with the block,
// (which happens even before saving the state)
func (cs *ConsensusState) reconstructLastCommit(state State) {
	if state.LastBlockHeight == 0 {
		return
	}
	seenCommit := cs.blockStore.LoadSeenCommit(state.LastBlockHeight)
	seenCommitC := ttypes.Commit{TendermintCommit: seenCommit}
	lastPrecommits := ttypes.NewVoteSet(state.ChainID, state.LastBlockHeight, seenCommitC.Round(), ttypes.VoteTypePrecommit, state.LastValidators)
	for _, item := range seenCommit.Precommits {
		if item == nil || len(item.Signature) == 0{
			continue
		}
		precommit := &ttypes.Vote{Vote:item}
		added, err := lastPrecommits.AddVote(precommit)
		if !added || err != nil {
			panic(fmt.Sprintf("Panicked on a Crisis: %v", fmt.Sprintf("Failed to reconstruct LastCommit: %v", err)))
		}
	}
	if !lastPrecommits.HasTwoThirdsMajority() {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", "Failed to reconstruct LastCommit: Does not have +2/3 maj"))
	}
	//cs.roundstateRWLock.Lock()
	cs.LastCommit = lastPrecommits
	//cs.roundstateRWLock.Unlock()
}

// Updates ConsensusState and increments height to match that of state.
// The round becomes 0 and cs.Step becomes ttypes.RoundStepNewHeight.
func (cs *ConsensusState) updateToState(state State) {
	if cs.CommitRound > -1 && 0 < cs.Height && cs.Height != state.LastBlockHeight {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("updateToState() expected state height of %v but found %v",
			cs.Height, state.LastBlockHeight)))
	}
	if !cs.state.IsEmpty() && cs.state.LastBlockHeight+1 != cs.Height {
		// This might happen when someone else is mutating cs.state.
		// Someone forgot to pass in state.Copy() somewhere?!
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("Inconsistent cs.state.LastBlockHeight+1 %v vs cs.Height %v",
			cs.state.LastBlockHeight+1, cs.Height)))
	}

	// If state isn't further out than cs.state, just ignore.
	// This happens when SwitchToConsensus() is called in the reactor.
	// We don't want to reset e.g. the Votes.
	if !cs.state.IsEmpty() && (state.LastBlockHeight <= cs.state.LastBlockHeight) {
		tendermintlog.Info("Ignoring updateToState()", "newHeight", state.LastBlockHeight+1, "oldHeight", cs.state.LastBlockHeight+1)
		return
	}

	// Reset fields based on state.
	validators := state.Validators
	lastPrecommits := (*ttypes.VoteSet)(nil)
	if cs.CommitRound > -1 && cs.Votes != nil {
		if !cs.Votes.Precommits(cs.CommitRound).HasTwoThirdsMajority() {
			panic(fmt.Sprintf("Panicked on a Sanity Check: %v", "updateToState(state) called but last Precommit round didn't have +2/3"))
		}
		lastPrecommits = cs.Votes.Precommits(cs.CommitRound)
	}

	// Next desired block height
	height := state.LastBlockHeight + 1

	// RoundState fields
	cs.updateHeight(height)
	cs.updateRoundStep(0, ttypes.RoundStepNewHeight)
	//cs.roundstateRWLock.Lock()
	if cs.CommitTime.IsZero() {
		// "Now" makes it easier to sync up dev nodes.
		// We add timeoutCommit to allow transactions
		// to be gathered for the first block.
		// And alternative solution that relies on clocks:
		//  cs.StartTime = state.LastBlockTime.Add(timeoutCommit)
		cs.StartTime = cs.Commit(time.Now())
	} else {
		cs.StartTime = cs.Commit(cs.CommitTime)
	}
	cs.Validators = validators
	cs.Proposal = nil
	cs.ProposalBlock = nil
	cs.LockedRound = 0
	cs.LockedBlock = nil
	cs.Votes = ttypes.NewHeightVoteSet(state.ChainID, height, validators)
	cs.CommitRound = -1
	cs.LastCommit = lastPrecommits
	cs.LastValidators = state.LastValidators
	//cs.roundstateRWLock.Unlock()
	//cs.precommitOK = false

	cs.state = state

	// Finally, broadcast RoundState
	cs.newStep()
}

func (cs *ConsensusState) newStep() {

	if cs.broadcastChannel != nil {
		cs.broadcastChannel <- MsgInfo{TypeID:ttypes.NewRoundStepID, Msg:cs.RoundStateMessage(), PeerID:cs.ourId, PeerIP:""}
	}
	//20180226 hg do it later
	//cs.wal.Save(rs)
	cs.nSteps++
}

func (cs *ConsensusState) NewTxsAvailable(height int64) {
	cs.syncMutex.Lock()
	defer cs.syncMutex.Unlock()
	cs.needSetFinish = true
	cs.NewTxsHeight <- height
}

//-----------------------------------------
// the main go routines

// receiveRoutine handles messages which may cause state transitions.
// it's argument (n) is the number of messages to process before exiting - use 0 to run forever
// It keeps the RoundState and is the only thing that updates it.
// Updates (state transitions) happen on timeouts, complete proposals, and 2/3 majorities.
// ConsensusState must be locked before any internal state is updated.
func (cs *ConsensusState) receiveRoutine(maxSteps int) {
	defer func() {
		if r := recover(); r != nil {
			tendermintlog.Error("CONSENSUS FAILURE!!!", "err", r, "stack", string(debug.Stack()))
		}
	}()

	for {
		if maxSteps > 0 {
			if cs.nSteps >= maxSteps {
				tendermintlog.Info("reached max steps. exiting receive routine")
				cs.nSteps = 0
				return
			}
		}
		rs := cs.RoundState
		var mi MsgInfo
		/*
			txs:=cs.client.RequestTx()
			if len(txs) != 0 {
				txs = cs.client.CheckTxDup(txs)
				lastBlock := cs.client.GetCurrentBlock()
				cs.handleTxsAvailable(lastBlock.Height + 1)
			}
		*/
		select {
		case height := <-cs.txsAvailable:
			cs.handleTxsAvailable(height)
		case mi = <-cs.peerMsgQueue:
			//cs.wal.Save(mi)
			// handles proposals, block parts, votes
			// may generate internal events (votes, complete proposals, 2/3 majorities)
			cs.handleMsg(mi)
		case mi = <-cs.internalMsgQueue:
			//cs.wal.Save(mi)
			// handles proposals, block parts, votes
			cs.handleMsg(mi)
		case ti := <-cs.timeoutTicker.Chan(): // tockChan:
			//cs.wal.Save(ti)
			// if the timeout is relevant to the rs
			// go to the next step
			cs.handleTimeout(ti, rs)
		case <-cs.Quit:

			// NOTE: the internalMsgQueue may have signed messages from our
			// priv_val that haven't hit the WAL, but its ok because
			// priv_val tracks LastSig

			// close wal now that we're done writing to it
			//cs.wal.Stop()

			close(cs.done)
			return
		}
	}
}

// state transitions on complete-proposal, 2/3-any, 2/3-one
func (cs *ConsensusState) handleMsg(mi MsgInfo) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	var err error
	msg, peerID, peerIP := mi.Msg, string(mi.PeerID), mi.PeerIP
	//tendermintlog.Info("handleMsg", "peerIP", peerIP, "msg", msg.TypeName())
	switch msg := msg.(type) {
	case *types.Proposal:
		// will not cause transition.
		// once proposal is set, we can receive block parts
		err = cs.setProposal(msg)
		if err != nil {
			tendermintlog.Error("handleMsg proposalMsg failed", "msg", msg, "error", err)
		}
	case *types.Vote:
		// attempt to add the vote and dupeout the validator if its a duplicate signature
		// if the vote gives us a 2/3-any or 2/3-one, we transition
		err := cs.tryAddVote(msg, peerID)
		if err == ErrAddingVote {
			// TODO: punish peer
		}

		// NOTE: the vote is broadcast to peers by the reactor listening
		// for vote events

		// TODO: If rs.Height == vote.Height && rs.Round < vote.Round,
		// the peer is sending us CatchupCommit precommits.
		// We could make note of this and help filter in broadcastHasVoteMessage().
	default:
		tendermintlog.Error("Unknown msg type", msg.String(), "peerid", peerID, "peerip", peerIP)
	}
	if err != nil {
		tendermintlog.Error("Error with msg", "type", reflect.TypeOf(msg), "peerid", peerID, "peerip", peerIP, "err", err, "msg", msg)
	}
}

func (cs *ConsensusState) handleTimeout(ti timeoutInfo, rs ttypes.RoundState) {
	tendermintlog.Debug("Received tock", "timeout", ti.Duration, "height", ti.Height, "round", ti.Round, "step", ti.Step)

	// timeouts must be for current height, round, step
	if ti.Height != rs.Height || ti.Round < rs.Round || (ti.Round == rs.Round && ti.Step < rs.Step) {
		tendermintlog.Debug("Ignoring tock because we're ahead", "height", rs.Height, "round", rs.Round, "step", rs.Step)
		return
	}

	// the timeout will now cause a state transition
	cs.mtx.Lock()
	defer cs.mtx.Unlock()

	switch ti.Step {
	case ttypes.RoundStepNewHeight:
		// NewRound event fired from enterNewRound.
		// XXX: should we fire timeout here (for timeout commit)?
		cs.enterNewRound(ti.Height, 0)
	case ttypes.RoundStepNewRound:
		cs.enterPropose(ti.Height, 0)
	case ttypes.RoundStepPropose:
		//cs.eventBus.PublishEventTimeoutPropose(cs.RoundStateEvent())
		cs.enterPrevote(ti.Height, ti.Round)
	case ttypes.RoundStepPrevoteWait:
		//cs.eventBus.PublishEventTimeoutWait(cs.RoundStateEvent())
		cs.enterPrecommit(ti.Height, ti.Round)
	case ttypes.RoundStepPrecommitWait:
		//cs.eventBus.PublishEventTimeoutWait(cs.RoundStateEvent())
		cs.enterNewRound(ti.Height, ti.Round+1)
	default:
		panic(fmt.Sprintf("Invalid timeout step: %v", ti.Step))
	}

}

func (cs *ConsensusState) checkTxsAvailable() {
	for {
		select {
		case height := <-cs.client.TxsAvailable():
			tendermintlog.Info(fmt.Sprintf("checkTxsAvailable. Current: %v/%v/%v", cs.Height, cs.Round, cs.Step), "height", height)
			if cs.Height != height {
				tendermintlog.Warn("blockchain and consensus are not sync")
				break
			}
			if cs.isProposalComplete() {
				tendermintlog.Info("already has proposal")
				break
			}
			cs.txsAvailable <- height
		}
	}
}

func (cs *ConsensusState) handleTxsAvailable(height int64) {
	cs.mtx.Lock()
	defer cs.mtx.Unlock()
	// we only need to do this for round 0
	cs.begCons = time.Now()
	cs.enterPropose(height, 0)
}

//-----------------------------------------------------------------------------
// State functions
// Used internally by handleTimeout and handleMsg to make state transitions

// Enter: `timeoutNewHeight` by startTime (commitTime+timeoutCommit),
// 	or, if SkipTimeout==true, after receiving all precommits from (height,round-1)
// Enter: `timeoutPrecommits` after any +2/3 precommits from (height,round-1)
// Enter: +2/3 precommits for nil at (height,round-1)
// Enter: +2/3 prevotes any or +2/3 precommits for block or any from (height, round)
// NOTE: cs.StartTime was already set for height.
func (cs *ConsensusState) enterNewRound(height int64, round int) {
	if cs.Height != height || round < cs.Round || (cs.Round == round && cs.Step != ttypes.RoundStepNewHeight) {
		tendermintlog.Debug(fmt.Sprintf("enterNewRound(%v/%v): Invalid args. Current step: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))
		return
	}

	if now := time.Now(); cs.StartTime.After(now) {
		tendermintlog.Info("Need to set a buffer and log message here for sanity.", "startTime", cs.StartTime, "now", now)
	}

	tendermintlog.Info(fmt.Sprintf("enterNewRound(%v/%v). Current: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))

	// Increment validators if necessary
	validators := cs.Validators
	if cs.Round < round {
		validators = validators.Copy()
		validators.IncrementAccum(round - cs.Round)
		tendermintlog.Debug("enterNewRound validator changed", "csr", cs.Round, "round", round)
	}
	tendermintlog.Debug("enterNewRound proposer ", "proposer", validators.Proposer, "validators", validators)
	// Setup new round
	// we don't fire newStep for this step,
	// but we fire an event, so update the round step first
	cs.updateRoundStep(round, ttypes.RoundStepNewRound)
	//cs.roundstateRWLock.Lock()
	cs.Validators = validators
	//cs.roundstateRWLock.Unlock()
	if round == 0 {
		// We've already reset these upon new height,
		// and meanwhile we might have received a proposal
		// for round 0.
	} else {
		tendermintlog.Info("Resetting Proposal info")
		//cs.roundstateRWLock.Lock()
		cs.Proposal = nil
		cs.ProposalBlock = nil
		//cs.roundstateRWLock.Unlock()
		//cs.precommitOK = false
	}
	cs.Votes.SetRound(round + 1) // also track next round (round+1) to allow round-skipping

	//cs.eventBus.PublishEventNewRound(cs.RoundStateEvent())
	cs.broadcastChannel <- MsgInfo{TypeID:ttypes.NewRoundStepID, Msg:cs.RoundStateMessage(), PeerID:cs.ourId, PeerIP:""}

	// Wait for txs to be available in the mempool
	// before we enterPropose in round 0. If the last block changed the app hash,
	// we may need an empty "proof" block, and enterPropose immediately.

	waitForTxs := cs.WaitForTxs() && round == 0 && !cs.needProofBlock(height)
	if waitForTxs {
		if cs.client.Cfg.CreateEmptyBlocksInterval > 0 {
			cs.scheduleTimeout(cs.EmptyBlocksInterval(), height, round, ttypes.RoundStepNewRound)
		}
		go cs.proposalHeartbeat(height, round)
	} else {
		cs.enterPropose(height, round)
	}
}

// needProofBlock returns true on the first height (so the genesis app hash is signed right away)
// and where the last block (height-1) caused the app hash to change
func (cs *ConsensusState) needProofBlock(height int64) bool {
	if height == 1 {
		//return true
	}

	/*
		lastBlockMeta := cs.client.LoadBlockMeta(height - 1)
		tendermintlog.Info("needProofBlock", "lastBlockMeta", lastBlockMeta, "apphash", cs.state.AppHash, "headerhash", lastBlockMeta.Header.AppHash)
		return !bytes.Equal(cs.state.AppHash, lastBlockMeta.Header.AppHash)
	*/
	return false
}

func (cs *ConsensusState) proposalHeartbeat(height int64, round int) {
	counter := 0
	addr := cs.privValidator.GetAddress()
	valIndex, _ := cs.Validators.GetByAddress(addr)
	chainID := cs.state.ChainID
	for {
		rs := cs.GetRoundState()
		// if we've already moved on, no need to send more heartbeats
		if rs.Step > ttypes.RoundStepNewRound || rs.Round > round || rs.Height > height {
			if round == 0 {
				cs.begCons = time.Now()
			}
			return
		}
		heartbeat := &types.Heartbeat{
			Height:           rs.Height,
			Round:            int32(rs.Round),
			Sequence:         int32(counter),
			ValidatorAddress: addr,
			ValidatorIndex:   int32(valIndex),
		}
		heartbeatMsg := &ttypes.Heartbeat{Heartbeat:heartbeat}
		cs.privValidator.SignHeartbeat(chainID, heartbeatMsg)
		cs.broadcastChannel <- MsgInfo{TypeID:ttypes.ProposalHeartbeatID, Msg:heartbeat, PeerID:cs.ourId, PeerIP:""}
		counter++
		time.Sleep(proposalHeartbeatIntervalSeconds * time.Second)
	}
}

// Enter (CreateEmptyBlocks): from enterNewRound(height,round)
// Enter (CreateEmptyBlocks, CreateEmptyBlocksInterval > 0 ): after enterNewRound(height,round), after timeout of CreateEmptyBlocksInterval
// Enter (!CreateEmptyBlocks) : after enterNewRound(height,round), once txs are in the mempool
func (cs *ConsensusState) enterPropose(height int64, round int) {
	if cs.Height != height || round < cs.Round || (cs.Round == round && ttypes.RoundStepPropose <= cs.Step) {
		tendermintlog.Info(fmt.Sprintf("enterPropose(%v/%v): Invalid args. Current step: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))
		return
	}
	tendermintlog.Info(fmt.Sprintf("enterPropose(%v/%v). Current: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))

	defer func() {
		// Done enterPropose:
		cs.updateRoundStep(round, ttypes.RoundStepPropose)
		cs.newStep()

		// If we have the whole proposal + POL, then goto Prevote now.
		// else, we'll enterPrevote when the rest of the proposal is received (in AddProposalBlockPart),
		// or else after timeoutPropose
		if cs.isProposalComplete() {
			cs.enterPrevote(height, cs.Round)
		}
	}()

	// If we don't get the proposal and all block parts quick enough, enterPrevote
	cs.scheduleTimeout(cs.Propose(round), height, round, ttypes.RoundStepPropose)

	// Nothing more to do if we're not a validator
	if cs.privValidator == nil {
		tendermintlog.Debug("This node is not a validator")
		return
	}

	// if not a validator, we're done
	if !cs.Validators.HasAddress(cs.privValidator.GetAddress()) {
		tendermintlog.Debug("This node is not a validator", "addr", cs.privValidator.GetAddress(), "vals", cs.Validators)
		return
	}
	tendermintlog.Debug("This node is a validator")

	if cs.isProposer() {
		tendermintlog.Info("enterPropose: Our turn to propose", "proposer", cs.Validators.GetProposer().Address, "privValidator", cs.privValidator)
		cs.decideProposal(height, round)
	} else {
		tendermintlog.Info("enterPropose: Not our turn to propose", "proposer", cs.Validators.GetProposer().Address, "privValidator", cs.privValidator)
	}

}

func (cs *ConsensusState) isProposer() bool {
	return bytes.Equal(cs.Validators.GetProposer().Address, cs.privValidator.GetAddress())
}

func (cs *ConsensusState) defaultDecideProposal(height int64, round int) {

	var block *ttypes.TendermintBlock

	// Decide on block
	if cs.LockedBlock != nil {
		// If we're locked onto a block, just choose that.
		block = cs.LockedBlock
	} else {
		// Create a new proposal block from state/txs from the mempool.
		block = cs.createProposalBlock()
		if block == nil { // on error
			return
		}
	}

	// Make proposal
	polRound, polBlockID := cs.Votes.POLInfo()
	proposal := ttypes.NewProposal(height, round, block.TendermintBlock, polRound, polBlockID.BlockID)
	if err := cs.privValidator.SignProposal(cs.state.ChainID, proposal); err == nil {
		// Set fields
		/*  fields set by setProposal and addBlockPart
		cs.Proposal = proposal
		cs.ProposalBlock = block
		cs.ProposalBlockParts = blockParts
		*/

		// send proposal and block parts on internal msg queue
		cs.sendInternalMessage(MsgInfo{ttypes.ProposalID,&proposal.Proposal, cs.ourId, ""})
		tendermintlog.Debug("Signed proposal", "height", height, "round", round, "proposal", proposal)
	} else {
		if !cs.replayMode {
			tendermintlog.Error("enterPropose: Error signing proposal", "height", height, "round", round, "err", err)
		}
	}
}

// Returns true if the proposal block is complete &&
// (if POLRound was proposed, we have +2/3 prevotes from there).
func (cs *ConsensusState) isProposalComplete() bool {
	if cs.Proposal == nil || cs.ProposalBlock == nil {
		return false
	}
	// we have the proposal. if there's a POLRound,
	// make sure we have the prevotes from it too
	if cs.Proposal.POLRound < 0 {
		return true
	}
	// if this is false the proposer is lying or we haven't received the POL yet
	return cs.Votes.Prevotes(int(cs.Proposal.POLRound)).HasTwoThirdsMajority()
}

// Create the next block to propose and return it.
// Returns nil block upon error.
// NOTE: keep it side-effect free for clarity.
func (cs *ConsensusState) createProposalBlock() (block *ttypes.TendermintBlock) {
	var commit *types.TendermintCommit
	if cs.Height == 1 {
		// We're creating a proposal for the first block.
		// The commit is empty, but not nil.
		commit = &types.TendermintCommit{}
	} else if cs.LastCommit.HasTwoThirdsMajority() {
		// Make the commit from LastCommit
		commit = cs.LastCommit.MakeCommit()
	} else {
		// This shouldn't happen.
		tendermintlog.Error("enterPropose: Cannot propose anything: No commit for the previous block.")
		return
	}

	// Mempool validated transactions
	beg := time.Now()
	pblock := cs.client.BuildBlock()
	if pblock == nil {
		tendermintlog.Error("No new block to propose, will change Proposer", "height", cs.Height)
		return
	}
	tendermintlog.Info(fmt.Sprintf("createProposalBlock BuildBlock. Current: %v/%v/%v", cs.Height, cs.Round, cs.Step), "txs-len", len(pblock.Txs), "cost", time.Since(beg))

	block = cs.state.MakeBlock(cs.Height, pblock.Txs, commit)
	tendermintlog.Info("createProposalBlock block", "txs-len", len(block.Txs))
	evidence := cs.evpool.PendingEvidence()
	block.AddEvidence(evidence)
	return block
}

// Enter: `timeoutPropose` after entering Propose.
// Enter: proposal block and POL is ready.
// Enter: any +2/3 prevotes for future round.
// Prevote for LockedBlock if we're locked, or ProposalBlock if valid.
// Otherwise vote nil.
func (cs *ConsensusState) enterPrevote(height int64, round int) {
	if cs.Height != height || round < cs.Round || (cs.Round == round && ttypes.RoundStepPrevote <= cs.Step) {
		tendermintlog.Debug(fmt.Sprintf("enterPrevote(%v/%v): Invalid args. Current step: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))
		return
	}

	defer func() {
		// Done enterPrevote:
		cs.updateRoundStep(round, ttypes.RoundStepPrevote)
		cs.newStep()
	}()

	// fire event for how we got here
	if cs.isProposalComplete() {
		//only used to test hg 20180227
		//cs.eventBus.PublishEventCompleteProposal(cs.RoundStateEvent())
	} else {
		// we received +2/3 prevotes for a future round
		// TODO: catchup event?
	}

	tendermintlog.Info(fmt.Sprintf("enterPrevote(%v/%v). Current: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step), "cost", time.Since(cs.begCons))

	// Sign and broadcast vote as necessary
	cs.doPrevote(height, round)

	// Once `addVote` hits any +2/3 prevotes, we will go to PrevoteWait
	// (so we have more time to try and collect +2/3 prevotes for a single block)
}

func (cs *ConsensusState) defaultDoPrevote(height int64, round int) {
	// If a block is locked, prevote that.
	if cs.LockedBlock != nil {
		tendermintlog.Info("enterPrevote: Block was locked")
		cs.signAddVote(ttypes.VoteTypePrevote, cs.LockedBlock.Hash())
		return
	}

	// If ProposalBlock is nil, prevote nil.
	if cs.ProposalBlock == nil {
		tendermintlog.Info("enterPrevote: ProposalBlock is nil")
		cs.signAddVote(ttypes.VoteTypePrevote, nil)
		return
	}

	// Validate proposal block
	err := cs.blockExec.ValidateBlock(cs.state, cs.ProposalBlock)
	if err != nil {
		// ProposalBlock is invalid, prevote nil.
		tendermintlog.Error("enterPrevote: ProposalBlock is invalid", "err", err)
		cs.signAddVote(ttypes.VoteTypePrevote, nil)
		return
	}

	// Prevote cs.ProposalBlock
	// NOTE: the proposal signature is validated when it is received,
	// and the proposal block parts are validated as they are received (against the merkle hash in the proposal)
	tendermintlog.Info("enterPrevote: ProposalBlock is valid")
	cs.signAddVote(ttypes.VoteTypePrevote, cs.ProposalBlock.Hash())
}

// Enter: any +2/3 prevotes at next round.
func (cs *ConsensusState) enterPrevoteWait(height int64, round int) {
	if cs.Height != height || round < cs.Round || (cs.Round == round && ttypes.RoundStepPrevoteWait <= cs.Step) {
		tendermintlog.Debug(fmt.Sprintf("enterPrevoteWait(%v/%v): Invalid args. Current step: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))
		return
	}
	if !cs.Votes.Prevotes(round).HasTwoThirdsAny() {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("enterPrevoteWait(%v/%v), but Prevotes does not have any +2/3 votes", height, round)))
	}
	tendermintlog.Info(fmt.Sprintf("enterPrevoteWait(%v/%v). Current: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))

	defer func() {
		// Done enterPrevoteWait:
		cs.updateRoundStep(round, ttypes.RoundStepPrevoteWait)
		cs.newStep()
	}()

	// Wait for some more prevotes; enterPrecommit
	cs.scheduleTimeout(cs.Prevote(round), height, round, ttypes.RoundStepPrevoteWait)
}

// Enter: `timeoutPrevote` after any +2/3 prevotes.
// Enter: +2/3 precomits for block or nil.
// Enter: any +2/3 precommits for next round.
// Lock & precommit the ProposalBlock if we have enough prevotes for it (a POL in this round)
// else, unlock an existing lock and precommit nil if +2/3 of prevotes were nil,
// else, precommit nil otherwise.
func (cs *ConsensusState) enterPrecommit(height int64, round int) {
	if cs.Height != height || round < cs.Round || (cs.Round == round && ttypes.RoundStepPrecommit <= cs.Step) {
		tendermintlog.Debug(fmt.Sprintf("enterPrecommit(%v/%v): Invalid args. Current step: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))
		return
	}

	tendermintlog.Info(fmt.Sprintf("enterPrecommit(%v/%v). Current: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step), "cost", time.Since(cs.begCons))

	defer func() {
		// Done enterPrecommit:
		cs.updateRoundStep(round, ttypes.RoundStepPrecommit)
		cs.newStep()
	}()

	blockID, ok := cs.Votes.Prevotes(round).TwoThirdsMajority()

	// If we don't have a polka, we must precommit nil
	if !ok {
		if cs.LockedBlock != nil {
			tendermintlog.Info("enterPrecommit: No +2/3 prevotes during enterPrecommit while we're locked. Precommitting nil")
		} else {
			tendermintlog.Info("enterPrecommit: No +2/3 prevotes during enterPrecommit. Precommitting nil.")
		}
		cs.signAddVote(ttypes.VoteTypePrecommit, nil)
		return
	}

	// At this point +2/3 prevoted for a particular block or nil
	//cs.eventBus.PublishEventPolka(cs.RoundStateEvent())

	// the latest POLRound should be this round
	polRound, _ := cs.Votes.POLInfo()
	if polRound < round {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("This POLRound should be %v but got %", round, polRound)))
	}

	// +2/3 prevoted nil. Unlock and precommit nil.
	if len(blockID.Hash) == 0 {
		if cs.LockedBlock == nil {
			tendermintlog.Info("enterPrecommit: +2/3 prevoted for nil.")
		} else {
			tendermintlog.Info("enterPrecommit: +2/3 prevoted for nil. Unlocking")
			//cs.roundstateRWLock.Lock()
			cs.LockedRound = 0
			cs.LockedBlock = nil
			//cs.roundstateRWLock.Unlock()
			//cs.eventBus.PublishEventUnlock(cs.RoundStateEvent())
		}
		cs.signAddVote(ttypes.VoteTypePrecommit, nil)
		return
	}

	// At this point, +2/3 prevoted for a particular block.

	// If we're already locked on that block, precommit it, and update the LockedRound
	if cs.LockedBlock.HashesTo(blockID.Hash) {
		tendermintlog.Info("enterPrecommit: +2/3 prevoted locked block. Relocking")
		//cs.roundstateRWLock.Lock()
		cs.LockedRound = round
		//cs.roundstateRWLock.Unlock()
		//cs.eventBus.PublishEventRelock(cs.RoundStateEvent())
		cs.signAddVote(ttypes.VoteTypePrecommit, blockID.Hash)
		return
	}

	// If +2/3 prevoted for proposal block, stage and precommit it
	if cs.ProposalBlock.HashesTo(blockID.Hash) {
		tendermintlog.Info("enterPrecommit: +2/3 prevoted proposal block. Locking", "hash", blockID.Hash)
		// Validate the block.
		if err := cs.blockExec.ValidateBlock(cs.state, cs.ProposalBlock); err != nil {
			panic(fmt.Sprintf("Panicked on a Consensus Failure: %v", fmt.Sprintf("enterPrecommit: +2/3 prevoted for an invalid block: %v", err)))
		}
		//cs.roundstateRWLock.Lock()
		cs.LockedRound = round
		cs.LockedBlock = cs.ProposalBlock
		//cs.roundstateRWLock.Unlock()
		//cs.eventBus.PublishEventLock(cs.RoundStateEvent())
		cs.signAddVote(ttypes.VoteTypePrecommit, blockID.Hash)
		return
	}

	// There was a polka in this round for a block we don't have.
	// Fetch that block, unlock, and precommit nil.
	// The +2/3 prevotes for this round is the POL for our unlock.
	// TODO: In the future save the POL prevotes for justification.
	//cs.roundstateRWLock.Lock()
	cs.LockedRound = 0
	cs.LockedBlock = nil
	//cs.roundstateRWLock.Unlock()
	//cs.eventBus.PublishEventUnlock(cs.RoundStateEvent())
	cs.signAddVote(ttypes.VoteTypePrecommit, nil)
}

// Enter: any +2/3 precommits for next round.
func (cs *ConsensusState) enterPrecommitWait(height int64, round int) {
	if cs.Height != height || round < cs.Round || (cs.Round == round && ttypes.RoundStepPrecommitWait <= cs.Step) {
		tendermintlog.Debug(fmt.Sprintf("enterPrecommitWait(%v/%v): Invalid args. Current step: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))
		return
	}
	if !cs.Votes.Precommits(round).HasTwoThirdsAny() {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("enterPrecommitWait(%v/%v), but Precommits does not have any +2/3 votes", height, round)))
	}
	tendermintlog.Info(fmt.Sprintf("enterPrecommitWait(%v/%v). Current: %v/%v/%v", height, round, cs.Height, cs.Round, cs.Step))

	defer func() {
		// Done enterPrecommitWait:
		cs.updateRoundStep(round, ttypes.RoundStepPrecommitWait)
		cs.newStep()
	}()

	// Wait for some more precommits; enterNewRound
	cs.scheduleTimeout(cs.Precommit(round), height, round, ttypes.RoundStepPrecommitWait)

}

// Enter: +2/3 precommits for block
func (cs *ConsensusState) enterCommit(height int64, commitRound int) {
	if cs.Height != height || ttypes.RoundStepCommit <= cs.Step {
		tendermintlog.Debug(fmt.Sprintf("enterCommit(%v/%v): Invalid args. Current step: %v/%v/%v", height, commitRound, cs.Height, cs.Round, cs.Step))
		return
	}
	tendermintlog.Info(fmt.Sprintf("enterCommit(%v/%v). Current: %v/%v/%v", height, commitRound, cs.Height, cs.Round, cs.Step), "cost", time.Since(cs.begCons))

	defer func() {
		// Done enterCommit:
		// keep cs.Round the same, commitRound points to the right Precommits set.
		cs.updateRoundStep(cs.Round, ttypes.RoundStepCommit)
		//cs.roundstateRWLock.Lock()
		cs.CommitRound = commitRound
		cs.CommitTime = time.Now()
		//cs.roundstateRWLock.Unlock()
		cs.newStep()

		// Maybe finalize immediately.
		cs.tryFinalizeCommit(height)
	}()

	blockID, ok := cs.Votes.Precommits(commitRound).TwoThirdsMajority()
	if !ok {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", "RunActionCommit() expects +2/3 precommits"))
	}

	// The Locked* fields no longer matter.
	// Move them over to ProposalBlock if they match the commit hash,
	// otherwise they'll be cleared in updateToState.
	if cs.LockedBlock.HashesTo(blockID.Hash) {
		tendermintlog.Info("Commit is for locked block. Set ProposalBlock=LockedBlock", "blockHash", blockID.Hash)
		//cs.roundstateRWLock.Lock()
		cs.ProposalBlock = cs.LockedBlock
		//cs.roundstateRWLock.Unlock()
	}
}

// If we have the block AND +2/3 commits for it, finalize.
func (cs *ConsensusState) tryFinalizeCommit(height int64) {
	if cs.Height != height {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("tryFinalizeCommit() cs.Height: %v vs height: %v", cs.Height, height)))
	}

	blockID, ok := cs.Votes.Precommits(cs.CommitRound).TwoThirdsMajority()
	if !ok || len(blockID.Hash) == 0 {
		tendermintlog.Error("Attempt to finalize failed. There was no +2/3 majority, or +2/3 was for <nil>.")
		tendermintlog.Info(fmt.Sprintf("Continue consensus. Current: %v/%v/%v", cs.Height, cs.CommitRound, cs.Step), "cost", time.Since(cs.begCons))
		cs.enterNewRound(cs.Height, cs.CommitRound+1)
		return
	}
	if !cs.ProposalBlock.HashesTo(blockID.Hash) {
		// TODO: this happens every time if we're not a validator (ugly logs)
		// TODO: ^^ wait, why does it matter that we're a validator?
		tendermintlog.Error("Attempt to finalize failed. We don't have the commit block.", "proposal-block", cs.ProposalBlock.Hash(), "commit-block", blockID.Hash)
		tendermintlog.Info(fmt.Sprintf("Continue consensus. Current: %v/%v/%v", cs.Height, cs.CommitRound, cs.Step), "cost", time.Since(cs.begCons))
		return
	}

	//	go
	cs.finalizeCommit(height)
}

func (cs *ConsensusState) commitReturn(result bool) {
	cs.syncMutex.Lock()
	if cs.needSetFinish {
		cs.NewTxsFinished <- result
		cs.needSetFinish = false
	}
	cs.syncMutex.Unlock()
	tendermintlog.Debug("performance: NewTxsFinished set ", "result", result)
}

// Increment height and goto ttypes.RoundStepNewHeight
func (cs *ConsensusState) finalizeCommit(height int64) {
	if cs.Height != height || cs.Step != ttypes.RoundStepCommit {
		tendermintlog.Debug(fmt.Sprintf("finalizeCommit(%v): Invalid args. Current step: %v/%v/%v", height, cs.Height, cs.Round, cs.Step))
		return
	}

	blockID, ok := cs.Votes.Precommits(cs.CommitRound).TwoThirdsMajority()
	block := cs.ProposalBlock

	if !ok {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("Cannot finalizeCommit, commit does not have two thirds majority")))
	}

	if !block.HashesTo(blockID.Hash) {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("Cannot finalizeCommit, ProposalBlock does not hash to commit hash")))
	}
	if err := cs.blockExec.ValidateBlock(cs.state, block); err != nil {
		panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("+2/3 committed an invalid block: %v", err)))
	}

	tendermintlog.Debug("performance: Finalizing commit of block ", "tx_numbers", block.Header.NumTxs,
		"height", block.Header.Height, "hash", block.Hash(), "root", block.Header.AppHash)

	stateCopy := cs.state.Copy()
	// NOTE: the block.AppHash wont reflect these txs until the next block
	var err error
	stateCopy, err = cs.blockExec.ApplyBlock(stateCopy, ttypes.BlockID{BlockID:types.BlockID{Hash:block.Hash()}}, block)
	if err != nil {
		tendermintlog.Error("Error on ApplyBlock. Did the application crash? Please restart tendermint", "err", err)
		//windows not support
		err := ttypes.Kill()
		if err != nil {
			tendermintlog.Error("Failed to kill this process - please do so manually", "err", err)
		}
		return
	}

	lastCommit := block.LastCommit

	precommits := cs.Votes.Precommits(cs.CommitRound)
	seenCommit := precommits.MakeCommit()

	newState := SaveState(stateCopy)

	//tendermintlog.Info("blockid info", "seen commit", seenCommit.StringIndented("seen"),"last commit", lastCommit.StringIndented("last"), "state", stateCopy)
	tendermintlog.Info(fmt.Sprintf("Save consensus state. Current: %v/%v/%v", cs.Height, cs.CommitRound, cs.Step), "cost", time.Since(cs.begCons))
	//hg 20180302
	if cs.isProposer() {
		newProposal := cs.Proposal
		tendermintlog.Info("finalizeCommit get hash of txs of proposal", "height", block.Header.Height, "hash", merkle.CalcMerkleRoot(block.Txs))
		commitBlock := &types.Block{}
		commitBlock.Height = block.Header.Height
		commitBlock.Txs = make([]*types.Transaction, 1, len(block.Txs)+1)
		commitBlock.Txs = append(commitBlock.Txs, block.Txs...)

		tx0 := ttypes.CreateBlockInfoTx(cs.blockStore.GetPubkey(), lastCommit, seenCommit, newState, newProposal)
		commitBlock.Txs[0] = tx0
		err = cs.client.CommitBlock(commitBlock)
		if err != nil {
			cs.LockedRound = 0
			cs.LockedBlock = nil
			tendermintlog.Info(fmt.Sprintf("Proposer continue consensus. Current: %v/%v/%v", cs.Height, cs.Round, cs.Step), "cost", time.Since(cs.begCons))
			cs.enterNewRound(cs.Height, cs.CommitRound+1)
			return
		}
		tendermintlog.Info(fmt.Sprintf("Proposer reach consensus. Current: %v/%v/%v", cs.Height, cs.CommitRound, cs.Step),
			"tx-lens", len(commitBlock.Txs), "cost", time.Since(cs.begCons), "proposer-addr", fmt.Sprintf("%X", ttypes.Fingerprint(cs.privValidator.GetAddress())))
	} else {
		reachCons, err := cs.client.CheckCommit(block.Header.Height)
		if err != nil {
			panic("Not proposer, and exit consensus")
		}
		if !reachCons {
			cs.LockedRound = 0
			cs.LockedBlock = nil
			tendermintlog.Info(fmt.Sprintf("Not-Proposer continue consensus. Current: %v/%v/%v", cs.Height, cs.CommitRound, cs.Step), "cost", time.Since(cs.begCons))
			cs.enterNewRound(cs.Height, cs.CommitRound+1)
			return
		}
		tendermintlog.Info(fmt.Sprintf("Not-Proposer reach consensus. Current: %v/%v/%v", cs.Height, cs.CommitRound, cs.Step),
			"tx-lens", block.Header.NumTxs+1, "cost", time.Since(cs.begCons), "proposer-addr", fmt.Sprintf("%X", ttypes.Fingerprint(cs.Validators.GetProposer().Address)))
	}

	//fail.Fail() // XXX

	// NewHeightStep!
	cs.updateToState(stateCopy)

	//fail.Fail() // XXX

	// cs.StartTime is already set.
	// Schedule Round0 to start soon.
	cs.scheduleRound0(&cs.RoundState)

	//cs.client.ConsResult() <- cs.Height

	// By here,
	// * cs.Height has been increment to height+1
	// * cs.Step is now ttypes.RoundStepNewHeight
	// * cs.StartTime is set to when we will start round0.
	// Execute and commit the block, update and save the state, and update the mempool.

}

//-----------------------------------------------------------------------------


func (cs *ConsensusState) defaultSetProposal(proposal *types.Proposal) error {
	tendermintlog.Info(fmt.Sprintf("Consensus receive proposal. Current: %v/%v/%v", cs.Height, cs.Round, cs.Step), "proposal", fmt.Sprintf("%v/%v", proposal.Height, proposal.Round))
	// Already have one
	// TODO: possibly catch double proposals
	if cs.Proposal != nil {
		tendermintlog.Debug("defaultSetProposal: proposal is not nil")
		return nil
	}

	// Does not apply
	if proposal.Height != cs.Height || int(proposal.Round) != cs.Round {
		tendermintlog.Error("defaultSetProposal:height is not equal or round is not equal", "proposal-height", proposal.Height, "cs-height", cs.Height, "proposal-round", proposal.Round, "cs-round", cs.Round)
		return nil
	}

	// We don't care about the proposal if we're already in ttypes.RoundStepCommit.
	if ttypes.RoundStepCommit <= cs.Step {
		//tendermintlog.Error("defaultSetProposal:", "msg", "step not equal")
		//return nil
	}

	// Verify POLRound, which must be -1 or between 0 and proposal.Round exclusive.
	if proposal.POLRound != -1 &&
		(proposal.POLRound < 0 || proposal.Round <= proposal.POLRound) {
		return ErrInvalidProposalPOLRound
	}

	// Verify signature
	pubkey, err := ttypes.ConsensusCrypto.PubKeyFromBytes(cs.Validators.GetProposer().PubKey)
	if err != nil {
		return errors.New(fmt.Sprintf("Error pubkey from bytes:%v", err))
	}
	proposalTmp := &ttypes.Proposal{Proposal:*proposal}
	signature, err := ttypes.ConsensusCrypto.SignatureFromBytes(proposal.Signature)
	if err != nil {
		return fmt.Errorf("defaultSetProposal Error: SIGA[%v] to signature failed:%v", proposal.Signature, err)
	}
	if !pubkey.VerifyBytes(ttypes.SignBytes(cs.state.ChainID, proposalTmp), signature) {
		return ErrInvalidProposalSignature
	}
	cs.Proposal = proposal

	cs.ProposalBlock = &ttypes.TendermintBlock{TendermintBlock: proposal.Block}

	// NOTE: it's possible to receive complete proposal blocks for future rounds without having the proposal
	tendermintlog.Error(fmt.Sprintf("Set proposal block. Current: %v/%v/%v", cs.Height, cs.Round, cs.Step), "height", cs.ProposalBlock.Header.Height,
		"hash", fmt.Sprintf("%X", cs.ProposalBlock.Hash()))

	if cs.isProposer() {
		//cs.broadcastChannel <- MsgInfo{TypeID:ttypes.ProposalID, Msg:cs.Proposal, PeerID:cs.ourId, PeerIP:""}
	}

	if cs.Step <= ttypes.RoundStepPropose {
		if cs.Round == 0 {
			// receive proposal before txsAvailable, reset begCons
			cs.begCons = time.Now()
		}
		// Move onto the next step
		cs.enterPrevote(cs.Height, cs.Round)
	} else if cs.Step == ttypes.RoundStepCommit {
		// If we're waiting on the proposal block...
		cs.tryFinalizeCommit(cs.Height)
	}
	return nil
}

// Attempt to add the vote. if its a duplicate signature, dupeout the validator
func (cs *ConsensusState) tryAddVote(voteRaw *types.Vote, peerID string) error {
	vote := &ttypes.Vote{Vote:voteRaw}
	_, err := cs.addVote(vote, peerID)
	if err != nil {
		// If the vote height is off, we'll just ignore it,
		// But if it's a conflicting sig, add it to the cs.evpool.
		// If it's otherwise invalid, punish peer.
		if err == ErrVoteHeightMismatch {
			return err
		} else if voteErr, ok := err.(*ttypes.ErrVoteConflictingVotes); ok {
			if bytes.Equal(vote.ValidatorAddress, cs.privValidator.GetAddress()) {
				tendermintlog.Error("Found conflicting vote from ourselves. Did you unsafe_reset a validator?", "height", vote.Height, "round", vote.Round, "type", vote.Type)
				return err
			}
			cs.evpool.AddEvidence(voteErr.DuplicateVoteEvidence)
			return err
		} else {
			// Probably an invalid signature / Bad peer.
			// Seems this can also err sometimes with "Unexpected step" - perhaps not from a bad peer ?
			tendermintlog.Error("Error attempting to add vote", "err", err)
			return ErrAddingVote
		}
	}
	return nil
}

//-----------------------------------------------------------------------------

func (cs *ConsensusState) addVote(vote *ttypes.Vote, peerID string) (added bool, err error) {
	tendermintlog.Debug("addVote", "voteHeight", vote.Height, "voteType", vote.Type, "valIndex", vote.ValidatorIndex, "csHeight", cs.Height, "peerid", peerID)

	// A precommit for the previous height?
	// These come in while we wait timeoutCommit
	if vote.Height+1 == cs.Height {
		if !(cs.Step == ttypes.RoundStepNewHeight && vote.Type == uint32(ttypes.VoteTypePrecommit)) {
			// TODO: give the reason ..
			// fmt.Errorf("tryAddVote: Wrong height, not a LastCommit straggler commit.")
			return added, ErrVoteHeightMismatch
		}
		added, err = cs.LastCommit.AddVote(vote)
		if !added {
			return added, err
		}

		cs.broadcastChannel <- MsgInfo{TypeID:ttypes.VoteID, Msg:vote.Vote, PeerID:cs.ourId, PeerIP:""}

		// if we can skip timeoutCommit and have all the votes now,
		if cs.client.Cfg.SkipTimeoutCommit && cs.LastCommit.HasAll() {
			// go straight to new round (skip timeout commit)
			// cs.scheduleTimeout(time.Duration(0), cs.Height, 0, ttypes.RoundStepNewHeight)
			cs.enterNewRound(cs.Height, 0)
		}
		return
	}

	// A prevote/precommit for this height?
	if vote.Height == cs.Height {
		height := cs.Height
		added, err = cs.Votes.AddVote(vote, peerID)
		if added {
			cs.broadcastChannel <- MsgInfo{TypeID:ttypes.VoteID, Msg:vote.Vote, PeerID:cs.ourId, PeerIP:""}

			switch vote.Type {
			case uint32(ttypes.VoteTypePrevote):
				prevotes := cs.Votes.Prevotes(int(vote.Round))
				tendermintlog.Info("Added to prevote", "vote", vote, "prevotes", prevotes.StringShort())

				// If +2/3 prevotes for a block or nil for *any* round:
				if blockID, ok := prevotes.TwoThirdsMajority(); ok {

					// There was a polka!
					// If we're locked but this is a recent polka, unlock.
					// If it matches our ProposalBlock, update the ValidBlock

					// Unlock if `cs.LockedRound < vote.Round <= cs.Round`
					// NOTE: If vote.Round > cs.Round, we'll deal with it when we get to vote.Round
					if (cs.LockedBlock != nil) &&
						(cs.LockedRound < int(vote.Round)) &&
						(int(vote.Round) <= cs.Round) &&
						!cs.LockedBlock.HashesTo(blockID.Hash) {

						tendermintlog.Info("Unlocking because of POL.", "lockedRound", cs.LockedRound, "POLRound", vote.Round)
						cs.LockedRound = 0
						cs.LockedBlock = nil
						//cs.LockedBlockParts = nil
						//cs.eventBus.PublishEventUnlock(cs.RoundStateEvent())
					}
				}

				// If +2/3 prevotes for *anything* for this or future round:
				if cs.Round <= int(vote.Round) && prevotes.HasTwoThirdsAny() {
					// Round-skip over to PrevoteWait or goto Precommit.
					cs.enterNewRound(height, int(vote.Round)) // if the vote is ahead of us
					if prevotes.HasTwoThirdsMajority() {
						cs.enterPrecommit(height, int(vote.Round))
					} else {
						cs.enterPrevote(height, int(vote.Round)) // if the vote is ahead of us
						cs.enterPrevoteWait(height, int(vote.Round))
					}
				} else if cs.Proposal != nil && 0 <= cs.Proposal.POLRound && cs.Proposal.POLRound == vote.Round {
					// If the proposal is now complete, enter prevote of cs.Round.
					if cs.isProposalComplete() {
						cs.enterPrevote(height, cs.Round)
					}
				}

			case uint32(ttypes.VoteTypePrecommit):
				precommits := cs.Votes.Precommits(int(vote.Round))
				tendermintlog.Info("Added to precommit", "vote", vote, "precommits", precommits.StringShort())
				blockID, ok := precommits.TwoThirdsMajority()
				if ok {
					if len(blockID.Hash) == 0 {
						cs.enterNewRound(height, int(vote.Round)+1)
					} else {
						cs.enterNewRound(height, int(vote.Round))
						cs.enterPrecommit(height, int(vote.Round))
						cs.enterCommit(height, int(vote.Round))

						if cs.client.Cfg.SkipTimeoutCommit && precommits.HasAll() {
							// if we have all the votes now,
							// go straight to new round (skip timeout commit)
							// cs.scheduleTimeout(time.Duration(0), cs.Height, 0, cstypes.RoundStepNewHeight)
							cs.enterNewRound(cs.Height, 0)
						}

					}
				} else if cs.Round <= int(vote.Round) && precommits.HasAll() {
					// workaround: need have all the votes
					cs.enterNewRound(height, int(vote.Round))
					cs.enterPrecommit(height, int(vote.Round))
					cs.enterPrecommitWait(height, int(vote.Round))
				}

			default:
				panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf("Unexpected vote type %X", vote.Type))) // Should not happen.
			}
		}
		// Either duplicate, or error upon cs.Votes.AddByIndex()
		return
	} else {
		err = ErrVoteHeightMismatch
	}

	// Height mismatch, bad peer?
	tendermintlog.Info("Vote ignored and not added", "voteType", vote.Type, "voteHeight", vote.Height, "csHeight", cs.Height, "err", err)
	return
}

func (cs *ConsensusState) signVote(type_ byte, hash []byte) (*ttypes.Vote, error) {
	addr := cs.privValidator.GetAddress()
	valIndex, _ := cs.Validators.GetByAddress(addr)
	tVote := &types.Vote{}
	tVote.BlockID = &types.BlockID{Hash: hash}
	vote := &ttypes.Vote{
		Vote: &types.Vote{
		ValidatorAddress: addr,
		ValidatorIndex:   int32(valIndex),
		Height:           cs.Height,
		Round:            int32(cs.Round),
		Timestamp:        time.Now().UnixNano(),
		Type:             uint32(type_),
		BlockID:          &types.BlockID{Hash: hash},
		Signature:nil,
	},
	}
	err := cs.privValidator.SignVote(cs.state.ChainID, vote)
	return vote, err
}

// sign the vote and publish on internalMsgQueue
func (cs *ConsensusState) signAddVote(type_ byte, hash []byte) *ttypes.Vote {
	// if we don't have a key or we're not in the validator set, do nothing
	if cs.privValidator == nil || !cs.Validators.HasAddress(cs.privValidator.GetAddress()) {
		return nil
	}
	vote, err := cs.signVote(type_, hash)
	if err == nil {
		cs.sendInternalMessage(MsgInfo{TypeID:ttypes.VoteID, Msg: vote.Vote, PeerID:cs.ourId, PeerIP:""})
		tendermintlog.Info("Signed and pushed vote", "height", cs.Height, "round", cs.Round, "vote", vote, "err", err)
		return vote
	}

	tendermintlog.Error("Error signing vote", "height", cs.Height, "round", cs.Round, "vote", vote, "err", err)
	return nil

}

//---------------------------------------------------------

func CompareHRS(h1 int64, r1 int, s1 ttypes.RoundStepType, h2 int64, r2 int, s2 ttypes.RoundStepType) int {
	if h1 < h2 {
		return -1
	} else if h1 > h2 {
		return 1
	}
	if r1 < r2 {
		return -1
	} else if r1 > r2 {
		return 1
	}
	if s1 < s2 {
		return -1
	} else if s1 > s2 {
		return 1
	}
	return 0
}

// Commit returns the amount of time to wait for straggler votes after receiving +2/3 precommits for a single block (ie. a commit).
func (cs *ConsensusState) Commit(t time.Time) time.Time {
	return t.Add(time.Duration(cs.client.Cfg.TimeoutCommit) * time.Millisecond)
}

// Propose returns the amount of time to wait for a proposal
func (cs *ConsensusState) Propose(round int) time.Duration {
	return time.Duration(cs.client.Cfg.TimeoutPropose+cs.client.Cfg.TimeoutProposeDelta*int32(round)) * time.Millisecond
}

// Prevote returns the amount of time to wait for straggler votes after receiving any +2/3 prevotes
func (cs *ConsensusState) Prevote(round int) time.Duration {
	return time.Duration(cs.client.Cfg.TimeoutPrevote+cs.client.Cfg.TimeoutPrevoteDelta*int32(round)) * time.Millisecond
}

// Precommit returns the amount of time to wait for straggler votes after receiving any +2/3 precommits
func (cs *ConsensusState) Precommit(round int) time.Duration {
	return time.Duration(cs.client.Cfg.TimeoutPrecommit+cs.client.Cfg.TimeoutPrecommitDelta*int32(round)) * time.Millisecond
}

// WaitForTxs returns true if the consensus should wait for transactions before entering the propose step
func (cs *ConsensusState) WaitForTxs() bool {
	return !cs.client.Cfg.CreateEmptyBlocks || cs.client.Cfg.CreateEmptyBlocksInterval > 0
}

// EmptyBlocks returns the amount of time to wait before proposing an empty block or starting the propose timer if there are no txs available
func (cs *ConsensusState) EmptyBlocksInterval() time.Duration {
	return time.Duration(cs.client.Cfg.CreateEmptyBlocksInterval) * time.Second
}

// PeerGossipSleep returns the amount of time to sleep if there is nothing to send from the ConsensusReactor
func (cs *ConsensusState) PeerGossipSleep() time.Duration {
	return time.Duration( /*cs.client.Cfg.PeerGossipSleepDuration*/ 100) * time.Millisecond
}

// PeerQueryMaj23Sleep returns the amount of time to sleep after each VoteSetMaj23Message is sent in the ConsensusReactor
func (cs *ConsensusState) PeerQueryMaj23Sleep() time.Duration {
	return time.Duration( /*cs.PeerQueryMaj23SleepDuration*/ 2000) * time.Millisecond
}

func (cs *ConsensusState) IsProposer() bool {
	return cs.isProposer()
}

func (cs *ConsensusState) GetPrevotesState(height int64, round int, blockID *types.BlockID) *ttypes.BitArray {
	if height != cs.Height {
		return nil
	}
	return cs.Votes.Prevotes(round).BitArrayByBlockID(blockID)
}

func (cs *ConsensusState) GetPrecommitsState(height int64, round int, blockID *types.BlockID) *ttypes.BitArray {
	if height != cs.Height {
		return nil
	}
	return cs.Votes.Precommits(round).BitArrayByBlockID(blockID)
}

func (cs *ConsensusState) SetPeerMaj23(height int64, round int, type_ byte, peerID ID, blockID *types.BlockID) {
	if height == cs.Height {
		cs.Votes.SetPeerMaj23(round, type_, string(peerID), blockID)
	}
}

func (cs *ConsensusState) checkProposalToCommit() {
	for {
		select {
		case info, ok := <-cs.needCommitByInfo:
			if ok {
				if cs.Proposal != nil {
					cs.enterCommit(info.height, info.round)
				}
			} else {
				tendermintlog.Error("checkProposalToCommit channel closed")
				cs.needCommitByInfo = nil
				return
			}
		}
	}
}
