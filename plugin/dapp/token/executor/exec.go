package executor

import (
	"gitlab.33.cn/chain33/chain33/account"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *token) Exec_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	token := payload.GetCointoken()
	db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
	if err != nil {
		return nil, err
	}
	tokenAction := tokenty.TokenAction{
		Ty: tokenty.ActionTransfer,
		Value: &tokenty.TokenAction_Transfer{
			payload,
		},
	}
	return t.ExecTransWithdraw(db, tx, &tokenAction, index)
}

func (t *token) Exec_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	token := payload.GetCointoken()
	db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
	if err != nil {
		return nil, err
	}
	tokenAction := tokenty.TokenAction{
		Ty: tokenty.ActionWithdraw,
		Value: &tokenty.TokenAction_Withdraw{
			payload,
		},
	}
	return t.ExecTransWithdraw(db, tx, &tokenAction, index)
}

func (t *token) Exec_TokenPreCreate(payload *tokenty.TokenPreCreate, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := newTokenAction(t, "", tx)
	return action.preCreate(payload)
}

func (t *token) Exec_TokenFinishCreate(payload *tokenty.TokenFinishCreate, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := newTokenAction(t, types.GetFundAddr(), tx)
	return action.finishCreate(payload)
}

func (t *token) Exec_TokenRevokeCreate(payload *tokenty.TokenRevokeCreate, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := newTokenAction(t, "", tx)
	return action.revokeCreate(payload)
}

func (t *token) Exec_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, index int) (*types.Receipt, error) {
	token := payload.GetCointoken()
	db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
	if err != nil {
		return nil, err
	}
	tokenAction := tokenty.TokenAction{
		Ty: tokenty.TokenActionTransferToExec,
		Value: &tokenty.TokenAction_TransferToExec{
			payload,
		},
	}
	return t.ExecTransWithdraw(db, tx, &tokenAction, index)
}
