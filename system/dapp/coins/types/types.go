package types

import (
	"reflect"

	"github.com/inconshreveable/log15"
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
	tlog        = log15.New("module", "exectype.coins")

	/*
		对应 proto type 的字段
		//	*CoinsAction_Transfer
		//	*CoinsAction_Withdraw
		//	*CoinsAction_Genesis
		//	*CoinsAction_TransferToExec
	*/
	actionName = map[string]int32{
		"Transfer":       CoinsActionTransfer,
		"TransferToExec": CoinsActionTransferToExec,
		"Withdraw":       CoinsActionWithdraw,
		"Genesis":        CoinsActionGenesis,
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerCoins)
}

func Init() {
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

func (coins *CoinsType) GetPayload() types.Message {
	return &CoinsAction{}
}

func (coins *CoinsType) GetLogMap() map[int64]reflect.Type {
	return nil
}

func (c *CoinsType) GetTypeMap() map[string]int32 {
	return actionName
}
