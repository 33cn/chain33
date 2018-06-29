package tendermint

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"

	cmn "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/common"

	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/p2p"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"encoding/json"
)

const (
	StateChannel       = byte(0x20)
	DataChannel        = byte(0x21)
	VoteChannel        = byte(0x22)
	VoteSetBitsChannel = byte(0x23)
)

var (
	ErrAlreadyStarted = errors.New("already started")
	ErrAlreadyStopped = errors.New("already stopped")
)

//-----------------------------------------------------------------------------

// ConsensusReactor defines a reactor for the consensus service.
type ConsensusReactor struct {
	p2p.BaseReactor // BaseService + p2p.Switch

	conS *ConsensusState

	mtx      sync.RWMutex
	fastSync bool

	eventBus *types.EventBus
}

// NewConsensusReactor returns a new ConsensusReactor with the given consensusState.
func NewConsensusReactor(consensusState *ConsensusState, fastSync bool) *ConsensusReactor {
	conR := &ConsensusReactor{
		conS:     consensusState,
		fastSync: fastSync,

		eventBus: nil,
	}

	conR.BaseReactor = *p2p.NewBaseReactor("ConsensusReactor", conR)
	return conR
}

// OnStart implements BaseService.
func (conR *ConsensusReactor) OnStart() error {
	if err := conR.BaseReactor.OnStart(); err != nil {
		return err
	}
	err := conR.startBroadcastRoutine()
	if err != nil {
		return err
	}

	if !conR.FastSync() {
		err := conR.conS.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

// OnStop implements BaseService
func (conR *ConsensusReactor) OnStop() {
	conR.BaseReactor.OnStop()
	conR.conS.Stop()
}

// SwitchToConsensus switches from fast_sync mode to consensus mode.
// It resets the state, turns off fast_sync, and starts the consensus state-machine
func (conR *ConsensusReactor) SwitchToConsensus(state State, blocksSynced int) {
	conR.conS.reconstructLastCommit(state)
	// NOTE: The line below causes broadcastNewRoundStepRoutine() to
	// broadcast a types.NewRoundStepMessage.
	conR.conS.updateToState(state)

	conR.mtx.Lock()
	conR.fastSync = false
	conR.mtx.Unlock()

	if blocksSynced > 0 {
		// dont bother with the WAL if we fast synced
		conR.conS.doWALCatchup = false
	}
	err := conR.conS.Start()
	if err != nil {
		tendermintlog.Error("Error starting conS", "err", err)
	}
}

// GetChannels implements Reactor
func (conR *ConsensusReactor) GetChannels() []*p2p.ChannelDescriptor {
	// TODO optimize
	return []*p2p.ChannelDescriptor{
		{
			ID:                StateChannel,
			Priority:          5,
			SendQueueCapacity: 100,
		},
		{
			ID:                 DataChannel, // maybe split between gossiping current block and catchup stuff
			Priority:           10,          // once we gossip the whole block there's nothing left to send until next height or round
			SendQueueCapacity:  10240,		 // 100
			RecvBufferCapacity: 50 * 4096,
		},
		{
			ID:                 VoteChannel,
			Priority:           5,
			SendQueueCapacity:  100,
			RecvBufferCapacity: 100 * 100,
		},
		{
			ID:                 VoteSetBitsChannel,
			Priority:           1,
			SendQueueCapacity:  2,
			RecvBufferCapacity: 1024,
		},
	}
}

// AddPeer implements Reactor
func (conR *ConsensusReactor) AddPeer(peer p2p.Peer) {

	if !conR.IsRunning() {
		return
	}

	// Create peerState for peer
	peerState := NewPeerState(peer)
	peer.Set(types.PeerStateKey, peerState)

	// Begin routines for this peer.
	go conR.gossipDataRoutine(peer, peerState)
	go conR.gossipVotesRoutine(peer, peerState)
	go conR.queryMaj23Routine(peer, peerState)

	// Send our state to peer.
	// If we're fast_syncing, broadcast a RoundStepMessage later upon SwitchToConsensus().
	if !conR.FastSync() {
		conR.sendNewRoundStepMessages(peer)
	}
}

// RemovePeer implements Reactor
func (conR *ConsensusReactor) RemovePeer(peer p2p.Peer, reason interface{}) {

	if !conR.IsRunning() {
		return
	}

	// TODO
	//peer.Get(PeerStateKey).(*PeerState).Disconnect()
}

// Receive implements Reactor
// NOTE: We process these messages even when we're fast_syncing.
// Messages affect either a peer state or the consensus state.
// Peer state updates can happen in parallel, but processing of
// proposals, block parts, and votes are ordered by the receiveRoutine
// NOTE: blocks on consensus state for proposals, block parts, and votes
func (conR *ConsensusReactor) Receive(chID byte, src p2p.Peer, msgBytes []byte) {
	if !conR.IsRunning() {
		tendermintlog.Error("Receive", "src", src, "chId", chID, "bytes", msgBytes)
		return
	}
	envelope := types.MsgEnvelope{}
	err := json.Unmarshal(msgBytes, &envelope)

	//_, msg, err := DecodeMessage(msgBytes)
	if err != nil {
		tendermintlog.Error("Error decoding message", "src", src, "chId", chID, "err", err, "bytes", msgBytes)
		// TODO punish peer?
		return
	}

	//tendermintlog.Debug("Receive", "src", src, "chId", chID, "msg", envelope.Kind)

	if v, ok := types.MsgMap[envelope.Kind]; ok {
		msg := v.(types.ReactorMsg).Copy()
		err = json.Unmarshal(*envelope.Data, &msg)
		if err != nil {
			tendermintlog.Error("ConsensusReactor Receive Unmarshal data failed:%v\n", err)
			return
		}
		// Get peer states
		ps := src.Get(types.PeerStateKey).(*PeerState)
		switch chID {
		case StateChannel:
			switch msg := msg.(type) {
			case *types.NewRoundStepMessage:
				ps.ApplyNewRoundStepMessage(msg)
			case *types.CommitStepMessage:
				ps.ApplyCommitStepMessage(msg)
			case *types.HasVoteMessage:
				ps.ApplyHasVoteMessage(msg)
			case *types.VoteSetMaj23Message:
				cs := conR.conS
				cs.mtx.Lock()
				height, votes := cs.Height, cs.Votes
				cs.mtx.Unlock()
				if height != msg.Height {
					return
				}
				// Peer claims to have a maj23 for some BlockID at H,R,S,
				votes.SetPeerMaj23(msg.Round, msg.Type, ps.Peer.Key(), msg.BlockID)
				// Respond with a types.VoteSetBitsMessage showing which votes we have.
				// (and consequently shows which we don't have)
				var ourVotes *cmn.BitArray
				switch msg.Type {
				case types.VoteTypePrevote:
					ourVotes = votes.Prevotes(msg.Round).BitArrayByBlockID(msg.BlockID)
				case types.VoteTypePrecommit:
					ourVotes = votes.Precommits(msg.Round).BitArrayByBlockID(msg.BlockID)
				default:
					tendermintlog.Error("Bad types.VoteSetBitsMessage field Type")
					return
				}
				src.TrySend(VoteSetBitsChannel, &types.VoteSetBitsMessage{
					Height:  msg.Height,
					Round:   msg.Round,
					Type:    msg.Type,
					BlockID: msg.BlockID,
					Votes:   ourVotes,
				})
			case *types.ProposalHeartbeatMessage:
				hb := msg.Heartbeat
				tendermintlog.Debug("Received proposal heartbeat message",
					"height", hb.Height, "round", hb.Round, "sequence", hb.Sequence,
					"valIdx", hb.ValidatorIndex, "valAddr", hb.ValidatorAddress)
			default:
				tendermintlog.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
			}

		case DataChannel:
			if conR.FastSync() {
				tendermintlog.Info("Ignoring message received during fastSync", "msg", msg)
				return
			}

			switch msg := msg.(type) {
			case *types.ProposalMessage:
				ps.SetHasProposal(msg.Proposal)
				conR.conS.peerMsgQueue <- msgInfo{msg, src.Key()}
			case *types.ProposalPOLMessage:
				ps.ApplyProposalPOLMessage(msg)
			default:
				tendermintlog.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
			}

		case VoteChannel:
			if conR.FastSync() {
				tendermintlog.Info("Ignoring message received during fastSync", "msg", msg)
				return
			}

			switch msg := msg.(type) {
			case *types.VoteMessage:
				cs := conR.conS
				cs.mtx.Lock()
				height, valSize, lastCommitSize := cs.Height, cs.Validators.Size(), cs.LastCommit.Size()
				cs.mtx.Unlock()
				ps.EnsureVoteBitArrays(height, valSize)
				ps.EnsureVoteBitArrays(height-1, lastCommitSize)
				ps.SetHasVote(msg.Vote)
				cs.peerMsgQueue <- msgInfo{msg, src.Key()}

			default:
				// don't punish (leave room for soft upgrades)
				tendermintlog.Error(cmn.Fmt("Unknown message type %v", reflect.TypeOf(msg)))
			}

		case VoteSetBitsChannel:
			if conR.FastSync() {
				tendermintlog.Info("Ignoring message received during fastSync", "msg", msg)
				return
			}
			switch msg := msg.(type) {
			case *types.VoteSetBitsMessage:
				cs := conR.conS
				cs.mtx.Lock()
				height, votes := cs.Height, cs.Votes
				cs.mtx.Unlock()

				if height == msg.Height {
					var ourVotes *cmn.BitArray
					switch msg.Type {
					case types.VoteTypePrevote:
						ourVotes = votes.Prevotes(msg.Round).BitArrayByBlockID(msg.BlockID)
					case types.VoteTypePrecommit:
						ourVotes = votes.Precommits(msg.Round).BitArrayByBlockID(msg.BlockID)
					default:
						tendermintlog.Error("Bad types.VoteSetBitsMessage field Type")
						return
					}
					ps.ApplyVoteSetBitsMessage(msg, ourVotes)
				} else {
					ps.ApplyVoteSetBitsMessage(msg, nil)
				}
			default:
				// don't punish (leave room for soft upgrades)
				tendermintlog.Error(cmn.Fmt("Unknown message type %v", msg.TypeName()))
			}

		default:
			tendermintlog.Error(cmn.Fmt("Unknown chId %X", chID))
		}
	} else {
	tendermintlog.Error("not find ReactorMsg kind %v", envelope.Kind)
	}
}

// SetEventBus sets event bus.
func (conR *ConsensusReactor) SetEventBus(b *types.EventBus) {
	conR.eventBus = b
	conR.conS.SetEventBus(b)
}

// FastSync returns whether the consensus reactor is in fast-sync mode.
func (conR *ConsensusReactor) FastSync() bool {
	conR.mtx.RLock()
	defer conR.mtx.RUnlock()
	return conR.fastSync
}

//--------------------------------------

// startBroadcastRoutine subscribes for new round steps, votes and proposal
// heartbeats using the event bus and starts a go routine to broadcasts events
// to peers upon receiving them.
func (conR *ConsensusReactor) startBroadcastRoutine() error {
	const subscriber = "consensus-reactor"
	ctx := context.Background()

	// new round steps
	stepsCh := make(chan interface{})
	err := conR.eventBus.Subscribe(ctx, subscriber, &types.SimpleEventQuery{Name: "tm.event=NewRoundStep"}, stepsCh)
	if err != nil {
		return errors.Wrapf(err, "failed to subscribe %s to NewRoundgoStep", subscriber)
	}

	// votes
	votesCh := make(chan interface{})
	err = conR.eventBus.Subscribe(ctx, subscriber, &types.SimpleEventQuery{Name: "tm.event=Vote"}, votesCh)
	if err != nil {
		return errors.Wrapf(err, "failed to subscribe %s to Vote", subscriber)
	}

	// proposal heartbeats
	heartbeatsCh := make(chan interface{})
	err = conR.eventBus.Subscribe(ctx, subscriber, &types.SimpleEventQuery{Name: "tm.event=ProposalHeartbeat"}, heartbeatsCh)
	if err != nil {
		return errors.Wrapf(err, "failed to subscribe %s to ProposalHeartbeat", subscriber)
	}

	go func() {
		for {
			select {
			case data, ok := <-stepsCh:
				if ok { // a receive from a closed channel returns the zero value immediately
					edrs := data.(types.TMEventData).Unwrap().(types.EventDataRoundState)
					conR.broadcastNewRoundStep(edrs.RoundState.(*types.RoundState))
				}
			case data, ok := <-votesCh:
				if ok {
					edv := data.(types.TMEventData).Unwrap().(types.EventDataVote)
						conR.broadcastHasVoteMessage(edv.Vote)
				}
			case data, ok := <-heartbeatsCh:
				if ok {
					edph := data.(types.TMEventData).Unwrap().(types.EventDataProposalHeartbeat)
					conR.broadcastProposalHeartbeatMessage(edph)
				}
			case <-conR.Quit:
				conR.eventBus.UnsubscribeAll(ctx, subscriber)
				return
			}
		}
	}()

	return nil
}

func (conR *ConsensusReactor) broadcastProposalHeartbeatMessage(heartbeat types.EventDataProposalHeartbeat) {
	hb := heartbeat.Heartbeat
	tendermintlog.Debug("Broadcasting proposal heartbeat message",
		"height", hb.Height, "round", hb.Round, "sequence", hb.Sequence)
	msg := &types.ProposalHeartbeatMessage{hb}
	conR.Switch.Broadcast(StateChannel, msg)
}

func (conR *ConsensusReactor) broadcastNewRoundStep(rs *types.RoundState) {
	nrsMsg, csMsg := makeRoundStepMessages(rs)
	if nrsMsg != nil {
		conR.Switch.Broadcast(StateChannel, nrsMsg)
	}
	if csMsg != nil {
		conR.Switch.Broadcast(StateChannel, csMsg)
	}
}

// Broadcasts types.HasVoteMessage to peers that care.
func (conR *ConsensusReactor) broadcastHasVoteMessage(vote *types.Vote) {
	msg := &types.HasVoteMessage{
		Height: vote.Height,
		Round:  vote.Round,
		Type:   vote.Type,
		Index:  vote.ValidatorIndex,
	}
	conR.Switch.Broadcast(StateChannel, msg)
	/*
		// TODO: Make this broadcast more selective.
		for _, peer := range conR.Switch.Peers().List() {
			ps := peer.Get(PeerStateKey).(*PeerState)
			prs := ps.GetRoundState()
			if prs.Height == vote.Height {
				// TODO: Also filter on round?
				peer.TrySend(StateChannel, struct{ ConsensusMessage }{msg})
			} else {
				// Height doesn't match
				// TODO: check a field, maybe CatchupCommitRound?
				// TODO: But that requires changing the struct field comment.
			}
		}
	*/
}

func makeRoundStepMessages(rs *types.RoundState) (nrsMsg *types.NewRoundStepMessage, csMsg *types.CommitStepMessage) {
	nrsMsg = &types.NewRoundStepMessage{
		Height: rs.Height,
		Round:  rs.Round,
		Step:   rs.Step,
		SecondsSinceStartTime: int(time.Since(rs.StartTime).Seconds()),
		LastCommitRound:       rs.LastCommit.Round(),
	}
	if rs.Step == types.RoundStepCommit {
		csMsg = &types.CommitStepMessage{
			Height:           rs.Height,
		}
	}
	return
}

func (conR *ConsensusReactor) sendNewRoundStepMessages(peer p2p.Peer) {
	rs := conR.conS.GetRoundState()
	nrsMsg, csMsg := makeRoundStepMessages(rs)
	if nrsMsg != nil {
		peer.Send(StateChannel, nrsMsg)
	}
	if csMsg != nil {
		peer.Send(StateChannel, csMsg)
	}
}

func (conR *ConsensusReactor) gossipDataRoutine(peer p2p.Peer, ps *PeerState) {

OUTER_LOOP:
	for {
		// Manage disconnects from self or peer.

		if !peer.IsRunning() || !conR.IsRunning() {
			tendermintlog.Error("Stopping gossipDataRoutine for peer")
			return
		}

		rs := conR.conS.GetRoundState()
		prs := ps.GetRoundState()

		// If height and round don't match, sleep.
		if (rs.Height != prs.Height) || (rs.Round != prs.Round) {
			//tendermintlog.Debug("Peer Height|Round mismatch, sleeping", "myheight", rs.Height, "myround", rs.Round, "peerHeight", prs.Height, "peerRound", prs.Round, "peer", peer.Key())
			time.Sleep(conR.conS.PeerGossipSleep())
			continue OUTER_LOOP
		}

		// By here, height and round match.
		// Proposal block parts were already matched and sent if any were wanted.
		// (These can match on hash so the round doesn't matter)
		// Now consider sending other things, like the Proposal itself.

		// Send Proposal && ProposalPOL BitArray?
		if rs.Proposal != nil && !prs.Proposal{
			// Proposal: share the proposal metadata with peer.
			{
				proposalTrans := types.ProposalToProposalTrans(rs.Proposal)
				msg := &types.ProposalMessage{Proposal: proposalTrans}
				tendermintlog.Debug("Sending proposal", "peer",peer.Key(), "height", prs.Height, "round", prs.Round)
				if peer.Send(DataChannel, msg) {
					ps.SetHasProposal(proposalTrans)
				}
			}
			// ProposalPOL: lets peer know which POL votes we have so far.
			// Peer must receive types.ProposalMessage first.
			// rs.Proposal was validated, so rs.Proposal.POLRound <= rs.Round,
			// so we definitely have rs.Votes.Prevotes(rs.Proposal.POLRound).
			if 0 <= rs.Proposal.POLRound {
				msg := &types.ProposalPOLMessage{
					Height:           rs.Height,
					ProposalPOLRound: rs.Proposal.POLRound,
					ProposalPOL:      rs.Votes.Prevotes(rs.Proposal.POLRound).BitArray(),
				}
				tendermintlog.Debug("Sending POL", "height", prs.Height, "round", prs.Round)
				peer.Send(DataChannel, msg)
			}
			continue OUTER_LOOP
		}

		// Nothing to do. Sleep.
		time.Sleep(conR.conS.PeerGossipSleep())
		continue OUTER_LOOP
	}
}

func (conR *ConsensusReactor) gossipVotesRoutine(peer p2p.Peer, ps *PeerState) {
	// Simple hack to throttle logs upon sleep.
	var sleeping = 0

OUTER_LOOP:
	for {
		// Manage disconnects from self or peer.
		if !peer.IsRunning() || !conR.IsRunning() {
			tendermintlog.Info("Stopping gossipVotesRoutine for peer")
			return
		}

		rs := conR.conS.GetRoundState()
		prs := ps.GetRoundState()

		switch sleeping {
		case 1: // First sleep
			sleeping = 2
		case 2: // No more sleep
			sleeping = 0
		}

		//logger.Debug("gossipVotesRoutine", "rsHeight", rs.Height, "rsRound", rs.Round,
		//	"prsHeight", prs.Height, "prsRound", prs.Round, "prsStep", prs.Step)

		// If height matches, then send LastCommit, Prevotes, Precommits.
		if rs.Height == prs.Height {
			if conR.gossipVotesForHeight(rs, prs, ps) {
				continue OUTER_LOOP
			}
		}

		if (0 < prs.Height) && (prs.Height < rs.Height) && !prs.Proposal {
			proposal, err := conR.conS.LastProposals.QueryElem(prs.Height)
			if err == nil && proposal != nil {
				proposalTrans := types.ProposalToProposalTrans(proposal)
				msg := &types.ProposalMessage{Proposal: proposalTrans}
				tendermintlog.Debug("peer height behind, Sending old proposal", "peer",peer.Key(), "height", prs.Height, "round", prs.Round, "proopsal-height", proposalTrans.Height)
				if peer.Send(DataChannel, msg) {
					ps.SetHasProposal(proposalTrans)
					time.Sleep(conR.conS.PeerGossipSleep())
				} else {
					tendermintlog.Error("peer height behind Sending old proposal failed")
				}
			}
			continue OUTER_LOOP
		}

		// Special catchup logic.
		// If peer is lagging by height 1, send LastCommit.
		if prs.Height != 0 && rs.Height == prs.Height+1 {
			if ps.PickSendVote(rs.LastCommit) {
				continue OUTER_LOOP
			}
		}

		// Catchup logic
		// If peer is lagging by more than 1, send Commit.
		if prs.Height != 0 && rs.Height >= prs.Height+2 {
			// Load the block commit for prs.Height,
			// which contains precommit signatures for prs.Height.
			commit := conR.conS.blockStore.LoadBlockCommit(prs.Height + 1)
			if ps.PickSendVote(commit) {
				continue OUTER_LOOP
			}
		}

		if sleeping == 0 {
			// We sent nothing. Sleep...
			sleeping = 1
			tendermintlog.Debug("No votes to send, sleeping", "peer", peer.Key(), "rs.Height", rs.Height, "prs.Height", prs.Height,
				"localPV", rs.Votes.Prevotes(rs.Round).BitArray(), "peerPV", prs.Prevotes,
				"localPC", rs.Votes.Precommits(rs.Round).BitArray(), "peerPC", prs.Precommits)
		} else if sleeping == 2 {
			// Continued sleep...
			sleeping = 1
		}

		time.Sleep(conR.conS.PeerGossipSleep())
		continue OUTER_LOOP
	}
}

func (conR *ConsensusReactor) gossipVotesForHeight(rs *types.RoundState, prs *types.PeerRoundState, ps *PeerState) bool {

	// If there are lastCommits to send...
	if prs.Step == types.RoundStepNewHeight {
		if ps.PickSendVote(rs.LastCommit) {
			tendermintlog.Debug("Picked rs.LastCommit to send")
			return true
		}
	}
	if !prs.Proposal && rs.Proposal != nil{
		tendermintlog.Info("peer proposal not set sleeping")
		time.Sleep(conR.conS.PeerGossipSleep())
		return true
	}
	// If there are prevotes to send...
	if prs.Step <= types.RoundStepPrevote && prs.Round != -1 && prs.Round <= rs.Round {
		if ps.PickSendVote(rs.Votes.Prevotes(prs.Round)) {
			tendermintlog.Debug("Picked rs.Prevotes(prs.Round) to send", "round", prs.Round)
			return true
		}
	}
	// If there are precommits to send...
	if prs.Step <= types.RoundStepPrecommit && prs.Round != -1 && prs.Round <= rs.Round {
		if ps.PickSendVote(rs.Votes.Precommits(prs.Round)) {
			tendermintlog.Debug("Picked rs.Precommits(prs.Round) to send", "round", prs.Round)
			return true
		}
	}
	// If there are POLPrevotes to send...
	if prs.ProposalPOLRound != -1 {
		if polPrevotes := rs.Votes.Prevotes(prs.ProposalPOLRound); polPrevotes != nil {
			if ps.PickSendVote(polPrevotes) {
				tendermintlog.Debug("Picked rs.Prevotes(prs.ProposalPOLRound) to send", "round", prs.ProposalPOLRound)
				return true
			}
		}
	}
	return false
}

// NOTE: `queryMaj23Routine` has a simple crude design since it only comes
// into play for liveness when there's a signature DDoS attack happening.
func (conR *ConsensusReactor) queryMaj23Routine(peer p2p.Peer, ps *PeerState) {

OUTER_LOOP:
	for {
		// Manage disconnects from self or peer.
		if !peer.IsRunning() || !conR.IsRunning() {
			tendermintlog.Info("Stopping queryMaj23Routine for peer")
			return
		}

		// Maybe send Height/Round/Prevotes
		{
			rs := conR.conS.GetRoundState()
			prs := ps.GetRoundState()
			if rs.Height == prs.Height {
				if maj23, ok := rs.Votes.Prevotes(prs.Round).TwoThirdsMajority(); ok {
					peer.TrySend(StateChannel, &types.VoteSetMaj23Message{
						Height:  prs.Height,
						Round:   prs.Round,
						Type:    types.VoteTypePrevote,
						BlockID: maj23,
					})
					time.Sleep(conR.conS.PeerQueryMaj23Sleep())
				}
			}
		}

		// Maybe send Height/Round/Precommits
		{
			rs := conR.conS.GetRoundState()
			prs := ps.GetRoundState()
			if rs.Height == prs.Height {
				if maj23, ok := rs.Votes.Precommits(prs.Round).TwoThirdsMajority(); ok {
					peer.TrySend(StateChannel, &types.VoteSetMaj23Message{
						Height:  prs.Height,
						Round:   prs.Round,
						Type:    types.VoteTypePrecommit,
						BlockID: maj23,
					})
					time.Sleep(conR.conS.PeerQueryMaj23Sleep())
				}
			}
		}

		// Maybe send Height/Round/ProposalPOL
		{
			rs := conR.conS.GetRoundState()
			prs := ps.GetRoundState()
			if rs.Height == prs.Height && prs.ProposalPOLRound >= 0 {
				if maj23, ok := rs.Votes.Prevotes(prs.ProposalPOLRound).TwoThirdsMajority(); ok {
					peer.TrySend(StateChannel, &types.VoteSetMaj23Message{
						Height:  prs.Height,
						Round:   prs.ProposalPOLRound,
						Type:    types.VoteTypePrevote,
						BlockID: maj23,
					})
					time.Sleep(conR.conS.PeerQueryMaj23Sleep())
				}
			}
		}

		// Little point sending LastCommitRound/LastCommit,
		// These are fleeting and non-blocking.

		// Maybe send Height/CatchupCommitRound/CatchupCommit.
		{
			prs := ps.GetRoundState()
			if prs.CatchupCommitRound != -1 && 0 < prs.Height && prs.Height <= conR.conS.blockStore.Height() {
				commit := conR.conS.LoadCommit(prs.Height)
				peer.TrySend(StateChannel, &types.VoteSetMaj23Message{
					Height:  prs.Height,
					Round:   commit.Round(),
					Type:    types.VoteTypePrecommit,
					BlockID: commit.BlockID,
				})
				time.Sleep(conR.conS.PeerQueryMaj23Sleep())
			}
		}

		time.Sleep(conR.conS.PeerQueryMaj23Sleep())

		continue OUTER_LOOP
	}
}

// String returns a string representation of the ConsensusReactor.
// NOTE: For now, it is just a hard-coded string to avoid accessing unprotected shared variables.
// TODO: improve!
func (conR *ConsensusReactor) String() string {
	// better not to access shared variables
	return "ConsensusReactor" // conR.StringIndented("")
}

// StringIndented returns an indented string representation of the ConsensusReactor
func (conR *ConsensusReactor) StringIndented(indent string) string {
	s := "ConsensusReactor{\n"
	s += indent + "  " + conR.conS.StringIndented(indent+"  ") + "\n"
	for _, peer := range conR.Switch.Peers().List() {
		ps := peer.Get(types.PeerStateKey).(*PeerState)
		s += indent + "  " + ps.StringIndented(indent+"  ") + "\n"
	}
	s += indent + "}"
	return s
}

//-----------------------------------------------------------------------------

var (
	ErrPeerStateHeightRegression = errors.New("Error peer state height regression")
	ErrPeerStateInvalidStartTime = errors.New("Error peer state invalid startTime")
)

// PeerState contains the known state of a peer, including its connection
// and threadsafe access to its PeerRoundState.
type PeerState struct {
	Peer   p2p.Peer

	mtx sync.Mutex
	types.PeerRoundState
}

// NewPeerState returns a new PeerState for the given Peer
func NewPeerState(peer p2p.Peer) *PeerState {
	return &PeerState{
		Peer:   peer,
		PeerRoundState: types.PeerRoundState{
			Round:              -1,
			ProposalPOLRound:   -1,
			LastCommitRound:    -1,
			CatchupCommitRound: -1,
		},
	}
}

// GetRoundState returns an atomic snapshot of the PeerRoundState.
// There's no point in mutating it since it won't change PeerState.
func (ps *PeerState) GetRoundState() *types.PeerRoundState {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	prs := ps.PeerRoundState // copy
	return &prs
}

// GetHeight returns an atomic snapshot of the PeerRoundState's height
// used by the mempool to ensure peers are caught up before broadcasting new txs
func (ps *PeerState) GetHeight() int64 {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return ps.PeerRoundState.Height
}

// SetHasProposal sets the given proposal as known for the peer.
func (ps *PeerState) SetHasProposal(proposal *types.ProposalTrans) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != proposal.Height || ps.Round != proposal.Round {
		return
	}
	if ps.Proposal {
		return
	}

	ps.Proposal = true

	ps.ProposalPOLRound = proposal.POLRound
	ps.ProposalPOL = nil // Nil until types.ProposalPOLMessage received.
}

// PickSendVote picks a vote and sends it to the peer.
// Returns true if vote was sent.
func (ps *PeerState) PickSendVote(votes types.VoteSetReader) bool {
	if vote, ok := ps.PickVoteToSend(votes); ok {
		msg := &types.VoteMessage{vote}
		tendermintlog.Debug("Sending vote message", "ps", ps, "vote", vote)
		return ps.Peer.Send(VoteChannel, msg)
	}
	return false
}

// PickVoteToSend picks a vote to send to the peer.
// Returns true if a vote was picked.
// NOTE: `votes` must be the correct Size() for the Height().
func (ps *PeerState) PickVoteToSend(votes types.VoteSetReader) (vote *types.Vote, ok bool) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if votes.Size() == 0 {
		return nil, false
	}

	height, round, type_, size := votes.Height(), votes.Round(), votes.Type(), votes.Size()

	// Lazily set data using 'votes'.
	if votes.IsCommit() {
		ps.ensureCatchupCommitRound(height, round, size)
	}
	ps.ensureVoteBitArrays(height, size)

	psVotes := ps.getVoteBitArray(height, round, type_)
	if psVotes == nil {
		return nil, false // Not something worth sending
	}
	if index, ok := votes.BitArray().Sub(psVotes).PickRandom(); ok {
		ps.setHasVote(height, round, type_, index)
		return votes.GetByIndex(index), true
	}
	return nil, false
}

func (ps *PeerState) getVoteBitArray(height int64, round int, type_ byte) *cmn.BitArray {
	if !types.IsVoteTypeValid(type_) {
		return nil
	}

	if ps.Height == height {
		if ps.Round == round {
			switch type_ {
			case types.VoteTypePrevote:
				return ps.Prevotes
			case types.VoteTypePrecommit:
				return ps.Precommits
			}
		}
		if ps.CatchupCommitRound == round {
			switch type_ {
			case types.VoteTypePrevote:
				return nil
			case types.VoteTypePrecommit:
				return ps.CatchupCommit
			}
		}
		if ps.ProposalPOLRound == round {
			switch type_ {
			case types.VoteTypePrevote:
				return ps.ProposalPOL
			case types.VoteTypePrecommit:
				return nil
			}
		}
		return nil
	}
	if ps.Height == height+1 {
		if ps.LastCommitRound == round {
			switch type_ {
			case types.VoteTypePrevote:
				return nil
			case types.VoteTypePrecommit:
				return ps.LastCommit
			}
		}
		return nil
	}
	return nil
}

// 'round': A round for which we have a +2/3 commit.
func (ps *PeerState) ensureCatchupCommitRound(height int64, round int, numValidators int) {
	if ps.Height != height {
		return
	}
	/*
		NOTE: This is wrong, 'round' could change.
		e.g. if orig round is not the same as block LastCommit round.
		if ps.CatchupCommitRound != -1 && ps.CatchupCommitRound != round {
			cmn.PanicSanity(cmn.Fmt("Conflicting CatchupCommitRound. Height: %v, Orig: %v, New: %v", height, ps.CatchupCommitRound, round))
		}
	*/
	if ps.CatchupCommitRound == round {
		return // Nothing to do!
	}
	ps.CatchupCommitRound = round
	if round == ps.Round {
		ps.CatchupCommit = ps.Precommits
	} else {
		ps.CatchupCommit = cmn.NewBitArray(numValidators)
	}
}

// EnsureVoteVitArrays ensures the bit-arrays have been allocated for tracking
// what votes this peer has received.
// NOTE: It's important to make sure that numValidators actually matches
// what the node sees as the number of validators for height.
func (ps *PeerState) EnsureVoteBitArrays(height int64, numValidators int) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	ps.ensureVoteBitArrays(height, numValidators)
}

func (ps *PeerState) ensureVoteBitArrays(height int64, numValidators int) {
	if ps.Height == height {
		if ps.Prevotes == nil {
			ps.Prevotes = cmn.NewBitArray(numValidators)
		}
		if ps.Precommits == nil {
			ps.Precommits = cmn.NewBitArray(numValidators)
		}
		if ps.CatchupCommit == nil {
			ps.CatchupCommit = cmn.NewBitArray(numValidators)
		}
		if ps.ProposalPOL == nil {
			ps.ProposalPOL = cmn.NewBitArray(numValidators)
		}
	} else if ps.Height == height+1 {
		if ps.LastCommit == nil {
			ps.LastCommit = cmn.NewBitArray(numValidators)
		}
	}
}

// SetHasVote sets the given vote as known by the peer
func (ps *PeerState) SetHasVote(vote *types.Vote) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	ps.setHasVote(vote.Height, vote.Round, vote.Type, vote.ValidatorIndex)
}

func (ps *PeerState) setHasVote(height int64, round int, type_ byte, index int) {
	// NOTE: some may be nil BitArrays -> no side effects.
	psVotes := ps.getVoteBitArray(height, round, type_)
	if psVotes != nil {
		psVotes.SetIndex(index, true)
	}
}

// ApplyNewRoundStepMessage updates the peer state for the new round.
func (ps *PeerState) ApplyNewRoundStepMessage(msg *types.NewRoundStepMessage) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	// Ignore duplicates or decreases
	if CompareHRS(msg.Height, msg.Round, msg.Step, ps.Height, ps.Round, ps.Step) <= 0 {
		return
	}

	// Just remember these values.
	psHeight := ps.Height
	psRound := ps.Round
	//psStep := ps.Step
	psCatchupCommitRound := ps.CatchupCommitRound
	psCatchupCommit := ps.CatchupCommit

	startTime := time.Now().Add(-1 * time.Duration(msg.SecondsSinceStartTime) * time.Second)
	ps.Height = msg.Height
	ps.Round = msg.Round
	ps.Step = msg.Step
	ps.StartTime = startTime
	if psHeight != msg.Height || psRound != msg.Round {
		ps.Proposal = false
		ps.ProposalPOLRound = -1
		ps.ProposalPOL = nil
		// We'll update the BitArray capacity later.
		ps.Prevotes = nil
		ps.Precommits = nil
	}
	if psHeight == msg.Height && psRound != msg.Round && msg.Round == psCatchupCommitRound {
		// Peer caught up to CatchupCommitRound.
		// Preserve psCatchupCommit!
		// NOTE: We prefer to use prs.Precommits if
		// pr.Round matches pr.CatchupCommitRound.
		ps.Precommits = psCatchupCommit
	}
	if psHeight != msg.Height {
		// Shift Precommits to LastCommit.
		if psHeight+1 == msg.Height && psRound == msg.LastCommitRound {
			ps.LastCommitRound = msg.LastCommitRound
			ps.LastCommit = ps.Precommits
		} else {
			ps.LastCommitRound = msg.LastCommitRound
			ps.LastCommit = nil
		}
		// We'll update the BitArray capacity later.
		ps.CatchupCommitRound = -1
		ps.CatchupCommit = nil
	}
}

// ApplyCommitStepMessage updates the peer state for the new commit.
func (ps *PeerState) ApplyCommitStepMessage(msg *types.CommitStepMessage) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != msg.Height {
		return
	}

}

// ApplyProposalPOLMessage updates the peer state for the new proposal POL.
func (ps *PeerState) ApplyProposalPOLMessage(msg *types.ProposalPOLMessage) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != msg.Height {
		return
	}
	if ps.ProposalPOLRound != msg.ProposalPOLRound {
		return
	}

	// TODO: Merge onto existing ps.ProposalPOL?
	// We might have sent some prevotes in the meantime.
	ps.ProposalPOL = msg.ProposalPOL
}

// ApplyHasVoteMessage updates the peer state for the new vote.
func (ps *PeerState) ApplyHasVoteMessage(msg *types.HasVoteMessage) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	if ps.Height != msg.Height {
		return
	}

	ps.setHasVote(msg.Height, msg.Round, msg.Type, msg.Index)
}

// ApplyVoteSetBitsMessage updates the peer state for the bit-array of votes
// it claims to have for the corresponding BlockID.
// `ourVotes` is a BitArray of votes we have for msg.BlockID
// NOTE: if ourVotes is nil (e.g. msg.Height < rs.Height),
// we conservatively overwrite ps's votes w/ msg.Votes.
func (ps *PeerState) ApplyVoteSetBitsMessage(msg *types.VoteSetBitsMessage, ourVotes *cmn.BitArray) {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()

	votes := ps.getVoteBitArray(msg.Height, msg.Round, msg.Type)
	if votes != nil {
		if ourVotes == nil {
			votes.Update(msg.Votes)
		} else {
			otherVotes := votes.Sub(ourVotes)
			hasVotes := otherVotes.Or(msg.Votes)
			votes.Update(hasVotes)
		}
	}
}

// String returns a string representation of the PeerState
func (ps *PeerState) String() string {
	return ps.StringIndented("")
}

// StringIndented returns a string representation of the PeerState
func (ps *PeerState) StringIndented(indent string) string {
	ps.mtx.Lock()
	defer ps.mtx.Unlock()
	return fmt.Sprintf(`PeerState{
%s  Key %v
%s  PRS %v
%s}`,
		indent, ps.Peer.Key(),
		indent, ps.PeerRoundState.StringIndented(indent+"  "),
		indent)
}
