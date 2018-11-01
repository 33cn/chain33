package types

import (
	"reflect"

	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(TokenX))
	types.RegistorExecutor(TokenX, NewType())
	types.RegisterDappFork(TokenX, "Enable", 100899)
	types.RegisterDappFork(TokenX, "ForkTokenBlackList", 190000)
	types.RegisterDappFork(TokenX, "ForkBadTokenSymbol", 184000)
	types.RegisterDappFork(TokenX, "ForkTokenPrice", 560000)
}

// exec
type TokenType struct {
	types.ExecTypeBase
}

func NewType() *TokenType {
	c := &TokenType{}
	c.SetChild(c)
	return c
}

func (t *TokenType) GetPayload() types.Message {
	return &TokenAction{}
}

func (t *TokenType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Transfer":          ActionTransfer,
		"Genesis":           ActionGenesis,
		"Withdraw":          ActionWithdraw,
		"TokenPreCreate":    TokenActionPreCreate,
		"TokenFinishCreate": TokenActionFinishCreate,
		"TokenRevokeCreate": TokenActionRevokeCreate,
		"TransferToExec":    TokenActionTransferToExec,
	}
}

func (t *TokenType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{
		TyLogTokenTransfer:        {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogTokenTransfer"},
		TyLogTokenDeposit:         {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogTokenDeposit"},
		TyLogTokenExecTransfer:    {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogTokenExecTransfer"},
		TyLogTokenExecWithdraw:    {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogTokenExecWithdraw"},
		TyLogTokenExecDeposit:     {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogTokenExecDeposit"},
		TyLogTokenExecFrozen:      {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogTokenExecFrozen"},
		TyLogTokenExecActive:      {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogTokenExecActive"},
		TyLogTokenGenesisTransfer: {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogTokenGenesisTransfer"},
		TyLogTokenGenesisDeposit:  {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogTokenGenesisDeposit"},
		TyLogPreCreateToken:       {reflect.TypeOf(ReceiptToken{}), "LogPreCreateToken"},
		TyLogFinishCreateToken:    {reflect.TypeOf(ReceiptToken{}), "LogFinishCreateToken"},
		TyLogRevokeCreateToken:    {reflect.TypeOf(ReceiptToken{}), "LogRevokeCreateToken"},
	}
}

func (t *TokenType) RPC_Default_Process(action string, msg interface{}) (*types.Transaction, error) {
	var create *types.CreateTx
	if _, ok := msg.(*types.CreateTx); !ok {
		return nil, types.ErrInvalidParam
	}
	create = msg.(*types.CreateTx)
	if !create.IsToken {
		return nil, types.ErrNotSupport
	}
	tx, err := t.AssertCreate(create)
	if err != nil {
		return nil, err
	}
	//to地址的问题,如果是主链交易，to地址就是直接是设置to
	if !types.IsPara() {
		tx.To = create.To
	}
	return tx, err
}
