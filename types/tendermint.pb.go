// Code generated by protoc-gen-go. DO NOT EDIT.
// source: tendermint.proto

package types

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type BlockID struct {
	Hash []byte `protobuf:"bytes,1,opt,name=Hash,proto3" json:"Hash,omitempty"`
}

func (m *BlockID) Reset()                    { *m = BlockID{} }
func (m *BlockID) String() string            { return proto.CompactTextString(m) }
func (*BlockID) ProtoMessage()               {}
func (*BlockID) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{0} }

func (m *BlockID) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

type TendermintBitArray struct {
	Bits  int32    `protobuf:"varint,1,opt,name=Bits" json:"Bits,omitempty"`
	Elems []uint64 `protobuf:"varint,2,rep,packed,name=Elems" json:"Elems,omitempty"`
}

func (m *TendermintBitArray) Reset()                    { *m = TendermintBitArray{} }
func (m *TendermintBitArray) String() string            { return proto.CompactTextString(m) }
func (*TendermintBitArray) ProtoMessage()               {}
func (*TendermintBitArray) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{1} }

func (m *TendermintBitArray) GetBits() int32 {
	if m != nil {
		return m.Bits
	}
	return 0
}

func (m *TendermintBitArray) GetElems() []uint64 {
	if m != nil {
		return m.Elems
	}
	return nil
}

type Vote struct {
	ValidatorAddress []byte   `protobuf:"bytes,1,opt,name=ValidatorAddress,proto3" json:"ValidatorAddress,omitempty"`
	ValidatorIndex   int32    `protobuf:"varint,2,opt,name=ValidatorIndex" json:"ValidatorIndex,omitempty"`
	Height           int64    `protobuf:"varint,3,opt,name=Height" json:"Height,omitempty"`
	Round            int32    `protobuf:"varint,4,opt,name=Round" json:"Round,omitempty"`
	Timestamp        int64    `protobuf:"varint,5,opt,name=Timestamp" json:"Timestamp,omitempty"`
	Type             uint32   `protobuf:"varint,6,opt,name=Type" json:"Type,omitempty"`
	BlockID          *BlockID `protobuf:"bytes,7,opt,name=BlockID" json:"BlockID,omitempty"`
	Signature        []byte   `protobuf:"bytes,8,opt,name=Signature,proto3" json:"Signature,omitempty"`
}

func (m *Vote) Reset()                    { *m = Vote{} }
func (m *Vote) String() string            { return proto.CompactTextString(m) }
func (*Vote) ProtoMessage()               {}
func (*Vote) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{2} }

func (m *Vote) GetValidatorAddress() []byte {
	if m != nil {
		return m.ValidatorAddress
	}
	return nil
}

func (m *Vote) GetValidatorIndex() int32 {
	if m != nil {
		return m.ValidatorIndex
	}
	return 0
}

func (m *Vote) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *Vote) GetRound() int32 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *Vote) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Vote) GetType() uint32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *Vote) GetBlockID() *BlockID {
	if m != nil {
		return m.BlockID
	}
	return nil
}

func (m *Vote) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

type TendermintCommit struct {
	BlockID    *BlockID `protobuf:"bytes,1,opt,name=BlockID" json:"BlockID,omitempty"`
	Precommits []*Vote  `protobuf:"bytes,2,rep,name=Precommits" json:"Precommits,omitempty"`
}

func (m *TendermintCommit) Reset()                    { *m = TendermintCommit{} }
func (m *TendermintCommit) String() string            { return proto.CompactTextString(m) }
func (*TendermintCommit) ProtoMessage()               {}
func (*TendermintCommit) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{3} }

func (m *TendermintCommit) GetBlockID() *BlockID {
	if m != nil {
		return m.BlockID
	}
	return nil
}

func (m *TendermintCommit) GetPrecommits() []*Vote {
	if m != nil {
		return m.Precommits
	}
	return nil
}

type TendermintBlockInfo struct {
	SeenCommit *TendermintCommit `protobuf:"bytes,1,opt,name=SeenCommit" json:"SeenCommit,omitempty"`
	LastCommit *TendermintCommit `protobuf:"bytes,2,opt,name=LastCommit" json:"LastCommit,omitempty"`
	State      *State            `protobuf:"bytes,3,opt,name=State" json:"State,omitempty"`
	Proposal   *Proposal         `protobuf:"bytes,4,opt,name=Proposal" json:"Proposal,omitempty"`
}

func (m *TendermintBlockInfo) Reset()                    { *m = TendermintBlockInfo{} }
func (m *TendermintBlockInfo) String() string            { return proto.CompactTextString(m) }
func (*TendermintBlockInfo) ProtoMessage()               {}
func (*TendermintBlockInfo) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{4} }

func (m *TendermintBlockInfo) GetSeenCommit() *TendermintCommit {
	if m != nil {
		return m.SeenCommit
	}
	return nil
}

func (m *TendermintBlockInfo) GetLastCommit() *TendermintCommit {
	if m != nil {
		return m.LastCommit
	}
	return nil
}

func (m *TendermintBlockInfo) GetState() *State {
	if m != nil {
		return m.State
	}
	return nil
}

func (m *TendermintBlockInfo) GetProposal() *Proposal {
	if m != nil {
		return m.Proposal
	}
	return nil
}

type BlockSize struct {
	MaxBytes int32 `protobuf:"varint,1,opt,name=MaxBytes" json:"MaxBytes,omitempty"`
	MaxTxs   int32 `protobuf:"varint,2,opt,name=MaxTxs" json:"MaxTxs,omitempty"`
	MaxGas   int64 `protobuf:"varint,3,opt,name=MaxGas" json:"MaxGas,omitempty"`
}

func (m *BlockSize) Reset()                    { *m = BlockSize{} }
func (m *BlockSize) String() string            { return proto.CompactTextString(m) }
func (*BlockSize) ProtoMessage()               {}
func (*BlockSize) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{5} }

func (m *BlockSize) GetMaxBytes() int32 {
	if m != nil {
		return m.MaxBytes
	}
	return 0
}

func (m *BlockSize) GetMaxTxs() int32 {
	if m != nil {
		return m.MaxTxs
	}
	return 0
}

func (m *BlockSize) GetMaxGas() int64 {
	if m != nil {
		return m.MaxGas
	}
	return 0
}

type TxSize struct {
	MaxBytes int32 `protobuf:"varint,1,opt,name=MaxBytes" json:"MaxBytes,omitempty"`
	MaxGas   int64 `protobuf:"varint,2,opt,name=MaxGas" json:"MaxGas,omitempty"`
}

func (m *TxSize) Reset()                    { *m = TxSize{} }
func (m *TxSize) String() string            { return proto.CompactTextString(m) }
func (*TxSize) ProtoMessage()               {}
func (*TxSize) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{6} }

func (m *TxSize) GetMaxBytes() int32 {
	if m != nil {
		return m.MaxBytes
	}
	return 0
}

func (m *TxSize) GetMaxGas() int64 {
	if m != nil {
		return m.MaxGas
	}
	return 0
}

type BlockGossip struct {
	BlockPartSizeBytes int32 `protobuf:"varint,1,opt,name=BlockPartSizeBytes" json:"BlockPartSizeBytes,omitempty"`
}

func (m *BlockGossip) Reset()                    { *m = BlockGossip{} }
func (m *BlockGossip) String() string            { return proto.CompactTextString(m) }
func (*BlockGossip) ProtoMessage()               {}
func (*BlockGossip) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{7} }

func (m *BlockGossip) GetBlockPartSizeBytes() int32 {
	if m != nil {
		return m.BlockPartSizeBytes
	}
	return 0
}

type EvidenceParams struct {
	MaxAge int64 `protobuf:"varint,1,opt,name=MaxAge" json:"MaxAge,omitempty"`
}

func (m *EvidenceParams) Reset()                    { *m = EvidenceParams{} }
func (m *EvidenceParams) String() string            { return proto.CompactTextString(m) }
func (*EvidenceParams) ProtoMessage()               {}
func (*EvidenceParams) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{8} }

func (m *EvidenceParams) GetMaxAge() int64 {
	if m != nil {
		return m.MaxAge
	}
	return 0
}

type ConsensusParams struct {
	BlockSize      *BlockSize      `protobuf:"bytes,1,opt,name=BlockSize" json:"BlockSize,omitempty"`
	TxSize         *TxSize         `protobuf:"bytes,2,opt,name=TxSize" json:"TxSize,omitempty"`
	BlockGossip    *BlockGossip    `protobuf:"bytes,3,opt,name=BlockGossip" json:"BlockGossip,omitempty"`
	EvidenceParams *EvidenceParams `protobuf:"bytes,4,opt,name=EvidenceParams" json:"EvidenceParams,omitempty"`
}

func (m *ConsensusParams) Reset()                    { *m = ConsensusParams{} }
func (m *ConsensusParams) String() string            { return proto.CompactTextString(m) }
func (*ConsensusParams) ProtoMessage()               {}
func (*ConsensusParams) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{9} }

func (m *ConsensusParams) GetBlockSize() *BlockSize {
	if m != nil {
		return m.BlockSize
	}
	return nil
}

func (m *ConsensusParams) GetTxSize() *TxSize {
	if m != nil {
		return m.TxSize
	}
	return nil
}

func (m *ConsensusParams) GetBlockGossip() *BlockGossip {
	if m != nil {
		return m.BlockGossip
	}
	return nil
}

func (m *ConsensusParams) GetEvidenceParams() *EvidenceParams {
	if m != nil {
		return m.EvidenceParams
	}
	return nil
}

type Validator struct {
	Address     []byte `protobuf:"bytes,1,opt,name=Address,proto3" json:"Address,omitempty"`
	PubKey      []byte `protobuf:"bytes,2,opt,name=PubKey,proto3" json:"PubKey,omitempty"`
	VotingPower int64  `protobuf:"varint,3,opt,name=VotingPower" json:"VotingPower,omitempty"`
	Accum       int64  `protobuf:"varint,4,opt,name=Accum" json:"Accum,omitempty"`
}

func (m *Validator) Reset()                    { *m = Validator{} }
func (m *Validator) String() string            { return proto.CompactTextString(m) }
func (*Validator) ProtoMessage()               {}
func (*Validator) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{10} }

func (m *Validator) GetAddress() []byte {
	if m != nil {
		return m.Address
	}
	return nil
}

func (m *Validator) GetPubKey() []byte {
	if m != nil {
		return m.PubKey
	}
	return nil
}

func (m *Validator) GetVotingPower() int64 {
	if m != nil {
		return m.VotingPower
	}
	return 0
}

func (m *Validator) GetAccum() int64 {
	if m != nil {
		return m.Accum
	}
	return 0
}

type ValidatorSet struct {
	Validators []*Validator `protobuf:"bytes,1,rep,name=Validators" json:"Validators,omitempty"`
	Proposer   *Validator   `protobuf:"bytes,2,opt,name=Proposer" json:"Proposer,omitempty"`
}

func (m *ValidatorSet) Reset()                    { *m = ValidatorSet{} }
func (m *ValidatorSet) String() string            { return proto.CompactTextString(m) }
func (*ValidatorSet) ProtoMessage()               {}
func (*ValidatorSet) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{11} }

func (m *ValidatorSet) GetValidators() []*Validator {
	if m != nil {
		return m.Validators
	}
	return nil
}

func (m *ValidatorSet) GetProposer() *Validator {
	if m != nil {
		return m.Proposer
	}
	return nil
}

type State struct {
	ChainID                          string           `protobuf:"bytes,1,opt,name=ChainID" json:"ChainID,omitempty"`
	LastBlockHeight                  int64            `protobuf:"varint,2,opt,name=LastBlockHeight" json:"LastBlockHeight,omitempty"`
	LastBlockTotalTx                 int64            `protobuf:"varint,3,opt,name=LastBlockTotalTx" json:"LastBlockTotalTx,omitempty"`
	LastBlockID                      *BlockID         `protobuf:"bytes,4,opt,name=LastBlockID" json:"LastBlockID,omitempty"`
	LastBlockTime                    int64            `protobuf:"varint,5,opt,name=LastBlockTime" json:"LastBlockTime,omitempty"`
	Validators                       *ValidatorSet    `protobuf:"bytes,6,opt,name=Validators" json:"Validators,omitempty"`
	LastValidators                   *ValidatorSet    `protobuf:"bytes,7,opt,name=LastValidators" json:"LastValidators,omitempty"`
	LastHeightValidatorsChanged      int64            `protobuf:"varint,8,opt,name=LastHeightValidatorsChanged" json:"LastHeightValidatorsChanged,omitempty"`
	ConsensusParams                  *ConsensusParams `protobuf:"bytes,9,opt,name=ConsensusParams" json:"ConsensusParams,omitempty"`
	LastHeightConsensusParamsChanged int64            `protobuf:"varint,10,opt,name=LastHeightConsensusParamsChanged" json:"LastHeightConsensusParamsChanged,omitempty"`
	LastResultsHash                  []byte           `protobuf:"bytes,11,opt,name=LastResultsHash,proto3" json:"LastResultsHash,omitempty"`
	AppHash                          []byte           `protobuf:"bytes,12,opt,name=AppHash,proto3" json:"AppHash,omitempty"`
}

func (m *State) Reset()                    { *m = State{} }
func (m *State) String() string            { return proto.CompactTextString(m) }
func (*State) ProtoMessage()               {}
func (*State) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{12} }

func (m *State) GetChainID() string {
	if m != nil {
		return m.ChainID
	}
	return ""
}

func (m *State) GetLastBlockHeight() int64 {
	if m != nil {
		return m.LastBlockHeight
	}
	return 0
}

func (m *State) GetLastBlockTotalTx() int64 {
	if m != nil {
		return m.LastBlockTotalTx
	}
	return 0
}

func (m *State) GetLastBlockID() *BlockID {
	if m != nil {
		return m.LastBlockID
	}
	return nil
}

func (m *State) GetLastBlockTime() int64 {
	if m != nil {
		return m.LastBlockTime
	}
	return 0
}

func (m *State) GetValidators() *ValidatorSet {
	if m != nil {
		return m.Validators
	}
	return nil
}

func (m *State) GetLastValidators() *ValidatorSet {
	if m != nil {
		return m.LastValidators
	}
	return nil
}

func (m *State) GetLastHeightValidatorsChanged() int64 {
	if m != nil {
		return m.LastHeightValidatorsChanged
	}
	return 0
}

func (m *State) GetConsensusParams() *ConsensusParams {
	if m != nil {
		return m.ConsensusParams
	}
	return nil
}

func (m *State) GetLastHeightConsensusParamsChanged() int64 {
	if m != nil {
		return m.LastHeightConsensusParamsChanged
	}
	return 0
}

func (m *State) GetLastResultsHash() []byte {
	if m != nil {
		return m.LastResultsHash
	}
	return nil
}

func (m *State) GetAppHash() []byte {
	if m != nil {
		return m.AppHash
	}
	return nil
}

type DuplicateVoteEvidence struct {
	PubKey string `protobuf:"bytes,1,opt,name=pubKey" json:"pubKey,omitempty"`
	VoteA  *Vote  `protobuf:"bytes,2,opt,name=voteA" json:"voteA,omitempty"`
	VoteB  *Vote  `protobuf:"bytes,3,opt,name=voteB" json:"voteB,omitempty"`
}

func (m *DuplicateVoteEvidence) Reset()                    { *m = DuplicateVoteEvidence{} }
func (m *DuplicateVoteEvidence) String() string            { return proto.CompactTextString(m) }
func (*DuplicateVoteEvidence) ProtoMessage()               {}
func (*DuplicateVoteEvidence) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{13} }

func (m *DuplicateVoteEvidence) GetPubKey() string {
	if m != nil {
		return m.PubKey
	}
	return ""
}

func (m *DuplicateVoteEvidence) GetVoteA() *Vote {
	if m != nil {
		return m.VoteA
	}
	return nil
}

func (m *DuplicateVoteEvidence) GetVoteB() *Vote {
	if m != nil {
		return m.VoteB
	}
	return nil
}

type EvidenceEnvelope struct {
	TypeName string `protobuf:"bytes,1,opt,name=typeName" json:"typeName,omitempty"`
	Data     []byte `protobuf:"bytes,2,opt,name=data,proto3" json:"data,omitempty"`
}

func (m *EvidenceEnvelope) Reset()                    { *m = EvidenceEnvelope{} }
func (m *EvidenceEnvelope) String() string            { return proto.CompactTextString(m) }
func (*EvidenceEnvelope) ProtoMessage()               {}
func (*EvidenceEnvelope) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{14} }

func (m *EvidenceEnvelope) GetTypeName() string {
	if m != nil {
		return m.TypeName
	}
	return ""
}

func (m *EvidenceEnvelope) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type EvidenceData struct {
	Evidence []*EvidenceEnvelope `protobuf:"bytes,1,rep,name=evidence" json:"evidence,omitempty"`
}

func (m *EvidenceData) Reset()                    { *m = EvidenceData{} }
func (m *EvidenceData) String() string            { return proto.CompactTextString(m) }
func (*EvidenceData) ProtoMessage()               {}
func (*EvidenceData) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{15} }

func (m *EvidenceData) GetEvidence() []*EvidenceEnvelope {
	if m != nil {
		return m.Evidence
	}
	return nil
}

type TendermintBlockHeader struct {
	ChainID         string   `protobuf:"bytes,1,opt,name=chainID" json:"chainID,omitempty"`
	Height          int64    `protobuf:"varint,2,opt,name=height" json:"height,omitempty"`
	Time            int64    `protobuf:"varint,3,opt,name=time" json:"time,omitempty"`
	NumTxs          int64    `protobuf:"varint,4,opt,name=numTxs" json:"numTxs,omitempty"`
	LastBlockID     *BlockID `protobuf:"bytes,5,opt,name=lastBlockID" json:"lastBlockID,omitempty"`
	TotalTxs        int64    `protobuf:"varint,6,opt,name=totalTxs" json:"totalTxs,omitempty"`
	LastCommitHash  []byte   `protobuf:"bytes,7,opt,name=lastCommitHash,proto3" json:"lastCommitHash,omitempty"`
	ValidatorsHash  []byte   `protobuf:"bytes,8,opt,name=validatorsHash,proto3" json:"validatorsHash,omitempty"`
	ConsensusHash   []byte   `protobuf:"bytes,9,opt,name=consensusHash,proto3" json:"consensusHash,omitempty"`
	AppHash         []byte   `protobuf:"bytes,10,opt,name=appHash,proto3" json:"appHash,omitempty"`
	LastResultsHash []byte   `protobuf:"bytes,11,opt,name=lastResultsHash,proto3" json:"lastResultsHash,omitempty"`
	EvidenceHash    []byte   `protobuf:"bytes,12,opt,name=evidenceHash,proto3" json:"evidenceHash,omitempty"`
}

func (m *TendermintBlockHeader) Reset()                    { *m = TendermintBlockHeader{} }
func (m *TendermintBlockHeader) String() string            { return proto.CompactTextString(m) }
func (*TendermintBlockHeader) ProtoMessage()               {}
func (*TendermintBlockHeader) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{16} }

func (m *TendermintBlockHeader) GetChainID() string {
	if m != nil {
		return m.ChainID
	}
	return ""
}

func (m *TendermintBlockHeader) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *TendermintBlockHeader) GetTime() int64 {
	if m != nil {
		return m.Time
	}
	return 0
}

func (m *TendermintBlockHeader) GetNumTxs() int64 {
	if m != nil {
		return m.NumTxs
	}
	return 0
}

func (m *TendermintBlockHeader) GetLastBlockID() *BlockID {
	if m != nil {
		return m.LastBlockID
	}
	return nil
}

func (m *TendermintBlockHeader) GetTotalTxs() int64 {
	if m != nil {
		return m.TotalTxs
	}
	return 0
}

func (m *TendermintBlockHeader) GetLastCommitHash() []byte {
	if m != nil {
		return m.LastCommitHash
	}
	return nil
}

func (m *TendermintBlockHeader) GetValidatorsHash() []byte {
	if m != nil {
		return m.ValidatorsHash
	}
	return nil
}

func (m *TendermintBlockHeader) GetConsensusHash() []byte {
	if m != nil {
		return m.ConsensusHash
	}
	return nil
}

func (m *TendermintBlockHeader) GetAppHash() []byte {
	if m != nil {
		return m.AppHash
	}
	return nil
}

func (m *TendermintBlockHeader) GetLastResultsHash() []byte {
	if m != nil {
		return m.LastResultsHash
	}
	return nil
}

func (m *TendermintBlockHeader) GetEvidenceHash() []byte {
	if m != nil {
		return m.EvidenceHash
	}
	return nil
}

type TendermintBlock struct {
	Header     *TendermintBlockHeader `protobuf:"bytes,1,opt,name=header" json:"header,omitempty"`
	Txs        []*Transaction  `protobuf:"bytes,2,rep,name=txs" json:"txs,omitempty"`
	Evidence   *EvidenceData          `protobuf:"bytes,3,opt,name=evidence" json:"evidence,omitempty"`
	LastCommit *TendermintCommit      `protobuf:"bytes,4,opt,name=lastCommit" json:"lastCommit,omitempty"`
}

func (m *TendermintBlock) Reset()                    { *m = TendermintBlock{} }
func (m *TendermintBlock) String() string            { return proto.CompactTextString(m) }
func (*TendermintBlock) ProtoMessage()               {}
func (*TendermintBlock) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{17} }

func (m *TendermintBlock) GetHeader() *TendermintBlockHeader {
	if m != nil {
		return m.Header
	}
	return nil
}

func (m *TendermintBlock) GetTxs() []*Transaction {
	if m != nil {
		return m.Txs
	}
	return nil
}

func (m *TendermintBlock) GetEvidence() *EvidenceData {
	if m != nil {
		return m.Evidence
	}
	return nil
}

func (m *TendermintBlock) GetLastCommit() *TendermintCommit {
	if m != nil {
		return m.LastCommit
	}
	return nil
}

type Proposal struct {
	Height     int64            `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
	Round      int32            `protobuf:"varint,2,opt,name=round" json:"round,omitempty"`
	Timestamp  int64            `protobuf:"varint,3,opt,name=timestamp" json:"timestamp,omitempty"`
	POLRound   int32            `protobuf:"varint,4,opt,name=POLRound" json:"POLRound,omitempty"`
	POLBlockID *BlockID         `protobuf:"bytes,5,opt,name=POLBlockID" json:"POLBlockID,omitempty"`
	Signature  []byte           `protobuf:"bytes,6,opt,name=signature,proto3" json:"signature,omitempty"`
	Block      *TendermintBlock `protobuf:"bytes,7,opt,name=block" json:"block,omitempty"`
}

func (m *Proposal) Reset()                    { *m = Proposal{} }
func (m *Proposal) String() string            { return proto.CompactTextString(m) }
func (*Proposal) ProtoMessage()               {}
func (*Proposal) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{18} }

func (m *Proposal) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *Proposal) GetRound() int32 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *Proposal) GetTimestamp() int64 {
	if m != nil {
		return m.Timestamp
	}
	return 0
}

func (m *Proposal) GetPOLRound() int32 {
	if m != nil {
		return m.POLRound
	}
	return 0
}

func (m *Proposal) GetPOLBlockID() *BlockID {
	if m != nil {
		return m.POLBlockID
	}
	return nil
}

func (m *Proposal) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *Proposal) GetBlock() *TendermintBlock {
	if m != nil {
		return m.Block
	}
	return nil
}

type NewRoundStepMsg struct {
	Height                int64 `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
	Round                 int32 `protobuf:"varint,2,opt,name=round" json:"round,omitempty"`
	Step                  int32 `protobuf:"varint,3,opt,name=step" json:"step,omitempty"`
	SecondsSinceStartTime int32 `protobuf:"varint,4,opt,name=secondsSinceStartTime" json:"secondsSinceStartTime,omitempty"`
	LastCommitRound       int32 `protobuf:"varint,5,opt,name=lastCommitRound" json:"lastCommitRound,omitempty"`
}

func (m *NewRoundStepMsg) Reset()                    { *m = NewRoundStepMsg{} }
func (m *NewRoundStepMsg) String() string            { return proto.CompactTextString(m) }
func (*NewRoundStepMsg) ProtoMessage()               {}
func (*NewRoundStepMsg) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{19} }

func (m *NewRoundStepMsg) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *NewRoundStepMsg) GetRound() int32 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *NewRoundStepMsg) GetStep() int32 {
	if m != nil {
		return m.Step
	}
	return 0
}

func (m *NewRoundStepMsg) GetSecondsSinceStartTime() int32 {
	if m != nil {
		return m.SecondsSinceStartTime
	}
	return 0
}

func (m *NewRoundStepMsg) GetLastCommitRound() int32 {
	if m != nil {
		return m.LastCommitRound
	}
	return 0
}

type CommitStepMsg struct {
	Height int64 `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
}

func (m *CommitStepMsg) Reset()                    { *m = CommitStepMsg{} }
func (m *CommitStepMsg) String() string            { return proto.CompactTextString(m) }
func (*CommitStepMsg) ProtoMessage()               {}
func (*CommitStepMsg) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{20} }

func (m *CommitStepMsg) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

type ProposalPOLMsg struct {
	Height           int64               `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
	ProposalPOLRound int32               `protobuf:"varint,2,opt,name=proposalPOLRound" json:"proposalPOLRound,omitempty"`
	ProposalPOL      *TendermintBitArray `protobuf:"bytes,3,opt,name=proposalPOL" json:"proposalPOL,omitempty"`
}

func (m *ProposalPOLMsg) Reset()                    { *m = ProposalPOLMsg{} }
func (m *ProposalPOLMsg) String() string            { return proto.CompactTextString(m) }
func (*ProposalPOLMsg) ProtoMessage()               {}
func (*ProposalPOLMsg) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{21} }

func (m *ProposalPOLMsg) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *ProposalPOLMsg) GetProposalPOLRound() int32 {
	if m != nil {
		return m.ProposalPOLRound
	}
	return 0
}

func (m *ProposalPOLMsg) GetProposalPOL() *TendermintBitArray {
	if m != nil {
		return m.ProposalPOL
	}
	return nil
}

type HasVoteMsg struct {
	Height int64 `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
	Round  int32 `protobuf:"varint,2,opt,name=round" json:"round,omitempty"`
	Type   int32 `protobuf:"varint,3,opt,name=type" json:"type,omitempty"`
	Index  int32 `protobuf:"varint,4,opt,name=index" json:"index,omitempty"`
}

func (m *HasVoteMsg) Reset()                    { *m = HasVoteMsg{} }
func (m *HasVoteMsg) String() string            { return proto.CompactTextString(m) }
func (*HasVoteMsg) ProtoMessage()               {}
func (*HasVoteMsg) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{22} }

func (m *HasVoteMsg) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *HasVoteMsg) GetRound() int32 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *HasVoteMsg) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *HasVoteMsg) GetIndex() int32 {
	if m != nil {
		return m.Index
	}
	return 0
}

type VoteSetMaj23Msg struct {
	Height  int64    `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
	Round   int32    `protobuf:"varint,2,opt,name=round" json:"round,omitempty"`
	Type    int32    `protobuf:"varint,3,opt,name=type" json:"type,omitempty"`
	BlockID *BlockID `protobuf:"bytes,4,opt,name=blockID" json:"blockID,omitempty"`
}

func (m *VoteSetMaj23Msg) Reset()                    { *m = VoteSetMaj23Msg{} }
func (m *VoteSetMaj23Msg) String() string            { return proto.CompactTextString(m) }
func (*VoteSetMaj23Msg) ProtoMessage()               {}
func (*VoteSetMaj23Msg) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{23} }

func (m *VoteSetMaj23Msg) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *VoteSetMaj23Msg) GetRound() int32 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *VoteSetMaj23Msg) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *VoteSetMaj23Msg) GetBlockID() *BlockID {
	if m != nil {
		return m.BlockID
	}
	return nil
}

type VoteSetBitsMsg struct {
	Height  int64               `protobuf:"varint,1,opt,name=height" json:"height,omitempty"`
	Round   int32               `protobuf:"varint,2,opt,name=round" json:"round,omitempty"`
	Type    int32               `protobuf:"varint,3,opt,name=type" json:"type,omitempty"`
	BlockID *BlockID            `protobuf:"bytes,4,opt,name=blockID" json:"blockID,omitempty"`
	Votes   *TendermintBitArray `protobuf:"bytes,5,opt,name=votes" json:"votes,omitempty"`
}

func (m *VoteSetBitsMsg) Reset()                    { *m = VoteSetBitsMsg{} }
func (m *VoteSetBitsMsg) String() string            { return proto.CompactTextString(m) }
func (*VoteSetBitsMsg) ProtoMessage()               {}
func (*VoteSetBitsMsg) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{24} }

func (m *VoteSetBitsMsg) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *VoteSetBitsMsg) GetRound() int32 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *VoteSetBitsMsg) GetType() int32 {
	if m != nil {
		return m.Type
	}
	return 0
}

func (m *VoteSetBitsMsg) GetBlockID() *BlockID {
	if m != nil {
		return m.BlockID
	}
	return nil
}

func (m *VoteSetBitsMsg) GetVotes() *TendermintBitArray {
	if m != nil {
		return m.Votes
	}
	return nil
}

type Heartbeat struct {
	ValidatorAddress []byte `protobuf:"bytes,1,opt,name=validatorAddress,proto3" json:"validatorAddress,omitempty"`
	ValidatorIndex   int32  `protobuf:"varint,2,opt,name=validatorIndex" json:"validatorIndex,omitempty"`
	Height           int64  `protobuf:"varint,3,opt,name=height" json:"height,omitempty"`
	Round            int32  `protobuf:"varint,4,opt,name=round" json:"round,omitempty"`
	Sequence         int32  `protobuf:"varint,5,opt,name=sequence" json:"sequence,omitempty"`
	Signature        []byte `protobuf:"bytes,6,opt,name=signature,proto3" json:"signature,omitempty"`
}

func (m *Heartbeat) Reset()                    { *m = Heartbeat{} }
func (m *Heartbeat) String() string            { return proto.CompactTextString(m) }
func (*Heartbeat) ProtoMessage()               {}
func (*Heartbeat) Descriptor() ([]byte, []int) { return fileDescriptor16, []int{25} }

func (m *Heartbeat) GetValidatorAddress() []byte {
	if m != nil {
		return m.ValidatorAddress
	}
	return nil
}

func (m *Heartbeat) GetValidatorIndex() int32 {
	if m != nil {
		return m.ValidatorIndex
	}
	return 0
}

func (m *Heartbeat) GetHeight() int64 {
	if m != nil {
		return m.Height
	}
	return 0
}

func (m *Heartbeat) GetRound() int32 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *Heartbeat) GetSequence() int32 {
	if m != nil {
		return m.Sequence
	}
	return 0
}

func (m *Heartbeat) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func init() {
	proto.RegisterType((*BlockID)(nil), "types.BlockID")
	proto.RegisterType((*TendermintBitArray)(nil), "types.TendermintBitArray")
	proto.RegisterType((*Vote)(nil), "types.Vote")
	proto.RegisterType((*TendermintCommit)(nil), "types.TendermintCommit")
	proto.RegisterType((*TendermintBlockInfo)(nil), "types.TendermintBlockInfo")
	proto.RegisterType((*BlockSize)(nil), "types.BlockSize")
	proto.RegisterType((*TxSize)(nil), "types.TxSize")
	proto.RegisterType((*BlockGossip)(nil), "types.BlockGossip")
	proto.RegisterType((*EvidenceParams)(nil), "types.EvidenceParams")
	proto.RegisterType((*ConsensusParams)(nil), "types.ConsensusParams")
	proto.RegisterType((*Validator)(nil), "types.Validator")
	proto.RegisterType((*ValidatorSet)(nil), "types.ValidatorSet")
	proto.RegisterType((*State)(nil), "types.State")
	proto.RegisterType((*DuplicateVoteEvidence)(nil), "types.DuplicateVoteEvidence")
	proto.RegisterType((*EvidenceEnvelope)(nil), "types.EvidenceEnvelope")
	proto.RegisterType((*EvidenceData)(nil), "types.EvidenceData")
	proto.RegisterType((*TendermintBlockHeader)(nil), "types.TendermintBlockHeader")
	proto.RegisterType((*TendermintBlock)(nil), "types.TendermintBlock")
	proto.RegisterType((*Proposal)(nil), "types.Proposal")
	proto.RegisterType((*NewRoundStepMsg)(nil), "types.NewRoundStepMsg")
	proto.RegisterType((*CommitStepMsg)(nil), "types.CommitStepMsg")
	proto.RegisterType((*ProposalPOLMsg)(nil), "types.ProposalPOLMsg")
	proto.RegisterType((*HasVoteMsg)(nil), "types.HasVoteMsg")
	proto.RegisterType((*VoteSetMaj23Msg)(nil), "types.VoteSetMaj23Msg")
	proto.RegisterType((*VoteSetBitsMsg)(nil), "types.VoteSetBitsMsg")
	proto.RegisterType((*Heartbeat)(nil), "types.Heartbeat")
}

func init() { proto.RegisterFile("tendermint.proto", fileDescriptor16) }

var fileDescriptor16 = []byte{
	// 1416 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xbc, 0x58, 0xdd, 0x6e, 0x1b, 0xc5,
	0x17, 0xd7, 0xc6, 0x76, 0x12, 0x1f, 0x3b, 0x1f, 0xff, 0x69, 0xd3, 0xfa, 0x5f, 0x8a, 0x64, 0x46,
	0x05, 0xac, 0xb6, 0x4a, 0xab, 0xa4, 0x12, 0x17, 0xa5, 0xa8, 0x71, 0x52, 0x35, 0x85, 0xa4, 0xb5,
	0xc6, 0x56, 0xb9, 0x9e, 0xd8, 0x83, 0xbd, 0x60, 0xef, 0x2e, 0x3b, 0x63, 0xd7, 0x41, 0xe2, 0x86,
	0x37, 0xe0, 0x11, 0x78, 0x00, 0xae, 0x78, 0x08, 0x9e, 0x00, 0xc1, 0x0d, 0xaf, 0xc0, 0x33, 0xa0,
	0x39, 0x33, 0xbb, 0x3b, 0xbb, 0x76, 0x5c, 0x40, 0x88, 0xbb, 0x3d, 0xbf, 0xf9, 0xcd, 0x9c, 0x39,
	0x9f, 0x33, 0xb3, 0xb0, 0xab, 0x44, 0x30, 0x10, 0xf1, 0xc4, 0x0f, 0xd4, 0x7e, 0x14, 0x87, 0x2a,
	0x24, 0x15, 0x75, 0x19, 0x09, 0x79, 0xeb, 0x7f, 0x2a, 0xe6, 0x81, 0xe4, 0x7d, 0xe5, 0x87, 0x81,
	0x19, 0xa1, 0xef, 0xc2, 0x46, 0x7b, 0x1c, 0xf6, 0xbf, 0x7a, 0x71, 0x42, 0x08, 0x94, 0x4f, 0xb9,
	0x1c, 0x35, 0xbc, 0xa6, 0xd7, 0xaa, 0x33, 0xfc, 0xa6, 0x9f, 0x00, 0xe9, 0xa5, 0x8b, 0xb5, 0x7d,
	0x75, 0x14, 0xc7, 0xfc, 0x52, 0x33, 0xdb, 0xbe, 0x92, 0xc8, 0xac, 0x30, 0xfc, 0x26, 0xd7, 0xa1,
	0xf2, 0x6c, 0x2c, 0x26, 0xb2, 0xb1, 0xd6, 0x2c, 0xb5, 0xca, 0xcc, 0x08, 0xf4, 0xbb, 0x35, 0x28,
	0xbf, 0x0e, 0x95, 0x20, 0x77, 0x61, 0xf7, 0x35, 0x1f, 0xfb, 0x03, 0xae, 0xc2, 0xf8, 0x68, 0x30,
	0x88, 0x85, 0x94, 0x56, 0xd1, 0x02, 0x4e, 0x3e, 0x80, 0xed, 0x14, 0x7b, 0x11, 0x0c, 0xc4, 0xbc,
	0xb1, 0x86, 0x8a, 0x0a, 0x28, 0xb9, 0x01, 0xeb, 0xa7, 0xc2, 0x1f, 0x8e, 0x54, 0xa3, 0xd4, 0xf4,
	0x5a, 0x25, 0x66, 0x25, 0xbd, 0x15, 0x16, 0x4e, 0x83, 0x41, 0xa3, 0x8c, 0xd3, 0x8c, 0x40, 0x6e,
	0x43, 0xb5, 0xe7, 0x4f, 0x84, 0x54, 0x7c, 0x12, 0x35, 0x2a, 0x38, 0x21, 0x03, 0xb4, 0x49, 0xbd,
	0xcb, 0x48, 0x34, 0xd6, 0x9b, 0x5e, 0x6b, 0x8b, 0xe1, 0x37, 0x69, 0xa5, 0xbe, 0x69, 0x6c, 0x34,
	0xbd, 0x56, 0xed, 0x60, 0x7b, 0x1f, 0xfd, 0xb8, 0x6f, 0x51, 0x96, 0xba, 0xee, 0x36, 0x54, 0xbb,
	0xfe, 0x30, 0xe0, 0x6a, 0x1a, 0x8b, 0xc6, 0x26, 0x9a, 0x95, 0x01, 0xd4, 0x87, 0xdd, 0xcc, 0x89,
	0xc7, 0xe1, 0x64, 0xe2, 0x2b, 0x77, 0x6d, 0x6f, 0xf5, 0xda, 0xf7, 0x00, 0x3a, 0xb1, 0xe8, 0xe3,
	0x34, 0xe3, 0xdd, 0xda, 0x41, 0xcd, 0x92, 0xb5, 0x6b, 0x99, 0x33, 0x4c, 0x7f, 0xf3, 0xe0, 0x9a,
	0x13, 0x30, 0x5c, 0x22, 0xf8, 0x22, 0x24, 0x1f, 0x01, 0x74, 0x85, 0x08, 0x8c, 0x72, 0xab, 0xf1,
	0xa6, 0x5d, 0xa4, 0xb8, 0x37, 0xe6, 0x50, 0xf5, 0xc4, 0x33, 0x2e, 0xed, 0x08, 0xc6, 0x61, 0xd5,
	0xc4, 0x8c, 0x4a, 0x28, 0x54, 0xba, 0x8a, 0x2b, 0x81, 0xb1, 0xa9, 0x1d, 0xd4, 0xed, 0x1c, 0xc4,
	0x98, 0x19, 0x22, 0xf7, 0x60, 0xb3, 0x13, 0x87, 0x51, 0x28, 0xf9, 0x18, 0x63, 0x55, 0x3b, 0xd8,
	0xb1, 0xb4, 0x04, 0x66, 0x29, 0x81, 0x7e, 0x0e, 0x55, 0xb4, 0xa7, 0xeb, 0x7f, 0x23, 0xc8, 0x2d,
	0xd8, 0x3c, 0xe7, 0xf3, 0xf6, 0xa5, 0x12, 0x49, 0x16, 0xa6, 0xb2, 0x4e, 0x8b, 0x73, 0x3e, 0xef,
	0xcd, 0xa5, 0x4d, 0x1b, 0x2b, 0x59, 0xfc, 0x39, 0x97, 0x49, 0xba, 0x18, 0x89, 0x7e, 0x0c, 0xeb,
	0xbd, 0xf9, 0x5f, 0x5c, 0x55, 0xcf, 0x5e, 0xcb, 0xcd, 0x7e, 0x02, 0x35, 0xdc, 0xd6, 0xf3, 0x50,
	0x4a, 0x3f, 0x22, 0xfb, 0x40, 0x50, 0xec, 0xf0, 0x58, 0xe9, 0x35, 0xdd, 0xc5, 0x96, 0x8c, 0xd0,
	0x16, 0x6c, 0x3f, 0x9b, 0xf9, 0x03, 0x11, 0xf4, 0x45, 0x87, 0xc7, 0x7c, 0x92, 0x28, 0x3a, 0x1a,
	0x0a, 0x9c, 0x65, 0x14, 0x1d, 0x0d, 0x05, 0xfd, 0xdd, 0x83, 0x9d, 0xe3, 0x30, 0x90, 0x22, 0x90,
	0x53, 0x69, 0xb9, 0xfb, 0x8e, 0x4f, 0x6c, 0x54, 0x77, 0xdd, 0x3c, 0xd2, 0x38, 0x73, 0xdc, 0xf6,
	0x7e, 0x62, 0xaa, 0x8d, 0xe4, 0x56, 0x12, 0x49, 0x04, 0x59, 0xe2, 0x87, 0x47, 0x39, 0x9b, 0x6c,
	0x04, 0x89, 0xbb, 0xb0, 0x19, 0x61, 0x39, 0xd3, 0x9f, 0x14, 0x4d, 0xb1, 0x31, 0xdd, 0xb3, 0x13,
	0xf3, 0x83, 0xac, 0x40, 0xa6, 0x53, 0xa8, 0xa6, 0xf5, 0x4d, 0x1a, 0xb0, 0x91, 0xef, 0x12, 0x89,
	0xa8, 0xdd, 0xd3, 0x99, 0x5e, 0x7c, 0x26, 0x2e, 0xd1, 0x84, 0x3a, 0xb3, 0x12, 0x69, 0x42, 0xed,
	0x75, 0xa8, 0xfc, 0x60, 0xd8, 0x09, 0xdf, 0x88, 0xd8, 0x86, 0xd8, 0x85, 0x74, 0x5b, 0x38, 0xea,
	0xf7, 0xa7, 0x13, 0xdc, 0x56, 0x89, 0x19, 0x81, 0x06, 0x50, 0x4f, 0xd5, 0x76, 0x85, 0x22, 0x0f,
	0x01, 0x52, 0x59, 0x2b, 0x2f, 0x39, 0x3e, 0x4d, 0x07, 0x98, 0xc3, 0x21, 0xf7, 0x93, 0x2c, 0x16,
	0xb1, 0x75, 0xeb, 0x22, 0x3f, 0x65, 0xd0, 0x5f, 0xca, 0xb6, 0x30, 0xb4, 0x8d, 0xc7, 0x23, 0xee,
	0x07, 0xb6, 0x05, 0x54, 0x59, 0x22, 0x92, 0x16, 0xec, 0xe8, 0x4a, 0x42, 0xe7, 0xda, 0x0e, 0x67,
	0x92, 0xae, 0x08, 0xeb, 0xb6, 0x9a, 0x42, 0xbd, 0x50, 0xf1, 0x71, 0x6f, 0x6e, 0x4d, 0x5f, 0xc0,
	0xc9, 0x43, 0xa8, 0xa5, 0xd8, 0x8b, 0x13, 0x1b, 0x9c, 0x62, 0xdb, 0x71, 0x29, 0xe4, 0x0e, 0x6c,
	0x65, 0xab, 0xf8, 0x13, 0x61, 0xdb, 0x66, 0x1e, 0x24, 0x87, 0x39, 0x8f, 0xad, 0xe3, 0xb2, 0xd7,
	0x8a, 0x1e, 0xe8, 0x0a, 0x95, 0x73, 0xda, 0x63, 0xd8, 0xd6, 0xab, 0x38, 0x13, 0x37, 0xae, 0x9e,
	0x58, 0xa0, 0x92, 0xa7, 0xf0, 0x8e, 0x46, 0x8c, 0x0f, 0x32, 0xfc, 0x78, 0xc4, 0x83, 0xa1, 0x18,
	0x60, 0x03, 0x2e, 0xb1, 0x55, 0x14, 0xf2, 0x74, 0xa1, 0x96, 0x1a, 0x55, 0xd4, 0x7f, 0xc3, 0xea,
	0x2f, 0x8c, 0xb2, 0x85, 0xd2, 0xfb, 0x14, 0x9a, 0x99, 0x82, 0xc2, 0x60, 0xb2, 0x11, 0xc0, 0x8d,
	0xbc, 0x95, 0x97, 0xc4, 0x9b, 0x09, 0x39, 0x1d, 0x2b, 0x89, 0x87, 0x70, 0x0d, 0x93, 0xbb, 0x08,
	0x63, 0x5d, 0x44, 0x11, 0x32, 0xea, 0xb6, 0x2e, 0x8c, 0x48, 0xa7, 0xb0, 0x77, 0x32, 0x8d, 0xc6,
	0x7e, 0x9f, 0x2b, 0xa1, 0x8f, 0x85, 0xa4, 0xba, 0x74, 0xc1, 0x44, 0xa6, 0x60, 0x4c, 0x96, 0x59,
	0x89, 0xbc, 0x07, 0x95, 0x59, 0xa8, 0xc4, 0x91, 0xcd, 0xd9, 0xdc, 0x91, 0x62, 0x46, 0x12, 0x4a,
	0xdb, 0x76, 0x80, 0x45, 0x4a, 0x9b, 0xb6, 0x61, 0x37, 0xd1, 0xf4, 0x2c, 0x98, 0x89, 0x71, 0x18,
	0x61, 0x1b, 0xd5, 0xc4, 0x97, 0x7c, 0x22, 0xac, 0xce, 0x54, 0xd6, 0xe7, 0xec, 0x80, 0x2b, 0x6e,
	0x8b, 0x17, 0xbf, 0xe9, 0x31, 0xd4, 0x93, 0x35, 0x4e, 0xb8, 0xe2, 0xe4, 0x10, 0x36, 0x85, 0x95,
	0x6d, 0x01, 0xde, 0x2c, 0xb4, 0x90, 0x44, 0x15, 0x4b, 0x89, 0xf4, 0x87, 0x12, 0xec, 0x15, 0x4e,
	0xbe, 0x53, 0xc1, 0x07, 0x02, 0x7b, 0x49, 0x3f, 0x5f, 0x67, 0x56, 0xd4, 0xae, 0x19, 0xb9, 0xe5,
	0x65, 0x25, 0xbd, 0x49, 0xa5, 0xd3, 0xdd, 0x54, 0x12, 0x7e, 0x6b, 0x6e, 0x30, 0x9d, 0xe8, 0x53,
	0xc5, 0xb4, 0x0f, 0x2b, 0xe9, 0xaa, 0x1a, 0x3b, 0x55, 0x55, 0x59, 0x5e, 0x55, 0x0e, 0x05, 0xdd,
	0x63, 0x4a, 0xd2, 0x54, 0x4b, 0x89, 0xa5, 0xb2, 0xbe, 0xfa, 0x8c, 0xd3, 0x33, 0x14, 0xc3, 0xbc,
	0x81, 0x8e, 0x2a, 0xa0, 0x9a, 0x37, 0x4b, 0x93, 0x1a, 0x79, 0xe6, 0xd6, 0x51, 0x40, 0x75, 0x05,
	0xf7, 0x93, 0x9c, 0x43, 0x5a, 0x15, 0x69, 0x79, 0x50, 0x7b, 0x88, 0xdb, 0xac, 0x02, 0x93, 0x55,
	0x56, 0xd4, 0x99, 0x39, 0x5e, 0x9e, 0x99, 0x05, 0x98, 0x50, 0xa8, 0x27, 0xb1, 0x70, 0xd2, 0x33,
	0x87, 0xd1, 0x5f, 0x3d, 0xd8, 0x29, 0xc4, 0x88, 0x3c, 0xd2, 0x31, 0xd0, 0x71, 0xb2, 0xe7, 0xd7,
	0xed, 0x85, 0xcb, 0x85, 0x13, 0x4b, 0x66, 0xb9, 0xe4, 0x0e, 0x94, 0xd4, 0x3c, 0xb9, 0x0d, 0x25,
	0x27, 0x53, 0x2f, 0xbb, 0xdd, 0x32, 0x3d, 0x4c, 0x1e, 0x38, 0x89, 0x54, 0xca, 0xb5, 0x17, 0x37,
	0xdf, 0xb2, 0x24, 0xd2, 0xb7, 0x9d, 0xcc, 0xd1, 0xb6, 0x43, 0x5e, 0x7d, 0xdb, 0xc9, 0xa8, 0xf4,
	0x0f, 0x2f, 0xbb, 0xca, 0x38, 0x69, 0xe5, 0xe5, 0xd2, 0xea, 0x3a, 0x54, 0x62, 0xbc, 0x97, 0x9a,
	0x7b, 0x89, 0x11, 0xf4, 0xdd, 0x51, 0xa5, 0xf7, 0x52, 0x93, 0x71, 0x19, 0xa0, 0x93, 0xa5, 0xf3,
	0xea, 0xcc, 0xbd, 0xce, 0xa6, 0x32, 0xd9, 0x07, 0xe8, 0xbc, 0x3a, 0x5b, 0x9d, 0x79, 0x0e, 0x43,
	0x6b, 0x92, 0xe9, 0x2d, 0x75, 0xdd, 0xdc, 0x52, 0x53, 0x80, 0xdc, 0x87, 0xca, 0x85, 0x26, 0xda,
	0x46, 0x7c, 0x63, 0x79, 0x1c, 0x98, 0x21, 0xd1, 0x9f, 0x3c, 0xd8, 0x79, 0x29, 0xde, 0xe0, 0x46,
	0xba, 0x4a, 0x44, 0xe7, 0x72, 0xf8, 0x37, 0xed, 0x26, 0x50, 0x96, 0x4a, 0x18, 0x93, 0x2b, 0x0c,
	0xbf, 0xc9, 0x23, 0xd8, 0x93, 0xa2, 0x1f, 0x06, 0x03, 0xd9, 0xf5, 0x83, 0xbe, 0xe8, 0x2a, 0x1e,
	0x2b, 0x3c, 0x78, 0x8c, 0xe9, 0xcb, 0x07, 0x93, 0x24, 0xb5, 0x61, 0x41, 0x4d, 0x15, 0xe4, 0x17,
	0x61, 0xfa, 0x21, 0x6c, 0x19, 0xf1, 0x2d, 0x5b, 0xa6, 0xdf, 0x7b, 0xb0, 0x9d, 0xc4, 0xb3, 0xf3,
	0xea, 0x6c, 0x95, 0x75, 0x77, 0x61, 0x37, 0xca, 0x98, 0xcc, 0x31, 0x74, 0x01, 0x27, 0x8f, 0xa1,
	0xe6, 0x60, 0x36, 0x27, 0xff, 0xbf, 0xe8, 0x69, 0xfb, 0xd0, 0x62, 0x2e, 0x9b, 0x0e, 0x00, 0x4e,
	0xb9, 0xd4, 0xcd, 0xf7, 0x1f, 0x39, 0x5b, 0x2b, 0x49, 0x9c, 0xad, 0xbf, 0x35, 0xd3, 0xc7, 0xd7,
	0x95, 0x7d, 0x26, 0xa1, 0x40, 0xbf, 0x85, 0x1d, 0xad, 0xa2, 0x2b, 0xd4, 0x39, 0xff, 0xf2, 0xe0,
	0xf0, 0xdf, 0x51, 0xd5, 0x82, 0x8d, 0x8b, 0x95, 0xd7, 0x8e, 0x64, 0x98, 0xfe, 0xe8, 0xc1, 0xb6,
	0xd5, 0xaf, 0x9f, 0x95, 0xff, 0xb1, 0x7a, 0xf2, 0xc0, 0x9c, 0x78, 0xd2, 0x56, 0xd3, 0x8a, 0xd0,
	0x18, 0x1e, 0xfd, 0xd9, 0x83, 0xea, 0xa9, 0xe0, 0xb1, 0xba, 0x10, 0x1c, 0x73, 0x61, 0x76, 0xc5,
	0x2b, 0x77, 0xb6, 0xe4, 0x95, 0x3b, 0x5b, 0xfa, 0xca, 0x9d, 0x2d, 0xbc, 0x72, 0x47, 0xb9, 0x57,
	0x6e, 0xd1, 0xfc, 0xb2, 0x6b, 0xfe, 0x2d, 0xd8, 0x94, 0xe2, 0xeb, 0x29, 0xb6, 0x3c, 0x53, 0x04,
	0xa9, 0xbc, 0xba, 0xfe, 0x2f, 0xd6, 0xf1, 0x87, 0xc0, 0xe1, 0x9f, 0x01, 0x00, 0x00, 0xff, 0xff,
	0x92, 0xda, 0x50, 0x23, 0x3e, 0x10, 0x00, 0x00,
}
