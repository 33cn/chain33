package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	cmn "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"encoding/json"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"github.com/golang/protobuf/proto"
)

var (
	blocklog = log15.New("module", "tendermint-block")
)

// Block defines the atomic unit of a Tendermint blockchain.
// TODO: add Version byte
type Block struct {
	*Header    `json:"header"`
	Evidence   EvidenceData `json:"evidence"`
	LastCommit *Commit      `json:"last_commit"`
	BlockBytes  []byte       `json:"block_bytes"`
}

// MakeBlock returns a new block with an empty header, except what can be computed from itself.
// It populates the same set of fields validated by ValidateBasic
func MakeBlock(height int64, txs []*types.Transaction, commit *Commit) *Block {
	curTime := time.Now().Unix()
    oriBlock:= &types.Block{
		Height:    height,
		BlockTime: curTime,
		Txs:       txs,
		TxHash:    merkle.CalcMerkleRoot(txs),
	}
	blockByte, err := proto.Marshal(oriBlock)
	if err != nil {
		blocklog.Error("MakeBlock failed", "error", err)
		return nil
	}

	block := &Block{
		Header: &Header{
			Height: height,
			Time : curTime,
			NumTxs: int64(len(txs)),
		},
		LastCommit: commit,
		BlockBytes: blockByte,
	}
	block.FillHeader()
	return block
}

// AddEvidence appends the given evidence to the block
func (b *Block) AddEvidence(evidence []Evidence) {
	for _, item := range evidence {
		data, err := json.Marshal(item)
		if err != nil {
			blocklog.Error("AddEvidence marshal failed", "error", err)
			return
		}
		msg := json.RawMessage(data)
		enve := MsgEnvelope{
			Kind: item.TypeName(),
			Data: &msg,
		}
		b.Evidence.Evidence = append(b.Evidence.Evidence, enve)
	}
}

// ValidateBasic performs basic validation that doesn't involve state data.
// It checks the internal consistency of the block.
func (b *Block) ValidateBasic() (int64,error) {
	block := types.Block{}
	err := proto.Unmarshal(b.BlockBytes, &block)
	if err != nil {
		blocklog.Error("ValidateBasic unmarshal failed", "error", err)
		return 0, err
	}
	newTxs := int64(len(block.Txs))

	if b.NumTxs != newTxs {
		return 0, fmt.Errorf("Wrong Block.Header.NumTxs. Expected %v, got %v", newTxs, b.NumTxs)
	}
	if !bytes.Equal(b.LastCommitHash, b.LastCommit.Hash()) {
		return 0, fmt.Errorf("Wrong Block.Header.LastCommitHash.  Expected %v, got %v", b.LastCommitHash, b.LastCommit.Hash())
	}
	if b.Height != 1 {
		if err := b.LastCommit.ValidateBasic(); err != nil {
			return 0, err
		}
	}

	calHash := merkle.CalcMerkleRoot(block.Txs)
	if !bytes.Equal(block.TxHash, calHash) {
		return 0, fmt.Errorf("Wrong Block.Header.DataHash.  Expected %v, got %v", block.TxHash, calHash)
	}

	if !bytes.Equal(b.EvidenceHash, b.Evidence.Hash()) {
		return 0, errors.New(fmt.Sprintf("Wrong Block.Header.EvidenceHash.  Expected %v, got %v", b.EvidenceHash, b.Evidence.Hash()))
	}
	return newTxs, nil
}

// FillHeader fills in any remaining header fields that are a function of the block data
func (b *Block) FillHeader() {
	if b.LastCommitHash == nil {
		b.LastCommitHash = b.LastCommit.Hash()
	}
	if b.EvidenceHash == nil {
		b.EvidenceHash = b.Evidence.Hash()
	}
}

// Hash computes and returns the block hash.
// If the block is incomplete, block hash is nil for safety.
func (b *Block) Hash() []byte {
	if b == nil || b.Header == nil || b.LastCommit == nil {
		return nil
	}
	b.FillHeader()
	return b.Header.Hash()
}

// HashesTo is a convenience function that checks if a block hashes to the given argument.
// A nil block never hashes to anything, and nothing hashes to a nil hash.
func (b *Block) HashesTo(hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	if b == nil {
		return false
	}
	return bytes.Equal(b.Hash(), hash)
}

// String returns a string representation of the block
func (b *Block) String() string {
	return b.StringIndented("")
}

// StringIndented returns a string representation of the block
func (b *Block) StringIndented(indent string) string {
	if b == nil {
		return "nil-Block"
	}
	return fmt.Sprintf(`Block{
%s  %v
%s  %v
%s  %v
%s}#%v`,
		indent, b.Header.StringIndented(indent+"  "),
		indent, b.Evidence.StringIndented(indent+"  "),
		indent, b.LastCommit.StringIndented(indent+"  "),
		indent, b.Hash())
}

// StringShort returns a shortened string representation of the block
func (b *Block) StringShort() string {
	if b == nil {
		return "nil-Block"
	} else {
		return fmt.Sprintf("Block#%v", b.Hash())
	}
}

//-----------------------------------------------------------------------------

// Header defines the structure of a Tendermint block header
// TODO: limit header size
// NOTE: changes to the Header should be duplicated in the abci Header
type Header struct {
	// basic block info
	ChainID string    `json:"chain_id"`
	Height  int64     `json:"height"`
	Time    int64     `json:"time"`
	NumTxs  int64     `json:"num_txs"`

	// prev block info
	LastBlockID BlockID `json:"last_block_id"`
	TotalTxs    int64   `json:"total_txs"`

	// hashes of block data
	LastCommitHash []byte `json:"last_commit_hash"` // commit from validators from the last block

	// hashes from the app output from the prev block
	ValidatorsHash  []byte `json:"validators_hash"`   // validators for the current block
	ConsensusHash   []byte `json:"consensus_hash"`    // consensus params for current block
	AppHash         []byte `json:"app_hash"`          // state after txs from the previous block
	LastResultsHash []byte `json:"last_results_hash"` // root hash of all results from the txs from the previous block

	// consensus info
	EvidenceHash []byte `json:"evidence_hash"` // evidence included in the block
}

// Hash returns the hash of the header.
// Returns nil if ValidatorHash is missing.
func (h *Header) Hash() []byte {
	if len(h.ValidatorsHash) == 0 {
		return nil
	}
	bytes, err := json.Marshal(h)
	if err != nil {
		blocklog.Error("block header Hash() marshal failed", "error", err)
		return nil
	}
	return crypto.Ripemd160(bytes)
}

// StringIndented returns a string representation of the header
func (h *Header) StringIndented(indent string) string {
	if h == nil {
		return "nil-Header"
	}
	return fmt.Sprintf(`Header{
%s  ChainID:        %v
%s  Height:         %v
%s  Time:           %v
%s  NumTxs:         %v
%s  TotalTxs:       %v
%s  LastBlockID:    %v
%s  LastCommit:     %v
%s  Validators:     %v
%s  App:            %v
%s  Conensus:       %v
%s  Results:        %v
%s  Evidence:       %v
%s}#%v`,
		indent, h.ChainID,
		indent, h.Height,
		indent, time.Unix(0,h.Time),
		indent, h.NumTxs,
		indent, h.TotalTxs,
		indent, h.LastBlockID,
		indent, h.LastCommitHash,
		indent, h.ValidatorsHash,
		indent, h.AppHash,
		indent, h.ConsensusHash,
		indent, h.LastResultsHash,
		indent, h.EvidenceHash,
		indent, h.Hash())
}

//-------------------------------------

// Commit contains the evidence that a block was committed by a set of validators.
// NOTE: Commit is empty for height 1, but never nil.
type Commit struct {
	// NOTE: The Precommits are in order of address to preserve the bonded ValidatorSet order.
	// Any peer with a block can gossip precommits by index with a peer without recalculating the
	// active ValidatorSet.
	BlockID    BlockID `json:"blockID"`
	Precommits []*Vote `json:"precommits"`

	// Volatile
	firstPrecommit *Vote
	hash           []byte
	bitArray       *cmn.BitArray
}

// FirstPrecommit returns the first non-nil precommit in the commit
func (commit *Commit) FirstPrecommit() *Vote {
	if len(commit.Precommits) == 0 {
		return nil
	}
	if commit.firstPrecommit != nil {
		return commit.firstPrecommit
	}
	for _, precommit := range commit.Precommits {
		if precommit != nil {
			commit.firstPrecommit = precommit
			return precommit
		}
	}
	return nil
}

// Height returns the height of the commit
func (commit *Commit) Height() int64 {
	if len(commit.Precommits) == 0 {
		return 0
	}
	return commit.FirstPrecommit().Height
}

// Round returns the round of the commit
func (commit *Commit) Round() int {
	if len(commit.Precommits) == 0 {
		return 0
	}
	return commit.FirstPrecommit().Round
}

// Type returns the vote type of the commit, which is always VoteTypePrecommit
func (commit *Commit) Type() byte {
	return VoteTypePrecommit
}

// Size returns the number of votes in the commit
func (commit *Commit) Size() int {
	if commit == nil {
		return 0
	}
	return len(commit.Precommits)
}

// BitArray returns a BitArray of which validators voted in this commit
func (commit *Commit) BitArray() *cmn.BitArray {
	if commit.bitArray == nil {
		commit.bitArray = cmn.NewBitArray(len(commit.Precommits))
		for i, precommit := range commit.Precommits {
			// TODO: need to check the BlockID otherwise we could be counting conflicts,
			// not just the one with +2/3 !
			commit.bitArray.SetIndex(i, precommit != nil)
		}
	}
	return commit.bitArray
}

// GetByIndex returns the vote corresponding to a given validator index
func (commit *Commit) GetByIndex(index int) *Vote {
	return commit.Precommits[index]
}

// IsCommit returns true if there is at least one vote
func (commit *Commit) IsCommit() bool {
	return len(commit.Precommits) != 0
}

// ValidateBasic performs basic validation that doesn't involve state data.
func (commit *Commit) ValidateBasic() error {
	if commit.BlockID.IsZero() {
		return errors.New("Commit cannot be for nil block")
	}
	if len(commit.Precommits) == 0 {
		return errors.New("No precommits in commit")
	}
	height, round := commit.Height(), commit.Round()

	// validate the precommits
	for _, precommit := range commit.Precommits {
		// It's OK for precommits to be missing.
		if precommit == nil {
			continue
		}
		// Ensure that all votes are precommits
		if precommit.Type != VoteTypePrecommit {
			return fmt.Errorf("Invalid commit vote. Expected precommit, got %v",
				precommit.Type)
		}
		// Ensure that all heights are the same
		if precommit.Height != height {
			return fmt.Errorf("Invalid commit precommit height. Expected %v, got %v",
				height, precommit.Height)
		}
		// Ensure that all rounds are the same
		if precommit.Round != round {
			return fmt.Errorf("Invalid commit precommit round. Expected %v, got %v",
				round, precommit.Round)
		}
	}
	return nil
}

// Hash returns the hash of the commit
func (commit *Commit) Hash() []byte {
	if commit.hash == nil {
		bs := make([][]byte, len(commit.Precommits))
		for i, precommit := range commit.Precommits {
			bs[i] = precommit.Hash()
		}
		commit.hash = merkle.GetMerkleRoot(bs)
	}
	return commit.hash
}

// StringIndented returns a string representation of the commit
func (commit *Commit) StringIndented(indent string) string {
	if commit == nil {
		return "nil-Commit"
	}
	precommitStrings := make([]string, len(commit.Precommits))
	for i, precommit := range commit.Precommits {
		precommitStrings[i] = precommit.String()
	}
	return fmt.Sprintf(`Commit{
%s  BlockID:    %v
%s  Precommits: %v
%s}#%v`,
		indent, commit.BlockID,
		indent, strings.Join(precommitStrings, "\n"+indent+"  "),
		indent, commit.hash)
}

//-----------------------------------------------------------------------------

// SignedHeader is a header along with the commits that prove it
type SignedHeader struct {
	Header *Header `json:"header"`
	Commit *Commit `json:"commit"`
}

//-----------------------------------------------------------------------------

// EvidenceData contains any evidence of malicious wrong-doing by validators
type EvidenceEnvelopeList []MsgEnvelope

func (env MsgEnvelope) Hash() []byte {
	if v, ok := EvidenceType2Obj[env.Kind]; ok {
		tmp := v.(Evidence).Copy()
		err := json.Unmarshal(*env.Data, &tmp)
		if err != nil {
			blocklog.Error("envelop hash unmarshal failed", "error", err)
			return nil
		}
		return tmp.Hash()
	}
	blocklog.Error("envelop hash not find evidence kind", "kind", env.Kind)
	return nil
}

func (env MsgEnvelope) String() string {
	if v, ok := EvidenceType2Obj[env.Kind]; ok {
		tmp := v.(Evidence).Copy()
		err := json.Unmarshal(*env.Data, &tmp)
		if err != nil {
			blocklog.Error("envelop String unmarshal failed", "error", err)
			return ""
		}
		return tmp.String()
	}
	blocklog.Error("envelop String not find evidence kind", "kind", env.Kind)
	return ""
}

// Hash returns the simple merkle root hash of the EvidenceList.
func (evl EvidenceEnvelopeList) Hash() []byte {
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

func (evl EvidenceEnvelopeList) String() string {
	s := ""
	for _, e := range evl {
		s += fmt.Sprintf("%s\t\t", e)
	}
	return s
}

// Has returns true if the evidence is in the EvidenceList.
func (evl EvidenceEnvelopeList) Has(evidence Evidence) bool {
	for _, ev := range evl {
		if v, ok := EvidenceType2Obj[ev.Kind]; ok {
			tmp := v.(Evidence).Copy()
			err := json.Unmarshal(*ev.Data, &tmp)
			if err != nil {
				blocklog.Error("envelop has unmarshal failed", "error", err)
				return false
			}
			if tmp.Equal(evidence) {
				return true
			}
			return false
		}
		blocklog.Error("envelop has not find evidence kind", "kind", ev.Kind)
		return false
	}
	return false
}

type EvidenceData struct {
	Evidence EvidenceEnvelopeList `json:"evidence"`

	// Volatile
	hash []byte
}

// Hash returns the hash of the data.
func (data *EvidenceData) Hash() []byte {
	if data.hash == nil {
		data.hash = data.Evidence.Hash()
	}
	return data.hash
}

// StringIndented returns a string representation of the evidence.
func (data *EvidenceData) StringIndented(indent string) string {
	if data == nil {
		return "nil-Evidence"
	}
	evStrings := make([]string, cmn.MinInt(len(data.Evidence), 21))
	for i, ev := range data.Evidence {
		if i == 20 {
			evStrings[i] = fmt.Sprintf("... (%v total)", len(data.Evidence))
			break
		}
		evStrings[i] = fmt.Sprintf("Evidence:%v", ev)
	}
	return fmt.Sprintf(`Data{
%s  %v
%s}#%v`,
		indent, strings.Join(evStrings, "\n"+indent+"  "),
		indent, data.hash)
	return ""
}

//--------------------------------------------------------------------------------

// BlockID defines the unique ID of a block as its Hash and its PartSetHeader
type BlockID struct {
	Hash        []byte    `json:"hash"`
	//PartsHeader PartSetHeader `json:"parts"`
}

// IsZero returns true if this is the BlockID for a nil-block
func (blockID BlockID) IsZero() bool {
	return len(blockID.Hash) == 0
}

// Equals returns true if the BlockID matches the given BlockID
func (blockID BlockID) Equals(other BlockID) bool {
	return bytes.Equal(blockID.Hash, other.Hash)
}

// Key returns a machine-readable string representation of the BlockID
func (blockID BlockID) Key() string {
	return string(blockID.Hash)
}

// WriteSignBytes writes the canonical bytes of the BlockID to the given writer for digital signing
func (blockID BlockID) WriteSignBytes(w io.Writer, n *int, err *error) {
	if *err != nil {
		return
	}
	if blockID.IsZero() {
		n_, err_ := w.Write([]byte("null"))
		*n = n_
		*err = err_
		return
	} else {
		canonical := CanonicalBlockID(blockID)
		byteBlockID,e := json.Marshal(&canonical)
		if e != nil {
			*err = e
			return
		}
		n_, err_ := w.Write(byteBlockID)
		*n = n_
		*err = err_
		return
	}
}

// String returns a human readable string representation of the BlockID
func (blockID BlockID) String() string {
	return fmt.Sprintf(`%v`, blockID.Hash)
}

//------------------------------------------------------
// evidence pool

// EvidencePool defines the EvidencePool interface used by the ConsensusState.
// UNSTABLE
type EvidencePool interface {
	PendingEvidence() []Evidence
	AddEvidence(Evidence) error
	Update(*Block)
}

// MockMempool is an empty implementation of a Mempool, useful for testing.
// UNSTABLE
type MockEvidencePool struct {
}

func (m MockEvidencePool) PendingEvidence() []Evidence { return nil }
func (m MockEvidencePool) AddEvidence(Evidence) error  { return nil }
func (m MockEvidencePool) Update(*Block)               {}
