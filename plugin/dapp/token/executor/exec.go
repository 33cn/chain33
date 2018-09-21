package executor

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *token) Exec_TokenPreCreate(payload *types.TokenPreCreate, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := newTokenAction(t, "", tx)
	return action.preCreate(payload)
}

func (t *token) Exec_Tokenfinishcreate(payload *types.TokenFinishCreate, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := newTokenAction(t, types.FundKeyAddr, tx)
	return action.finishCreate(payload)
}

func (t *token) Exec_Tokenrevokecreate(payload *types.TokenRevokeCreate, tx *types.Transaction, index int) (*types.Receipt, error) {
	action := newTokenAction(t, "", tx)
	return action.revokeCreate(payload)
}

func (t *token) Exec_Transfer(payload *types.AssetsTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	var tokenAction types.TokenAction
	err := types.Decode(tx.Payload, &tokenAction)
	if err != nil {
		return nil, err
	}
	if tokenAction.GetTransfer() == nil {
		return nil, types.ErrInputPara
	}
	token := tokenAction.GetTransfer().GetCointoken()
	db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
	if err != nil {
		return nil, err
	}
	return t.ExecTransWithdraw(db, tx, &tokenAction, index)
}

func (t *token) Exec_Withdraw(payload *types.AssetsWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	var tokenAction types.TokenAction
	err := types.Decode(tx.Payload, &tokenAction)
	if err != nil {
		return nil, err
	}
	token := tokenAction.GetWithdraw().GetCointoken()
	db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
	if err != nil {
		return nil, err
	}
	return t.ExecTransWithdraw(db, tx, &tokenAction, index)
}

func (t *token) Exec_Genesis(payload *types.AssetsGenesis, tx *types.Transaction, index int) (*types.Receipt, error) {
	return nil, types.ErrActionNotSupport
}

func (t *token) Exec_TransferToExec(payload *types.AssetsTransferToExec, tx *types.Transaction, index int) (*types.Receipt, error) {
	var tokenAction types.TokenAction
	err := types.Decode(tx.Payload, &tokenAction)
	if err != nil {
		return nil, err
	}
	if tokenAction.GetTransferToExec() == nil {
		return nil, types.ErrInputPara
	}
	token := tokenAction.GetTransferToExec().GetCointoken()
	db, err := account.NewAccountDB(t.GetName(), token, t.GetStateDB())
	if err != nil {
		return nil, err
	}
	return t.ExecTransWithdraw(db, tx, &tokenAction, index)
}
