package types

import (
	"bytes"
	"encoding/json"
	"fmt"

	"reflect"

	"github.com/gogo/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
)

// ErrEvidenceInvalid wraps a piece of evidence and the error denoting how or why it is invalid.
type ErrEvidenceInvalid struct {
	Evidence   Evidence
	ErrorValue error
}

func NewEvidenceInvalidErr(ev Evidence, err error) *ErrEvidenceInvalid {
	return &ErrEvidenceInvalid{ev, err}
}

// Error returns a string representation of the error.
func (err *ErrEvidenceInvalid) Error() string {
	return Fmt("Invalid evidence: %v. Evidence: %v", err.ErrorValue, err.Evidence)
}

//-------------------------------------------
const (
	DuplicateVote = "DuplicateVote"
	MockGood      = "MockGood"
	MockBad       = "MockBad"
)

var EvidenceType2Type map[string]reflect.Type
var EvidenceType2Obj map[string]Evidence

// Evidence represents any provable malicious activity by a validator
type Evidence interface {
	Height() int64               // height of the equivocation
	Address() []byte             // address of the equivocating validator
	Index() int                  // index of the validator in the validator set
	Hash() []byte                // hash of the evidence
	Verify(chainID string) error // verify the evidence
	Equal(Evidence) bool         // check equality of evidence

	String() string
	Copy() Evidence
	TypeName() string
	SetChild(child proto.Message)
	Child() proto.Message
}

//-------------------------------------------

// EvidenceList is a list of Evidence. Evidences is not a word.
type EvidenceList []Evidence

// Hash returns the simple merkle root hash of the EvidenceList.
func (evl EvidenceList) Hash() []byte {
	// Recursive impl.
	// Copied from tmlibs/merkle to avoid allocations
	switch len(evl) {
	case 0:
		return nil
	case 1:
		return evl[0].Hash()
	default:
		left := evl[:(len(evl)+1)/2].Hash()
		right := evl[(len(evl)+1)/2:].Hash()
		return merkle.GetHashFromTwoHash(left, right)
	}
}

func (evl EvidenceList) String() string {
	s := ""
	for _, e := range evl {
		s += Fmt("%s\t\t", e)
	}
	return s
}

// Has returns true if the evidence is in the EvidenceList.
func (evl EvidenceList) Has(evidence Evidence) bool {
	for _, ev := range evl {
		if ev.Equal(evidence) {
			return true
		}
	}
	return false
}

//-------------------------------------------

// DuplicateVoteEvidence contains evidence a validator signed two conflicting votes.
type DuplicateVoteEvidence struct {
	*tmtypes.DuplicateVoteEvidence
}

// String returns a string representation of the evidence.
func (dve *DuplicateVoteEvidence) String() string {
	return Fmt("VoteA: %v; VoteB: %v", dve.VoteA, dve.VoteB)

}

// Height returns the height this evidence refers to.
func (dve *DuplicateVoteEvidence) Height() int64 {
	return dve.VoteA.Height
}

// Address returns the address of the validator.
func (dve *DuplicateVoteEvidence) Address() []byte {
	pubkey, err := PubKeyFromString(dve.PubKey)
	if err != nil {
		return nil
	}
	return GenAddressByPubKey(pubkey)
}

// Index returns the index of the validator.
func (dve *DuplicateVoteEvidence) Index() int {
	return int(dve.VoteA.ValidatorIndex)
}

// Hash returns the hash of the evidence.
func (dve *DuplicateVoteEvidence) Hash() []byte {
	return SimpleHashFromBinary(dve)
}

// Verify returns an error if the two votes aren't conflicting.
// To be conflicting, they must be from the same validator, for the same H/R/S, but for different blocks.
func (dve *DuplicateVoteEvidence) Verify(chainID string) error {
	// H/R/S must be the same
	if dve.VoteA.Height != dve.VoteB.Height ||
		dve.VoteA.Round != dve.VoteB.Round ||
		dve.VoteA.Type != dve.VoteB.Type {
		return fmt.Errorf("DuplicateVoteEvidence Error: H/R/S does not match. Got %v and %v", dve.VoteA, dve.VoteB)
	}

	// Address must be the same
	if !bytes.Equal(dve.VoteA.ValidatorAddress, dve.VoteB.ValidatorAddress) {
		return fmt.Errorf("DuplicateVoteEvidence Error: Validator addresses do not match. Got %X and %X", dve.VoteA.ValidatorAddress, dve.VoteB.ValidatorAddress)
	}
	// XXX: Should we enforce index is the same ?
	if dve.VoteA.ValidatorIndex != dve.VoteB.ValidatorIndex {
		return fmt.Errorf("DuplicateVoteEvidence Error: Validator indices do not match. Got %d and %d", dve.VoteA.ValidatorIndex, dve.VoteB.ValidatorIndex)
	}

	blockIDA := BlockID{
		*dve.VoteA.BlockID,
	}
	blockIDB := BlockID{
		*dve.VoteB.BlockID,
	}
	// BlockIDs must be different
	if blockIDA.Equals(blockIDB) {
		return fmt.Errorf("DuplicateVoteEvidence Error: BlockIDs are the same (%v) - not a real duplicate vote!", dve.VoteA.BlockID)
	}

	// Signatures must be valid
	pubkey, err := PubKeyFromString(dve.PubKey)
	if err != nil {
		return fmt.Errorf("DuplicateVoteEvidence Error: pubkey[%v] to PubKey failed:%v", dve.PubKey, err)
	}
	sigA, err := ConsensusCrypto.SignatureFromBytes(dve.VoteA.Signature)
	if err != nil {
		return fmt.Errorf("DuplicateVoteEvidence Error: SIGA[%v] to signature failed:%v", dve.VoteA.Signature, err)
	}
	sigB, err := ConsensusCrypto.SignatureFromBytes(dve.VoteB.Signature)
	if err != nil {
		return fmt.Errorf("DuplicateVoteEvidence Error: SIGB[%v] to signature failed:%v", dve.VoteB.Signature, err)
	}
	vote := &Vote{
		dve.VoteA,
	}
	if !pubkey.VerifyBytes(SignBytes(chainID, vote), sigA) {
		return fmt.Errorf("DuplicateVoteEvidence Error verifying VoteA: %v", ErrVoteInvalidSignature)
	}
	vote = &Vote{
		dve.VoteB,
	}
	if !pubkey.VerifyBytes(SignBytes(chainID, vote), sigB) {
		return fmt.Errorf("DuplicateVoteEvidence Error verifying VoteB: %v", ErrVoteInvalidSignature)
	}

	return nil
}

// Equal checks if two pieces of evidence are equal.
func (dve *DuplicateVoteEvidence) Equal(ev Evidence) bool {
	if _, ok := ev.(*DuplicateVoteEvidence); !ok {
		return false
	}
	if dve == nil {
		return false
	}
	// just check their hashes
	return bytes.Equal(SimpleHashFromBinary(dve), SimpleHashFromBinary(ev.(*DuplicateVoteEvidence)))
}

func (dve *DuplicateVoteEvidence) TypeName() string {
	return DuplicateVote
}

func (dve *DuplicateVoteEvidence) Copy() Evidence {
	return &DuplicateVoteEvidence{}
}

func (dve *DuplicateVoteEvidence) SetChild(child proto.Message) {
	dve.DuplicateVoteEvidence = child.(*tmtypes.DuplicateVoteEvidence)
}

func (dve *DuplicateVoteEvidence) Child() proto.Message {
	return dve.DuplicateVoteEvidence
}

func SimpleHashFromBinary(item *DuplicateVoteEvidence) []byte {
	bytes, e := json.Marshal(item)
	if e != nil {
		//commonlog.Error("SimpleHashFromBinary marshal failed", "type", item, "error", e)
		panic(Fmt("SimpleHashFromBinary marshal failed, err:%v", e))
	}
	return crypto.Ripemd160(bytes)

}

func EvidenceEnvelope2Evidence(envelope *tmtypes.EvidenceEnvelope) Evidence {
	if v, ok := EvidenceType2Type[envelope.TypeName]; ok {
		realMsg2 := reflect.New(v).Interface()
		err := proto.Unmarshal(envelope.Data, realMsg2.(proto.Message))
		if err != nil {
			panic(Fmt("Evidence is not valid", "evidenceType", envelope.TypeName, "err", err))
		}
		if evidence, ok2 := EvidenceType2Obj[envelope.TypeName]; ok2 {
			evidence = evidence.Copy()
			evidence.SetChild(realMsg2.(proto.Message))
			return evidence.(Evidence)
		}
	}
	return nil
}

//-----------------------------------------------------------------

// UNSTABLE
type MockGoodEvidence struct {
	Height_  int64
	Address_ []byte
	Index_   int
}

// UNSTABLE
func NewMockGoodEvidence(height int64, index int, address []byte) MockGoodEvidence {
	return MockGoodEvidence{height, address, index}
}

func (e MockGoodEvidence) Height() int64   { return e.Height_ }
func (e MockGoodEvidence) Address() []byte { return e.Address_ }
func (e MockGoodEvidence) Index() int      { return e.Index_ }
func (e MockGoodEvidence) Hash() []byte {
	return []byte(Fmt("%d-%d", e.Height_, e.Index_))
}
func (e MockGoodEvidence) Verify(chainID string) error { return nil }
func (e MockGoodEvidence) Equal(ev Evidence) bool {
	e2 := ev.(MockGoodEvidence)
	return e.Height_ == e2.Height_ &&
		bytes.Equal(e.Address_, e2.Address_) &&
		e.Index_ == e2.Index_
}
func (e MockGoodEvidence) String() string {
	return Fmt("GoodEvidence: %d/%s/%d", e.Height_, e.Address_, e.Index_)
}
func (e MockGoodEvidence) TypeName() string {
	return MockGood
}
func (e MockGoodEvidence) Copy() Evidence {
	return &MockGoodEvidence{}
}
func (e MockGoodEvidence) SetChild(proto.Message) {}
func (e MockGoodEvidence) Child() proto.Message {
	return nil
}

// UNSTABLE
type MockBadEvidence struct {
	MockGoodEvidence
}

func (e MockBadEvidence) Verify(chainID string) error { return fmt.Errorf("MockBadEvidence") }
func (e MockBadEvidence) Equal(ev Evidence) bool {
	e2 := ev.(MockBadEvidence)
	return e.Height_ == e2.Height_ &&
		bytes.Equal(e.Address_, e2.Address_) &&
		e.Index_ == e2.Index_
}
func (e MockBadEvidence) String() string {
	return Fmt("BadEvidence: %d/%s/%d", e.Height_, e.Address_, e.Index_)
}
func (e MockBadEvidence) TypeName() string {
	return MockBad
}
func (e MockBadEvidence) Copy() Evidence {
	return &MockBadEvidence{}
}
func (e MockBadEvidence) SetChild(proto.Message) {}
func (e MockBadEvidence) Child() proto.Message {
	return nil
}

//------------------------------------------------------
// evidence pool

// EvidencePool defines the EvidencePool interface used by the ConsensusState.
// UNSTABLE
type EvidencePool interface {
	PendingEvidence() []Evidence
	AddEvidence(Evidence) error
	Update(*TendermintBlock)
}

// MockMempool is an empty implementation of a Mempool, useful for testing.
// UNSTABLE
type MockEvidencePool struct {
}

func (m MockEvidencePool) PendingEvidence() []Evidence { return nil }
func (m MockEvidencePool) AddEvidence(Evidence) error  { return nil }
func (m MockEvidencePool) Update(*TendermintBlock)     {}
