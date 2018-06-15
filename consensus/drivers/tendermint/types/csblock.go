package types

import (
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"fmt"
)

var bslog = log.New("module", "tendermint-blockstore")

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

func (bs *BlockStore) LoadSeenCommit(height int64) *Commit {
	oldBlock, err := bs.client.RequestBlock(height)
	if err != nil {
		bslog.Error("LoadSeenCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(oldBlock)
	if err != nil {
		panic(fmt.Sprintf("LoadSeenCommit GetBlockInfo failed:%v",err))
	}
	if blockInfo == nil {
		bslog.Error("LoadSeenCommit get nil block info")
		return nil
	}
	seenCommit := blockInfo.GetSeenCommit()
	if seenCommit != nil {
		votesCopy := make([]*Vote, len(seenCommit.GetPrecommits()))
		LoadVotes(votesCopy, seenCommit.GetPrecommits())
		if seenCommit.GetBlockID() != nil {
			return &Commit{
				BlockID: BlockID{
					Hash: seenCommit.BlockID.Hash,
				},
				Precommits: votesCopy,
			}
		}
	}

	return nil
}

func (bs *BlockStore) LoadBlockCommit(height int64) *Commit {
	oldBlock, err := bs.client.RequestBlock(height)
	if err != nil {
		bslog.Error("LoadBlockCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo, err := GetBlockInfo(oldBlock)
	if err != nil {
		panic(fmt.Sprintf("LoadBlockCommit GetBlockInfo failed:%v",err))
	}
	if blockInfo == nil {
		bslog.Error("LoadBlockCommit get nil block info")
		return nil
	}
	lastCommit := blockInfo.GetLastCommit()
	if lastCommit != nil {
		votesCopy := make([]*Vote, len(lastCommit.GetPrecommits()))
		LoadVotes(votesCopy, lastCommit.GetPrecommits())
		if lastCommit.GetBlockID() != nil {
			return &Commit{
				BlockID: BlockID{
					Hash: lastCommit.BlockID.Hash,
				},
				Precommits: votesCopy,
			}
		}
	}
	return nil
}

func (bs *BlockStore) Height() int64 {
	return bs.client.GetCurrentHeight()
}

func (bs *BlockStore) GetPubkey() string {
	return bs.pubkey
}