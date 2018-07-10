package types

import (
	"encoding/json"
	"fmt"
	"time"
)

var (
	MsgMap map[byte]interface{}
)

const (
	EvidenceListMsg      = "EvidenceList"
	
	NewRoundStepMsg      = "NewRoundStep"
	CommitStepMsg        = "CommitStep"
	ProposalMsg          = "Proposal"
	ProposalPOLMsg       = "ProposalPOL"
	VoteMsg              = "Vote"
	HasVoteMsg      	 = "HasVote"
	VoteSetMaj23Msg 	 = "VoteSetMaj23"
	VoteSetBitsMsg       = "VoteSetBits"
	ProposalHeartbeatMsg = "ProposalHeartbeat"


	EvidenceListID      = byte(0x01)

	NewRoundStepID      = byte(0x02)
	CommitStepID        = byte(0x03)
	ProposalID          = byte(0x04)
	ProposalPOLID       = byte(0x05)
	VoteID              = byte(0x06)
	HasVoteID      	    = byte(0x07)
	VoteSetMaj23ID 	    = byte(0X08)
	VoteSetBitsID       = byte(0x09)
	ProposalHeartbeatID = byte(0x0a)

	PacketTypePing           = byte(0xff)
	PacketTypePong           = byte(0xfe)
)

type PeerRoundState struct {
	Height                   int64         // Height peer is at
	Round                    int           // Round peer is at, -1 if unknown.
	Step                     RoundStepType // Step peer is at
	StartTime                time.Time     // Estimated start of round 0 at this height
	Proposal                 bool          // True if peer has proposal for this round
	ProposalPOLRound         int           // Proposal's POL round. -1 if none.
	ProposalPOL              *BitArray // nil until ProposalPOLMessage received.
	Prevotes                 *BitArray // All votes peer has for this round
	Precommits               *BitArray // All precommits peer has for this round
	LastCommitRound          int           // Round of commit for last height. -1 if none.
	LastCommit               *BitArray // All commit precommits of commit for last height.
	CatchupCommitRound       int           // Round that we have commit for. Not necessarily unique. -1 if none.
	CatchupCommit            *BitArray // All commit precommits peer has for this height & CatchupCommitRound
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

type ReactorMsg interface {
	TypeID()  byte
	TypeName() string
	Copy() ReactorMsg
}

type MsgEnvelope struct {
	Kind string           `json:"kind"`
	Data *json.RawMessage `json:"data"`
}

func InitMessageMap() {
	MsgMap = map[byte]interface{}{
		EvidenceListID:      &EvidenceListMessage{},
		NewRoundStepID:      &NewRoundStepMessage{},
		CommitStepID:        &CommitStepMessage{},
		ProposalID:          &ProposalMessage{},
		ProposalPOLID:       &ProposalPOLMessage{},
		VoteID:              &VoteMessage{},
		HasVoteID:           &HasVoteMessage{},
		VoteSetMaj23ID:      &VoteSetMaj23Message{},
		VoteSetBitsID:       &VoteSetBitsMessage{},
		ProposalHeartbeatID: &ProposalHeartbeatMessage{},
	}
}

type EvidenceListMessage struct {
	Evidence []Evidence
}

// String returns a string representation of the EvidenceListMessage.
func (m *EvidenceListMessage) String() string {
	return fmt.Sprintf("[EvidenceListMessage %v]", m.Evidence)
}

func (m *EvidenceListMessage) TypeName() string {
	return 	EvidenceListMsg
}

func (m *EvidenceListMessage) Copy() ReactorMsg {
	return 	&EvidenceListMessage{}
}

func (m *EvidenceListMessage) TypeID() byte {
	return EvidenceListID
}

type NewRoundStepMessage struct {
	Height                int64
	Round                 int
	Step                  RoundStepType
	SecondsSinceStartTime int
	LastCommitRound       int
}

// String returns a string representation.
func (m *NewRoundStepMessage) String() string {
	return fmt.Sprintf("[NewRoundStep H:%v R:%v S:%v LCR:%v]",
		m.Height, m.Round, m.Step, m.LastCommitRound)
}

func (m *NewRoundStepMessage) TypeName() string {
	return NewRoundStepMsg
}

func (m *NewRoundStepMessage) Copy() ReactorMsg {
	return &NewRoundStepMessage{}
}

func (m *NewRoundStepMessage) TypeID() byte {
	return NewRoundStepID
}

//-------------------------------------

// CommitStepMessage is sent when a block is committed.
type CommitStepMessage struct {
	Height           int64
}

// String returns a string representation.
func (m *CommitStepMessage) String() string {
	return fmt.Sprintf("[CommitStep H:%v]", m.Height)
}

func (m *CommitStepMessage) TypeName() string {
	return CommitStepMsg
}

func (m *CommitStepMessage) Copy() ReactorMsg {
	return &CommitStepMessage{}
}

func (m *CommitStepMessage) TypeID() byte {
	return CommitStepID
}
//-------------------------------------

// ProposalMessage is sent when a new block is proposed.
type ProposalMessage struct {
	//Proposal *Proposal
	Proposal *ProposalTrans
}

// String returns a string representation.
func (m *ProposalMessage) String() string {
	return fmt.Sprintf("[Proposal %v]", m.Proposal)
}

func (m *ProposalMessage) TypeName() string {
	return ProposalMsg
}

func (m *ProposalMessage) Copy() ReactorMsg {
	return &ProposalMessage{}
}

func (m *ProposalMessage) TypeID() byte {
	return ProposalID
}
//-------------------------------------

// ProposalPOLMessage is sent when a previous proposal is re-proposed.
type ProposalPOLMessage struct {
	Height           int64
	ProposalPOLRound int
	ProposalPOL      *BitArray
}

// String returns a string representation.
func (m *ProposalPOLMessage) String() string {
	return fmt.Sprintf("[ProposalPOL H:%v POLR:%v POL:%v]", m.Height, m.ProposalPOLRound, m.ProposalPOL)
}

func (m *ProposalPOLMessage) TypeName() string {
	return ProposalPOLMsg
}

func (m *ProposalPOLMessage) Copy() ReactorMsg {
	return &ProposalPOLMessage{}
}

func (m *ProposalPOLMessage) TypeID() byte {
	return ProposalPOLID
}
//-------------------------------------

// VoteMessage is sent when voting for a proposal (or lack thereof).
type VoteMessage struct {
	Vote *Vote
}

// String returns a string representation.
func (m *VoteMessage) String() string {
	return fmt.Sprintf("[Vote %v]", m.Vote)
}

func (m *VoteMessage) TypeName() string {
	return VoteMsg
}

func (m *VoteMessage) Copy() ReactorMsg {
	return &VoteMessage{}
}

func (m *VoteMessage) TypeID() byte {
	return VoteID
}
//-------------------------------------

// HasVoteMessage is sent to indicate that a particular vote has been received.
type HasVoteMessage struct {
	Height int64
	Round  int
	Type   byte
	Index  int
}

// String returns a string representation.
func (m *HasVoteMessage) String() string {
	return fmt.Sprintf("[HasVote VI:%v V:{%v/%02d/%v}]", m.Index, m.Height, m.Round, m.Type)
}

func (m *HasVoteMessage) TypeName() string {
	return HasVoteMsg
}

func (m *HasVoteMessage) Copy() ReactorMsg {
	return &HasVoteMessage{}
}

func (m *HasVoteMessage) TypeID() byte {
	return HasVoteID
}

//-------------------------------------

// VoteSetMaj23Message is sent to indicate that a given BlockID has seen +2/3 votes.
type VoteSetMaj23Message struct {
	Height  int64
	Round   int
	Type    byte
	BlockID BlockID
}

// String returns a string representation.
func (m *VoteSetMaj23Message) String() string {
	return fmt.Sprintf("[VSM23 %v/%02d/%v %v]", m.Height, m.Round, m.Type, m.BlockID)
}

func (m *VoteSetMaj23Message) TypeName() string {
	return VoteSetMaj23Msg
}

func (m *VoteSetMaj23Message) Copy() ReactorMsg {
	return &VoteSetMaj23Message{}
}

func (m *VoteSetMaj23Message) TypeID() byte {
	return VoteSetMaj23ID
}
//-------------------------------------

// VoteSetBitsMessage is sent to communicate the bit-array of votes seen for the BlockID.
type VoteSetBitsMessage struct {
	Height  int64
	Round   int
	Type    byte
	BlockID BlockID
	Votes   *BitArray
}

// String returns a string representation.
func (m *VoteSetBitsMessage) String() string {
	return fmt.Sprintf("[VSB %v/%02d/%v %v %v]", m.Height, m.Round, m.Type, m.BlockID, m.Votes)
}

func (m *VoteSetBitsMessage) TypeName() string {
	return VoteSetBitsMsg
}

func (m *VoteSetBitsMessage) Copy() ReactorMsg {
	return &VoteSetBitsMessage{}
}

func (m *VoteSetBitsMessage) TypeID() byte {
	return VoteSetBitsID
}
//-------------------------------------

// ProposalHeartbeatMessage is sent to signal that a node is alive and waiting for transactions for a proposal.
type ProposalHeartbeatMessage struct {
	Heartbeat *Heartbeat
}

// String returns a string representation.
func (m *ProposalHeartbeatMessage) String() string {
	return fmt.Sprintf("[HEARTBEAT %v]", m.Heartbeat)
}

func (m *ProposalHeartbeatMessage) TypeName() string {
	return ProposalHeartbeatMsg
}

func (m *ProposalHeartbeatMessage) Copy() ReactorMsg {
	return &ProposalHeartbeatMessage{}
}

func (m *ProposalHeartbeatMessage) TypeID() byte {
	return ProposalHeartbeatID
}