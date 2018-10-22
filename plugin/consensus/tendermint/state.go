package tendermint

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"sync"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	ttypes "gitlab.33.cn/chain33/chain33/plugin/consensus/tendermint/types"
	tmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	"gitlab.33.cn/chain33/chain33/types"
)

const fee = 1e6

var (
	r *rand.Rand
)

// State is a short description of the latest committed block of the Tendermint consensus.
// It keeps all information necessary to validate new blocks,
// including the last validator set and the consensus params.
// All fields are exposed so the struct can be easily serialized,
// but none of them should be mutated directly.
// Instead, use state.Copy() or state.NextState(...).
// NOTE: not goroutine-safe.
type State struct {
	// Immutable
	ChainID string

	// LastBlockHeight=0 at genesis (ie. block(H=0) does not exist)
	LastBlockHeight  int64
	LastBlockTotalTx int64
	LastBlockID      ttypes.BlockID
	LastBlockTime    int64

	// LastValidators is used to validate block.LastCommit.
	// Validators are persisted to the database separately every time they change,
	// so we can query for historical validator sets.
	// Note that if s.LastBlockHeight causes a valset change,
	// we set s.LastHeightValidatorsChanged = s.LastBlockHeight + 1
	Validators                  *ttypes.ValidatorSet
	LastValidators              *ttypes.ValidatorSet
	LastHeightValidatorsChanged int64

	// Consensus parameters used for validating blocks.
	// Changes returned by EndBlock and updated after Commit.
	ConsensusParams                  ttypes.ConsensusParams
	LastHeightConsensusParamsChanged int64

	// Merkle root of the results from executing prev block
	LastResultsHash []byte

	// The latest AppHash we've received from calling abci.Commit()
	AppHash []byte
}

// Copy makes a copy of the State for mutating.
func (s State) Copy() State {
	return State{
		ChainID: s.ChainID,

		LastBlockHeight:  s.LastBlockHeight,
		LastBlockTotalTx: s.LastBlockTotalTx,
		LastBlockID:      s.LastBlockID,
		LastBlockTime:    s.LastBlockTime,

		Validators:                  s.Validators.Copy(),
		LastValidators:              s.LastValidators.Copy(),
		LastHeightValidatorsChanged: s.LastHeightValidatorsChanged,

		ConsensusParams:                  s.ConsensusParams,
		LastHeightConsensusParamsChanged: s.LastHeightConsensusParamsChanged,

		AppHash: s.AppHash,

		LastResultsHash: s.LastResultsHash,
	}
}

// Equals returns true if the States are identical.
func (s State) Equals(s2 State) bool {
	return bytes.Equal(s.Bytes(), s2.Bytes())
}

// Bytes serializes the State using go-wire.
func (s State) Bytes() []byte {
	sbytes, err := json.Marshal(s)
	if err != nil {
		fmt.Printf("Error reading GenesisDoc: %v", err)
		return nil
	}
	return sbytes
}

// IsEmpty returns true if the State is equal to the empty State.
func (s State) IsEmpty() bool {
	return s.Validators == nil // XXX can't compare to Empty
}

// GetValidators returns the last and current validator sets.
func (s State) GetValidators() (last *ttypes.ValidatorSet, current *ttypes.ValidatorSet) {
	return s.LastValidators, s.Validators
}

//------------------------------------------------------------------------
// Create a block from the latest state

// MakeBlock builds a block with the given txs and commit from the current state.
func (s State) MakeBlock(height int64, round int64, Txs []*types.Transaction, commit *tmtypes.TendermintCommit) *ttypes.TendermintBlock {
	// build base block
	block := ttypes.MakeBlock(height, round, Txs, commit)

	// fill header with state data
	block.Header.ChainID = s.ChainID
	block.Header.TotalTxs = s.LastBlockTotalTx + block.Header.NumTxs
	block.Header.LastBlockID = &s.LastBlockID.BlockID
	block.Header.ValidatorsHash = s.Validators.Hash()
	block.Header.AppHash = s.AppHash
	block.Header.ConsensusHash = s.ConsensusParams.Hash()
	block.Header.LastResultsHash = s.LastResultsHash

	return block
}

//------------------------------------------------------------------------
// Genesis

// MakeGenesisStateFromFile reads and unmarshals state from the given
// file.
//
// Used during replay and in tests.
func MakeGenesisStateFromFile(genDocFile string) (State, error) {
	genDoc, err := MakeGenesisDocFromFile(genDocFile)
	if err != nil {
		return State{}, err
	}
	return MakeGenesisState(genDoc)
}

// MakeGenesisDocFromFile reads and unmarshals genesis doc from the given file.
func MakeGenesisDocFromFile(genDocFile string) (*ttypes.GenesisDoc, error) {
	genDocJSON, err := ioutil.ReadFile(genDocFile)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read GenesisDoc file: %v", err)
	}
	genDoc, err := ttypes.GenesisDocFromJSON(genDocJSON)
	if err != nil {
		return nil, fmt.Errorf("Error reading GenesisDoc: %v", err)
	}
	return genDoc, nil
}

// MakeGenesisState creates state from ttypes.GenesisDoc.
func MakeGenesisState(genDoc *ttypes.GenesisDoc) (State, error) {
	err := genDoc.ValidateAndComplete()
	if err != nil {
		return State{}, fmt.Errorf("Error in genesis file: %v", err)
	}

	// Make validators slice
	validators := make([]*ttypes.Validator, len(genDoc.Validators))
	for i, val := range genDoc.Validators {
		pubKey, err := ttypes.PubKeyFromString(val.PubKey.Data)
		if err != nil {
			return State{}, fmt.Errorf("Error validate[i] in genesis file: %v", i, err)
		}

		// Make validator
		validators[i] = &ttypes.Validator{
			Address:     ttypes.GenAddressByPubKey(pubKey),
			PubKey:      pubKey.Bytes(),
			VotingPower: val.Power,
		}
	}

	return State{

		ChainID: genDoc.ChainID,

		LastBlockHeight: 0,
		LastBlockID:     ttypes.BlockID{},
		LastBlockTime:   genDoc.GenesisTime.UnixNano(),

		Validators:                  ttypes.NewValidatorSet(validators),
		LastValidators:              ttypes.NewValidatorSet(nil),
		LastHeightValidatorsChanged: 1,

		ConsensusParams:                  *genDoc.ConsensusParams,
		LastHeightConsensusParamsChanged: 1,

		AppHash: genDoc.AppHash,
	}, nil
}

//-------------------stateDB------------------------
type CSStateDB struct {
	client *TendermintClient
	state  State
	mtx    sync.Mutex
}

func NewStateDB(client *TendermintClient, state State) *CSStateDB {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &CSStateDB{
		client: client,
		state:  state,
	}
}

func LoadState(state *tmtypes.State) State {
	stateTmp := State{
		ChainID:                          state.GetChainID(),
		LastBlockHeight:                  state.GetLastBlockHeight(),
		LastBlockTotalTx:                 state.GetLastBlockTotalTx(),
		LastBlockTime:                    state.LastBlockTime,
		Validators:                       nil,
		LastValidators:                   nil,
		LastHeightValidatorsChanged:      state.LastHeightValidatorsChanged,
		ConsensusParams:                  ttypes.ConsensusParams{BlockSize: ttypes.BlockSize{}, TxSize: ttypes.TxSize{}, BlockGossip: ttypes.BlockGossip{}, EvidenceParams: ttypes.EvidenceParams{}},
		LastHeightConsensusParamsChanged: state.LastHeightConsensusParamsChanged,
		LastResultsHash:                  state.LastResultsHash,
		AppHash:                          state.AppHash,
	}
	if validators := state.GetValidators(); validators != nil {
		if array := validators.GetValidators(); array != nil {
			targetArray := make([]*ttypes.Validator, len(array))
			LoadValidators(targetArray, array)
			stateTmp.Validators = &ttypes.ValidatorSet{Validators: targetArray, Proposer: nil}
		}
		if proposer := validators.GetProposer(); proposer != nil {
			if stateTmp.Validators == nil {
				tendermintlog.Error("LoadState validator is nil but proposer")
			} else {
				if val, err := LoadProposer(proposer); err == nil {
					stateTmp.Validators.Proposer = val
				}
			}
		}
	}
	if lastValidators := state.GetLastValidators(); lastValidators != nil {
		if array := lastValidators.GetValidators(); array != nil {
			targetArray := make([]*ttypes.Validator, len(array))
			LoadValidators(targetArray, array)
			stateTmp.LastValidators = &ttypes.ValidatorSet{Validators: targetArray, Proposer: nil}
		}
		if proposer := lastValidators.GetProposer(); proposer != nil {
			if stateTmp.LastValidators == nil {
				tendermintlog.Error("LoadState last validator is nil but proposer")
			} else {
				if val, err := LoadProposer(proposer); err == nil {
					stateTmp.LastValidators.Proposer = val
				}
			}
		}
	}
	if consensusParams := state.GetConsensusParams(); consensusParams != nil {
		if consensusParams.GetBlockSize() != nil {
			stateTmp.ConsensusParams.BlockSize.MaxBytes = int(consensusParams.BlockSize.MaxBytes)
			stateTmp.ConsensusParams.BlockSize.MaxGas = consensusParams.BlockSize.MaxGas
			stateTmp.ConsensusParams.BlockSize.MaxTxs = int(consensusParams.BlockSize.MaxTxs)
		}
		if consensusParams.GetTxSize() != nil {
			stateTmp.ConsensusParams.TxSize.MaxGas = consensusParams.TxSize.MaxGas
			stateTmp.ConsensusParams.TxSize.MaxBytes = int(consensusParams.TxSize.MaxBytes)
		}
		if consensusParams.GetBlockGossip() != nil {
			stateTmp.ConsensusParams.BlockGossip.BlockPartSizeBytes = int(consensusParams.BlockGossip.BlockPartSizeBytes)
		}
		if consensusParams.GetEvidenceParams() != nil {
			stateTmp.ConsensusParams.EvidenceParams.MaxAge = consensusParams.EvidenceParams.MaxAge
		}
	}

	return stateTmp
}

func (csdb *CSStateDB) SaveState(state State) {
	csdb.mtx.Lock()
	defer csdb.mtx.Unlock()
	csdb.state = state.Copy()
}

func (csdb *CSStateDB) LoadState() State {
	csdb.mtx.Lock()
	defer csdb.mtx.Unlock()
	return csdb.state
}

func (csdb *CSStateDB) LoadValidators(height int64) (*ttypes.ValidatorSet, error) {
	if height == 0 {
		return nil, nil
	}
	if csdb.state.LastBlockHeight+1 == height {
		return csdb.state.Validators, nil
	}

	blockInfo, err := csdb.client.QueryBlockInfoByHeight(height)
	if err != nil {
		tendermintlog.Error("LoadValidators GetBlockInfo failed", "error", err)
		panic(fmt.Sprintf("LoadValidators GetBlockInfo failed:%v", err))
	}

	var state State
	if blockInfo == nil {
		tendermintlog.Error("LoadValidators", "msg", "block height is not 0 but blockinfo is nil")
		panic(fmt.Sprintf("LoadValidators block height is %v but block info is nil", height))
	} else {
		csState := blockInfo.GetState()
		if csState == nil {
			tendermintlog.Error("LoadValidators", "msg", "blockInfo.GetState is nil")
			return nil, fmt.Errorf("LoadValidators get state from block info is nil")
		}
		state = LoadState(csState)
	}
	return state.Validators.Copy(), nil
}

func saveConsensusParams(dest *tmtypes.ConsensusParams, source ttypes.ConsensusParams) {
	dest.BlockSize.MaxBytes = int32(source.BlockSize.MaxBytes)
	dest.BlockSize.MaxTxs = int32(source.BlockSize.MaxTxs)
	dest.BlockSize.MaxGas = source.BlockSize.MaxGas
	dest.TxSize.MaxGas = source.TxSize.MaxGas
	dest.TxSize.MaxBytes = int32(source.TxSize.MaxBytes)
	dest.BlockGossip.BlockPartSizeBytes = int32(source.BlockGossip.BlockPartSizeBytes)
	dest.EvidenceParams.MaxAge = source.EvidenceParams.MaxAge
}

func saveValidators(dest []*tmtypes.Validator, source []*ttypes.Validator) []*tmtypes.Validator {
	for _, item := range source {
		if item == nil {
			dest = append(dest, &tmtypes.Validator{})
		} else {
			validator := &tmtypes.Validator{
				Address:     item.Address,
				PubKey:      item.PubKey,
				VotingPower: item.VotingPower,
				Accum:       item.Accum,
			}
			dest = append(dest, validator)
		}
	}
	return dest
}

func saveProposer(dest *tmtypes.Validator, source *ttypes.Validator) {
	if source != nil {
		dest.Address = source.Address
		dest.PubKey = source.PubKey
		dest.VotingPower = source.VotingPower
		dest.Accum = source.Accum
	}
}

func SaveState(state State) *tmtypes.State {
	newState := tmtypes.State{
		ChainID:                          state.ChainID,
		LastBlockHeight:                  state.LastBlockHeight,
		LastBlockTotalTx:                 state.LastBlockTotalTx,
		LastBlockTime:                    state.LastBlockTime,
		Validators:                       &tmtypes.ValidatorSet{Validators: make([]*tmtypes.Validator, 0), Proposer: &tmtypes.Validator{}},
		LastValidators:                   &tmtypes.ValidatorSet{Validators: make([]*tmtypes.Validator, 0), Proposer: &tmtypes.Validator{}},
		LastHeightValidatorsChanged:      state.LastHeightValidatorsChanged,
		ConsensusParams:                  &tmtypes.ConsensusParams{BlockSize: &tmtypes.BlockSize{}, TxSize: &tmtypes.TxSize{}, BlockGossip: &tmtypes.BlockGossip{}, EvidenceParams: &tmtypes.EvidenceParams{}},
		LastHeightConsensusParamsChanged: state.LastHeightConsensusParamsChanged,
		LastResultsHash:                  state.LastResultsHash,
		AppHash:                          state.AppHash,
	}
	if state.Validators != nil {
		newState.Validators.Validators = saveValidators(newState.Validators.Validators, state.Validators.Validators)
		saveProposer(newState.Validators.Proposer, state.Validators.Proposer)
	}
	if state.LastValidators != nil {
		newState.LastValidators.Validators = saveValidators(newState.LastValidators.Validators, state.LastValidators.Validators)
		saveProposer(newState.LastValidators.Proposer, state.LastValidators.Proposer)
	}
	saveConsensusParams(newState.ConsensusParams, state.ConsensusParams)
	return &newState
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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

func LoadValidators(des []*ttypes.Validator, source []*tmtypes.Validator) {
	for i, item := range source {
		if item.GetAddress() == nil || len(item.GetAddress()) == 0 {
			tendermintlog.Warn("LoadValidators get address is nil or empty")
			continue
		} else if item.GetPubKey() == nil || len(item.GetPubKey()) == 0 {
			tendermintlog.Warn("LoadValidators get pubkey is nil or empty")
			continue
		}
		des[i] = &ttypes.Validator{}
		des[i].Address = item.GetAddress()
		pub := item.GetPubKey()
		if pub == nil {
			tendermintlog.Error("LoadValidators get validator pubkey is nil", "item", i)
		} else {
			des[i].PubKey = pub
		}
		des[i].VotingPower = item.VotingPower
		des[i].Accum = item.Accum
	}
}

func LoadProposer(source *tmtypes.Validator) (*ttypes.Validator, error) {
	if source.GetAddress() == nil || len(source.GetAddress()) == 0 {
		tendermintlog.Warn("LoadProposer get address is nil or empty")
		return nil, errors.New("LoadProposer get address is nil or empty")
	} else if source.GetPubKey() == nil || len(source.GetPubKey()) == 0 {
		tendermintlog.Warn("LoadProposer get pubkey is nil or empty")
		return nil, errors.New("LoadProposer get pubkey is nil or empty")
	}

	des := &ttypes.Validator{}
	des.Address = source.GetAddress()
	pub := source.GetPubKey()
	if pub == nil {
		tendermintlog.Error("LoadProposer get pubkey is nil")
	} else {
		des.PubKey = pub
	}
	des.VotingPower = source.VotingPower
	des.Accum = source.Accum
	return des, nil
}

func CreateBlockInfoTx(pubkey string, lastCommit *tmtypes.TendermintCommit, seenCommit *tmtypes.TendermintCommit, state *tmtypes.State, proposal *tmtypes.Proposal, block *tmtypes.TendermintBlock) *types.Transaction {
	blockNoTxs := *block
	blockNoTxs.Txs = make([]*types.Transaction, 0)
	blockInfo := &tmtypes.TendermintBlockInfo{
		SeenCommit: seenCommit,
		LastCommit: lastCommit,
		State:      state,
		Proposal:   proposal,
		Block:      &blockNoTxs,
	}
	tendermintlog.Debug("CreateBlockInfoTx", "validators", blockInfo.State.Validators.Validators, "block", block, "block-notxs", blockNoTxs)

	nput := &tmtypes.ValNodeAction_BlockInfo{BlockInfo: blockInfo}
	action := &tmtypes.ValNodeAction{Value: nput, Ty: tmtypes.ValNodeActionBlockInfo}
	tx := &types.Transaction{Execer: []byte("valnode"), Payload: types.Encode(action), Fee: fee}
	tx.To = address.ExecAddress("valnode")
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(pubkey))

	return tx
}
