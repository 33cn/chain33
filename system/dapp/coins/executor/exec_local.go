package executor

import (
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Coins) ExecLocal_Transfer(transfer *cty.CoinsTransfer, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecLocal_TransferToExec(transfer *cty.CoinsTransferToExec, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecLocal_Withdraw(withdraw *cty.CoinsWithdraw, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	from := tx.From()
	kv, err := updateAddrReciver(c.GetLocalDB(), from, withdraw.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecLocal_Genesis(gen *cty.CoinsGenesis, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), gen.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}
