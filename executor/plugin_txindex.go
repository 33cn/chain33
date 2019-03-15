// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"fmt"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/types"
)

func init() {
	RegisterPlugin("txindex", &txindexPlugin{})
}

type txindexPlugin struct {
	pluginBase
}

func (p *txindexPlugin) CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	return nil, true, nil
}

func (p *txindexPlugin) ExecLocal(executor *executor, data *types.BlockDetail) (kvs []*types.KeyValue, err error) {
	for i := 0; i < len(data.Block.Txs); i++ {
		tx := data.Block.Txs[i]
		receipt := data.Receipts[i]
		kv := getTx(executor, tx, receipt, i)
		kvs = append(kvs, kv...)
	}
	return kvs, nil
}

func (p *txindexPlugin) ExecDelLocal(executor *executor, data *types.BlockDetail) (kvs []*types.KeyValue, err error) {
	for i := 0; i < len(data.Block.Txs); i++ {
		tx := data.Block.Txs[i]
		receipt := data.Receipts[i]
		//del：tx
		kvdel := getTx(executor, tx, receipt, i)
		for k := range kvdel {
			kvdel[k].Value = nil
		}
		kvs = append(kvs, kvdel...)
	}
	return kvs, nil
}

//获取公共的信息
func getTx(executor *executor, tx *types.Transaction, receipt *types.ReceiptData, index int) []*types.KeyValue {
	txhash := tx.Hash()
	//构造txresult 信息保存到db中
	var txresult types.TxResult
	txresult.Height = executor.height
	txresult.Index = int32(index)
	txresult.Tx = tx
	txresult.Receiptdate = receipt
	txresult.Blocktime = executor.blocktime
	txresult.ActionName = tx.ActionName()
	var kvlist []*types.KeyValue
	kvlist = append(kvlist, &types.KeyValue{Key: types.CalcTxKey(txhash), Value: types.Encode(&txresult)})
	if types.IsEnable("quickIndex") {
		kvlist = append(kvlist, &types.KeyValue{Key: types.CalcTxShortKey(txhash), Value: []byte("1")})
	}
	return kvlist
}

type txIndex struct {
	from      string
	to        string
	heightstr string
	index     *types.ReplyTxInfo
}

//交易中 from/to 的索引
func getTxIndex(executor *executor, tx *types.Transaction, receipt *types.ReceiptData, index int) *txIndex {
	var txIndexInfo txIndex
	var txinf types.ReplyTxInfo
	txinf.Hash = tx.Hash()
	txinf.Height = executor.height
	txinf.Index = int64(index)
	ety := types.LoadExecutorType(string(tx.Execer))
	// none exec has not execType
	if ety != nil {
		var err error
		txinf.Assets, err = ety.GetAssets(tx)
		if err != nil {
			elog.Error("getTxIndex ", "GetAssets err", err)
		}
	}

	txIndexInfo.index = &txinf
	heightstr := fmt.Sprintf("%018d", executor.height*types.MaxTxsPerBlock+int64(index))
	txIndexInfo.heightstr = heightstr

	txIndexInfo.from = address.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	txIndexInfo.to = tx.GetRealToAddr()
	return &txIndexInfo
}
