package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"encoding/json"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/crypto"
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

func NewConflictingVoteError(val *Validator, voteA, voteB *Vote) *ErrVoteConflictingVotes {
	keyString := fmt.Sprintf("%X", val.PubKey)
	return &ErrVoteConflictingVotes{
		&DuplicateVoteEvidence{
			PubKey: keyString,
			VoteA:  voteA,
			VoteB:  voteB,
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
	ValidatorAddress []byte    `json:"validator_address"`
	ValidatorIndex   int       `json:"validator_index"`
	Height           int64     `json:"height"`
	Round            int       `json:"round"`
	Timestamp        time.Time `json:"timestamp"`
	Type             byte      `json:"type"`
	BlockID          BlockID   `json:"block_id"` // zero if vote is nil.
	Signature        []byte    `json:"signature"`
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
	return
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
	switch vote.Type {
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
		CanonicalTime(vote.Timestamp))
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
