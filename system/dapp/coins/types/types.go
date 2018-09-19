package types

import (
	"reflect"

	"gitlab.33.cn/chain33/chain33/types"
)

const (
	CoinsActionTransfer       = 1
	CoinsActionGenesis        = 2
	CoinsActionWithdraw       = 3
	CoinsActionTransferToExec = 10
)

var (
	CoinsX      = "coins"
	ExecerCoins = []byte(CoinsX)
	actionName  = map[string]int32{
		"Transfer":       CoinsActionTransfer,
		"TransferToExec": CoinsActionTransferToExec,
		"Withdraw":       CoinsActionWithdraw,
		"Genesis":        CoinsActionGenesis,
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerCoins)
	types.RegistorExecutor("coins", NewType())
}

type CoinsType struct {
	types.ExecTypeBase
}

func NewType() *CoinsType {
	c := &CoinsType{}
	c.SetChild(c)
	return c
}

func (coins *CoinsType) GetName() string {
	return CoinsX
}

func (coins *CoinsType) GetPayload() types.Message {
	return &CoinsAction{}
}

func (coins *CoinsType) GetLogMap() map[int64]reflect.Type {
	return nil
}

func (c *CoinsType) GetTypeMap() map[string]int32 {
	return actionName
}

func (c *CoinsType) RPC_Default_Process(action string, msg interface{}) (*types.Transaction, error) {
	var create *types.CreateTx
	if _, ok := msg.(*types.CreateTx); !ok {
		return nil, types.ErrInvalidParam
	}
	create = msg.(*types.CreateTx)
	if create.IsToken {
		return nil, types.ErrNotSupport
	}
	return c.AssertCreate(create)
}
