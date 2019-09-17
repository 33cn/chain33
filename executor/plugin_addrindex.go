// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	dbm "github.com/33cn/chain33/common/db"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

func init() {
	RegisterPlugin("addrindex", &addrindexPlugin{})
}

type addrindexPlugin struct {
	pluginBase
}

func (p *addrindexPlugin) CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	return nil, enable, nil
}

func (p *addrindexPlugin) ExecLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	b := data.Block
	var set types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		receipt := data.Receipts[i]
		txindex := getTxIndex(executor, tx, receipt, i)
		txinfobyte := types.Encode(txindex.index)
		if len(txindex.from) != 0 {
			fromkey1 := types.CalcTxAddrDirHashKey(txindex.from, drivers.TxIndexFrom, txindex.heightstr)
			fromkey2 := types.CalcTxAddrHashKey(txindex.from, txindex.heightstr)
			set.KV = append(set.KV, &types.KeyValue{Key: fromkey1, Value: txinfobyte})
			set.KV = append(set.KV, &types.KeyValue{Key: fromkey2, Value: txinfobyte})
			kv, err := updateAddrTxsCount(executor.localDB, txindex.from, 1, true)
			if err == nil && kv != nil {
				set.KV = append(set.KV, kv)
			}
		}
		if len(txindex.to) != 0 {
			tokey1 := types.CalcTxAddrDirHashKey(txindex.to, drivers.TxIndexTo, txindex.heightstr)
			tokey2 := types.CalcTxAddrHashKey(txindex.to, txindex.heightstr)
			set.KV = append(set.KV, &types.KeyValue{Key: tokey1, Value: txinfobyte})
			set.KV = append(set.KV, &types.KeyValue{Key: tokey2, Value: txinfobyte})
			kv, err := updateAddrTxsCount(executor.localDB, txindex.to, 1, true)
			if err == nil && kv != nil {
				set.KV = append(set.KV, kv)
			}
		}
	}
	return set.KV, nil
}

func (p *addrindexPlugin) ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	b := data.Block
	var set types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		receipt := data.Receipts[i]
		//del: addr index
		txindex := getTxIndex(executor, tx, receipt, i)
		if len(txindex.from) != 0 {
			fromkey1 := types.CalcTxAddrDirHashKey(txindex.from, drivers.TxIndexFrom, txindex.heightstr)
			fromkey2 := types.CalcTxAddrHashKey(txindex.from, txindex.heightstr)
			set.KV = append(set.KV, &types.KeyValue{Key: fromkey1, Value: nil})
			set.KV = append(set.KV, &types.KeyValue{Key: fromkey2, Value: nil})
			kv, err := updateAddrTxsCount(executor.localDB, txindex.from, 1, false)
			if err == nil && kv != nil {
				set.KV = append(set.KV, kv)
			}
		}
		if len(txindex.to) != 0 {
			tokey1 := types.CalcTxAddrDirHashKey(txindex.to, drivers.TxIndexTo, txindex.heightstr)
			tokey2 := types.CalcTxAddrHashKey(txindex.to, txindex.heightstr)
			set.KV = append(set.KV, &types.KeyValue{Key: tokey1, Value: nil})
			set.KV = append(set.KV, &types.KeyValue{Key: tokey2, Value: nil})
			kv, err := updateAddrTxsCount(executor.localDB, txindex.to, 1, false)
			if err == nil && kv != nil {
				set.KV = append(set.KV, kv)
			}
		}
	}
	return set.KV, nil
}

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
	ver := types.GInt("dbversion")
	if ver == 0 {
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
	err = setAddrTxsCount(cachedb, addr, txscount)
	if err != nil {
		return nil, err
	}
	//keyvalue
	return getAddrTxsCountKV(addr, txscount), nil
}
