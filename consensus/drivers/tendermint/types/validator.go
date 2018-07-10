package types

import (
	"bytes"
	"fmt"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"encoding/binary"
)

// Volatile state for each Validator
// NOTE: The Accum is not included in Validator.Hash();
// make sure to update that method if changes are made here
type Validator struct {
	Address     []byte    `json:"address"`
	PubKey      []byte    `json:"pub_key"`
	VotingPower int64     `json:"voting_power"`

	Accum int64 `json:"accum"`
}

func NewValidator(pubKey crypto.PubKey, votingPower int64) *Validator {
	return &Validator{
		Address:     GenAddressByPubKey(pubKey),
		PubKey:      pubKey.Bytes(),
		VotingPower: votingPower,
		Accum:       0,
	}
}

// Creates a new copy of the validator so we can mutate accum.
// Panics if the validator is nil.
func (v *Validator) Copy() *Validator {
	vCopy := *v
	return &vCopy
}

// Returns the one with higher Accum.
func (v *Validator) CompareAccum(other *Validator) *Validator {
	if v == nil {
		return other
	}
	if v.Accum > other.Accum {
		return v
	} else if v.Accum < other.Accum {
		return other
	} else {
		if bytes.Compare(v.Address, other.Address) < 0 {
			return v
		} else if bytes.Compare(v.Address, other.Address) > 0 {
			return other
		} else {
			PanicSanity("Cannot compare identical validators")
			return nil
		}
	}
}

func (v *Validator) String() string {
	if v == nil {
		return "nil-Validator"
	}
	return fmt.Sprintf("Validator{%v %v VP:%v A:%v}",
		v.Address,
		v.PubKey,
		v.VotingPower,
		v.Accum)
}

// Hash computes the unique ID of a validator with a given voting power.
// It excludes the Accum value, which changes with every round.
func (v *Validator) Hash() []byte {
	hashBytes := v.Address
	hashBytes = append(hashBytes, v.PubKey...)
	bPowder := make([]byte,8)
	binary.BigEndian.PutUint64(bPowder, uint64(v.VotingPower))
	hashBytes = append(hashBytes, bPowder...)
	return crypto.Ripemd160(hashBytes)
}

//--------------------------------------------------------------------------------
// For testing...

// RandValidator returns a randomized validator, useful for testing.
// UNSTABLE
func RandValidator(randPower bool, minPower int64) (*Validator, *PrivValidatorImp) {
	_, tempFilePath := Tempfile("priv_validator_")
	privVal := GenPrivValidatorImp(tempFilePath)
	votePower := minPower
	if randPower {
		votePower += int64(Randgen.Uint32())
	}
	val := NewValidator(privVal.PubKey, votePower)
	return val, privVal
}
