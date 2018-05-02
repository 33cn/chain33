package state

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	wire "github.com/tendermint/go-wire"

	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"os"
)

// database keys
var (
	stateKey = []byte("stateKey")
)

//-----------------------------------------------------------------------------

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
	LastBlockID      types.BlockID
	LastBlockTime    time.Time

	// LastValidators is used to validate block.LastCommit.
	// Validators are persisted to the database separately every time they change,
	// so we can query for historical validator sets.
	// Note that if s.LastBlockHeight causes a valset change,
	// we set s.LastHeightValidatorsChanged = s.LastBlockHeight + 1
	Validators                  *types.ValidatorSet
	LastValidators              *types.ValidatorSet
	LastHeightValidatorsChanged int64

	// Consensus parameters used for validating blocks.
	// Changes returned by EndBlock and updated after Commit.
	ConsensusParams                  types.ConsensusParams
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
	return wire.BinaryBytes(s)
}

// IsEmpty returns true if the State is equal to the empty State.
func (s State) IsEmpty() bool {
	return s.Validators == nil // XXX can't compare to Empty
}

// GetValidators returns the last and current validator sets.
func (s State) GetValidators() (last *types.ValidatorSet, current *types.ValidatorSet) {
	return s.LastValidators, s.Validators
}

//------------------------------------------------------------------------
// Create a block from the latest state

// MakeBlock builds a block with the given txs and commit from the current state.
func (s State) MakeBlock(height int64, txs []types.Tx, commit *types.Commit, lastParentHash []byte, lastBlockTime int64, lastStateHash []byte) (*types.Block, *types.PartSet) {
	// build base block
	block := types.MakeBlock(height, txs, commit, lastParentHash, lastBlockTime, lastStateHash)

	// fill header with state data
	block.ChainID = s.ChainID
	block.TotalTxs = s.LastBlockTotalTx + block.NumTxs
	block.LastBlockID = s.LastBlockID
	block.ValidatorsHash = s.Validators.Hash()
	block.AppHash = s.AppHash
	block.ConsensusHash = s.ConsensusParams.Hash()
	block.LastResultsHash = s.LastResultsHash

	return block, block.MakePartSet(s.ConsensusParams.BlockGossip.BlockPartSizeBytes)
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
func MakeGenesisDocFromFile(genDocFile string) (*types.GenesisDoc, error) {
	genDocJSON, err := ioutil.ReadFile(genDocFile)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read GenesisDoc file: %v", err)
	}
	genDoc, err := types.GenesisDocFromJSON(genDocJSON)
	if err != nil {
		return nil, fmt.Errorf("Error reading GenesisDoc: %v", err)
	}
	return genDoc, nil
}

// MakeGenesisState creates state from types.GenesisDoc.
func MakeGenesisState(genDoc *types.GenesisDoc) (State, error) {
	err := genDoc.ValidateAndComplete()
	if err != nil {
		return State{}, fmt.Errorf("Error in genesis file: %v", err)
	}

	// Make validators slice
	validators := make([]*types.Validator, len(genDoc.Validators))
	for i, val := range genDoc.Validators {
		pubKey := val.PubKey
		address := pubKey.Address()

		// Make validator
		validators[i] = &types.Validator{
			Address:     address,
			PubKey:      pubKey,
			VotingPower: val.Power,
		}
	}

	return State{

		ChainID: genDoc.ChainID,

		LastBlockHeight: 0,
		LastBlockID:     types.BlockID{},
		LastBlockTime:   genDoc.GenesisTime,

		Validators:                  types.NewValidatorSet(validators),
		LastValidators:              types.NewValidatorSet(nil),
		LastHeightValidatorsChanged: 1,

		ConsensusParams:                  *genDoc.ConsensusParams,
		LastHeightConsensusParamsChanged: 1,

		AppHash: genDoc.AppHash,
	}, nil
}

// LoadStateFromDBOrGenesisFile loads the most recent state from the database,
// or creates a new one from the given genesisFilePath and persists the result
// to the database.
func LoadStateFromDBOrGenesisFile(stateDB dbm.DB, genesisFilePath string) (State, error) {
	state := LoadState(stateDB)
	if state.IsEmpty() {
		var err error
		state, err = MakeGenesisStateFromFile(genesisFilePath)
		if err != nil {
			return state, err
		}
		SaveState(stateDB, state)
	}

	return state, nil
}

// LoadStateFromDBOrGenesisDoc loads the most recent state from the database,
// or creates a new one from the given genesisDoc and persists the result
// to the database.
func LoadStateFromDBOrGenesisDoc(stateDB dbm.DB, genesisDoc *types.GenesisDoc) (State, error) {
	state := LoadState(stateDB)
	if state.IsEmpty() {
		var err error
		state, err = MakeGenesisState(genesisDoc)
		if err != nil {
			return state, err
		}
		SaveState(stateDB, state)
	}

	return state, nil
}

// LoadState loads the State from the database.
func LoadState(db dbm.DB) State {
	return loadState(db, stateKey)
}

func loadState(db dbm.DB, key []byte) (state State) {
	buf,e := db.Get(key)
	if e != nil {
		fmt.Printf(fmt.Sprintf(`LoadState: db get key %v failed:%v\n`, key, e))
	}
	if len(buf) == 0 {
		return state
	}

	r, n, err := bytes.NewReader(buf), new(int), new(error)
	wire.ReadBinaryPtr(&state, r, 0, n, err)
	if *err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		fmt.Printf(fmt.Sprintf(`LoadState: Data has been corrupted or its spec has changed:%v\n`, *err))
		os.Exit(1)
	}
	// TODO: ensure that buf is completely read.

	return state
}

// SaveState persists the State, the ValidatorsInfo, and the ConsensusParamsInfo to the database.
func SaveState(db dbm.DB, s State) {
	saveState(db, s, stateKey)
}

func saveState(db dbm.DB, s State, key []byte) {
	nextHeight := s.LastBlockHeight + 1
	saveValidatorsInfo(db, nextHeight, s.LastHeightValidatorsChanged, s.Validators)
	saveConsensusParamsInfo(db, nextHeight, s.LastHeightConsensusParamsChanged, s.ConsensusParams)
	db.SetSync(stateKey, s.Bytes())
}

// ValidatorsInfo represents the latest validator set, or the last height it changed
type ValidatorsInfo struct {
	ValidatorSet      *types.ValidatorSet
	LastHeightChanged int64
}

// Bytes serializes the ValidatorsInfo using go-wire
func (valInfo *ValidatorsInfo) Bytes() []byte {
	return wire.BinaryBytes(*valInfo)
}

// ConsensusParamsInfo represents the latest consensus params, or the last height it changed
type ConsensusParamsInfo struct {
	ConsensusParams   types.ConsensusParams
	LastHeightChanged int64
}

// Bytes serializes the ConsensusParamsInfo using go-wire
func (params ConsensusParamsInfo) Bytes() []byte {
	return wire.BinaryBytes(params)
}

func calcValidatorsKey(height int64) []byte {
	return []byte(fmt.Sprintf("validatorsKey:%v", height))
}

func calcConsensusParamsKey(height int64) []byte {
	return []byte(fmt.Sprintf("consensusParamsKey:%v", height))
}

// saveValidatorsInfo persists the validator set for the next block to disk.
// It should be called from s.Save(), right before the state itself is persisted.
// If the validator set did not change after processing the latest block,
// only the last height for which the validators changed is persisted.
func saveValidatorsInfo(db dbm.DB, nextHeight, changeHeight int64, valSet *types.ValidatorSet) {
	valInfo := &ValidatorsInfo{
		LastHeightChanged: changeHeight,
	}
	if changeHeight == nextHeight {
		valInfo.ValidatorSet = valSet
	}
	db.SetSync(calcValidatorsKey(nextHeight), valInfo.Bytes())
}

// saveConsensusParamsInfo persists the consensus params for the next block to disk.
// It should be called from s.Save(), right before the state itself is persisted.
// If the consensus params did not change after processing the latest block,
// only the last height for which they changed is persisted.
func saveConsensusParamsInfo(db dbm.DB, nextHeight, changeHeight int64, params types.ConsensusParams) {
	paramsInfo := &ConsensusParamsInfo{
		LastHeightChanged: changeHeight,
	}
	if changeHeight == nextHeight {
		paramsInfo.ConsensusParams = params
	}
	db.SetSync(calcConsensusParamsKey(nextHeight), paramsInfo.Bytes())
}

func loadValidatorsInfo(db dbm.DB, height int64) *ValidatorsInfo {
	buf, e := db.Get(calcValidatorsKey(height))
	if e != nil {
		fmt.Printf(fmt.Sprintf(`LoadValidators: db get height key %v failed:
                %v\n`, calcValidatorsKey(height), e))
	}
	if len(buf) == 0 {
		return nil
	}

	v := new(ValidatorsInfo)
	r, n, err := bytes.NewReader(buf), new(int), new(error)
	wire.ReadBinaryPtr(v, r, 0, n, err)
	if *err != nil {
		// DATA HAS BEEN CORRUPTED OR THE SPEC HAS CHANGED
		fmt.Printf(fmt.Sprintf(`LoadValidators: Data has been corrupted or its spec has changed:
                %v\n`, *err))
		os.Exit(1)
	}
	// TODO: ensure that buf is completely read.

	return v
}

// LoadValidators loads the ValidatorSet for a given height.
// Returns ErrNoValSetForHeight if the validator set can't be found for this height.
func LoadValidators(db dbm.DB, height int64) (*types.ValidatorSet, error) {
	valInfo := loadValidatorsInfo(db, height)
	if valInfo == nil {
		return nil, ErrNoValSetForHeight{height}
	}

	if valInfo.ValidatorSet == nil {
		valInfo = loadValidatorsInfo(db, valInfo.LastHeightChanged)
		if valInfo == nil {
			panic(fmt.Sprintf("Panicked on a Sanity Check: %v",fmt.Sprintf(`Couldn't find validators at height %d as
                        last changed from height %d`, valInfo.LastHeightChanged, height)))
		}
	}

	return valInfo.ValidatorSet, nil
}
