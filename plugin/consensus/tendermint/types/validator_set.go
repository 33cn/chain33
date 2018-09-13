package types

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"sort"
	"strings"

	"github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
)

var validatorsetlog = log15.New("module", "tendermint-val")

type Validator struct {
	Address     []byte `json:"address"`
	PubKey      []byte `json:"pub_key"`
	VotingPower int64  `json:"voting_power"`

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
	return Fmt("Validator{%v %v VP:%v A:%v}",
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
	bPowder := make([]byte, 8)
	binary.BigEndian.PutUint64(bPowder, uint64(v.VotingPower))
	hashBytes = append(hashBytes, bPowder...)
	return crypto.Ripemd160(hashBytes)
}

// ValidatorSet represent a set of *Validator at a given height.
// The validators can be fetched by address or index.
// The index is in order of .Address, so the indices are fixed
// for all rounds of a given blockchain height.
// On the other hand, the .AccumPower of each validator and
// the designated .GetProposer() of a set changes every round,
// upon calling .IncrementAccum().
// NOTE: Not goroutine-safe.
// NOTE: All get/set to validators should copy the value for safety.
// TODO: consider validator Accum overflow
type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*Validator `json:"validators"`
	Proposer   *Validator   `json:"proposer"`

	// cached (unexported)
	totalVotingPower int64
}

func NewValidatorSet(vals []*Validator) *ValidatorSet {
	validators := make([]*Validator, len(vals))
	for i, val := range vals {
		validators[i] = val.Copy()
	}
	sort.Sort(ValidatorsByAddress(validators))
	vs := &ValidatorSet{
		Validators: validators,
	}

	if vals != nil {
		vs.IncrementAccum(1)
	}

	return vs
}

// incrementAccum and update the proposer
// TODO: mind the overflow when times and votingPower shares too large.
func (valSet *ValidatorSet) IncrementAccum(times int) {
	// Add VotingPower * times to each validator and order into heap.
	validatorsHeap := NewHeap()
	for _, val := range valSet.Validators {
		val.Accum += val.VotingPower * int64(times) // TODO: mind overflow
		validatorsHeap.Push(val, accumComparable{val})
	}

	// Decrement the validator with most accum times times
	for i := 0; i < times; i++ {
		mostest := validatorsHeap.Peek().(*Validator)
		if i == times-1 {
			valSet.Proposer = mostest
		}
		mostest.Accum -= valSet.TotalVotingPower()
		validatorsHeap.Update(mostest, accumComparable{mostest})
	}
}

func (valSet *ValidatorSet) Copy() *ValidatorSet {
	validators := make([]*Validator, len(valSet.Validators))
	for i, val := range valSet.Validators {
		// NOTE: must copy, since IncrementAccum updates in place.
		validators[i] = val.Copy()
	}
	return &ValidatorSet{
		Validators:       validators,
		Proposer:         valSet.Proposer,
		totalVotingPower: valSet.totalVotingPower,
	}
}

func (valSet *ValidatorSet) HasAddress(address []byte) bool {
	idx := sort.Search(len(valSet.Validators), func(i int) bool {
		return bytes.Compare(address, valSet.Validators[i].Address) <= 0
	})
	return idx != len(valSet.Validators) && bytes.Equal(valSet.Validators[idx].Address, address)
}

func (valSet *ValidatorSet) GetByAddress(address []byte) (index int, val *Validator) {
	idx := sort.Search(len(valSet.Validators), func(i int) bool {
		return bytes.Compare(address, valSet.Validators[i].Address) <= 0
	})
	if idx != len(valSet.Validators) && bytes.Equal(valSet.Validators[idx].Address, address) {
		return idx, valSet.Validators[idx].Copy()
	} else {
		return -1, nil
	}
}

// GetByIndex returns the validator by index.
// It returns nil values if index < 0 or
// index >= len(ValidatorSet.Validators)
func (valSet *ValidatorSet) GetByIndex(index int) (address []byte, val *Validator) {
	if index < 0 || index >= len(valSet.Validators) {
		return nil, nil
	}
	val = valSet.Validators[index]
	return val.Address, val.Copy()
}

func (valSet *ValidatorSet) Size() int {
	return len(valSet.Validators)
}

func (valSet *ValidatorSet) TotalVotingPower() int64 {
	if valSet.totalVotingPower == 0 {
		for _, val := range valSet.Validators {
			valSet.totalVotingPower += val.VotingPower
		}
	}
	return valSet.totalVotingPower
}

func (valSet *ValidatorSet) GetProposer() (proposer *Validator) {
	if len(valSet.Validators) == 0 {
		return nil
	}
	if valSet.Proposer == nil {
		valSet.Proposer = valSet.findProposer()
	}
	return valSet.Proposer.Copy()
}

func (valSet *ValidatorSet) findProposer() *Validator {
	var proposer *Validator
	for _, val := range valSet.Validators {
		if proposer == nil || !bytes.Equal(val.Address, proposer.Address) {
			proposer = proposer.CompareAccum(val)
		}
	}
	return proposer
}

func (valSet *ValidatorSet) Hash() []byte {
	if len(valSet.Validators) == 0 {
		return nil
	}
	hashables := make([][]byte, len(valSet.Validators))
	for i, val := range valSet.Validators {
		hashables[i] = val.Hash()
	}
	return merkle.GetMerkleRoot(hashables)
}

func (valSet *ValidatorSet) Add(val *Validator) (added bool) {
	val = val.Copy()
	idx := sort.Search(len(valSet.Validators), func(i int) bool {
		return bytes.Compare(val.Address, valSet.Validators[i].Address) <= 0
	})
	if idx == len(valSet.Validators) {
		valSet.Validators = append(valSet.Validators, val)
		// Invalidate cache
		valSet.Proposer = nil
		valSet.totalVotingPower = 0
		return true
	} else if bytes.Equal(valSet.Validators[idx].Address, val.Address) {
		return false
	} else {
		newValidators := make([]*Validator, len(valSet.Validators)+1)
		copy(newValidators[:idx], valSet.Validators[:idx])
		newValidators[idx] = val
		copy(newValidators[idx+1:], valSet.Validators[idx:])
		valSet.Validators = newValidators
		// Invalidate cache
		valSet.Proposer = nil
		valSet.totalVotingPower = 0
		return true
	}
}

func (valSet *ValidatorSet) Update(val *Validator) (updated bool) {
	index, sameVal := valSet.GetByAddress(val.Address)
	if sameVal == nil {
		return false
	} else {
		valSet.Validators[index] = val.Copy()
		// Invalidate cache
		valSet.Proposer = nil
		valSet.totalVotingPower = 0
		return true
	}
}

func (valSet *ValidatorSet) Remove(address []byte) (val *Validator, removed bool) {
	idx := sort.Search(len(valSet.Validators), func(i int) bool {
		return bytes.Compare(address, valSet.Validators[i].Address) <= 0
	})
	if idx == len(valSet.Validators) || !bytes.Equal(valSet.Validators[idx].Address, address) {
		return nil, false
	} else {
		removedVal := valSet.Validators[idx]
		newValidators := valSet.Validators[:idx]
		if idx+1 < len(valSet.Validators) {
			newValidators = append(newValidators, valSet.Validators[idx+1:]...)
		}
		valSet.Validators = newValidators
		// Invalidate cache
		valSet.Proposer = nil
		valSet.totalVotingPower = 0
		return removedVal, true
	}
}

func (valSet *ValidatorSet) Iterate(fn func(index int, val *Validator) bool) {
	for i, val := range valSet.Validators {
		stop := fn(i, val.Copy())
		if stop {
			break
		}
	}
}

// Verify that +2/3 of the set had signed the given signBytes
func (valSet *ValidatorSet) VerifyCommit(chainID string, blockID BlockID, height int64, commit *Commit) error {
	if valSet.Size() != len(commit.Precommits) {
		return fmt.Errorf("Invalid commit -- wrong set size: %v vs %v", valSet.Size(), len(commit.Precommits))
	}
	validatorsetlog.Debug("VerifyCommit will get commit height", "height", commit.Height())
	commitHeight := commit.Height()
	if height != commitHeight {
		return fmt.Errorf("VerifyCommit 1 Invalid commit -- wrong height: %v vs %v", height, commitHeight)
	}

	talliedVotingPower := int64(0)
	round := commit.Round()

	for idx, item := range commit.Precommits {
		// may be nil if validator skipped.
		if item == nil || len(item.Signature) == 0 {
			continue
		}
		precommit := &Vote{Vote: item}
		if precommit.Height != height {
			return fmt.Errorf("VerifyCommit 2 Invalid commit -- wrong height: %v vs %v", height, precommit.Height)
		}
		if int(precommit.Round) != round {
			return fmt.Errorf("Invalid commit -- wrong round: %v vs %v", round, precommit.Round)
		}
		if precommit.Type != uint32(VoteTypePrecommit) {
			return fmt.Errorf("Invalid commit -- not precommit @ index %v", idx)
		}
		_, val := valSet.GetByIndex(idx)
		// Validate signature

		precommitSignBytes := SignBytes(chainID, precommit)
		sig, err := ConsensusCrypto.SignatureFromBytes(precommit.Signature)
		if err != nil {
			return fmt.Errorf("VerifyCommit SignatureFromBytes [%X] failed:%v", precommit.Signature, err)
		}
		pubkey, err := ConsensusCrypto.PubKeyFromBytes(val.PubKey)
		if err != nil {
			return fmt.Errorf("VerifyCommit PubKeyFromBytes [%X] failed:%v", val.PubKey, err)
		}
		if !pubkey.VerifyBytes(precommitSignBytes, sig) {
			return fmt.Errorf("Invalid commit -- invalid signature: %v", precommit)
		}
		if !bytes.Equal(blockID.Hash, precommit.BlockID.Hash) {
			continue // Not an error, but doesn't count
		}
		// Good precommit!
		talliedVotingPower += val.VotingPower
	}

	if talliedVotingPower > valSet.TotalVotingPower()*2/3 {
		return nil
	} else {
		return fmt.Errorf("Invalid commit -- insufficient voting power: got %v, needed %v",
			talliedVotingPower, (valSet.TotalVotingPower()*2/3 + 1))
	}
}

// VerifyCommitAny will check to see if the set would
// be valid with a different validator set.
//
// valSet is the validator set that we know
// * over 2/3 of the power in old signed this block
//
// newSet is the validator set that signed this block
// * only votes from old are sufficient for 2/3 majority
//   in the new set as well
//
// That means that:
// * 10% of the valset can't just declare themselves kings
// * If the validator set is 3x old size, we need more proof to trust
func (valSet *ValidatorSet) VerifyCommitAny(newSet *ValidatorSet, chainID string,
	blockID BlockID, height int64, commit *Commit) error {

	if newSet.Size() != len(commit.Precommits) {
		return errors.Errorf("Invalid commit -- wrong set size: %v vs %v", newSet.Size(), len(commit.Precommits))
	}
	if height != commit.Height() {
		return errors.Errorf("Invalid commit -- wrong height: %v vs %v", height, commit.Height())
	}

	oldVotingPower := int64(0)
	newVotingPower := int64(0)
	seen := map[int]bool{}
	round := commit.Round()

	for idx, item := range commit.Precommits {
		// first check as in VerifyCommit
		if item == nil || len(item.Signature) == 0 {
			continue
		}
		precommit := &Vote{Vote: item}
		if precommit.Height != height {
			// return certerr.ErrHeightMismatch(height, precommit.Height)
			return errors.Errorf("Blocks don't match - %d vs %d", round, precommit.Round)
		}
		if int(precommit.Round) != round {
			return errors.Errorf("Invalid commit -- wrong round: %v vs %v", round, precommit.Round)
		}
		if precommit.Type != uint32(VoteTypePrecommit) {
			return errors.Errorf("Invalid commit -- not precommit @ index %v", idx)
		}
		if !bytes.Equal(blockID.Hash, precommit.BlockID.Hash) {
			continue // Not an error, but doesn't count
		}

		// we only grab by address, ignoring unknown validators
		vi, ov := valSet.GetByAddress(precommit.ValidatorAddress)
		if ov == nil || seen[vi] {
			continue // missing or double vote...
		}
		seen[vi] = true

		// Validate signature old school
		precommitSignBytes := SignBytes(chainID, precommit)
		sig, err := ConsensusCrypto.SignatureFromBytes(precommit.Signature)
		if err != nil {
			return fmt.Errorf("VerifyCommitAny SignatureFromBytes [%X] failed:%v", precommit.Signature, err)
		}
		pubkey, err := ConsensusCrypto.PubKeyFromBytes(ov.PubKey)
		if err != nil {
			return fmt.Errorf("VerifyCommitAny PubKeyFromBytes 1 [%X] failed:%v", ov.PubKey, err)
		}
		if !pubkey.VerifyBytes(precommitSignBytes, sig) {
			return errors.Errorf("Invalid commit -- invalid signature: %v", precommit)
		}
		// Good precommit!
		oldVotingPower += ov.VotingPower

		// check new school
		_, cv := newSet.GetByIndex(idx)
		cvPubKey, err := ConsensusCrypto.PubKeyFromBytes(cv.PubKey)
		if err != nil {
			return fmt.Errorf("VerifyCommitAny PubKeyFromBytes 2 [%X] failed:%v", cv.PubKey, err)
		}
		if cvPubKey.Equals(pubkey) {
			// make sure this is properly set in the current block as well
			newVotingPower += cv.VotingPower
		}
	}

	if oldVotingPower <= valSet.TotalVotingPower()*2/3 {
		return errors.Errorf("Invalid commit -- insufficient old voting power: got %v, needed %v",
			oldVotingPower, (valSet.TotalVotingPower()*2/3 + 1))
	} else if newVotingPower <= newSet.TotalVotingPower()*2/3 {
		return errors.Errorf("Invalid commit -- insufficient cur voting power: got %v, needed %v",
			newVotingPower, (newSet.TotalVotingPower()*2/3 + 1))
	}
	return nil
}

func (valSet *ValidatorSet) String() string {
	return valSet.StringIndented("")
}

func (valSet *ValidatorSet) StringIndented(indent string) string {
	if valSet == nil {
		return "nil-ValidatorSet"
	}
	valStrings := []string{}
	valSet.Iterate(func(index int, val *Validator) bool {
		valStrings = append(valStrings, val.String())
		return false
	})
	return Fmt(`ValidatorSet{
%s  Proposer: %v
%s  Validators:
%s    %v
%s}`,
		indent, valSet.GetProposer().String(),
		indent,
		indent, strings.Join(valStrings, "\n"+indent+"    "),
		indent)

}

//-------------------------------------
// Implements sort for sorting validators by address.

type ValidatorsByAddress []*Validator

func (vs ValidatorsByAddress) Len() int {
	return len(vs)
}

func (vs ValidatorsByAddress) Less(i, j int) bool {
	return bytes.Compare(vs[i].Address, vs[j].Address) == -1
}

func (vs ValidatorsByAddress) Swap(i, j int) {
	it := vs[i]
	vs[i] = vs[j]
	vs[j] = it
}

//-------------------------------------
// Use with Heap for sorting validators by accum

type accumComparable struct {
	*Validator
}

// We want to find the validator with the greatest accum.
func (ac accumComparable) Less(o interface{}) bool {
	other := o.(accumComparable).Validator
	larger := ac.CompareAccum(other)
	return bytes.Equal(larger.Address, ac.Address)
}
