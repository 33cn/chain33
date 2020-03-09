// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txindex

import (
	"fmt"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/index"
	"github.com/33cn/chain33/types"
)

var (
	name = "txindex"
	elog = log.New("module", "system/index/txindex")
)

func init() {
	plugin.RegisterPlugin(name, newTxindex)
	plugin.RegisterQuery("HasTx", name)
	plugin.RegisterQuery("GetTx", name)
}

type txindexPlugin struct {
	*plugin.Base
}

func newTxindex() plugin.Plugin {
	p := &txindexPlugin{
		Base: &plugin.Base{},
	}
	p.SetName(name)
	return p
}

func (p *txindexPlugin) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "HasTx" {
		var req types.ReqKey
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return p.HasTx(req.Key)
	} else if funcName == "GetTx" {
		var req types.ReqKey
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return p.GetTx(req.Key)
	}
	return nil, types.ErrQueryNotSupport
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
		elog.Info("txindexPlugin plugin done", "name", i)
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
	kvlist = append(kvlist, &types.KeyValue{Key: CalcTxKey(cfg, name, txhash), Value: cfg.CalcTxKeyValue(&txresult)})
	if cfg.IsEnable("quickIndex") {
		kvlist = append(kvlist, &types.KeyValue{Key: CalcTxShortKey(name, txhash), Value: []byte("1")})
	}
	return kvlist
}

// Query impl

//HasTx 是否包含该交易
func (p *txindexPlugin) HasTx(key []byte) (types.Message, error) {
	api := p.GetAPI()
	types.AssertConfig(api)
	cfg := api.GetConfig()

	if cfg.IsEnable("quickIndex") {
		if _, err := p.GetLocalDB().Get(CalcTxShortKey(name, key)); err != nil {
			if err == types.ErrNotFound {
				return &types.Int32{Data: 0}, nil
			}
			return &types.Int32{Data: 0}, err
		}
		//通过短hash查询交易存在时，需要再通过全hash索引查询一下。
		//避免短hash重复，而全hash不一样的情况
		//return true, nil
	}
	if _, err := p.GetLocalDB().Get(CalcTxKey(cfg, name, key)); err != nil {
		if err == types.ErrNotFound {
			return &types.Int32{Data: 0}, nil
		}
		return &types.Int32{Data: 0}, err
	}
	return &types.Int32{Data: 1}, nil
}

// GetTx 获得交易
func (p *txindexPlugin) GetTx(key []byte) (types.Message, error) {
	api := p.GetAPI()
	types.AssertConfig(api)
	cfg := api.GetConfig()

	result, err := p.GetLocalDB().Get(CalcTxKey(cfg, name, key))
	if err != nil {
		return nil, err
	}
	var r types.TxResult
	err = types.Decode(result, &r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// CalcTxKey Calc Tx key
func CalcTxKey(cfg *types.Chain33Config, name string, txHash []byte) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:%s", types.LocalPluginPrefix, name, "TX", string(txHash)))
}

// CalcTxShortKey Calc Tx Short key
func CalcTxShortKey(name string, txHash []byte) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:%s", types.LocalPluginPrefix, name, "STX", string(txHash[:8])))
}
