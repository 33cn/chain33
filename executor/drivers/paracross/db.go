package paracross

import (
	"gitlab.33.cn/chain33/chain33/blockchain"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

func getTitle(db dbm.KV, key []byte) (*types.ParacrossStatus, error) {
	val, err := db.Get(key)
	if err != nil {
		if err != dbm.ErrNotFoundInDb {
			return nil, err
		}
		// 平行链如果是从其他链上移过来的，  需要增加配置， 对应title的平行链的起始高度
		clog.Info("first time load title", string(key))
		return &types.ParacrossStatus{Height: -1}, nil
	}

	var title types.ParacrossStatus
	err = types.Decode(val, &title)
	return &title, err
}

func saveTitle(db dbm.KV, key []byte, title *types.ParacrossStatus) error {
	val := types.Encode(title)
	return db.Set(key, val)
}

func getTitleHeight(db dbm.KV, key []byte) (*types.ParacrossHeightStatus, error) {
	val, err := db.Get(key)
	if err != nil {
		// 对应高度第一次提交commit
		if err == dbm.ErrNotFoundInDb {
			clog.Info("paracross.Commit first commit", "key", string(key))
		}
		return nil, err
	}
	var heightStatus types.ParacrossHeightStatus
	err = types.Decode(val, &heightStatus)
	return &heightStatus, err
}

func saveTitleHeight(db dbm.KV, key []byte, heightStatus types.Message /* heightStatus *types.ParacrossHeightStatus*/) error {
	// use as a types.Message
	val := types.Encode(heightStatus)
	return db.Set(key, val)
}

func GetBlockBody(db dbm.KVDB, blockHash []byte) (*types.BlockBody, error) {
	data, err := db.Get(blockchain.CalcHashToBlockBodyKey(blockHash))
	if err != nil {
		return nil, err
	}
	var block types.BlockBody
	err = types.Decode(data, &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func getBlockHeader(db dbm.KVDB, blockHash []byte) (*types.Header, error) {
	data, err := db.Get(blockchain.CalcHashToBlockHeaderKey(blockHash))
	if err != nil {
		return nil, err
	}
	var block types.Header
	err = types.Decode(data, &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}
