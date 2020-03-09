// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fee

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/index"
	"github.com/33cn/chain33/types"
)

var (
	name = "fee"
	elog = log.New("module", "system/index/fee")
)

func init() {
	plugin.RegisterPlugin(name, newFee)
}

type feePlugin struct {
	*plugin.Base
}

func newFee() plugin.Plugin {
	fee := &feePlugin{
		Base: &plugin.Base{},
	}
	fee.SetName(name)
	return fee
}

func (p *feePlugin) CheckEnable(enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	return nil, true, nil
}

func (p *feePlugin) ExecLocal(data *types.BlockDetail) ([]*types.KeyValue, error) {
	fee := &types.TotalFee{}
	for i := 0; i < len(data.Block.Txs); i++ {
		tx := data.Block.Txs[i]
		fee.Fee += tx.Fee
		fee.TxCount++
	}
	kv, err := saveFee(p.GetLocalDB(), fee, data.Block.ParentHash, data.Block.Hash(p.GetAPI().GetConfig()))
	if err != nil {
		return nil, err
	}
	return []*types.KeyValue{kv}, err
}

func (p *feePlugin) ExecDelLocal(data *types.BlockDetail) ([]*types.KeyValue, error) {
	kv, err := delFee(p.GetLocalDB(), data.Block.Hash(p.GetAPI().GetConfig()))
	if err != nil {
		return nil, err
	}
	return []*types.KeyValue{kv}, err
}

func saveFee(localdb dbm.KVDB, fee *types.TotalFee, parentHash, hash []byte) (*types.KeyValue, error) {
	totalFee := &types.TotalFee{}
	totalFeeBytes, err := localdb.Get(CalcTotalFeeKey(name, parentHash))
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
	return &types.KeyValue{Key: CalcTotalFeeKey(name, hash), Value: types.Encode(totalFee)}, nil
}

func delFee(localdb dbm.KVDB, hash []byte) (*types.KeyValue, error) {
	return &types.KeyValue{Key: CalcTotalFeeKey(name, hash)}, nil
}

// CalcTotalFeeKey 存储地址参与的交易数量。add时加一，del时减一
func CalcTotalFeeKey(name string, hash []byte) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:%s", types.LocalPluginPrefix, name, "Fee", string(hash)))
}
