package drivers

import (
	"fmt"

	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	TxIndexFrom = 1
	TxIndexTo   = 2
)

//用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
//地址下面所有的交易
func CalcTxAddrHashKey(addr string, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrHash:%s:%s", addr, heightindex))
}

//用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
//地址下面某个分类的交易
func CalcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrDirHash:%s:%d:%s", addr, flag, heightindex))
}

//存储地址参与的交易数量。add时加一，del时减一
func calcAddrTxsCountKey(addr string) []byte {
	return []byte(fmt.Sprintf("AddrTxsCount:%s", addr))
}

func getAddrTxsCountKV(addr string, count int64) *types.KeyValue {
	counts := &types.Int64{Data: count}
	countbytes := types.Encode(counts)
	kv := &types.KeyValue{Key: calcAddrTxsCountKey(addr), Value: countbytes}
	return kv
}

func getAddrTxsCount(db dbm.KVDB, addr string) (int64, error) {
	count := types.Int64{}
	TxsCount, err := db.Get(calcAddrTxsCountKey(addr))
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
	blockchaindbver := getBlockChainDbVersion(cachedb)
	if blockchaindbver == 0 {
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

func getBlockChainDbVersion(db dbm.KVDB) int64 {
	ver := types.Int64{}
	version, err := db.Get(version.BlockChainVerKey)
	if err != nil && err != types.ErrNotFound {
		return 0
	}
	if len(version) == 0 {
		return 0
	}
	err = types.Decode(version, &ver)
	if err != nil {
		return 0
	}
	return ver.Data
}
