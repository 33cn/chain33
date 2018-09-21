package executor

import "gitlab.33.cn/chain33/chain33/types"

func (t *token) execDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var action types.TokenAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var set *types.LocalDBSet
	if action.Ty == types.ActionTransfer || action.Ty == types.ActionWithdraw {
		set, err = t.ExecDelLocalLocalTransWithdraw(tx, receipt, index)
		if err != nil {
			return nil, err
		}
		if types.GetSaveTokenTxList() {
			kvs, err := t.makeTokenTxKvs(tx, &action, receipt, index, true)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kvs...)
		}
	} else {
		set, err = t.DriverBase.ExecDelLocal(tx, receipt, index)
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
					tokenlog.Error("Failed to decode ReceiptToken in ExecDelLocal")
					continue
				}
				set.KV = append(set.KV, t.deleteLogs(&receipt)...)
			}
		}
	}

	return set, nil
}

func (t *token) ExecDelLocal_Tokenprecreate(payload *types.TokenPreCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execDelLocal(tx, receiptData, index)
}

func (t *token) ExecDelLocal_Tokenfinishcreate(payload *types.TokenFinishCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execDelLocal(tx, receiptData, index)
}

func (t *token) ExecDelLocal_Tokenrevokecreate(payload *types.TokenRevokeCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execDelLocal(tx, receiptData, index)
}

func (t *token) ExecDelLocal_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execDelLocal(tx, receiptData, index)
}

func (t *token) ExecDelLocal_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execDelLocal(tx, receiptData, index)
}

func (t *token) ExecDelLocal_Genesis(payload *types.AssetsGenesis, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execDelLocal(tx, receiptData, index)
}

func (t *token) ExecDelLocal_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execDelLocal(tx, receiptData, index)
}
