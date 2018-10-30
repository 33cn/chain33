package executor

import (
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func getTitle(db dbm.KV, key []byte) (*pt.ParacrossStatus, error) {
	val, err := db.Get(key)
	if err != nil {
		if !isNotFound(err) {
			return nil, err
		}
		// 平行链如果是从其他链上移过来的，  需要增加配置， 对应title的平行链的起始高度
		clog.Info("first time load title", "key", string(key))
		return &pt.ParacrossStatus{Height: -1}, nil
	}

	var title pt.ParacrossStatus
	err = types.Decode(val, &title)
	return &title, err
}

func saveTitle(db dbm.KV, key []byte, title *pt.ParacrossStatus) error {
	val := types.Encode(title)
	return db.Set(key, val)
}

func getTitleHeight(db dbm.KV, key []byte) (*pt.ParacrossHeightStatus, error) {
	val, err := db.Get(key)
	if err != nil {
		// 对应高度第一次提交commit
		if isNotFound(err) {
			clog.Info("paracross.Commit first commit", "key", string(key))
		}
		return nil, err
	}
	var heightStatus pt.ParacrossHeightStatus
	err = types.Decode(val, &heightStatus)
	return &heightStatus, err
}

func saveTitleHeight(db dbm.KV, key []byte, heightStatus types.Message /* heightStatus *types.ParacrossHeightStatus*/) error {
	// use as a types.Message
	val := types.Encode(heightStatus)
	return db.Set(key, val)
}

func GetBlock(api client.QueueProtocolAPI, blockHash []byte) (*types.BlockDetail, error) {
	blockDetails, err := api.GetBlockByHashes(&types.ReqHashes{[][]byte{blockHash}})
	if err != nil {
		clog.Error("paracross.Commit getBlockHeader", "db", err,
			"commit tx hash", common.Bytes2Hex(blockHash))
		return nil, err
	}
	if len(blockDetails.Items) != 1 {
		clog.Error("paracross.Commit getBlockHeader", "len in not 1", len(blockDetails.Items))
		return nil, pt.ErrParaBlockHashNoMatch
	}
	if blockDetails.Items[0] == nil {
		clog.Error("paracross.Commit getBlockHeader", "commit tx hash net found", common.Bytes2Hex(blockHash))
		return nil, pt.ErrParaBlockHashNoMatch
	}
	return blockDetails.Items[0], nil
}

func getBlockHash(api client.QueueProtocolAPI, height int64) (*types.ReplyHash, error) {
	hash, err := api.GetBlockHash(&types.ReqInt{Height: height})
	if err != nil {
		clog.Error("paracross.Commit getBlockHeader", "db", err,
			"commit height", height)
		return nil, err
	}
	return hash, nil
}

func isNotFound(err error) bool {
	if err != nil && (err == dbm.ErrNotFoundInDb || err == types.ErrNotFound) {
		return true
	}
	return false
}

func GetTx(api client.QueueProtocolAPI, txHash []byte) (*types.TransactionDetail, error) {
	txs, err := api.GetTransactionByHash(&types.ReqHashes{[][]byte{txHash}})
	if err != nil {
		clog.Error("paracross.Commit GetTx", "db", err,
			"commit tx hash", common.Bytes2Hex(txHash))
		return nil, err
	}
	if len(txs.Txs) != 1 {
		clog.Error("paracross.Commit GetTx", "len in not 1", len(txs.Txs))
		return nil, pt.ErrParaBlockHashNoMatch
	}
	if txs.Txs == nil {
		clog.Error("paracross.Commit GetTx", "commit tx hash net found", common.Bytes2Hex(txHash))
		return nil, pt.ErrParaBlockHashNoMatch
	}
	return txs.Txs[0], nil
}
