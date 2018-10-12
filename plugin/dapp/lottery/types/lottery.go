package types

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	nameX string
	llog  = log.New("module", "exectype."+types.LotteryX)

	actionTypeMap = map[string]int32{
		"Create": LotteryActionCreate,
		"Buy":    LotteryActionBuy,
		"Draw":   LotteryActionDraw,
		"Close":  LotteryActionClose,
	}
)

func init() {
	nameX = types.ExecName(types.LotteryX)

	// init executor type
	types.RegistorExecutor(types.LotteryX, NewType())

	// init log
	types.RegistorLog(types.TyLogLotteryCreate, &LotteryCreateLog{})
	types.RegistorLog(types.TyLogLotteryBuy, &LotteryBuyLog{})
	types.RegistorLog(types.TyLogLotteryDraw, &LotteryDrawLog{})
	types.RegistorLog(types.TyLogLotteryClose, &LotteryCloseLog{})

	// init query rpc
	types.RegisterRPCQueryHandle("GetLotteryNormalInfo", &LotteryGetInfo{})
	types.RegisterRPCQueryHandle("GetLotteryCurrentInfo", &LotteryGetInfo{})
	types.RegisterRPCQueryHandle("GetLotteryHistoryLuckyNumber", &LotteryGetInfo{})
	types.RegisterRPCQueryHandle("GetLotteryRoundLuckyNumber", &LotteryLuckyRoundInfo{})
	types.RegisterRPCQueryHandle("GetLotteryHistoryBuyInfo", &LotteryBuyInfo{})
	types.RegisterRPCQueryHandle("GetLotteryBuyRoundInfo", &LotteryBuyRoundInfo{})

}

type LotteryType struct {
	types.ExecTypeBase
}

func NewType() *LotteryType {
	c := &LotteryType{}
	c.SetChild(c)
	return c
}

func (at *LotteryType) GetPayload() types.Message {
	return &LotteryAction{}
}

func (lottery LotteryType) ActionName(tx *types.Transaction) string {
	var action LotteryAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == LotteryActionCreate && action.GetCreate() != nil {
		return "create"
	} else if action.Ty == LotteryActionBuy && action.GetBuy() != nil {
		return "buy"
	} else if action.Ty == LotteryActionDraw && action.GetDraw() != nil {
		return "draw"
	} else if action.Ty == LotteryActionClose && action.GetClose() != nil {
		return "close"
	}
	return "unknow"
}

func (lottery LotteryType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action LotteryAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (lottery LotteryType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

func (lottery LotteryType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	llog.Debug("lottery.CreateTx", "action", action)
	var tx *types.Transaction
	if action == "LotteryCreate" {
		var param LotteryCreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawLotteryCreateTx(&param)
	} else if action == "LotteryBuy" {
		var param LotteryBuyTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawLotteryBuyTx(&param)
	} else if action == "LotteryDraw" {
		var param LotteryDrawTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawLotteryDrawTx(&param)
	} else if action == "LotteryClose" {
		var param LotteryCloseTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			llog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawLotteryCloseTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

func (lott LotteryType) GetTypeMap() map[string]int32 {
	return actionTypeMap
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
		Execer:  []byte(types.ExecName(types.LotteryX)),
		Payload: types.Encode(create),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(types.LotteryX)),
	}

	err := tx.SetRealFee(types.MinFee)
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
		Execer:  []byte(types.ExecName(types.LotteryX)),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(types.LotteryX)),
	}

	err := tx.SetRealFee(types.MinFee)
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
		Execer:  []byte(types.ExecName(types.LotteryX)),
		Payload: types.Encode(draw),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(types.LotteryX)),
	}

	err := tx.SetRealFee(types.MinFee)
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
		Execer:  []byte(types.ExecName(types.LotteryX)),
		Payload: types.Encode(close),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(types.LotteryX)),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
