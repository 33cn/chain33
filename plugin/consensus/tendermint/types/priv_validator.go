package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/common/crypto"
)

// TODO: type ?
const (
	stepNone      = 0 // Used to distinguish the initial state
	stepPropose   = 1
	stepPrevote   = 2
	stepPrecommit = 3
)

type KeyText struct {
	Kind string `json:"type"`
	Data string `json:"data"`
}

func voteToStep(vote *Vote) int8 {
	switch vote.Type {
	case uint32(VoteTypePrevote):
		return stepPrevote
	case uint32(VoteTypePrecommit):
		return stepPrecommit
	default:
		PanicSanity("Unknown vote type")
		return 0
	}
}

// PrivValidator defines the functionality of a local Tendermint validator
// that signs votes, proposals, and heartbeats, and never double signs.
type PrivValidator interface {
	GetAddress() []byte // redundant since .PubKey().Address()
	GetPubKey() crypto.PubKey

	SignVote(chainID string, vote *Vote) error
	SignProposal(chainID string, proposal *Proposal) error
	SignHeartbeat(chainID string, heartbeat *Heartbeat) error

	GetLastHeight() int64
	GetLastRound() int
	GetLastStep() int8

	//reset height,round,step used by start to catch up
	ResetLastHeight(height int64)
}

// PrivValidatorFS implements PrivValidator using data persisted to disk
// to prevent double signing. The Signer itself can be mutated to use
// something besides the default, for instance a hardware signer.
type PrivValidatorFS struct {
	Address       string   `json:"address"`
	PubKey        KeyText  `json:"pub_key"`
	LastHeight    int64    `json:"last_height"`
	LastRound     int      `json:"last_round"`
	LastStep      int8     `json:"last_step"`
	LastSignature *KeyText `json:"last_signature,omitempty"` // so we dont lose signatures
	LastSignBytes string   `json:"last_signbytes,omitempty"` // so we dont lose signatures

	// PrivKey should be empty if a Signer other than the default is being used.
	PrivKey KeyText `json:"priv_key"`
}

type PrivValidatorImp struct {
	Address       []byte
	PubKey        crypto.PubKey
	LastHeight    int64
	LastRound     int
	LastStep      int8
	LastSignature crypto.Signature
	LastSignBytes []byte

	// PrivKey should be empty if a Signer other than the default is being used.
	PrivKey crypto.PrivKey
	Signer  `json:"-"`

	// For persistence.
	// Overloaded for testing.
	filePath string
	mtx      sync.Mutex
}

// Signer is an interface that defines how to sign messages.
// It is the caller's duty to verify the msg before calling Sign,
// eg. to avoid double signing.
// Currently, the only callers are SignVote, SignProposal, and SignHeartbeat.
type Signer interface {
	Sign(msg []byte) (crypto.Signature, error)
}

// DefaultSigner implements Signer.
// It uses a standard, unencrypted crypto.PrivKey.
type DefaultSigner struct {
	PrivKey crypto.PrivKey `json:"priv_key"`
}

// NewDefaultSigner returns an instance of DefaultSigner.
func NewDefaultSigner(priv crypto.PrivKey) *DefaultSigner {
	return &DefaultSigner{
		PrivKey: priv,
	}
}

// Sign implements Signer. It signs the byte slice with a private key.
func (ds *DefaultSigner) Sign(msg []byte) (crypto.Signature, error) {
	return ds.PrivKey.Sign(msg), nil
}

// GetAddress returns the address of the validator.
// Implements PrivValidator.
func (pv *PrivValidatorImp) GetAddress() []byte {
	return pv.Address
}

// GetPubKey returns the public key of the validator.
// Implements PrivValidator.
func (pv *PrivValidatorImp) GetPubKey() crypto.PubKey {
	return pv.PubKey
}

func GenAddressByPubKey(pubkey crypto.PubKey) []byte {
	//must add 3 bytes ahead to make compatibly
	typeAddr := append([]byte{byte(0x01), byte(0x01), byte(0x20)}, pubkey.Bytes()...)
	return crypto.Ripemd160(typeAddr)
}

func PubKeyFromString(pubkeystring string) (crypto.PubKey, error) {
	pub, err := hex.DecodeString(pubkeystring)
	if err != nil {
		return nil, errors.New(Fmt("PubKeyFromString:DecodeString:%v failed,err:%v", pubkeystring, err))
	}

	pubkey, err := ConsensusCrypto.PubKeyFromBytes(pub)
	if err != nil {
		return nil, errors.New(Fmt("PubKeyFromString:PubKeyFromBytes:%v failed,err:%v", pub, err))
	}
	return pubkey, nil
}

func SignatureFromString(sigString string) (crypto.Signature, error) {
	sigbyte, err := hex.DecodeString(sigString)
	if err != nil {
		return nil, errors.New(Fmt("PubKeyFromString:DecodeString:%v failed,err:%v", sigString, err))
	}
	sig, err := ConsensusCrypto.SignatureFromBytes(sigbyte)
	if err != nil {
		return nil, errors.New(Fmt("PubKeyFromString:SignatureFromBytes:%v failed,err:%v", sigbyte, err))
	}
	return sig, nil
}

// GenPrivValidatorImp generates a new validator with randomly generated private key
// and sets the filePath, but does not call Save().
func GenPrivValidatorImp(filePath string) *PrivValidatorImp {
	privKey, err := ConsensusCrypto.GenKey()
	if err != nil {
		panic(Fmt("GenPrivValidatorImp: GenKey failed:%v", err))
	}
	return &PrivValidatorImp{
		Address:  GenAddressByPubKey(privKey.PubKey()),
		PubKey:   privKey.PubKey(),
		PrivKey:  privKey,
		LastStep: stepNone,
		Signer:   NewDefaultSigner(privKey),
		filePath: filePath,
	}
}

// LoadPrivValidatorFS loads a PrivValidatorImp from the filePath.
func LoadPrivValidatorFS(filePath string) *PrivValidatorImp {
	return LoadPrivValidatorFSWithSigner(filePath, func(privVal PrivValidator) Signer {
		return NewDefaultSigner(privVal.(*PrivValidatorImp).PrivKey)
	})
}

// LoadOrGenPrivValidatorFS loads a PrivValidatorFS from the given filePath
// or else generates a new one and saves it to the filePath.
func LoadOrGenPrivValidatorFS(filePath string) *PrivValidatorImp {
	var privVal *PrivValidatorImp
	if _, err := os.Stat(filePath); err == nil {
		privVal = LoadPrivValidatorFS(filePath)
	} else {
		privVal = GenPrivValidatorImp(filePath)
		privVal.Save()
	}
	return privVal
}

// LoadPrivValidatorWithSigner loads a PrivValidatorFS with a custom
// signer object. The PrivValidatorFS handles double signing prevention by persisting
// data to the filePath, while the Signer handles the signing.
// If the filePath does not exist, the PrivValidatorFS must be created manually and saved.
func LoadPrivValidatorFSWithSigner(filePath string, signerFunc func(PrivValidator) Signer) *PrivValidatorImp {
	privValJSONBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		Exit(err.Error())
	}
	privVal := &PrivValidatorFS{}
	err = json.Unmarshal(privValJSONBytes, &privVal)
	if err != nil {
		Exit(Fmt("Error reading PrivValidator from %v: %v\n", filePath, err))
	}
	if len(privVal.PubKey.Data) == 0 {
		Exit("Error PrivValidator pubkey is empty\n")
	}
	if len(privVal.PrivKey.Data) == 0 {
		Exit("Error PrivValidator privkey is empty\n")
	}
	addr, err := hex.DecodeString(privVal.Address)
	if err != nil {
		Exit(Fmt("Error PrivValidator DecodeString failed:%v\n", err))
	}
	privValImp := &PrivValidatorImp{
		Address:    addr,
		LastHeight: privVal.LastHeight,
		LastRound:  privVal.LastRound,
		LastStep:   privVal.LastStep,
	}
	tmp, err := hex.DecodeString(privVal.PrivKey.Data)
	if err != nil {
		Exit(Fmt("Error DecodeString PrivKey data failed: %v\n", err))
	}
	privKey, err := ConsensusCrypto.PrivKeyFromBytes(tmp)
	if err != nil {
		Exit(Fmt("Error PrivKeyFromBytes failed: %v\n", err))
	}
	privValImp.PrivKey = privKey

	pubKey, err := PubKeyFromString(privVal.PubKey.Data)
	if err != nil {
		Exit(Fmt("Error PubKeyFromBytes failed: %v\n", err))
	}
	privValImp.PubKey = pubKey

	if len(privVal.LastSignBytes) != 0 {
		tmp, err = hex.DecodeString(privVal.LastSignBytes)
		if err != nil {
			Exit(Fmt("Error DecodeString LastSignBytes data failed: %v\n", err))
		}
		privValImp.LastSignBytes = tmp
	}
	if privVal.LastSignature != nil {
		signature, err := SignatureFromString(privVal.LastSignature.Data)
		if err != nil {
			Exit(Fmt("Error SignatureFromBytes failed: %v\n", err))
		}
		privValImp.LastSignature = signature
	} else {
		privValImp.LastSignature = nil
	}

	privValImp.filePath = filePath
	privValImp.Signer = signerFunc(privValImp)
	return privValImp
}

// Save persists the PrivValidatorFS to disk.
func (privVal *PrivValidatorImp) Save() {
	privVal.mtx.Lock()
	defer privVal.mtx.Unlock()
	privVal.save()
}

func (privVal *PrivValidatorImp) save() {
	if privVal.filePath == "" {
		PanicSanity("Cannot save PrivValidator: filePath not set")
	}
	addr := Fmt("%X", privVal.Address[:])

	privValFS := &PrivValidatorFS{
		Address:       addr,
		LastHeight:    privVal.LastHeight,
		LastRound:     privVal.LastRound,
		LastStep:      privVal.LastStep,
		LastSignature: nil,
	}
	privValFS.PrivKey = KeyText{Kind: "ed25519", Data: Fmt("%X", privVal.PrivKey.Bytes()[:])}
	privValFS.PubKey = KeyText{Kind: "ed25519", Data: privVal.PubKey.KeyString()}
	if len(privVal.LastSignBytes) != 0 {
		tmp := Fmt("%X", privVal.LastSignBytes[:])
		privValFS.LastSignBytes = tmp
	}
	if privVal.LastSignature != nil {
		sig := Fmt("%X", privVal.LastSignature.Bytes()[:])
		privValFS.LastSignature = &KeyText{Kind: "ed25519", Data: sig}
	}
	jsonBytes, err := json.Marshal(privValFS)
	if err != nil {
		// `@; BOOM!!!
		PanicCrisis(err)
	}
	err = WriteFileAtomic(privVal.filePath, jsonBytes, 0600)
	if err != nil {
		// `@; BOOM!!!
		PanicCrisis(err)
	}
}

// Reset resets all fields in the PrivValidatorFS.
// NOTE: Unsafe!
func (privVal *PrivValidatorImp) Reset() {
	privVal.LastHeight = 0
	privVal.LastRound = 0
	privVal.LastStep = 0
	privVal.LastSignature = nil
	privVal.LastSignBytes = nil
	privVal.Save()
}

// SignVote signs a canonical representation of the vote, along with the
// chainID. Implements PrivValidator.
func (privVal *PrivValidatorImp) SignVote(chainID string, vote *Vote) error {
	privVal.mtx.Lock()
	defer privVal.mtx.Unlock()
	signature, err := privVal.signBytesHRS(vote.Height, int(vote.Round), voteToStep(vote),
		SignBytes(chainID, vote), checkVotesOnlyDifferByTimestamp)
	if err != nil {
		return errors.New(Fmt("Error signing vote: %v", err))
	}
	vote.Signature = signature.Bytes()
	return nil
}

// SignProposal signs a canonical representation of the proposal, along with
// the chainID. Implements PrivValidator.
func (privVal *PrivValidatorImp) SignProposal(chainID string, proposal *Proposal) error {
	privVal.mtx.Lock()
	defer privVal.mtx.Unlock()
	signature, err := privVal.signBytesHRS(proposal.Height, int(proposal.Round), stepPropose,
		SignBytes(chainID, proposal), checkProposalsOnlyDifferByTimestamp)
	if err != nil {
		return fmt.Errorf("Error signing proposal: %v", err)
	}
	proposal.Signature = signature.Bytes()
	return nil
}

// returns error if HRS regression or no LastSignBytes. returns true if HRS is unchanged
func (privVal *PrivValidatorImp) checkHRS(height int64, round int, step int8) (bool, error) {
	if privVal.LastHeight > height {
		return false, errors.New("Height regression")
	}

	if privVal.LastHeight == height {
		if privVal.LastRound > round {
			return false, errors.New("Round regression")
		}

		if privVal.LastRound == round {
			if privVal.LastStep > step {
				return false, errors.New("Step regression")
			} else if privVal.LastStep == step {
				if privVal.LastSignBytes != nil {
					if privVal.LastSignature == nil {
						panic("privVal: LastSignature is nil but LastSignBytes is not!")
					}
					return true, nil
				}
				return false, errors.New("No LastSignature found")
			}
		}
	}
	return false, nil
}

// signBytesHRS signs the given signBytes if the height/round/step (HRS) are
// greater than the latest state. If the HRS are equal and the only thing changed is the timestamp,
// it returns the privValidator.LastSignature. Else it returns an error.
func (privVal *PrivValidatorImp) signBytesHRS(height int64, round int, step int8,
	signBytes []byte, checkFn checkOnlyDifferByTimestamp) (crypto.Signature, error) {

	sameHRS, err := privVal.checkHRS(height, round, step)
	if err != nil {
		return nil, err
	}

	// We might crash before writing to the wal,
	// causing us to try to re-sign for the same HRS
	if sameHRS {
		// if they're the same or only differ by timestamp,
		// return the LastSignature. Otherwise, error
		if bytes.Equal(signBytes, privVal.LastSignBytes) ||
			checkFn(privVal.LastSignBytes, signBytes) {
			return privVal.LastSignature, nil
		}
		return nil, fmt.Errorf("Conflicting data")
	}

	sig, err := privVal.Sign(signBytes)
	if err != nil {
		return nil, err
	}
	//privVal.saveSigned(height, round, step, signBytes, sig)
	return sig, nil
}

// Persist height/round/step and signature
func (privVal *PrivValidatorImp) saveSigned(height int64, round int, step int8,
	signBytes []byte, sig crypto.Signature) {

	privVal.LastHeight = height
	privVal.LastRound = round
	privVal.LastStep = step
	privVal.LastSignature = sig
	privVal.LastSignBytes = signBytes
	privVal.save()
}

// SignHeartbeat signs a canonical representation of the heartbeat, along with the chainID.
// Implements PrivValidator.
func (privVal *PrivValidatorImp) SignHeartbeat(chainID string, heartbeat *Heartbeat) error {
	privVal.mtx.Lock()
	defer privVal.mtx.Unlock()
	sig, err := privVal.Sign(SignBytes(chainID, heartbeat))
	heartbeat.Signature = sig.Bytes()
	return err
}

// String returns a string representation of the PrivValidatorImp.
func (privVal *PrivValidatorImp) String() string {
	return Fmt("PrivValidator{%v LH:%v, LR:%v, LS:%v}", privVal.GetAddress(), privVal.LastHeight, privVal.LastRound, privVal.LastStep)
}

func (privVal *PrivValidatorImp) GetLastHeight() int64 {
	return privVal.LastHeight
}

func (privVal *PrivValidatorImp) GetLastRound() int {
	return privVal.LastRound
}

func (privVal *PrivValidatorImp) GetLastStep() int8 {
	return privVal.LastStep
}

func (privVal *PrivValidatorImp) ResetLastHeight(height int64) {
	privVal.LastHeight = height
	privVal.LastRound = 0
	privVal.LastStep = 0
}

//-------------------------------------

type PrivValidatorsByAddress []*PrivValidatorImp

func (pvs PrivValidatorsByAddress) Len() int {
	return len(pvs)
}

func (pvs PrivValidatorsByAddress) Less(i, j int) bool {
	return bytes.Compare(pvs[i].GetAddress(), pvs[j].GetAddress()) == -1
}

func (pvs PrivValidatorsByAddress) Swap(i, j int) {
	it := pvs[i]
	pvs[i] = pvs[j]
	pvs[j] = it
}

//-------------------------------------

type checkOnlyDifferByTimestamp func([]byte, []byte) bool

// returns true if the only difference in the votes is their timestamp
func checkVotesOnlyDifferByTimestamp(lastSignBytes, newSignBytes []byte) bool {
	var lastVote, newVote CanonicalJSONOnceVote
	if err := json.Unmarshal(lastSignBytes, &lastVote); err != nil {
		panic(Fmt("LastSignBytes cannot be unmarshalled into vote: %v", err))
	}
	if err := json.Unmarshal(newSignBytes, &newVote); err != nil {
		panic(Fmt("signBytes cannot be unmarshalled into vote: %v", err))
	}

	// set the times to the same value and check equality
	now := CanonicalTime(time.Now())
	lastVote.Vote.Timestamp = now
	newVote.Vote.Timestamp = now
	lastVoteBytes, _ := json.Marshal(lastVote)
	newVoteBytes, _ := json.Marshal(newVote)

	return bytes.Equal(newVoteBytes, lastVoteBytes)
}

// returns true if the only difference in the proposals is their timestamp
func checkProposalsOnlyDifferByTimestamp(lastSignBytes, newSignBytes []byte) bool {
	var lastProposal, newProposal CanonicalJSONOnceProposal
	if err := json.Unmarshal(lastSignBytes, &lastProposal); err != nil {
		panic(Fmt("LastSignBytes cannot be unmarshalled into proposal: %v", err))
	}
	if err := json.Unmarshal(newSignBytes, &newProposal); err != nil {
		panic(Fmt("signBytes cannot be unmarshalled into proposal: %v", err))
	}

	// set the times to the same value and check equality
	now := CanonicalTime(time.Now())
	lastProposal.Proposal.Timestamp = now
	newProposal.Proposal.Timestamp = now
	lastProposalBytes, _ := json.Marshal(lastProposal)
	newProposalBytes, _ := json.Marshal(newProposal)

	return bytes.Equal(newProposalBytes, lastProposalBytes)
}
