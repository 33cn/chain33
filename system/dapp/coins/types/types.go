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

	types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})
	types.RegistorLog(types.TyLogTransfer, &CoinsTransferLog{})
	types.RegistorLog(types.TyLogGenesis, &CoinsGenesisLog{})
	types.RegistorLog(types.TyLogExecTransfer, &CoinsExecTransferLog{})
	types.RegistorLog(types.TyLogExecWithdraw, &CoinsExecWithdrawLog{})
	types.RegistorLog(types.TyLogExecDeposit, &CoinsExecDepositLog{})
	types.RegistorLog(types.TyLogExecFrozen, &CoinsExecFrozenLog{})
	types.RegistorLog(types.TyLogExecActive, &CoinsExecActiveLog{})
	types.RegistorLog(types.TyLogGenesisTransfer, &CoinsGenesisTransferLog{})
	types.RegistorLog(types.TyLogGenesisDeposit, &CoinsGenesisDepositLog{})
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

func (coins *CoinsType) GetName() string {
	return CoinsX
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
	tx, err := c.AssertCreate(create)
	if err != nil {
		return nil, err
	}
	//to地址的问题,如果是主链交易，to地址就是直接是设置to
	if !types.IsPara() {
		tx.To = create.To
	}
	return tx, err
}

//重构后这部分会删除
type CoinsDepositLog struct {
}

func (l CoinsDepositLog) Name() string {
	return "LogDeposit"
}

func (l CoinsDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type CoinsGenesisLog struct {
}

func (l CoinsGenesisLog) Name() string {
	return "LogGenesis"
}

func (l CoinsGenesisLog) Decode(msg []byte) (interface{}, error) {
	return nil, nil
}

type CoinsTransferLog struct {
}

func (l CoinsTransferLog) Name() string {
	return "LogTransfer"
}

func (l CoinsTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecTransferLog struct {
}

func (l CoinsExecTransferLog) Name() string {
	return "LogExecTransfer"
}

func (l CoinsExecTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecWithdrawLog struct {
}

func (l CoinsExecWithdrawLog) Name() string {
	return "LogExecWithdraw"
}

func (l CoinsExecWithdrawLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecDepositLog struct {
}

func (l CoinsExecDepositLog) Name() string {
	return "LogExecDeposit"
}

func (l CoinsExecDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecFrozenLog struct {
}

func (l CoinsExecFrozenLog) Name() string {
	return "LogExecFrozen"
}

func (l CoinsExecFrozenLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecActiveLog struct {
}

func (l CoinsExecActiveLog) Name() string {
	return "LogExecActive"
}

func (l CoinsExecActiveLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsGenesisTransferLog struct {
}

func (l CoinsGenesisTransferLog) Name() string {
	return "LogGenesisTransfer"
}

func (l CoinsGenesisTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsGenesisDepositLog struct {
}

func (l CoinsGenesisDepositLog) Name() string {
	return "LogGenesisDeposit"
}

func (l CoinsGenesisDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}
