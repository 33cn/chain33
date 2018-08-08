package types

import (
	"fmt"
	"time"
	"gitlab.33.cn/chain33/chain33/types"
	"reflect"
)

var (
	MsgMap map[byte]reflect.Type
)

const (

	EvidenceListID = byte(0x01)

	NewRoundStepID      = byte(0x02)
	CommitStepID        = byte(0x03)
	ProposalID          = byte(0x04)
	ProposalPOLID       = byte(0x05)
	VoteID              = byte(0x06)
	HasVoteID           = byte(0x07)
	VoteSetMaj23ID      = byte(0X08)
	VoteSetBitsID       = byte(0x09)
	ProposalHeartbeatID = byte(0x0a)

	PacketTypePing = byte(0xff)
	PacketTypePong = byte(0xfe)
)

type PeerRoundState struct {
	Height             int64         // Height peer is at
	Round              int           // Round peer is at, -1 if unknown.
	Step               RoundStepType // Step peer is at
	StartTime          time.Time     // Estimated start of round 0 at this height
	Proposal           bool          // True if peer has proposal for this round
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
	return fmt.Sprintf(`PeerRoundState{
%s  %v/%v/%v @%v
%s  Proposal %v
%s  POL      %v (round %v)
%s  Prevotes   %v
%s  Precommits %v
%s  LastCommit %v (round %v)
%s  Catchup    %v (round %v)
%s}`,
		indent, prs.Height, prs.Round, prs.Step, prs.StartTime,
		indent, prs.Proposal,
		indent, prs.ProposalPOL, prs.ProposalPOLRound,
		indent, prs.Prevotes,
		indent, prs.Precommits,
		indent, prs.LastCommit, prs.LastCommitRound,
		indent, prs.CatchupCommit, prs.CatchupCommitRound,
		indent)
}

func InitMessageMap() {
	MsgMap = map[byte]reflect.Type{
		EvidenceListID:      reflect.TypeOf(types.EvidenceData{}),
		NewRoundStepID:      reflect.TypeOf(types.NewRoundStepMsg{}),
		CommitStepID:        reflect.TypeOf(types.CommitStepMsg{}),
		ProposalID:          reflect.TypeOf(types.Proposal{}),
		ProposalPOLID:       reflect.TypeOf(types.ProposalPOLMsg{}),
		VoteID:              reflect.TypeOf(types.Vote{}),
		HasVoteID:           reflect.TypeOf(types.HasVoteMsg{}),
		VoteSetMaj23ID:      reflect.TypeOf(types.VoteSetMaj23Msg{}),
		VoteSetBitsID:       reflect.TypeOf(types.VoteSetBitsMsg{}),
		ProposalHeartbeatID: reflect.TypeOf(types.Heartbeat{}),
	}
}
