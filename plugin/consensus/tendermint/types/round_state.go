package types

import (
	"time"

	"reflect"

	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
)

//-----------------------------------------------------------------------------
// RoundStepType enum type

// RoundStepType enumerates the state of the consensus state machine
type RoundStepType uint8 // These must be numeric, ordered.

var (
	MsgMap map[byte]reflect.Type
)

const (
	RoundStepNewHeight     = RoundStepType(0x01) // Wait til CommitTime + timeoutCommit
	RoundStepNewRound      = RoundStepType(0x02) // Setup new round and go to RoundStepPropose
	RoundStepPropose       = RoundStepType(0x03) // Did propose, gossip proposal
	RoundStepPrevote       = RoundStepType(0x04) // Did prevote, gossip prevotes
	RoundStepPrevoteWait   = RoundStepType(0x05) // Did receive any +2/3 prevotes, start timeout
	RoundStepPrecommit     = RoundStepType(0x06) // Did precommit, gossip precommits
	RoundStepPrecommitWait = RoundStepType(0x07) // Did receive any +2/3 precommits, start timeout
	RoundStepCommit        = RoundStepType(0x08) // Entered commit state machine
	// NOTE: RoundStepNewHeight acts as RoundStepCommitWait.

	EvidenceListID      = byte(0x01)
	NewRoundStepID      = byte(0x02)
	CommitStepID        = byte(0x03)
	ProposalID          = byte(0x04)
	ProposalPOLID       = byte(0x05)
	VoteID              = byte(0x06)
	HasVoteID           = byte(0x07)
	VoteSetMaj23ID      = byte(0X08)
	VoteSetBitsID       = byte(0x09)
	ProposalHeartbeatID = byte(0x0a)
	ProposalBlockID     = byte(0x0b)

	PacketTypePing = byte(0xff)
	PacketTypePong = byte(0xfe)
)

func InitMessageMap() {
	MsgMap = map[byte]reflect.Type{
		EvidenceListID:      reflect.TypeOf(tmtypes.EvidenceData{}),
		NewRoundStepID:      reflect.TypeOf(tmtypes.NewRoundStepMsg{}),
		CommitStepID:        reflect.TypeOf(tmtypes.CommitStepMsg{}),
		ProposalID:          reflect.TypeOf(tmtypes.Proposal{}),
		ProposalPOLID:       reflect.TypeOf(tmtypes.ProposalPOLMsg{}),
		VoteID:              reflect.TypeOf(tmtypes.Vote{}),
		HasVoteID:           reflect.TypeOf(tmtypes.HasVoteMsg{}),
		VoteSetMaj23ID:      reflect.TypeOf(tmtypes.VoteSetMaj23Msg{}),
		VoteSetBitsID:       reflect.TypeOf(tmtypes.VoteSetBitsMsg{}),
		ProposalHeartbeatID: reflect.TypeOf(tmtypes.Heartbeat{}),
		ProposalBlockID:     reflect.TypeOf(tmtypes.TendermintBlock{}),
	}
}

// String returns a string
func (rs RoundStepType) String() string {
	switch rs {
	case RoundStepNewHeight:
		return "RoundStepNewHeight"
	case RoundStepNewRound:
		return "RoundStepNewRound"
	case RoundStepPropose:
		return "RoundStepPropose"
	case RoundStepPrevote:
		return "RoundStepPrevote"
	case RoundStepPrevoteWait:
		return "RoundStepPrevoteWait"
	case RoundStepPrecommit:
		return "RoundStepPrecommit"
	case RoundStepPrecommitWait:
		return "RoundStepPrecommitWait"
	case RoundStepCommit:
		return "RoundStepCommit"
	default:
		return "RoundStepUnknown" // Cannot panic.
	}
}

//-----------------------------------------------------------------------------

// RoundState defines the internal consensus state.
// It is Immutable when returned from ConsensusState.GetRoundState()
// TODO: Actually, only the top pointer is copied,
// so access to field pointers is still racey
// NOTE: Not thread safe. Should only be manipulated by functions downstream
// of the cs.receiveRoutine
type RoundState struct {
	Height         int64 // Height we are working on
	Round          int
	Step           RoundStepType
	StartTime      time.Time
	CommitTime     time.Time // Subjective time when +2/3 precommits for Block at Round were found
	Validators     *ValidatorSet
	Proposal       *tmtypes.Proposal
	ProposalBlock  *TendermintBlock
	LockedRound    int
	LockedBlock    *TendermintBlock
	Votes          *HeightVoteSet
	CommitRound    int
	LastCommit     *VoteSet // Last precommits at Height-1
	LastValidators *ValidatorSet
}

func (rs *RoundState) RoundStateMessage() *tmtypes.NewRoundStepMsg {
	return &tmtypes.NewRoundStepMsg{
		Height: rs.Height,
		Round:  int32(rs.Round),
		Step:   int32(rs.Step),
		SecondsSinceStartTime: int32(time.Since(rs.StartTime).Seconds()),
		LastCommitRound:       int32(rs.LastCommit.Round()),
	}
}

// String returns a string
func (rs *RoundState) String() string {
	return rs.StringIndented("")
}

// StringIndented returns a string
func (rs *RoundState) StringIndented(indent string) string {
	return Fmt(`RoundState{
%s  H:%v R:%v S:%v
%s  StartTime:     %v
%s  CommitTime:    %v
%s  Validators:    %v
%s  Proposal:      %v
%s  ProposalBlock: %v
%s  LockedRound:   %v
%s  LockedBlock:   %v
%s  Votes:         %v
%s  LastCommit:    %v
%s  LastValidators:%v
%s}`,
		indent, rs.Height, rs.Round, rs.Step,
		indent, rs.StartTime,
		indent, rs.CommitTime,
		indent, rs.Validators.StringIndented(indent+"    "),
		indent, rs.Proposal,
		indent, rs.ProposalBlock.StringShort(),
		indent, rs.LockedRound,
		indent, rs.LockedBlock.StringShort(),
		indent, rs.Votes.StringIndented(indent+"    "),
		indent, rs.LastCommit.StringShort(),
		indent, rs.LastValidators.StringIndented(indent+"    "),
		indent)
}

// StringShort returns a string
func (rs *RoundState) StringShort() string {
	return Fmt(`RoundState{H:%v R:%v S:%v ST:%v}`,
		rs.Height, rs.Round, rs.Step, rs.StartTime)
}

//---------------------PeerRoundState----------------------------
type PeerRoundState struct {
	Height             int64         // Height peer is at
	Round              int           // Round peer is at, -1 if unknown.
	Step               RoundStepType // Step peer is at
	StartTime          time.Time     // Estimated start of round 0 at this height
	Proposal           bool          // True if peer has proposal for this round
	ProposalBlock      bool          // True if peer has proposal block for this round
	ProposalPOLRound   int           // Proposal's POL round. -1 if none.
	ProposalPOL        *BitArray     // nil until ProposalPOLMessage received.
	Prevotes           *BitArray     // All votes peer has for this round
	Precommits         *BitArray     // All precommits peer has for this round
	LastCommitRound    int           // Round of commit for last height. -1 if none.
	LastCommit         *BitArray     // All commit precommits of commit for last height.
	CatchupCommitRound int           // Round that we have commit for. Not necessarily unique. -1 if none.
	CatchupCommit      *BitArray     // All commit precommits peer has for this height & CatchupCommitRound
}

// String returns a string representation of the PeerRoundState
func (prs PeerRoundState) String() string {
	return prs.StringIndented("")
}

// StringIndented returns a string representation of the PeerRoundState
func (prs PeerRoundState) StringIndented(indent string) string {
	return Fmt(`PeerRoundState{
%s  %v/%v/%v @%v
%s  Proposal %v
%s  ProposalBlock %v
%s  POL      %v (round %v)
%s  Prevotes   %v
%s  Precommits %v
%s  LastCommit %v (round %v)
%s  Catchup    %v (round %v)
%s}`,
		indent, prs.Height, prs.Round, prs.Step, prs.StartTime,
		indent, prs.Proposal,
		indent, prs.ProposalBlock,
		indent, prs.ProposalPOL, prs.ProposalPOLRound,
		indent, prs.Prevotes,
		indent, prs.Precommits,
		indent, prs.LastCommit, prs.LastCommitRound,
		indent, prs.CatchupCommit, prs.CatchupCommitRound,
		indent)
}

//---------------------Canonical json-----------------------------------
type CanonicalJSONBlockID struct {
	Hash        []byte                     `json:"hash,omitempty"`
	PartsHeader CanonicalJSONPartSetHeader `json:"parts,omitempty"`
}

type CanonicalJSONPartSetHeader struct {
	Hash  []byte `json:"hash"`
	Total int    `json:"total"`
}

type CanonicalJSONProposal struct {
	BlockBytes []byte               `json:"block_parts_header"`
	Height     int64                `json:"height"`
	POLBlockID CanonicalJSONBlockID `json:"pol_block_id"`
	POLRound   int                  `json:"pol_round"`
	Round      int                  `json:"round"`
	Timestamp  string               `json:"timestamp"`
}

type CanonicalJSONVote struct {
	BlockID   CanonicalJSONBlockID `json:"block_id"`
	Height    int64                `json:"height"`
	Round     int                  `json:"round"`
	Timestamp string               `json:"timestamp"`
	Type      byte                 `json:"type"`
}

type CanonicalJSONHeartbeat struct {
	Height           int64  `json:"height"`
	Round            int    `json:"round"`
	Sequence         int    `json:"sequence"`
	ValidatorAddress []byte `json:"validator_address"`
	ValidatorIndex   int    `json:"validator_index"`
}

//------------------------------------
// Messages including a "chain id" can only be applied to one chain, hence "Once"

type CanonicalJSONOnceProposal struct {
	ChainID  string                `json:"chain_id"`
	Proposal CanonicalJSONProposal `json:"proposal"`
}

type CanonicalJSONOnceVote struct {
	ChainID string            `json:"chain_id"`
	Vote    CanonicalJSONVote `json:"vote"`
}

type CanonicalJSONOnceHeartbeat struct {
	ChainID   string                 `json:"chain_id"`
	Heartbeat CanonicalJSONHeartbeat `json:"heartbeat"`
}

//-----------------------------------
// Canonicalize the structs

func CanonicalBlockID(blockID BlockID) CanonicalJSONBlockID {
	return CanonicalJSONBlockID{
		Hash: blockID.Hash,
	}
}

func CanonicalProposal(proposal *Proposal) CanonicalJSONProposal {
	return CanonicalJSONProposal{
		//BlockBytes: proposal.BlockBytes,
		Height:    proposal.Height,
		Timestamp: CanonicalTime(time.Unix(0, proposal.Timestamp)),
		POLBlockID: CanonicalJSONBlockID{
			Hash: proposal.POLBlockID.Hash,
		},
		POLRound: int(proposal.POLRound),
		Round:    int(proposal.Round),
	}
}

func CanonicalVote(vote *Vote) CanonicalJSONVote {
	return CanonicalJSONVote{
		BlockID:   CanonicalJSONBlockID{Hash: vote.BlockID.Hash},
		Height:    vote.Height,
		Round:     int(vote.Round),
		Timestamp: CanonicalTime(time.Unix(0, vote.Timestamp)),
		Type:      byte(vote.Type),
	}
}

func CanonicalHeartbeat(heartbeat *Heartbeat) CanonicalJSONHeartbeat {
	return CanonicalJSONHeartbeat{
		heartbeat.Height,
		int(heartbeat.Round),
		int(heartbeat.Sequence),
		heartbeat.ValidatorAddress,
		int(heartbeat.ValidatorIndex),
	}
}

func CanonicalTime(t time.Time) string {
	// note that sending time over go-wire resets it to
	// local time, we need to force UTC here, so the
	// signatures match
	return t.UTC().Format(timeFormat)
}
