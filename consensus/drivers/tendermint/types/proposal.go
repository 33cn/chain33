package types

import (
	"errors"
	"fmt"
	"io"
	"time"

	"encoding/json"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
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
	types.Proposal
}

// NewProposal returns a new Proposal.
// If there is no POLRound, polRound should be -1.
func NewProposal(height int64, round int, blockhash []byte, polRound int, polBlockID types.BlockID) *Proposal {
	return &Proposal{types.Proposal{
		Height:     height,
		Round:      int32(round),
		Timestamp:  time.Now().UnixNano(),
		POLRound:   int32(polRound),
		POLBlockID: &polBlockID,
		Blockhash:  blockhash,
	},
	}
}

// String returns a string representation of the Proposal.
func (p *Proposal) String() string {
	return fmt.Sprintf("Proposal{%v/%v (%v,%v) %X %v @ %s}",
		p.Height, p.Round, p.POLRound, p.POLBlockID,
		p.Blockhash, p.Signature, CanonicalTime(time.Unix(0, p.Timestamp)))
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
