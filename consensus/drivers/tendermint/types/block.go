package types

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

const fee = 1e6

var (
	blocklog        = log15.New("module", "tendermint-block")
	r               *rand.Rand
	ConsensusCrypto crypto.Crypto
)

//-----------------------------------------------------------------------------
//BlockID
type BlockID struct {
	types.BlockID
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

// String returns a human readable string representation of the BlockID
func (blockID BlockID) String() string {
	return Fmt(`%v`, blockID.Hash)
}

//-----------------------------------------------------------------------------
//TendermintBlock
type TendermintBlock struct {
	*types.TendermintBlock
}

// MakeBlock returns a new block with an empty header, except what can be computed from itself.
// It populates the same set of fields validated by ValidateBasic
func MakeBlock(height int64, round int64, Txs []*types.Transaction, commit *types.TendermintCommit) *TendermintBlock {
	block := &TendermintBlock{&types.TendermintBlock{
		Header: &types.TendermintBlockHeader{
			Height: height,
			Round:  round,
			Time:   time.Now().UnixNano(),
			NumTxs: int64(len(Txs)),
		},
		Txs:        Txs,
		LastCommit: commit,
		Evidence:   &types.EvidenceData{Evidence: make([]*types.EvidenceEnvelope, 0)},
	},
	}
	block.FillHeader()
	return block
}

// AddEvidence appends the given evidence to the block
func (b *TendermintBlock) AddEvidence(evidence []Evidence) {
	for _, item := range evidence {
		ev := item.Child()
		if ev != nil {
			data, err := proto.Marshal(ev)
			if err != nil {
				blocklog.Error("AddEvidence marshal failed", "error", err)
				panic("AddEvidence marshal failed")
			}
			env := &types.EvidenceEnvelope{
				TypeName: item.TypeName(),
				Data:     data,
			}
			b.Evidence.Evidence = append(b.Evidence.Evidence, env)
		}
	}
}

// ValidateBasic performs basic validation that doesn't involve state data.
// It checks the internal consistency of the block.
func (b *TendermintBlock) ValidateBasic() (int64, error) {
	newTxs := int64(len(b.Txs))

	if b.Header.NumTxs != newTxs {
		return 0, fmt.Errorf("Wrong Block.Header.NumTxs. Expected %v, got %v", newTxs, b.Header.NumTxs)
	}
	lastCommit := Commit{
		TendermintCommit: b.LastCommit,
	}
	if !bytes.Equal(b.Header.LastCommitHash, lastCommit.Hash()) {
		return 0, fmt.Errorf("Wrong Block.Header.LastCommitHash.  Expected %v, got %v", b.Header.LastCommitHash, lastCommit.Hash())
	}
	if b.Header.Height != 1 {
		if err := lastCommit.ValidateBasic(); err != nil {
			return 0, err
		}
	}

	evidence := &EvidenceData{EvidenceData: b.Evidence}
	if !bytes.Equal(b.Header.EvidenceHash, evidence.Hash()) {
		return 0, errors.New(Fmt("Wrong Block.Header.EvidenceHash.  Expected %v, got %v", b.Header.EvidenceHash, evidence.Hash()))
	}
	return newTxs, nil
}

// FillHeader fills in any remaining header fields that are a function of the block data
func (b *TendermintBlock) FillHeader() {
	if b.Header.LastCommitHash == nil {
		lastCommit := &Commit{
			TendermintCommit: b.LastCommit,
		}
		b.Header.LastCommitHash = lastCommit.Hash()
	}
	if b.Header.EvidenceHash == nil {
		evidence := &EvidenceData{EvidenceData: b.Evidence}
		b.Header.EvidenceHash = evidence.Hash()
	}
}

// Hash computes and returns the block hash.
// If the block is incomplete, block hash is nil for safety.
func (b *TendermintBlock) Hash() []byte {
	if b == nil || b.Header == nil || b.LastCommit == nil {
		return nil
	}
	b.FillHeader()
	header := &Header{TendermintBlockHeader: b.Header}
	return header.Hash()
}

// HashesTo is a convenience function that checks if a block hashes to the given argument.
// A nil block never hashes to anything, and nothing hashes to a nil hash.
func (b *TendermintBlock) HashesTo(hash []byte) bool {
	if len(hash) == 0 {
		return false
	}
	if b == nil {
		return false
	}
	return bytes.Equal(b.Hash(), hash)
}

// String returns a string representation of the block
func (b *TendermintBlock) String() string {
	return b.StringIndented("")
}

// StringIndented returns a string representation of the block
func (b *TendermintBlock) StringIndented(indent string) string {
	if b == nil {
		return "nil-Block"
	}
	header := &Header{TendermintBlockHeader: b.Header}
	lastCommit := &Commit{TendermintCommit: b.LastCommit}
	return Fmt(`Block{
%s  %v
%s  %v
%s  %v
%s}#%v`,
		indent, header.StringIndented(indent+"  "),
		//		indent, b.Evidence.StringIndented(indent+"  "),
		indent, lastCommit.StringIndented(indent+"  "),
		indent, b.Hash())
}

// StringShort returns a shortened string representation of the block
func (b *TendermintBlock) StringShort() string {
	if b == nil {
		return "nil-Block"
	} else {
		return Fmt("Block#%v", b.Hash())
	}
}

//-----------------------------------------------------------------------------
// Header defines the structure of a Tendermint block header
// TODO: limit header size
// NOTE: changes to the Header should be duplicated in the abci Header
type Header struct {
	*types.TendermintBlockHeader
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
	return Fmt(`Header{
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
		indent, time.Unix(0, h.Time),
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

//-----------------------------------------------------------------------------
//Commit
type Commit struct {
	*types.TendermintCommit

	firstPrecommit *types.Vote
	hash           []byte
	bitArray       *BitArray
}

// FirstPrecommit returns the first non-nil precommit in the commit
func (commit *Commit) FirstPrecommit() *types.Vote {
	if len(commit.Precommits) == 0 {
		return nil
	}
	if commit.firstPrecommit != nil {
		return commit.firstPrecommit
	}
	for _, precommit := range commit.Precommits {
		if precommit != nil && len(precommit.Signature) > 0 {
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
	return int(commit.FirstPrecommit().Round)
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
func (commit *Commit) BitArray() *BitArray {
	if commit.bitArray == nil {
		commit.bitArray = NewBitArray(len(commit.Precommits))
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
	return &Vote{Vote: commit.Precommits[index]}
}

// IsCommit returns true if there is at least one vote
func (commit *Commit) IsCommit() bool {
	return len(commit.Precommits) != 0
}

// ValidateBasic performs basic validation that doesn't involve state data.
func (commit *Commit) ValidateBasic() error {
	blockID := &BlockID{BlockID: *commit.BlockID}
	if blockID.IsZero() {
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
		if byte(precommit.Type) != VoteTypePrecommit {
			return fmt.Errorf("Invalid commit vote. Expected precommit, got %v",
				precommit.Type)
		}
		// Ensure that all heights are the same
		if precommit.Height != height {
			return fmt.Errorf("Invalid commit precommit height. Expected %v, got %v",
				height, precommit.Height)
		}
		// Ensure that all rounds are the same
		if int(precommit.Round) != round {
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
		for i, item := range commit.Precommits {
			precommit := Vote{Vote: item}
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
	return Fmt(`Commit{
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
type EvidenceEnvelope struct {
	*types.EvidenceEnvelope
}

// EvidenceData contains any evidence of malicious wrong-doing by validators
type EvidenceEnvelopeList []EvidenceEnvelope

func (env EvidenceEnvelope) Hash() []byte {
	penv := env.EvidenceEnvelope
	evidence := EvidenceEnvelope2Evidence(penv)
	if evidence != nil {
		return evidence.Hash()
	}
	return nil
}

func (env EvidenceEnvelope) String() string {
	penv := env.EvidenceEnvelope
	evidence := EvidenceEnvelope2Evidence(penv)
	if evidence != nil {
		return evidence.String()
	}
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
		s += Fmt("%s\t\t", e)
	}
	return s
}

// Has returns true if the evidence is in the EvidenceList.
func (evl EvidenceEnvelopeList) Has(evidence Evidence) bool {
	for _, ev := range evl {
		penv := ev.EvidenceEnvelope
		tmp := EvidenceEnvelope2Evidence(penv)
		if tmp != nil {
			if tmp.Equal(evidence) {
				return true
			}
		}
	}
	return false
}

type EvidenceData struct {
	*types.EvidenceData
	hash []byte
}

// Hash returns the hash of the data.
func (data *EvidenceData) Hash() []byte {
	if data.hash == nil {
		if data.EvidenceData == nil {
			return nil
		}
		var evidence EvidenceEnvelopeList
		for _, item := range data.Evidence {
			elem := EvidenceEnvelope{
				EvidenceEnvelope: item,
			}
			evidence = append(evidence, elem)
		}
		data.hash = evidence.Hash()
	}
	return data.hash
}

// StringIndented returns a string representation of the evidence.
func (data *EvidenceData) StringIndented(indent string) string {
	if data == nil {
		return "nil-Evidence"
	}
	evStrings := make([]string, MinInt(len(data.Evidence), 21))
	for i, ev := range data.Evidence {
		if i == 20 {
			evStrings[i] = Fmt("... (%v total)", len(data.Evidence))
			break
		}
		evStrings[i] = Fmt("Evidence:%v", ev)
	}
	return Fmt(`Data{
%s  %v
%s}#%v`,
		indent, strings.Join(evStrings, "\n"+indent+"  "),
		indent, data.hash)
	return ""
}

//---------------------------------BlockStore---------------------------------------------
type BlockStore struct {
	client *drivers.BaseClient
	pubkey string
}

func NewBlockStore(client *drivers.BaseClient, pubkey string) *BlockStore {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &BlockStore{
		client: client,
		pubkey: pubkey,
	}
}

func (bs *BlockStore) LoadSeenCommit(height int64) *types.TendermintCommit {
	oldBlock, err := bs.client.RequestBlock(height)
	if err != nil {
		blocklog.Error("LoadSeenCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(oldBlock)
	if err != nil {
		panic(Fmt("LoadSeenCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		blocklog.Error("LoadSeenCommit get nil block info")
		return nil
	}
	return blockInfo.GetSeenCommit()
}

func (bs *BlockStore) LoadBlockCommit(height int64) *types.TendermintCommit {
	oldBlock, err := bs.client.RequestBlock(height)
	if err != nil {
		blocklog.Error("LoadBlockCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(oldBlock)
	if err != nil {
		panic(Fmt("LoadBlockCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		blocklog.Error("LoadBlockCommit get nil block info")
		return nil
	}
	return blockInfo.GetLastCommit()
}

func (bs *BlockStore) LoadProposal(height int64) *types.Proposal {
	block, err := bs.client.RequestBlock(height)
	if err != nil {
		blocklog.Error("LoadProposal by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(block)
	if err != nil {
		panic(Fmt("LoadProposal GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		blocklog.Error("LoadProposal get nil block info")
		return nil
	}
	proposal := blockInfo.GetProposal()
	return proposal
}

func (bs *BlockStore) LoadProposalBlock(height int64) *types.TendermintBlock {
	block, err := bs.client.RequestBlock(height)
	if err != nil {
		blocklog.Error("LoadProposal by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(block)
	if err != nil {
		panic(Fmt("LoadProposal GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		blocklog.Error("LoadProposal get nil block info")
		return nil
	}

	proposalBlock := blockInfo.GetBlock()
	if proposalBlock != nil {
		proposalBlock.Txs = append(proposalBlock.Txs, block.Txs[1:]...)
		txHash := merkle.CalcMerkleRoot(proposalBlock.Txs)
		blocklog.Info("LoadProposalBlock txs hash", "height", proposalBlock.Header.Height, "tx-hash", Fmt("%X", txHash))
	}
	return proposalBlock
}

func (bs *BlockStore) Height() int64 {
	return bs.client.GetCurrentHeight()
}

func (bs *BlockStore) GetPubkey() string {
	return bs.pubkey
}

func GetBlockInfo(block *types.Block) (*types.TendermintBlockInfo, error) {
	if len(block.Txs) == 0 || block.Height == 0 {
		return nil, nil
	}
	baseTx := block.Txs[0]
	//判断交易类型和执行情况
	var blockInfo types.TendermintBlockInfo
	nGet := &types.NormPut{}
	action := &types.NormAction{}
	err := types.Decode(baseTx.GetPayload(), action)
	if err != nil {
		blocklog.Error("GetBlockInfo decode payload failed", "error", err)
		return nil, errors.New(Fmt("GetBlockInfo decode payload failed:%v", err))
	}
	if nGet = action.GetNput(); nGet == nil {
		blocklog.Error("GetBlockInfo get nput failed")
		return nil, errors.New("GetBlockInfo get nput failed")
	}
	infobytes := nGet.GetValue()
	if infobytes == nil {
		blocklog.Error("GetBlockInfo get blockinfo value failed")
		return nil, errors.New("GetBlockInfo get blockinfo value failed")
	}
	err = types.Decode(infobytes, &blockInfo)
	if err != nil {
		blocklog.Error("GetBlockInfo decode blockinfo failed", "error", err)
		return nil, errors.New(Fmt("GetBlockInfo decode blockinfo failed:%v", err))
	}
	return &blockInfo, nil
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func LoadValidators(des []*Validator, source []*types.Validator) {
	for i, item := range source {
		if item.GetAddress() == nil || len(item.GetAddress()) == 0 {
			blocklog.Warn("LoadValidators get address is nil or empty")
			continue
		} else if item.GetPubKey() == nil || len(item.GetPubKey()) == 0 {
			blocklog.Warn("LoadValidators get pubkey is nil or empty")
			continue
		}
		des[i] = &Validator{}
		des[i].Address = item.GetAddress()
		pub := item.GetPubKey()
		if pub == nil {
			blocklog.Error("LoadValidators get validator pubkey is nil", "item", i)
		} else {
			des[i].PubKey = pub
		}
		des[i].VotingPower = item.VotingPower
		des[i].Accum = item.Accum
	}
}

func LoadProposer(source *types.Validator) (*Validator, error) {
	if source.GetAddress() == nil || len(source.GetAddress()) == 0 {
		blocklog.Warn("LoadProposer get address is nil or empty")
		return nil, errors.New("LoadProposer get address is nil or empty")
	} else if source.GetPubKey() == nil || len(source.GetPubKey()) == 0 {
		blocklog.Warn("LoadProposer get pubkey is nil or empty")
		return nil, errors.New("LoadProposer get pubkey is nil or empty")
	}

	des := &Validator{}
	des.Address = source.GetAddress()
	pub := source.GetPubKey()
	if pub == nil {
		blocklog.Error("LoadProposer get pubkey is nil")
	} else {
		des.PubKey = pub
	}
	des.VotingPower = source.VotingPower
	des.Accum = source.Accum
	return des, nil
}

func CreateBlockInfoTx(pubkey string, lastCommit *types.TendermintCommit, seenCommit *types.TendermintCommit, state *types.State, proposal *types.Proposal, block *types.TendermintBlock) *types.Transaction {
	blockNoTxs := *block
	blockNoTxs.Txs = make([]*types.Transaction, 0)
	blockInfo := &types.TendermintBlockInfo{
		SeenCommit: seenCommit,
		LastCommit: lastCommit,
		State:      state,
		Proposal:   proposal,
		Block:      &blockNoTxs,
	}
	blocklog.Debug("CreateBlockInfoTx", "validators", blockInfo.State.Validators.Validators, "block", block, "block-notxs", blockNoTxs)

	nput := &types.NormAction_Nput{&types.NormPut{Key: "BlockInfo", Value: types.Encode(blockInfo)}}
	action := &types.NormAction{Value: nput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(action), Fee: fee}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(pubkey))

	return tx
}
