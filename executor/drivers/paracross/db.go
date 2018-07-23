package paracross

import (
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

func getTitle(db dbm.KV, key []byte) (*types.ParacrossStatus, error) {
	val, err := db.Get(key)
	if err != nil {
		if err != types.ErrNotFound {
			return nil, err
		}
		// 平行链如果是从其他链上移过来的，  需要增加配置， 对应title的平行链的起始高度
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
