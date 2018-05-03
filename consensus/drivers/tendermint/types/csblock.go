package types

import (
	gtypes "gitlab.33.cn/chain33/chain33/types"
	crypto "github.com/tendermint/go-crypto"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"time"
	log "github.com/inconshreveable/log15"
	"math/rand"
)
var bslog = log.New("module", "tendermint-blockstore")
const fee = 1e6
var r *rand.Rand
//------------------------------------------------------------------------------
type BlockStore struct {
	client *drivers.BaseClient
	//LoadSeenCommit(height int64) *Commit
	//LoadBlockCommit(height int64) *Commit
	//Height() int64
	//GetCfg() *gtypes.Consensus
	//CreateCommitTx(lastCommit *Commit, seenCommit *Commit) *gtypes.Transaction
}

func NewBlockStore(client *drivers.BaseClient) *BlockStore {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	return &BlockStore {
		client : client,
	}
}
func GetCommitFromBlock(block *gtypes.Block) *gtypes.TendermintBlockInfo {
	if len(block.Txs) == 0 || block.Height == 0{
		return nil
	}
	baseTx := block.Txs[0]
	//判断交易类型和执行情况
	var blockInfo gtypes.TendermintBlockInfo
	nGet := &gtypes.NormPut{}
	action := &gtypes.NormAction{}
	err := gtypes.Decode(baseTx.GetPayload(), action)
	if err != nil {
		bslog.Error("GetCommitFromBlock get payload failed", "error", err)
		return nil
	}
	if nGet = action.GetNput(); nGet == nil {
		bslog.Error("GetCommitFromBlock get nput failed")
		return nil
	}
	infobytes := nGet.GetValue()
	if infobytes == nil {
		bslog.Error("GetCommitFromBlock get blockinfo failed")
		return nil
	}
	err = gtypes.Decode(infobytes, &blockInfo)
	if err != nil {
		bslog.Error("GetCommitFromBlock get payload failed", "error", err)
		return nil
	}
	return &blockInfo
}

func LoadVotes(des []*Vote, source []*gtypes.Vote) {
	for i, item := range source {
		des[i] = &Vote{}
		des[i].BlockID = BlockID{
			Hash:item.BlockID.Hash,
			PartsHeader:PartSetHeader{
				Total:int(item.BlockID.PartsHeader.Total), 
				Hash:item.BlockID.PartsHeader.Hash},
		}
		des[i].Height = item.Height
		des[i].Round = int(item.Round)
		sig ,err := crypto.SignatureFromBytes(item.Signature)
		if err != nil {
			bslog.Error("SignatureFromBytes failed", "err", err)
		} else {
			des[i].Signature = sig.Wrap()
		}

		des[i].Type = uint8(item.Type)
		des[i].ValidatorAddress = item.ValidatorAddress
		des[i].ValidatorIndex = int(item.ValidatorIndex)
		des[i].Timestamp = time.Unix(0, item.Timestamp)
		bslog.Info("load votes", "i", i, "source", item, "des", des[i])
	}
}

func (bs *BlockStore) LoadSeenCommit(height int64) *Commit {
	oldBlock, err := bs.client.RequestBlock(height)
	if err != nil {
		bslog.Error("LoadSeenCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo := GetCommitFromBlock(oldBlock)
	if blockInfo == nil {
		bslog.Error("LoadSeenCommit get nil block info")
		return nil
	}
	seenCommit := blockInfo.GetSeenCommit()
	if seenCommit != nil {
		votesCopy := make([]*Vote, len(seenCommit.GetPrecommits()))
		LoadVotes(votesCopy, seenCommit.GetPrecommits())
		if seenCommit.GetBlockID() != nil && seenCommit.GetBlockID().GetPartsHeader() != nil {
			return &Commit{
				BlockID: BlockID{
					Hash:seenCommit.BlockID.Hash,
					PartsHeader:PartSetHeader{
						Total:int(seenCommit.BlockID.PartsHeader.Total),
						Hash:seenCommit.BlockID.PartsHeader.Hash,
					},
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
		bslog.Error("LoadSeenCommit by height failed", "curHeight", bs.client.GetCurrentHeight(), "requestHeight", height, "error", err)
		return nil
	}
	blockInfo := GetCommitFromBlock(oldBlock)
	if blockInfo == nil {
		bslog.Error("LoadSeenCommit get nil block info")
		return nil
	}
	lastCommit := blockInfo.GetLastCommit()
	if lastCommit != nil {
		votesCopy := make([]*Vote, len(lastCommit.GetPrecommits()))
		LoadVotes(votesCopy, lastCommit.GetPrecommits())
		if lastCommit.GetBlockID() != nil && lastCommit.GetBlockID().GetPartsHeader() != nil{
			return &Commit{
				BlockID: BlockID{
					Hash:lastCommit.BlockID.Hash,
					PartsHeader:PartSetHeader{
						Total:int(lastCommit.BlockID.PartsHeader.Total),
						Hash:lastCommit.BlockID.PartsHeader.Hash,
					},
				},
				Precommits: votesCopy,
			}
		}
	}
	return nil
}

func (bs *BlockStore) CreateCommitTx(lastCommit *Commit, seenCommit *Commit ) *gtypes.Transaction {
	blockInfo := bs.SaveCommits(lastCommit, seenCommit)

	nput := &gtypes.NormAction_Nput{&gtypes.NormPut{Key: "BlockInfo", Value: gtypes.Encode(blockInfo)}}
	action := &gtypes.NormAction{Value: nput, Ty: gtypes.NormActionPut}
	tx := &gtypes.Transaction{Execer: []byte("norm"), Payload: gtypes.Encode(action), Fee: fee}
	tx.Nonce = r.Int63()
	//tx.Sign(gtypes.SECP256K1, getprivkey(privkey))

	return tx
}

func SaveVotes(des []*gtypes.Vote, source []*Vote) {
	lens := len(des)
	if lens > 0 {
		for i, item := range source {
			vote := gtypes.Vote{}
			des[i] = &vote
			partSetHeader := &gtypes.PartSetHeader{}
			partSetHeader.Hash = item.BlockID.PartsHeader.Hash
			partSetHeader.Total = int32(item.BlockID.PartsHeader.Total)
			blockID := &gtypes.BlockID{}
			blockID.Hash = item.BlockID.Hash
			blockID.PartsHeader = partSetHeader
			des[i].BlockID = blockID
			des[i].Height = item.Height
			des[i].Round = int32(item.Round)
			des[i].Signature = item.Signature.Unwrap().Bytes()
			des[i].Type = uint32(item.Type)
			des[i].ValidatorAddress = item.ValidatorAddress
			des[i].ValidatorIndex = int32(item.ValidatorIndex)
			des[i].Timestamp = item.Timestamp.UnixNano()
			bslog.Info("save votes", "i", i, "source",item, "des", des[i])
		}
	}
}

func (bs *BlockStore) SaveCommits(lastCommitVotes *Commit, seenCommitVotes *Commit) *gtypes.TendermintBlockInfo{
	newLastCommitVotes := make([]*gtypes.Vote, len(lastCommitVotes.Precommits))
	newSeenCommitVotes := make([]*gtypes.Vote, len(seenCommitVotes.Precommits))
	if len(lastCommitVotes.Precommits) > 0 {
		SaveVotes(newLastCommitVotes, lastCommitVotes.Precommits)
	}
	if len(seenCommitVotes.Precommits) > 0 {
		SaveVotes(newSeenCommitVotes, seenCommitVotes.Precommits)
	}
	lastCommit := &gtypes.TendermintCommit{
		BlockID:&gtypes.BlockID{
			Hash:lastCommitVotes.BlockID.Hash,
			PartsHeader:&gtypes.PartSetHeader{
				Total:int32(lastCommitVotes.BlockID.PartsHeader.Total), 
				Hash:lastCommitVotes.BlockID.PartsHeader.Hash,
				},
		},
		Precommits:newLastCommitVotes,
	}
	seenCommit := &gtypes.TendermintCommit{
		BlockID:&gtypes.BlockID{
			Hash:seenCommitVotes.BlockID.Hash,
			PartsHeader:&gtypes.PartSetHeader{
				Total:int32(seenCommitVotes.BlockID.PartsHeader.Total), 
				Hash:seenCommitVotes.BlockID.PartsHeader.Hash,
				},
		},
		Precommits:newSeenCommitVotes,
	}
	blockInfo := &gtypes.TendermintBlockInfo{
		SeenCommit:seenCommit,
		LastCommit:lastCommit,
	}
	return blockInfo
}

func (bs *BlockStore) Height() int64 {
	return bs.client.GetCurrentHeight()
}
