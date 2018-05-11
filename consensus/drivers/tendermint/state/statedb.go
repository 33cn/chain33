package state

import (
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"fmt"
	gtypes "gitlab.33.cn/chain33/chain33/types"
	"time"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
	"sync"
	"errors"
)

var csStateLog = log.New("module", "tendermint-stateDB")

type CSStateDB struct {
	client *drivers.BaseClient
	state  State
	mtx sync.Mutex
}

func NewStateDB(client *drivers.BaseClient, state State) *CSStateDB{
	return &CSStateDB{
		client:client,
		state: state,
	}
}

func LoadState(state *gtypes.State) State {
	stateTmp := State {
		ChainID: state.GetChainID(),
		LastBlockHeight: state.GetLastBlockHeight(),
		LastBlockTotalTx: state.GetLastBlockTotalTx(),
		LastBlockTime: time.Unix(0, state.LastBlockTime),
		Validators: nil,
		LastValidators:nil,
		LastHeightValidatorsChanged: state.LastHeightValidatorsChanged,
		ConsensusParams: types.ConsensusParams{BlockSize:types.BlockSize{}, TxSize:types.TxSize{}, BlockGossip:types.BlockGossip{}, EvidenceParams:types.EvidenceParams{}},
		LastHeightConsensusParamsChanged: state.LastHeightConsensusParamsChanged,
		LastResultsHash:state.LastResultsHash,
		AppHash:state.AppHash,

	}
	if validators := state.GetValidators(); validators != nil {
		if array := validators.GetValidators(); array != nil {
			targetArray := make([]*types.Validator, len(array))
			types.LoadValidators(targetArray, array)
			stateTmp.Validators = &types.ValidatorSet{Validators:targetArray, Proposer: nil}
		}
		if proposer := validators.GetProposer(); proposer != nil {
			if stateTmp.Validators == nil {
				csStateLog.Error("LoadState validator is nil but proposer")
			} else {
				val := types.Validator{}
				if err := types.LoadProposer(&val, proposer); err == nil {
					stateTmp.Validators.Proposer = &val
				}
			}
		}
	}
	if lastValidators := state.GetLastValidators();lastValidators != nil {
		if array := lastValidators.GetValidators(); array != nil {
			targetArray := make([]*types.Validator, len(array))
			types.LoadValidators(targetArray, array)
			stateTmp.LastValidators = &types.ValidatorSet{Validators:targetArray, Proposer: nil}
		}
		if proposer := lastValidators.GetProposer(); proposer != nil {
			if stateTmp.LastValidators == nil {
				csStateLog.Error("LoadState last validator is nil but proposer")
			} else {
				val := types.Validator{}
				if err := types.LoadProposer(&val, proposer); err == nil {
					stateTmp.LastValidators.Proposer = &val
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

func (csdb *CSStateDB) LoadValidators(height int64) (*types.ValidatorSet, error) {
	if height == 0 {
		return nil, nil
	}
	if csdb.state.LastBlockHeight + 1 == height {
		return csdb.state.Validators, nil
	}
	curHeight := csdb.client.GetCurrentHeight()
	block, err := csdb.client.RequestBlock(height)
	if err != nil {
		csStateLog.Error(fmt.Sprintf("LoadValidators : Couldn't find block at height %d as current height %d",height, curHeight))
		return nil, nil
	}
	blockInfo, err := types.GetBlockInfo(block)
	if err != nil {
		csStateLog.Error("LoadValidators GetBlockInfo failed", "error", err)
		panic(fmt.Sprintf("LoadValidators GetBlockInfo failed:%v",err))
	}

	var state State
	if blockInfo == nil {
		csStateLog.Error("LoadValidators", "msg", "block height is not 0 but blockinfo is nil")
		panic(fmt.Sprintf("LoadValidators block height is %v but block info is nil", block.Height))
	} else {
		csState := blockInfo.GetState()
		if csState == nil {
			csStateLog.Error("LoadValidators", "msg", "blockInfo.GetState is nil")
			return nil, errors.New(fmt.Sprintf("LoadValidators get state from block info is nil"))
		}
		state = LoadState(csState)
	}
	return state.Validators.Copy(), nil
}

func saveConsensusParams(dest *gtypes.ConsensusParams, source types.ConsensusParams) {
	dest.BlockSize.MaxBytes = int32(source.BlockSize.MaxBytes)
	dest.BlockSize.MaxTxs = int32(source.BlockSize.MaxTxs)
	dest.BlockSize.MaxGas = source.BlockSize.MaxGas
	dest.TxSize.MaxGas = source.TxSize.MaxGas
	dest.TxSize.MaxBytes = int32(source.TxSize.MaxBytes)
	dest.BlockGossip.BlockPartSizeBytes = int32(source.BlockGossip.BlockPartSizeBytes)
	dest.EvidenceParams.MaxAge = source.EvidenceParams.MaxAge
}

func saveValidators(dest []*gtypes.Validator, source []*types.Validator) []*gtypes.Validator{
	for _, item := range source {
		if item == nil {
			dest = append(dest, &gtypes.Validator{})
		} else {
			validator := &gtypes.Validator{
				Address: item.Address,
				PubKey: item.PubKey.Unwrap().Bytes(),
				VotingPower: item.VotingPower,
				Accum: item.Accum,
			}
			dest = append(dest, validator)
		}
	}
	return dest
}

func saveProposer(dest *gtypes.Validator, source *types.Validator){
	if source != nil {
		dest.Address = source.Address
		dest.PubKey = source.PubKey.Unwrap().Bytes()
		dest.VotingPower = source.VotingPower
		dest.Accum = source.Accum
		}
}

func SaveState(state State) *gtypes.State {
	newState := gtypes.State {
		ChainID: state.ChainID,
		LastBlockHeight: state.LastBlockHeight,
		LastBlockTotalTx: state.LastBlockTotalTx,
		LastBlockTime: state.LastBlockTime.UnixNano(),
		Validators: &gtypes.ValidatorSet{Validators:make([]*gtypes.Validator,0), Proposer:&gtypes.Validator{}},
		LastValidators:&gtypes.ValidatorSet{Validators:make([]*gtypes.Validator,0), Proposer:&gtypes.Validator{}},
		LastHeightValidatorsChanged: state.LastHeightValidatorsChanged,
		ConsensusParams: &gtypes.ConsensusParams{BlockSize:&gtypes.BlockSize{}, TxSize:&gtypes.TxSize{}, BlockGossip:&gtypes.BlockGossip{}, EvidenceParams:&gtypes.EvidenceParams{}},
		LastHeightConsensusParamsChanged: state.LastHeightConsensusParamsChanged,
		LastResultsHash:state.LastResultsHash,
		AppHash:state.AppHash,
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