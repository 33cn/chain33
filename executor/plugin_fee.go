// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import "github.com/33cn/chain33/types"

func init() {
	RegisterPlugin("fee", &feePlugin{})
}

type feePlugin struct {
	pluginBase
}

func (p *feePlugin) CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	return nil, true, nil
}

func (p *feePlugin) ExecLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	fee := &types.TotalFee{}
	for i := 0; i < len(data.Block.Txs); i++ {
		tx := data.Block.Txs[i]
		fee.Fee += tx.Fee
		fee.TxCount++
	}
	kv, err := saveFee(executor, fee, data.Block.ParentHash, data.Block.Hash())
	if err != nil {
		return nil, err
	}
	return []*types.KeyValue{kv}, err
}

func (p *feePlugin) ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	kv, err := delFee(executor, data.Block.Hash())
	if err != nil {
		return nil, err
	}
	return []*types.KeyValue{kv}, err
}

func saveFee(ex *executor, fee *types.TotalFee, parentHash, hash []byte) (*types.KeyValue, error) {
	totalFee := &types.TotalFee{}
	totalFeeBytes, err := ex.localDB.Get(types.TotalFeeKey(parentHash))
	if err == nil {
		err = types.Decode(totalFeeBytes, totalFee)
		if err != nil {
			return nil, err
		}
	} else if err != types.ErrNotFound {
		return nil, err
	}
	totalFee.Fee += fee.Fee
	totalFee.TxCount += fee.TxCount
	return &types.KeyValue{Key: types.TotalFeeKey(hash), Value: types.Encode(totalFee)}, nil
}

func delFee(ex *executor, hash []byte) (*types.KeyValue, error) {
	return &types.KeyValue{Key: types.TotalFeeKey(hash)}, nil
}
