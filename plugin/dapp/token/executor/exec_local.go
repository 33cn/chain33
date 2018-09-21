package executor

import "gitlab.33.cn/chain33/chain33/types"

func (t *token) execLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var action types.TokenAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var set *types.LocalDBSet
	if action.Ty == types.ActionTransfer || action.Ty == types.ActionWithdraw {
		set, err = t.ExecLocalTransWithdraw(tx, receipt, index)
		if err != nil {
			return nil, err
		}

		if action.Ty == types.ActionTransfer && action.GetTransfer() != nil {
			transfer := action.GetTransfer()
			// 添加个人资产列表
			//tokenlog.Info("ExecLocalTransWithdraw", "addr", tx.GetRealToAddr(), "asset", transfer.Cointoken)
			kv := AddTokenToAssets(tx.GetRealToAddr(), t.GetLocalDB(), transfer.Cointoken)
			if kv != nil {
				set.KV = append(set.KV, kv...)
			}
		}
	} else {
		set, err = t.DriverBase.ExecLocal(tx, receipt, index)
		if err != nil {
			return nil, err
		}
		if receipt.GetTy() != types.ExecOk {
			return set, nil
		}

		for i := 0; i < len(receipt.Logs); i++ {
			item := receipt.Logs[i]
			if item.Ty == types.TyLogPreCreateToken || item.Ty == types.TyLogFinishCreateToken || item.Ty == types.TyLogRevokeCreateToken {
				var receipt types.ReceiptToken
				err := types.Decode(item.Log, &receipt)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}

				receiptKV := t.saveLogs(&receipt)
				set.KV = append(set.KV, receiptKV...)

				// 添加个人资产列表
				if item.Ty == types.TyLogFinishCreateToken && action.GetTokenfinishcreate() != nil {
					kv := AddTokenToAssets(action.GetTokenfinishcreate().Owner, t.GetLocalDB(), action.GetTokenfinishcreate().Symbol)
					if kv != nil {
						set.KV = append(set.KV, kv...)
					}
				}
			}
		}
	}
	return set, nil
}

func (t *token) ExecLocal_TokenPreCreate(payload *types.TokenPreCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(tx, receiptData, index)
}

func (t *token) ExecLocal_Tokenfinishcreat(payload *types.TokenFinishCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(tx, receiptData, index)
}

func (t *token) ExecLocal_Tokenrevokecreate(payload *types.TokenRevokeCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(tx, receiptData, index)
}

func (t *token) ExecLocal_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(tx, receiptData, index)
}

func (t *token) ExecLocal_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(tx, receiptData, index)
}

func (t *token) ExecLocal_Genesis(payload *types.AssetsGenesis, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(tx, receiptData, index)
}

func (t *token) ExecLocal_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(tx, receiptData, index)
}
