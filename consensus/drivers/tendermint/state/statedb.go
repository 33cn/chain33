package state

import (
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"fmt"
	gtypes "gitlab.33.cn/chain33/chain33/types"
	"time"
	log "github.com/inconshreveable/log15"
	crypto "github.com/tendermint/go-crypto"
	"errors"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
)

var csStateLog = log.New("module", "tendermint-stateDB")

type CSStateDB struct {
	client *drivers.BaseClient
	state  State
}

func NewStateDB(client *drivers.BaseClient, state State) *CSStateDB{
	return &CSStateDB{
		client:client,
		state: state,
	}
}

func LoadValidators(des []*types.Validator, source []*gtypes.Validator) {
	for i, item := range source {
		if item.GetAddress() == nil || len(item.GetAddress()) == 0 {
			csStateLog.Warn("LoadValidators get address is nil or empty")
			continue
		} else if item.GetPubKey() == nil || len(item.GetPubKey()) == 0 {
			csStateLog.Warn("LoadValidators get pubkey is nil or empty")
			continue
		}
		des[i] = &types.Validator{}
		des[i].Address = item.GetAddress()
		pub, err := crypto.PubKeyFromBytes(item.GetPubKey())
		if err != nil {
			csStateLog.Error("LoadValidators get pubkey from byte failed", "err", err)
		} else {
			des[i].PubKey = pub.Wrap()
		}
		des[i].VotingPower = item.VotingPower
		des[i].Accum = item.Accum
	}
}

func LoadProposer(des *types.Validator, source *gtypes.Validator) error {
	if source.GetAddress() == nil || len(source.GetAddress()) == 0 {
		csStateLog.Warn("LoadProposer get address is nil or empty")
		return errors.New("LoadProposer get address is nil or empty")
	} else if source.GetPubKey() == nil || len(source.GetPubKey()) == 0 {
		csStateLog.Warn("LoadProposer get pubkey is nil or empty")
		return errors.New("LoadProposer get pubkey is nil or empty")
	}
	des = &types.Validator{}
	des.Address = source.GetAddress()
	pub, err := crypto.PubKeyFromBytes(source.GetPubKey())
	if err != nil {
		csStateLog.Error("LoadProposer get pubkey from byte failed", "err", err)
		return errors.New(fmt.Sprintf("LoadProposer get pubkey from byte failed, err:%v", err))
	} else {
		des.PubKey = pub.Wrap()
	}
	des.VotingPower = source.VotingPower
	des.Accum = source.Accum
	return nil
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
			LoadValidators(targetArray, array)
			stateTmp.Validators = &types.ValidatorSet{Validators:targetArray, Proposer: nil}
		}
		if proposer := validators.GetProposer(); proposer != nil {
			if stateTmp.Validators == nil {
				csStateLog.Error("LoadState validator is nil but proposer")
			} else {
				val := types.Validator{}
				if err := LoadProposer(&val, proposer); err == nil {
					stateTmp.Validators.Proposer = &val
				}
			}
		}
	}
	if lastValidators := state.GetLastValidators();lastValidators != nil {
		if array := lastValidators.GetValidators(); array != nil {
			targetArray := make([]*types.Validator, len(array))
			LoadValidators(targetArray, array)
			stateTmp.LastValidators = &types.ValidatorSet{Validators:targetArray, Proposer: nil}
		}
		if proposer := lastValidators.GetProposer(); proposer != nil {
			if stateTmp.LastValidators == nil {
				csStateLog.Error("LoadState last validator is nil but proposer")
			} else {
				val := types.Validator{}
				if err := LoadProposer(&val, proposer); err == nil {
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

func (csdb *CSStateDB) LoadState() State {
	return csdb.state
}

func (csdb *CSStateDB) loadValidatorsInfo(height int64) *ValidatorsInfo {
	curHeight := csdb.client.GetCurrentHeight()
	if curHeight != height {

	}
	return nil
}

func (csdb *CSStateDB) LoadValidators(height int64) (*types.ValidatorSet, error) {

	valInfo := csdb.loadValidatorsInfo(height)
	if valInfo == nil {
		return nil, ErrNoValSetForHeight{height}
	}

	if valInfo.ValidatorSet == nil {
		valInfo = csdb.loadValidatorsInfo(valInfo.LastHeightChanged)
		if valInfo == nil {
			panic(fmt.Sprintf("Panicked on a Sanity Check: %v", fmt.Sprintf(`Couldn't find validators at height %d as
                        last changed from height %d`, valInfo.LastHeightChanged, height)))
		}
	}

	return valInfo.ValidatorSet, nil
}