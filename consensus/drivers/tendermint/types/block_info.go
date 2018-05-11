package types

import (
	"time"

	log "github.com/inconshreveable/log15"
	crypto "github.com/tendermint/go-crypto"
	gtypes "gitlab.33.cn/chain33/chain33/types"
	gcrypto "gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common"
	"fmt"
	"errors"
)

var bilog = log.New("module", "tendermint-blockinfo")

func GetBlockInfo(block *gtypes.Block) (*gtypes.TendermintBlockInfo, error) {
	if len(block.Txs) == 0 || block.Height == 0 {
		return nil, nil
	}
	baseTx := block.Txs[0]
	//判断交易类型和执行情况
	var blockInfo gtypes.TendermintBlockInfo
	nGet := &gtypes.NormPut{}
	action := &gtypes.NormAction{}
	err := gtypes.Decode(baseTx.GetPayload(), action)
	if err != nil {
		bilog.Error("GetBlockInfo decode payload failed", "error", err)
		return nil, errors.New(fmt.Sprintf("GetBlockInfo decode payload failed:%v",err))
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
	err = gtypes.Decode(infobytes, &blockInfo)
	if err != nil {
		bilog.Error("GetBlockInfo decode blockinfo failed", "error", err)
		return nil, errors.New(fmt.Sprintf("GetBlockInfo decode blockinfo failed:%v", err))
	}
	return &blockInfo, nil
}

func LoadVotes(des []*Vote, source []*gtypes.Vote) {
	for i, item := range source {
		if item.Height == -1 {
			continue
		}
		des[i] = &Vote{}
		des[i].BlockID = BlockID{
			Hash: item.BlockID.Hash,
			PartsHeader: PartSetHeader{
				Total: int(item.BlockID.PartsHeader.Total),
				Hash:  item.BlockID.PartsHeader.Hash},
		}
		des[i].Height = item.Height
		des[i].Round = int(item.Round)
		sig, err := crypto.SignatureFromBytes(item.Signature)
		if err != nil {
			bilog.Error("SignatureFromBytes failed", "err", err)
		} else {
			des[i].Signature = sig.Wrap()
		}

		des[i].Type = uint8(item.Type)
		des[i].ValidatorAddress = item.ValidatorAddress
		des[i].ValidatorIndex = int(item.ValidatorIndex)
		des[i].Timestamp = time.Unix(0, item.Timestamp)
		bilog.Info("load votes", "i", i, "source", item, "des", des[i])
	}
}

func SaveVotes(des []*gtypes.Vote, source []*Vote) {
	for i, item := range source {

		if item == nil {
			des[i] = &gtypes.Vote{Height:-1, BlockID:&gtypes.BlockID{PartsHeader:&gtypes.PartSetHeader{}}}
			bilog.Info("SaveVotes-item=nil")
			continue
		}

		des[i] = &gtypes.Vote{}
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
		bilog.Info("save votes", "i", i, "source", item, "des", des[i])
	}

}

func SaveCommits(lastCommitVotes *Commit, seenCommitVotes *Commit) (*gtypes.TendermintCommit,*gtypes.TendermintCommit) {
	newLastCommitVotes := make([]*gtypes.Vote, len(lastCommitVotes.Precommits))
	newSeenCommitVotes := make([]*gtypes.Vote, len(seenCommitVotes.Precommits))
	if len(lastCommitVotes.Precommits) > 0 {
		bilog.Info("SaveCommits","lastCommitVotes",lastCommitVotes.StringIndented("last"))
		SaveVotes(newLastCommitVotes, lastCommitVotes.Precommits)
	}
	if len(seenCommitVotes.Precommits) > 0 {
		bilog.Info("SaveCommits","seenCommitVotes",seenCommitVotes.StringIndented("seen"))
		SaveVotes(newSeenCommitVotes, seenCommitVotes.Precommits)
	}
	lastCommit := &gtypes.TendermintCommit{
		BlockID: &gtypes.BlockID{
			Hash: lastCommitVotes.BlockID.Hash,
			PartsHeader: &gtypes.PartSetHeader{
				Total: int32(lastCommitVotes.BlockID.PartsHeader.Total),
				Hash:  lastCommitVotes.BlockID.PartsHeader.Hash,
			},
		},
		Precommits: newLastCommitVotes,
	}
	seenCommit := &gtypes.TendermintCommit{
		BlockID: &gtypes.BlockID{
			Hash: seenCommitVotes.BlockID.Hash,
			PartsHeader: &gtypes.PartSetHeader{
				Total: int32(seenCommitVotes.BlockID.PartsHeader.Total),
				Hash:  seenCommitVotes.BlockID.PartsHeader.Hash,
			},
		},
		Precommits: newSeenCommitVotes,
	}

	return seenCommit, lastCommit
}

func getprivkey(key string) gcrypto.PrivKey {
	cr, err := gcrypto.New(gtypes.GetSignatureTypeName(gtypes.SECP256K1))
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


func LoadValidators(des []*Validator, source []*gtypes.Validator) {
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
		pub, err := crypto.PubKeyFromBytes(item.GetPubKey())
		if err != nil {
			bilog.Error("LoadValidators get pubkey from byte failed", "err", err)
		} else {
			des[i].PubKey = pub.Wrap()
		}
		des[i].VotingPower = item.VotingPower
		des[i].Accum = item.Accum
	}
}

func LoadProposer(des *Validator, source *gtypes.Validator) error {
	if source.GetAddress() == nil || len(source.GetAddress()) == 0 {
		bilog.Warn("LoadProposer get address is nil or empty")
		return errors.New("LoadProposer get address is nil or empty")
	} else if source.GetPubKey() == nil || len(source.GetPubKey()) == 0 {
		bilog.Warn("LoadProposer get pubkey is nil or empty")
		return errors.New("LoadProposer get pubkey is nil or empty")
	}
	des = &Validator{}
	des.Address = source.GetAddress()
	pub, err := crypto.PubKeyFromBytes(source.GetPubKey())
	if err != nil {
		bilog.Error("LoadProposer get pubkey from byte failed", "err", err)
		return errors.New(fmt.Sprintf("LoadProposer get pubkey from byte failed, err:%v", err))
	} else {
		des.PubKey = pub.Wrap()
	}
	des.VotingPower = source.VotingPower
	des.Accum = source.Accum
	return nil
}


func CreateBlockInfoTx(pubkey string, lastCommit *gtypes.TendermintCommit, seenCommit *gtypes.TendermintCommit, state *gtypes.State) *gtypes.Transaction {
	blockInfo := & gtypes.TendermintBlockInfo{
		SeenCommit:seenCommit,
		LastCommit:lastCommit,
		State:state,
	}

	nput := &gtypes.NormAction_Nput{&gtypes.NormPut{Key: "BlockInfo", Value: gtypes.Encode(blockInfo)}}
	action := &gtypes.NormAction{Value: nput, Ty: gtypes.NormActionPut}
	tx := &gtypes.Transaction{Execer: []byte("norm"), Payload: gtypes.Encode(action), Fee: fee}
	tx.Nonce = r.Int63()
	tx.Sign(gtypes.SECP256K1, getprivkey(pubkey))

	return tx
}