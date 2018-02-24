package core

import (
	cstypes "code.aliyun.com/chain33/chain33/consensus/drivers/tendermint/types"
	"fmt"
)

// VoteMessage is sent when voting for a proposal (or lack thereof).
type VoteMessage struct {
	Vote *cstypes.Vote
}

// ProposalMessage is sent when a new block is proposed.
type ProposalMessage struct {
	Proposal *cstypes.Proposal
}

// String returns a string representation.
func (m *ProposalMessage) String() string {
	return fmt.Sprintf("[Proposal %v]", m.Proposal)
}

// String returns a string representation.
func (m *ProposalMessage) String() string {
	return fmt.Sprintf("[Proposal %v]", m.Proposal)
}

type ConsensusMessage interface{}
