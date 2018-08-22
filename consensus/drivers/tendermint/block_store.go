package tendermint

import (
	"time"
	"fmt"
	"math/rand"
	"errors"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	ttypes "gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint/types"
)

const fee = 1e6

var (
	r *rand.Rand
)

//---------------------------------BlockStore---------------------------------------------
type BlockStore struct {
	client *TendermintClient
	pubkey string
}

func NewBlockStore(client *TendermintClient) *BlockStore {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &BlockStore{
		client: client,
		pubkey: client.pubKey,
	}
}

func (bs *BlockStore) LoadSeenCommit(height int64) *types.TendermintCommit {
	blockInfo, err := bs.client.QueryBlockInfoByHeight(height)
	if err != nil {
		panic(fmt.Sprintf("LoadSeenCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		tendermintlog.Error("LoadSeenCommit get nil block info")
		return nil
	}
	return blockInfo.GetSeenCommit()
}

func (bs *BlockStore) LoadBlockCommit(height int64) *types.TendermintCommit {
	blockInfo, err := bs.client.QueryBlockInfoByHeight(height)
	if err != nil {
		panic(fmt.Sprintf("LoadBlockCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		tendermintlog.Error("LoadBlockCommit get nil block info")
		return nil
	}
	return blockInfo.GetLastCommit()
}

func (bs *BlockStore) LoadProposalBlock(height int64) *types.TendermintBlock {
	block, err := bs.client.RequestBlock(height)
	if err != nil {
		tendermintlog.Error("LoadProposal by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := bs.client.QueryBlockInfoByHeight(height)
	if err != nil {
		panic(fmt.Sprintf("LoadProposal GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		tendermintlog.Error("LoadProposal get nil block info")
		return nil
	}

	proposalBlock := blockInfo.GetBlock()
	if proposalBlock != nil {
		proposalBlock.Txs = append(proposalBlock.Txs, block.Txs[1:]...)
		txHash := merkle.CalcMerkleRoot(proposalBlock.Txs)
		tendermintlog.Info("LoadProposalBlock txs hash", "height", proposalBlock.Header.Height, "tx-hash", fmt.Sprintf("%X", txHash))
	}
	return proposalBlock
}

func (bs *BlockStore) Height() int64 {
	return bs.client.GetCurrentHeight()
}

func (bs *BlockStore) GetPubkey() string {
	return bs.pubkey
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

func LoadValidators(des []*ttypes.Validator, source []*types.Validator) {
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

func LoadProposer(source *types.Validator) (*ttypes.Validator, error) {
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
	tendermintlog.Debug("CreateBlockInfoTx", "validators", blockInfo.State.Validators.Validators, "block", block, "block-notxs", blockNoTxs)

	nput := &types.ValNodeAction_BlockInfo{BlockInfo: blockInfo}
	action := &types.ValNodeAction{Value: nput, Ty: types.ValNodeActionBlockInfo}
	tx := &types.Transaction{Execer: []byte("valnode"), Payload: types.Encode(action), Fee: fee}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, getprivkey(pubkey))

	return tx
}
