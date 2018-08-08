package types

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/merkle"
)

var bslog = log15.New("module", "tendermint-blockstore")

const fee = 1e6

var r *rand.Rand

//------------------------------------------------------------------------------
type BlockStore struct {
	client *drivers.BaseClient
	pubkey string
	//LoadSeenCommit(height int64) *Commit
	//LoadBlockCommit(height int64) *Commit
	//Height() int64
	//GetCfg() *gtypes.Consensus
	//CreateCommitTx(lastCommit *Commit, seenCommit *Commit) *gtypes.Transaction
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
		bslog.Error("LoadSeenCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(oldBlock)
	if err != nil {
		panic(fmt.Sprintf("LoadSeenCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		bslog.Error("LoadSeenCommit get nil block info")
		return nil
	}
	return blockInfo.GetSeenCommit()
}

func (bs *BlockStore) LoadBlockCommit(height int64) *types.TendermintCommit {
	oldBlock, err := bs.client.RequestBlock(height)
	if err != nil {
		bslog.Error("LoadBlockCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(oldBlock)
	if err != nil {
		panic(fmt.Sprintf("LoadBlockCommit GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		bslog.Error("LoadBlockCommit get nil block info")
		return nil
	}
	return blockInfo.GetLastCommit()
}

func (bs *BlockStore) LoadProposal(height int64) *types.Proposal {
	block, err := bs.client.RequestBlock(height)
	if err != nil {
		bslog.Error("LoadProposal by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(block)
	if err != nil {
		panic(fmt.Sprintf("LoadProposal GetBlockInfo failed:%v", err))
	}
	if blockInfo == nil {
		bslog.Error("LoadProposal get nil block info")
		return nil
	}
	proposal := blockInfo.GetProposal()
	if proposal != nil {
		proposal.Block.Txs = append(proposal.Block.Txs, block.Txs[1:]...)
		txHash := merkle.CalcMerkleRoot(proposal.Block.Txs)
		bslog.Info("LoadProposal get hash of txs of proposal", "height", proposal.Block.Header.Height, "hash", txHash)
	}
	return proposal
	//blockByte := proposalTrans.BlockBytes
	//var propBlock gtypes.Block
	//err = proto.Unmarshal(blockByte, &propBlock)
	//	if err != nil {
	//		panic(fmt.Sprintf("LoadProposal Unmarshal failed:%v", err))
	//	}
	//	propBlock.Txs = block.Txs[1:]
	//	propBlockByte, _ := proto.Marshal(&propBlock)
	//	proposalTrans.BlockBytes = propBlockByte
}

func (bs *BlockStore) Height() int64 {
	return bs.client.GetCurrentHeight()
}

func (bs *BlockStore) GetPubkey() string {
	return bs.pubkey
}
