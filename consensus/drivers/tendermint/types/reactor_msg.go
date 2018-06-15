package types

import (
	"encoding/json"
	"fmt"
	cmn "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/common"
)

var (
	MsgMap map[string]interface{}
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
	
)

type ReactorMsg interface {
	TypeName() string
	Copy() ReactorMsg
}

type MsgEnvelope struct {
	Kind string           `json:"kind"`
	Data *json.RawMessage `json:"data"`
}

func InitMessageMap() {
	MsgMap = map[string]interface{}{
		EvidenceListMsg:      &EvidenceListMessage{},
		NewRoundStepMsg:      &NewRoundStepMessage{},
		CommitStepMsg:        &CommitStepMessage{},
		ProposalMsg:          &ProposalMessage{},
		ProposalPOLMsg:       &ProposalPOLMessage{},
		VoteMsg:              &VoteMessage{},
		HasVoteMsg:           &HasVoteMessage{},
		VoteSetMaj23Msg:      &VoteSetMaj23Message{},
		VoteSetBitsMsg:       &VoteSetBitsMessage{},
		ProposalHeartbeatMsg: &ProposalHeartbeatMessage{},
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
//-------------------------------------

// ProposalPOLMessage is sent when a previous proposal is re-proposed.
type ProposalPOLMessage struct {
	Height           int64
	ProposalPOLRound int
	ProposalPOL      *cmn.BitArray
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
//-------------------------------------

// VoteSetBitsMessage is sent to communicate the bit-array of votes seen for the BlockID.
type VoteSetBitsMessage struct {
	Height  int64
	Round   int
	Type    byte
	BlockID BlockID
	Votes   *cmn.BitArray
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