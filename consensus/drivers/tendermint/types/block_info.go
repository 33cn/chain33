package types

import (
	"errors"
	"fmt"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

var bilog = log15.New("module", "tendermint-blockinfo")

var ConsensusCrypto crypto.Crypto

func GetBlockInfo(block *types.Block) (*types.TendermintBlockInfo, error) {
	if len(block.Txs) == 0 || block.Height == 0 {
		return nil, nil
	}
	//bilog.Info("GetBlockInfo", "txs", block.Txs)
	baseTx := block.Txs[0]
	//判断交易类型和执行情况
	var blockInfo types.TendermintBlockInfo
	nGet := &types.NormPut{}
	action := &types.NormAction{}
	err := types.Decode(baseTx.GetPayload(), action)
	if err != nil {
		bilog.Error("GetBlockInfo decode payload failed", "error", err)
		return nil, errors.New(fmt.Sprintf("GetBlockInfo decode payload failed:%v", err))
	}
	if nGet = action.GetNput(); nGet == nil {
		bilog.Error("GetBlockInfo get nput failed")
		return nil, errors.New("GetBlockInfo get nput failed")
	}
	infobytes := nGet.GetValue()
	if infobytes == nil {
		bilog.Error("GetBlockInfo get blockinfo value failed")
		return nil, errors.New("GetBlockInfo get blockinfo value failed")
	}
	err = types.Decode(infobytes, &blockInfo)
	if err != nil {
		bilog.Error("GetBlockInfo decode blockinfo failed", "error", err)
		return nil, errors.New(fmt.Sprintf("GetBlockInfo decode blockinfo failed:%v", err))
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
			bilog.Warn("LoadValidators get address is nil or empty")
			continue
		} else if item.GetPubKey() == nil || len(item.GetPubKey()) == 0 {
			bilog.Warn("LoadValidators get pubkey is nil or empty")
			continue
		}
		des[i] = &Validator{}
		des[i].Address = item.GetAddress()
		pub := item.GetPubKey()
		if pub == nil {
			bilog.Error("LoadValidators get validator pubkey is nil", "item", i)
		} else {
			des[i].PubKey = pub
		}
		des[i].VotingPower = item.VotingPower
		des[i].Accum = item.Accum
	}
}

func LoadProposer(source *types.Validator) (*Validator, error) {
	if source.GetAddress() == nil || len(source.GetAddress()) == 0 {
		bilog.Warn("LoadProposer get address is nil or empty")
		return nil, errors.New("LoadProposer get address is nil or empty")
	} else if source.GetPubKey() == nil || len(source.GetPubKey()) == 0 {
		bilog.Warn("LoadProposer get pubkey is nil or empty")
		return nil, errors.New("LoadProposer get pubkey is nil or empty")
	}

	des := &Validator{}
	des.Address = source.GetAddress()
	pub := source.GetPubKey()
	if pub == nil {
		bilog.Error("LoadProposer get pubkey is nil")
	} else {
		des.PubKey = pub
	}
	des.VotingPower = source.VotingPower
	des.Accum = source.Accum
	return des, nil
}

func CreateBlockInfoTx(pubkey string, lastCommit *types.TendermintCommit, seenCommit *types.TendermintCommit, state *types.State, proposal *types.Proposal) *types.Transaction {
	proposalNoTxs := *proposal
	proposalNoTxs.Block.Txs = make([]*types.Transaction, 0)
	blockInfo := &types.TendermintBlockInfo{
		SeenCommit: seenCommit,
		LastCommit: lastCommit,
		State:      state,
		Proposal:   &proposalNoTxs,
	}
	bilog.Debug("CreateBlockInfoTx", "validators", blockInfo.State.Validators.Validators, "proposal", proposal, "proposal-notxs", proposalNoTxs)

	nput := &types.NormAction_Nput{&types.NormPut{Key: "BlockInfo", Value: types.Encode(blockInfo)}}
	action := &types.NormAction{Value: nput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(action), Fee: fee}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(pubkey))

	return tx
}
