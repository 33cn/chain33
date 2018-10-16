package executor

import (
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *token) execLocal(receiptData *types.ReceiptData, addr, symbol string) ([]*types.KeyValue, error) {
	var set []*types.KeyValue
	for i := 0; i < len(receiptData.Logs); i++ {
		item := receiptData.Logs[i]
		if item.Ty == tokenty.TyLogPreCreateToken || item.Ty == tokenty.TyLogFinishCreateToken || item.Ty == tokenty.TyLogRevokeCreateToken {
			var receipt tokenty.ReceiptToken
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}

			receiptKV := t.saveLogs(&receipt)
			set = append(set, receiptKV...)

			// 添加个人资产列表
			if item.Ty == tokenty.TyLogFinishCreateToken {
				kv := AddTokenToAssets(addr, t.GetLocalDB(), symbol)
				if kv != nil {
					set = append(set, kv...)
				}
			}
		}
	}
	return set, nil
}

func (t *token) ExecLocal_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.ExecLocalTransWithdraw(tx, receiptData, index)
	if err != nil {
		return nil, err
	}
	// 添加个人资产列表
	//tokenlog.Info("ExecLocalTransWithdraw", "addr", tx.GetRealToAddr(), "asset", transfer.Cointoken)
	kv := AddTokenToAssets(tx.GetRealToAddr(), t.GetLocalDB(), payload.Cointoken)
	if kv != nil {
		set.KV = append(set.KV, kv...)
	}
	if types.GetSaveTokenTxList() {
		tokenAction := tokenty.TokenAction{
			Ty: tokenty.ActionTransfer,
			Value: &tokenty.TokenAction_Transfer{
				payload,
			},
		}
		kvs, err := t.makeTokenTxKvs(tx, &tokenAction, receiptData, index, false)
		if err != nil {
			return nil, err
		}
		set.KV = append(set.KV, kvs...)
	}
	return set, nil
}

func (t *token) ExecLocal_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.ExecLocalTransWithdraw(tx, receiptData, index)
	if err != nil {
		return nil, err
	}
	// 添加个人资产列表
	kv := AddTokenToAssets(tx.From(), t.GetLocalDB(), payload.Cointoken)
	if kv != nil {
		set.KV = append(set.KV, kv...)
	}
	if types.GetSaveTokenTxList() {
		tokenAction := tokenty.TokenAction{
			Ty: tokenty.ActionWithdraw,
			Value: &tokenty.TokenAction_Withdraw{
				payload,
			},
		}
		kvs, err := t.makeTokenTxKvs(tx, &tokenAction, receiptData, index, false)
		if err != nil {
			return nil, err
		}
		set.KV = append(set.KV, kvs...)
	}
	return set, nil
}

func (t *token) ExecLocal_Tokenprecreate(payload *tokenty.TokenPreCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := t.execLocal(receiptData, "", "")
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}

func (t *token) ExecLocal_Tokenfinishcreate(payload *tokenty.TokenFinishCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := t.execLocal(receiptData, payload.Owner, payload.Symbol)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}

func (t *token) ExecLocal_Tokenrevokecreate(payload *tokenty.TokenRevokeCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := t.execLocal(receiptData, "", "")
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}

func (t *token) ExecLocal_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := t.execLocal(receiptData, "", "")
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}
