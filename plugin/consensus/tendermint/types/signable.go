package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
)

var (
	ErrVoteUnexpectedStep            = errors.New("Unexpected step")
	ErrVoteInvalidValidatorIndex     = errors.New("Invalid validator index")
	ErrVoteInvalidValidatorAddress   = errors.New("Invalid validator address")
	ErrVoteInvalidSignature          = errors.New("Invalid signature")
	ErrVoteInvalidBlockHash          = errors.New("Invalid block hash")
	ErrVoteNonDeterministicSignature = errors.New("Non-deterministic signature")
	ErrVoteNil                       = errors.New("Nil vote")
	votelog                          = log15.New("module", "tendermint-vote")
)

// Signable is an interface for all signable things.
// It typically removes signatures before serializing.
type Signable interface {
	WriteSignBytes(chainID string, w io.Writer, n *int, err *error)
}

// SignBytes is a convenience method for getting the bytes to sign of a Signable.
func SignBytes(chainID string, o Signable) []byte {
	buf, n, err := new(bytes.Buffer), new(int), new(error)
	o.WriteSignBytes(chainID, buf, n, err)
	if *err != nil {
		PanicCrisis(err)
	}
	return buf.Bytes()
}

//----------------------Proposal----------------------
// Proposal defines a block proposal for the consensus.
// It refers to the block only by its PartSetHeader.
// It must be signed by the correct proposer for the given Height/Round
// to be considered valid. It may depend on votes from a previous round,
// a so-called Proof-of-Lock (POL) round, as noted in the POLRound and POLBlockID.
type Proposal struct {
	tmtypes.Proposal
}

// NewProposal returns a new Proposal.
// If there is no POLRound, polRound should be -1.
func NewProposal(height int64, round int, blockhash []byte, polRound int, polBlockID tmtypes.BlockID) *Proposal {
	return &Proposal{tmtypes.Proposal{
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
}

//-------------------heartbeat-------------------------
type Heartbeat struct {
	*tmtypes.Heartbeat
}

// WriteSignBytes writes the Heartbeat for signing.
// It panics if the Heartbeat is nil.
func (heartbeat *Heartbeat) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	if *err != nil {
		return
	}
	canonical := CanonicalJSONOnceHeartbeat{
		chainID,
		CanonicalHeartbeat(heartbeat),
	}
	byteHeartbeat, e := json.Marshal(&canonical)
	if e != nil {
		*err = e
		return
	}
	n_, err_ := w.Write(byteHeartbeat)
	*n = n_
	*err = err_
}

//----------------------vote-----------------------------
type ErrVoteConflictingVotes struct {
	*DuplicateVoteEvidence
}

func (err *ErrVoteConflictingVotes) Error() string {
	pubkey, error := PubKeyFromString(err.PubKey)
	if error != nil {
		return fmt.Sprintf("Conflicting votes from validator PubKey:%v,error:%v", err.PubKey, error)
	}
	addr := GenAddressByPubKey(pubkey)
	return fmt.Sprintf("Conflicting votes from validator %v", addr)
}

func NewConflictingVoteError(val *Validator, voteA, voteB *tmtypes.Vote) *ErrVoteConflictingVotes {
	keyString := fmt.Sprintf("%X", val.PubKey)
	return &ErrVoteConflictingVotes{
		&DuplicateVoteEvidence{
			&tmtypes.DuplicateVoteEvidence{
				PubKey: keyString,
				VoteA:  voteA,
				VoteB:  voteB,
			},
		},
	}
}

// Types of votes
// TODO Make a new type "VoteType"
const (
	VoteTypePrevote   = byte(0x01)
	VoteTypePrecommit = byte(0x02)
)

func IsVoteTypeValid(type_ byte) bool {
	switch type_ {
	case VoteTypePrevote:
		return true
	case VoteTypePrecommit:
		return true
	default:
		return false
	}
}

// Represents a prevote, precommit, or commit vote from validators for consensus.
type Vote struct {
	*tmtypes.Vote
}

func (vote *Vote) WriteSignBytes(chainID string, w io.Writer, n *int, err *error) {
	if *err != nil {
		return
	}
	canonical := CanonicalJSONOnceVote{
		chainID,
		CanonicalVote(vote),
	}
	byteVote, e := json.Marshal(&canonical)
	if e != nil {
		*err = e
		votelog.Error("vote WriteSignBytes marshal failed", "err", e)
		return
	}
	n_, err_ := w.Write(byteVote)
	*n = n_
	*err = err_
}

func (vote *Vote) Copy() *Vote {
	voteCopy := *vote
	return &voteCopy
}

func (vote *Vote) String() string {
	if vote == nil {
		return "nil-Vote"
	}
	var typeString string
	switch byte(vote.Type) {
	case VoteTypePrevote:
		typeString = "Prevote"
	case VoteTypePrecommit:
		typeString = "Precommit"
	default:
		PanicSanity("Unknown vote type")
	}

	return fmt.Sprintf("Vote{%v:%X %v/%02d/%v(%v) %X %v @ %s}",
		vote.ValidatorIndex, Fingerprint(vote.ValidatorAddress),
		vote.Height, vote.Round, vote.Type, typeString,
		Fingerprint(vote.BlockID.Hash), vote.Signature,
		CanonicalTime(time.Unix(0, vote.Timestamp)))
}

func (vote *Vote) Verify(chainID string, pubKey crypto.PubKey) error {
	addr := GenAddressByPubKey(pubKey)
	if !bytes.Equal(addr, vote.ValidatorAddress) {
		return ErrVoteInvalidValidatorAddress
	}

	sig, err := ConsensusCrypto.SignatureFromBytes(vote.Signature)
	if err != nil {
		votelog.Error("vote Verify failed", "err", err)
		return err
	}

	if !pubKey.VerifyBytes(SignBytes(chainID, vote), sig) {
		return ErrVoteInvalidSignature
	}
	return nil
}

func (vote *Vote) Hash() []byte {
	if vote == nil {
		//votelog.Error("vote hash is nil")
		return nil
	}
	bytes, err := json.Marshal(vote)
	if err != nil {
		votelog.Error("vote hash marshal failed", "err", err)
		return nil
	}

	return crypto.Ripemd160(bytes)
}
