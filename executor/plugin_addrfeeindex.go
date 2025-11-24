// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"fmt"

	"github.com/33cn/chain33/common"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

const addrFeeIndex = "addrfeeindex"

func init() {
	RegisterPlugin(addrFeeIndex, &addrFeeIndexPlugin{})
}

// 地址相关交易费列表
type addrFeeIndexPlugin struct {
	pluginBase
}

func (p *addrFeeIndexPlugin) CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	return nil, enable, nil
}

func (p *addrFeeIndexPlugin) ExecLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	b := data.Block
	var set types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		receipt := data.Receipts[i]
		txFeeIndex := getTxFeeIndex(executor, tx, receipt, i)
		txFeeInfoByte := types.Encode(txFeeIndex.index)
		if len(txFeeIndex.from) != 0 {
			txFeeKey := types.CalcTxFeeAddrDirHashKey(txFeeIndex.from, drivers.TxIndexFrom, txFeeIndex.heightstr)
			set.KV = append(set.KV, &types.KeyValue{Key: txFeeKey, Value: txFeeInfoByte})
		}

	}
	return set.KV, nil
}

func (p *addrFeeIndexPlugin) ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	b := data.Block
	var set types.LocalDBSet
	for i := 0; i < len(b.Txs); i++ {
		tx := b.Txs[i]
		receipt := data.Receipts[i]
		//del: addr index
		txFeeIndex := getTxFeeIndex(executor, tx, receipt, i)
		if len(txFeeIndex.from) != 0 {
			txFeeKey := types.CalcTxFeeAddrDirHashKey(txFeeIndex.from, drivers.TxIndexFrom, txFeeIndex.heightstr)
			set.KV = append(set.KV, &types.KeyValue{Key: txFeeKey, Value: nil})
		}
	}
	return set.KV, nil
}

type txFeeIndex struct {
	from      string
	to        string
	heightstr string
	index     *types.AddrTxFeeInfo
}

// 交易中 from fee 的索引
func getTxFeeIndex(executor *executor, tx *types.Transaction, receipt *types.ReceiptData, index int) *txFeeIndex {
	var txFeeIndexInfo txFeeIndex
	var txinf types.AddrTxFeeInfo
	txinf.TxHash = common.ToHex(tx.Hash())
	txinf.Height = executor.height
	txinf.Index = int64(index)
	txinf.Fee = tx.Fee
	txinf.Exec = string(tx.Execer)
	txinf.TxStatus = receipt.Ty
	txinf.FromAddr = tx.From()
	txinf.ToAddr = tx.GetRealToAddr()

	txFeeIndexInfo.index = &txinf
	heightstr := fmt.Sprintf("%018d", executor.height*types.MaxTxsPerBlock+int64(index))
	txFeeIndexInfo.heightstr = heightstr

	txFeeIndexInfo.from = txinf.FromAddr
	txFeeIndexInfo.to = tx.GetRealToAddr()
	return &txFeeIndexInfo
}
