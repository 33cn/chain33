package types

import (
	"errors"
	"fmt"
	"io"
	"time"

	"encoding/json"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/crypto"
)

var (
	ErrInvalidBlockPartSignature = errors.New("Error invalid block part signature")
	ErrInvalidBlockPartHash      = errors.New("Error invalid block part hash")

	proposallog = log15.New("module", "tendermint-proposal")
)

// Proposal defines a block proposal for the consensus.
// It refers to the block only by its PartSetHeader.
// It must be signed by the correct proposer for the given Height/Round
// to be considered valid. It may depend on votes from a previous round,
// a so-called Proof-of-Lock (POL) round, as noted in the POLRound and POLBlockID.
type Proposal struct {
	Height     int64            `json:"height"`
	Round      int              `json:"round"`
	Timestamp  time.Time        `json:"timestamp"`
	POLRound   int              `json:"pol_round"`    // -1 if null.
	POLBlockID BlockID          `json:"pol_block_id"` // zero if null.
	Signature  crypto.Signature `json:"signature"`
	BlockBytes []byte           `json:"block_bytes"`
}

type ProposalTrans struct {
	Height     int64     `json:"height"`
	Round      int       `json:"round"`
	Timestamp  time.Time `json:"timestamp"`
	POLRound   int       `json:"pol_round"`    // -1 if null.
	POLBlockID BlockID   `json:"pol_block_id"` // zero if null.
	Signature  string    `json:"signature"`
	BlockBytes []byte    `json:"block_bytes"`
}

func ProposalToProposalTrans(proposal *Proposal) *ProposalTrans {
	sig := fmt.Sprintf("%X", proposal.Signature.Bytes())
	proposalTrans := &ProposalTrans{
		POLRound:   proposal.POLRound,
		Height:     proposal.Height,
		Round:      proposal.Round,
		Timestamp:  proposal.Timestamp,
		POLBlockID: proposal.POLBlockID,
		Signature:  sig,
		BlockBytes: proposal.BlockBytes,
	}
	return proposalTrans
}

func ProposalTransToProposal(proposalTrans *ProposalTrans) (*Proposal, error) {
	sig, err := SignatureFromString(proposalTrans.Signature)
	if err != nil {
		return nil, err
	}
	proposal := &Proposal{
		POLRound:   proposalTrans.POLRound,
		Height:     proposalTrans.Height,
		Round:      proposalTrans.Round,
		Timestamp:  proposalTrans.Timestamp,
		POLBlockID: proposalTrans.POLBlockID,
		Signature:  sig,
		BlockBytes: proposalTrans.BlockBytes,
	}
	return proposal, nil
}

// NewProposal returns a new Proposal.
// If there is no POLRound, polRound should be -1.
func NewProposal(height int64, round int, block *Block, polRound int, polBlockID BlockID) *Proposal {
	blockByte, err := json.Marshal(block)
	if err != nil {
		proposallog.Error("NewProposal unmarshal failed", "error", err)
		return nil
	}
	return &Proposal{
		Height:     height,
		Round:      round,
		Timestamp:  time.Now().UTC(),
		POLRound:   polRound,
		POLBlockID: polBlockID,
		BlockBytes: blockByte,
	}
}

// String returns a string representation of the Proposal.
func (p *Proposal) String() string {
	return fmt.Sprintf("Proposal{%v/%v (%v,%v) %v @ %s}",
		p.Height, p.Round, p.POLRound,
		p.POLBlockID, p.Signature, CanonicalTime(p.Timestamp))
}

// WriteSignBytes writes the Proposal bytes for signing
func (p *Proposal) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	if *err != nil {
		return
	}
	canonical := CanonicalJSONOnceProposal{
		ChainID:  chainID,
		Proposal: CanonicalProposal(p),
	}
	byteOnceProposal, e := json.Marshal(&canonical)
	if e != nil {
		*err = e
		return
	}
	n_, err_ := w.Write(byteOnceProposal)
	*n = n_
	*err = err_
	return
}
