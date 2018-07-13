package tendermint

import (
	"fmt"

	"bytes"
	"encoding/json"
	"errors"
	"sync"

	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
)

type ValidatorCache struct {
	PubKey []byte
	Power  int64
}

type ValidatorsCache struct {
	mtx        sync.Mutex
	validators []*ValidatorCache
}

var (
	validatorsCache ValidatorsCache
)

func InitValidatorsCache(vals *types.ValidatorSet) {
	validatorsCache.mtx.Lock()
	defer validatorsCache.mtx.Unlock()
	validatorsCache.validators = make([]*ValidatorCache, len(vals.Validators))
	for i, item := range vals.Validators {
		validatorsCache.validators[i] = &ValidatorCache{
			PubKey: item.PubKey,
			Power:  item.VotingPower,
		}
	}
}

func GetValidatorsCache() []*ValidatorCache {
	validatorsCache.mtx.Lock()
	defer validatorsCache.mtx.Unlock()
	validators := make([]*ValidatorCache, len(validatorsCache.validators))
	for i, item := range validatorsCache.validators {
		validators[i] = &ValidatorCache{
			PubKey: item.PubKey,
			Power:  item.Power,
		}
	}
	return validators
}

func UpdateValidator2Cache(val ValidatorCache) {
	validatorsCache.mtx.Lock()
	defer validatorsCache.mtx.Unlock()
	for _, item := range validatorsCache.validators {
		if bytes.Equal(item.PubKey, val.PubKey) {
			item.Power = val.Power
			return
		}
	}
	validatorsCache.validators = append(validatorsCache.validators, &val)
}

//-----------------------------------------------------------------------------
// BlockExecutor handles block execution and state updates.
// It exposes ApplyBlock(), which validates & executes the block, updates state w/ ABCI responses,
// then commits and updates the mempool atomically, then saves state.

// BlockExecutor provides the context and accessories for properly executing a block.
type BlockExecutor struct {
	// save state, validators, consensus params, abci responses here
	db *CSStateDB

	// execute the app against this
	//proxyApp proxy.AppConnConsensus

	// update these with block results after commit
	//mempool types.Mempool
	evpool types.EvidencePool
}

// NewBlockExecutor returns a new BlockExecutor with a NopEventBus.
// Call SetEventBus to provide one.
func NewBlockExecutor(db *CSStateDB, evpool types.EvidencePool) *BlockExecutor {
	return &BlockExecutor{
		db:     db,
		evpool: evpool,
	}
}

// ValidateBlock validates the given block against the given state.
// If the block is invalid, it returns an error.
// Validation does not mutate state, but does require historical information from the stateDB,
// ie. to verify evidence from a validator at an old height.
func (blockExec *BlockExecutor) ValidateBlock(s State, block *types.Block) error {
	return validateBlock(blockExec.db, s, block)
}

// ApplyBlock validates the block against the state, executes it against the app,
// fires the relevant events, commits the app, and saves the new state and responses.
// It's the only function that needs to be called
// from outside this package to process and commit an entire block.
// It takes a blockID to avoid recomputing the parts hash.
func (blockExec *BlockExecutor) ApplyBlock(s State, blockID types.BlockID, block *types.Block) (State, error) {

	if err := blockExec.ValidateBlock(s, block); err != nil {
		return s, ErrInvalidBlock(err)
	}
	/*
		abciResponses, err := execBlockOnProxyApp(blockExec.logger, blockExec.proxyApp, block)
		if err != nil {
			return s, ErrProxyAppConn(err)
		}
	*/
	//fail.Fail() // XXX

	// save the results before we commit
	//saveABCIResponses(blockExec.db, block.Height, abciResponses)

	//fail.Fail() // XXX

	// update the state with the block and responses
	s, err := updateState(s, blockID, block)
	if err != nil {
		return s, fmt.Errorf("Commit failed for application: %v", err)
	}

	// lock mempool, commit state, update mempoool
	/*
		appHash, err := blockExec.Commit(block)
		if err != nil {
			return s, fmt.Errorf("Commit failed for application: %v", err)
		}
	*/
	//fail.Fail() // XXX

	// update the app hash and save the state
	//s.AppHash = appHash
	blockExec.db.SaveState(s)

	//fail.Fail() // XXX

	// Update evpool now that state is saved
	// TODO: handle the crash/recover scenario
	// ie. (may need to call Update for last block)
	blockExec.evpool.Update(block)

	return s, nil
}

// updateState returns a new State updated according to the header and responses.
func updateState(s State, blockID types.BlockID, block *types.Block) (State, error) {

	// copy the valset so we can apply changes from EndBlock
	// and update s.LastValidators and s.Validators
	prevValSet := s.Validators.Copy()
	nextValSet := prevValSet.Copy()

	// update the validator set with the latest abciResponses
	lastHeightValsChanged := s.LastHeightValidatorsChanged
	validatorUpdates := GetValidatorsCache()
	if len(validatorUpdates) > 0 {
		err := updateValidators(nextValSet, validatorUpdates)
		if err != nil {
			tendermintlog.Error("Error changing validator set", "error", err)
			//return s, fmt.Errorf("Error changing validator set: %v", err)
		}
		// change results from this height but only applies to the next height
		lastHeightValsChanged = block.Height + 1
	}
	// Update validator accums and set state variables
	nextValSet.IncrementAccum(1)

	// update the params with the latest abciResponses
	nextParams := s.ConsensusParams
	lastHeightParamsChanged := s.LastHeightConsensusParamsChanged

	// NOTE: the AppHash has not been populated.
	// It will be filled on state.Save.
	return State{
		ChainID:                          s.ChainID,
		LastBlockHeight:                  block.Header.Height,
		LastBlockTotalTx:                 s.LastBlockTotalTx + block.Header.NumTxs,
		LastBlockID:                      blockID,
		LastBlockTime:                    block.Header.Time,
		Validators:                       nextValSet,
		LastValidators:                   s.Validators.Copy(),
		LastHeightValidatorsChanged:      lastHeightValsChanged,
		ConsensusParams:                  nextParams,
		LastHeightConsensusParamsChanged: lastHeightParamsChanged,
		LastResultsHash:                  nil,
		AppHash:                          nil,
	}, nil
}

//----------------------------------------------------------------------------------------------------
// Execute block without state. TODO: eliminate

// ExecCommitBlock executes and commits a block on the proxyApp without validating or mutating the state.
// It returns the application root hash (result of abci.Commit).
/*
func ExecCommitBlock(appConnConsensus proxy.AppConnConsensus, block *types.Block, logger log.Logger) ([]byte, error) {
	_, err := execBlockOnProxyApp(logger, appConnConsensus, block)
	if err != nil {
		logger.Error("Error executing block on proxy app", "height", block.Height, "err", err)
		return nil, err
	}
	// Commit block, get hash back
	res, err := appConnConsensus.CommitSync()
	if err != nil {
		logger.Error("Client error during proxyAppConn.CommitSync", "err", res)
		return nil, err
	}
	if res.IsErr() {
		logger.Error("Error in proxyAppConn.CommitSync", "err", res)
		return nil, res
	}
	if res.Log != "" {
		logger.Info("Commit.Log: " + res.Log)
	}
	return res.Data, nil
}
*/
func updateValidators(currentSet *types.ValidatorSet, updates []*ValidatorCache) error {
	// If more or equal than 1/3 of total voting power changed in one block, then
	// a light client could never prove the transition externally. See
	// ./lite/doc.go for details on how a light client tracks validators.
	vp23, err := changeInVotingPowerMoreOrEqualToOneThird(currentSet, updates)
	if err != nil {
		return err
	}
	if vp23 {
		return errors.New("the change in voting power must be strictly less than 1/3")
	}

	for _, v := range updates {
		pubkey, err := types.ConsensusCrypto.PubKeyFromBytes(v.PubKey) // NOTE: expects go-wire encoded pubkey
		if err != nil {
			return err
		}

		address := types.GenAddressByPubKey(pubkey)
		power := int64(v.Power)
		// mind the overflow from int64
		if power < 0 {
			return fmt.Errorf("Power (%d) overflows int64", v.Power)
		}

		_, val := currentSet.GetByAddress(address)
		if val == nil {
			// add val
			added := currentSet.Add(types.NewValidator(pubkey, power))
			if !added {
				return fmt.Errorf("Failed to add new validator %X with voting power %d", address, power)
			}
		} else if v.Power == 0 {
			// remove val
			_, removed := currentSet.Remove(address)
			if !removed {
				return fmt.Errorf("Failed to remove validator %X", address)
			}
		} else {
			// update val
			val.VotingPower = power
			updated := currentSet.Update(val)
			if !updated {
				return fmt.Errorf("Failed to update validator %X with voting power %d", address, power)
			}
		}
	}
	return nil
}

func changeInVotingPowerMoreOrEqualToOneThird(currentSet *types.ValidatorSet, updates []*ValidatorCache) (bool, error) {
	threshold := currentSet.TotalVotingPower() * 1 / 3
	acc := int64(0)

	for _, v := range updates {
		pubkey, err := types.ConsensusCrypto.PubKeyFromBytes(v.PubKey) // NOTE: expects go-wire encoded pubkey
		if err != nil {
			return false, err
		}

		address := types.GenAddressByPubKey(pubkey)
		power := int64(v.Power)
		// mind the overflow from int64
		if power < 0 {
			return false, fmt.Errorf("Power (%d) overflows int64", v.Power)
		}

		_, val := currentSet.GetByAddress(address)
		if val == nil {
			acc += power
		} else {
			np := val.VotingPower - power
			if np < 0 {
				np = -np
			}
			acc += np
		}

		if acc >= threshold {
			return true, nil
		}
	}

	return false, nil
}

func validateBlock(stateDB *CSStateDB, s State, b *types.Block) error {
	// validate internal consistency
	newTxs, err := b.ValidateBasic()
	if err != nil {
		return err
	}

	// validate basic info
	if b.ChainID != s.ChainID {
		return fmt.Errorf("Wrong Block.Header.ChainID. Expected %v, got %v", s.ChainID, b.ChainID)
	}
	if b.Height != s.LastBlockHeight+1 {
		return fmt.Errorf("Wrong Block.Header.Height. Expected %v, got %v", s.LastBlockHeight+1, b.Height)
	}
	/*	TODO: Determine bounds for Time
		See blockchain/reactor "stopSyncingDurationMinutes"

		if !b.Time.After(lastBlockTime) {
			return errors.New("Invalid Block.Header.Time")
		}
	*/

	// validate prev block info
	if !b.LastBlockID.Equals(s.LastBlockID) {
		return fmt.Errorf("Wrong Block.Header.LastBlockID.  Expected %v, got %v", s.LastBlockID, b.LastBlockID)
	}

	if b.TotalTxs != s.LastBlockTotalTx+newTxs {
		return fmt.Errorf("Wrong Block.Header.TotalTxs. Expected %v, got %v", s.LastBlockTotalTx+newTxs, b.TotalTxs)
	}

	// validate app info
	if !bytes.Equal(b.AppHash, s.AppHash) {
		return fmt.Errorf("Wrong Block.Header.AppHash.  Expected %X, got %v", s.AppHash, b.AppHash)
	}
	if !bytes.Equal(b.ConsensusHash, s.ConsensusParams.Hash()) {
		return fmt.Errorf("Wrong Block.Header.ConsensusHash.  Expected %X, got %v", s.ConsensusParams.Hash(), b.ConsensusHash)
	}
	if !bytes.Equal(b.LastResultsHash, s.LastResultsHash) {
		return fmt.Errorf("Wrong Block.Header.LastResultsHash.  Expected %X, got %v", s.LastResultsHash, b.LastResultsHash)
	}
	if !bytes.Equal(b.ValidatorsHash, s.Validators.Hash()) {
		return fmt.Errorf("Wrong Block.Header.ValidatorsHash.  Expected %X, got %v", s.Validators.Hash(), b.ValidatorsHash)
	}

	// Validate block LastCommit.
	if b.Height == 1 {
		if len(b.LastCommit.Precommits) != 0 {
			return errors.New("Block at height 1 (first block) should have no LastCommit precommits")
		}
	} else {
		if len(b.LastCommit.Precommits) != s.LastValidators.Size() {
			return fmt.Errorf("Invalid block commit size. Expected %v, got %v",
				s.LastValidators.Size(), len(b.LastCommit.Precommits))
		}
		err := s.LastValidators.VerifyCommit(
			s.ChainID, s.LastBlockID, b.Height-1, b.LastCommit)
		if err != nil {
			return err
		}
	}

	for _, ev := range b.Evidence.Evidence {
		if v, ok := types.EvidenceType2Obj[ev.Kind]; ok {
			tmp := v.(types.Evidence).Copy()
			err := json.Unmarshal(*ev.Data, &tmp)
			if err != nil {
				return fmt.Errorf("validateBlock envelop unmarshal failed:%v", err)
			}
			if err := VerifyEvidence(stateDB, s, tmp); err != nil {
				return types.NewEvidenceInvalidErr(tmp, err)
			}
		}
	}

	return nil
}
