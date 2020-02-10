// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txindex

import (
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
)

var elog = log.New("module", "system/plugin/txindex")

func init() {
	plugin.RegisterPlugin("txindex", &txindexPlugin{})
}

type txindexPlugin struct {
	plugin.Base
}

func (p *txindexPlugin) CheckEnable(enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	return nil, true, nil
}

func (p *txindexPlugin) ExecLocal(data *types.BlockDetail) (kvs []*types.KeyValue, err error) {
	for i := 0; i < len(data.Block.Txs); i++ {
		tx := data.Block.Txs[i]
		receipt := data.Receipts[i]
		kv := getTx(p, tx, receipt, i)
		kvs = append(kvs, kv...)
	}
	return kvs, nil
}

func (p *txindexPlugin) ExecDelLocal(data *types.BlockDetail) (kvs []*types.KeyValue, err error) {
	for i := 0; i < len(data.Block.Txs); i++ {
		tx := data.Block.Txs[i]
		receipt := data.Receipts[i]
		//del：tx
		kvdel := getTx(p, tx, receipt, i)
		for k := range kvdel {
			kvdel[k].Value = nil
		}
		kvs = append(kvs, kvdel...)
	}
	return kvs, nil
}

//获取公共的信息
func getTx(p plugin.Plugin, tx *types.Transaction, receipt *types.ReceiptData, index int) []*types.KeyValue {
	api := p.GetAPI()
	types.AssertConfig(api)
	cfg := api.GetConfig()
	txhash := tx.Hash()
	//构造txresult 信息保存到db中
	var txresult types.TxResult
	txresult.Height = p.GetHeight()
	txresult.Index = int32(index)
	txresult.Tx = tx
	txresult.Receiptdate = receipt
	txresult.Blocktime = p.GetBlockTime()
	txresult.ActionName = tx.ActionName()
	var kvlist []*types.KeyValue
	kvlist = append(kvlist, &types.KeyValue{Key: cfg.CalcTxKey(txhash), Value: cfg.CalcTxKeyValue(&txresult)})
	if cfg.IsEnable("quickIndex") {
		kvlist = append(kvlist, &types.KeyValue{Key: types.CalcTxShortKey(txhash), Value: []byte("1")})
	}
	return kvlist
}
