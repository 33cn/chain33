package executor

import (
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	TxIndexFrom = 1
	TxIndexTo   = 2
)

func getAddrTxsCountKV(addr string, count int64) *types.KeyValue {
	counts := &types.Int64{Data: count}
	countbytes := types.Encode(counts)
	kv := &types.KeyValue{Key: types.CalcAddrTxsCountKey(addr), Value: countbytes}
	return kv
}

func getAddrTxsCount(db dbm.KVDB, addr string) (int64, error) {
	count := types.Int64{}
	TxsCount, err := db.Get(types.CalcAddrTxsCountKey(addr))
	if err != nil && err != types.ErrNotFound {
		return 0, err
	}
	if len(TxsCount) == 0 {
		return 0, nil
	}
	err = types.Decode(TxsCount, &count)
	if err != nil {
		return 0, err
	}
	return count.Data, nil
}

func setAddrTxsCount(db dbm.KVDB, addr string, count int64) error {
	kv := getAddrTxsCountKV(addr, count)
	return db.Set(kv.Key, kv.Value)
}

func updateAddrTxsCount(cachedb dbm.KVDB, addr string, amount int64, isadd bool) (*types.KeyValue, error) {
	//blockchaindb 数据库0版本不支持此功能
	ver, err := types.GetChainConfig("dbversion")
	if err != nil {
		return nil, err
	}
	if ver.(int64) == 0 {
		return nil, types.ErrNotFound
	}
	txscount, err := getAddrTxsCount(cachedb, addr)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	if isadd {
		txscount += amount
	} else {
		txscount -= amount
	}
	setAddrTxsCount(cachedb, addr, txscount)
	//keyvalue
	return getAddrTxsCountKV(addr, txscount), nil
}
