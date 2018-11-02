package types

import (
	"encoding/json"
	"reflect"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	llog = log.New("module", "exectype."+LotteryX)
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(LotteryX))
	types.RegistorExecutor(LotteryX, NewType())
	types.RegisterDappFork(LotteryX, "Enable", 0)
}

type LotteryType struct {
	types.ExecTypeBase
}

func NewType() *LotteryType {
	c := &LotteryType{}
	c.SetChild(c)
	return c
}

func (at *LotteryType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{
		TyLogLotteryCreate: {reflect.TypeOf(ReceiptLottery{}), "LogLotteryCreate"},
		TyLogLotteryBuy:    {reflect.TypeOf(ReceiptLottery{}), "LogLotteryBuy"},
		TyLogLotteryDraw:   {reflect.TypeOf(ReceiptLottery{}), "LogLotteryDraw"},
		TyLogLotteryClose:  {reflect.TypeOf(ReceiptLottery{}), "LogLotteryClose"},
	}
}

func (at *LotteryType) GetPayload() types.Message {
	return &LotteryAction{}
}

func (lottery LotteryType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	llog.Debug("lottery.CreateTx", "action", action)
	var tx *types.Transaction
	if action == "LotteryCreate" {
		var param LotteryCreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawLotteryCreateTx(&param)
	} else if action == "LotteryBuy" {
		var param LotteryBuyTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawLotteryBuyTx(&param)
	} else if action == "LotteryDraw" {
		var param LotteryDrawTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawLotteryDrawTx(&param)
	} else if action == "LotteryClose" {
		var param LotteryCloseTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawLotteryCloseTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

func (lott LotteryType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Create": LotteryActionCreate,
		"Buy":    LotteryActionBuy,
		"Draw":   LotteryActionDraw,
		"Close":  LotteryActionClose,
	}
}

func CreateRawLotteryCreateTx(parm *LotteryCreateTx) (*types.Transaction, error) {
	if parm == nil {
		llog.Error("CreateRawLotteryCreateTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &LotteryCreate{
		PurBlockNum:  parm.PurBlockNum,
		DrawBlockNum: parm.DrawBlockNum,
	}
	create := &LotteryAction{
		Ty:    LotteryActionCreate,
		Value: &LotteryAction_Create{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(LotteryX)),
		Payload: types.Encode(create),
		Fee:     parm.Fee,
		To:      address.ExecAddress(types.ExecName(LotteryX)),
	}
	name := types.ExecName(LotteryX)
	tx, err := types.FormatTx(name, tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func CreateRawLotteryBuyTx(parm *LotteryBuyTx) (*types.Transaction, error) {
	if parm == nil {
		llog.Error("CreateRawLotteryBuyTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &LotteryBuy{
		LotteryId: parm.LotteryId,
		Amount:    parm.Amount,
		Number:    parm.Number,
		Way:       parm.Way,
	}
	buy := &LotteryAction{
		Ty:    LotteryActionBuy,
		Value: &LotteryAction_Buy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(LotteryX)),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		To:      address.ExecAddress(types.ExecName(LotteryX)),
	}
	name := types.ExecName(LotteryX)
	tx, err := types.FormatTx(name, tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func CreateRawLotteryDrawTx(parm *LotteryDrawTx) (*types.Transaction, error) {
	if parm == nil {
		llog.Error("CreateRawLotteryDrawTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &LotteryDraw{
		LotteryId: parm.LotteryId,
	}
	draw := &LotteryAction{
		Ty:    LotteryActionDraw,
		Value: &LotteryAction_Draw{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(LotteryX)),
		Payload: types.Encode(draw),
		Fee:     parm.Fee,
		To:      address.ExecAddress(types.ExecName(LotteryX)),
	}
	name := types.ExecName(LotteryX)
	tx, err := types.FormatTx(name, tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func CreateRawLotteryCloseTx(parm *LotteryCloseTx) (*types.Transaction, error) {
	if parm == nil {
		llog.Error("CreateRawLotteryCloseTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &LotteryClose{
		LotteryId: parm.LotteryId,
	}
	close := &LotteryAction{
		Ty:    LotteryActionClose,
		Value: &LotteryAction_Close{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(LotteryX)),
		Payload: types.Encode(close),
		Fee:     parm.Fee,
		To:      address.ExecAddress(types.ExecName(LotteryX)),
	}

	name := types.ExecName(LotteryX)
	tx, err := types.FormatTx(name, tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
