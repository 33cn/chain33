package lottery

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string

var llog = log.New("module", name)

func Init() {
	name = types.ExecName(types.LotteryX)
	// init executor type
	types.RegistorExecutor(name, &LotteryType{})

	// init log
	types.RegistorLog(types.TyLogLotteryCreate, &LotteryCreateLog{})
	types.RegistorLog(types.TyLogLotteryBuy, &LotteryBuyLog{})
	types.RegistorLog(types.TyLogLotteryDraw, &LotteryDrawLog{})
	types.RegistorLog(types.TyLogLotteryClose, &LotteryCloseLog{})

	// init query rpc
	types.RegistorRpcType("GetLotteryNormalInfo", &LotteryGetInfo{})
	types.RegistorRpcType("GetLotteryCurrentInfo", &LotteryGetInfo{})
	types.RegistorRpcType("GetLotteryHistoryLuckyNumber", &LotteryGetInfo{})
	types.RegistorRpcType("GetLotteryRoundLuckyNumber", &LotteryLuckyRoundInfo{})
	types.RegistorRpcType("GetLotteryHistoryBuyInfo", &LotteryBuyInfo{})
	types.RegistorRpcType("GetLotteryBuyRoundInfo", &LotteryBuyRoundInfo{})

}

type LotteryType struct {
	types.ExecTypeBase
}

func (lottery LotteryType) ActionName(tx *types.Transaction) string {
	var action types.LotteryAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == types.LotteryActionCreate && action.GetCreate() != nil {
		return "create"
	} else if action.Ty == types.LotteryActionBuy && action.GetBuy() != nil {
		return "buy"
	} else if action.Ty == types.LotteryActionDraw && action.GetDraw() != nil {
		return "draw"
	} else if action.Ty == types.LotteryActionClose && action.GetClose() != nil {
		return "close"
	}
	return "unknow"
}

func (lottery LotteryType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action types.LotteryAction
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

type LotteryCreateLog struct {
}

func (l LotteryCreateLog) Name() string {
	return "LogLotteryCreate"
}

func (l LotteryCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type LotteryBuyLog struct {
}

func (l LotteryBuyLog) Name() string {
	return "LogLotteryBuy"
}

func (l LotteryBuyLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type LotteryDrawLog struct {
}

func (l LotteryDrawLog) Name() string {
	return "LogLotteryDraw"
}

func (l LotteryDrawLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type LotteryCloseLog struct {
}

func (l LotteryCloseLog) Name() string {
	return "LogLotteryClose"
}

func (l LotteryCloseLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type LotteryGetInfo struct {
}

func (t *LotteryGetInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqLotteryInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryGetInfo) ProtoToJson(reply interface{}) (interface{}, error) {
	return reply, nil
}

type LotteryLuckyRoundInfo struct {
}

func (t *LotteryLuckyRoundInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqLotteryLuckyInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryLuckyRoundInfo) ProtoToJson(reply interface{}) (interface{}, error) {
	return reply, nil
}

type LotteryBuyInfo struct {
}

func (t *LotteryBuyInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqLotteryBuyHistory
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryBuyInfo) ProtoToJson(reply interface{}) (interface{}, error) {
	return reply, nil
}

type LotteryBuyRoundInfo struct {
}

func (t *LotteryBuyRoundInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqLotteryBuyInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryBuyRoundInfo) ProtoToJson(reply interface{}) (interface{}, error) {
	return reply, nil
}

func CreateRawLotteryCreateTx(parm *LotteryCreateTx) (*types.Transaction, error) {
	if parm == nil {
		llog.Error("CreateRawLotteryCreateTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.LotteryCreate{
		PurBlockNum:  parm.PurBlockNum,
		DrawBlockNum: parm.DrawBlockNum,
	}
	create := &types.LotteryAction{
		Ty:    types.LotteryActionCreate,
		Value: &types.LotteryAction_Create{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(create),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
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

	v := &types.LotteryBuy{
		LotteryId: parm.LotteryId,
		Amount:    parm.Amount,
		Number:    parm.Number,
	}
	buy := &types.LotteryAction{
		Ty:    types.LotteryActionBuy,
		Value: &types.LotteryAction_Buy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
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

	v := &types.LotteryDraw{
		LotteryId: parm.LotteryId,
	}
	draw := &types.LotteryAction{
		Ty:    types.LotteryActionDraw,
		Value: &types.LotteryAction_Draw{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(draw),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
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

	v := &types.LotteryClose{
		LotteryId: parm.LotteryId,
	}
	close := &types.LotteryAction{
		Ty:    types.LotteryActionClose,
		Value: &types.LotteryAction_Close{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(close),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
