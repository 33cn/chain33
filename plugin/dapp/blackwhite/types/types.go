package types

import (
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/types/convertor"

	"gitlab.33.cn/chain33/chain33/types"
)

// blackwhite action type
const (
	BlackwhiteActionCreate = iota
	BlackwhiteActionPlay
	BlackwhiteActionShow
	BlackwhiteActionTimeoutDone
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerBlackwhite)

	// init executor type
	types.RegistorExecutor(BlackwhiteX, NewType())
	// init log
	types.RegistorLog(types.TyLogBlackwhiteCreate, &BlackwhiteCreateLog{})
	types.RegistorLog(types.TyLogBlackwhitePlay, &BlackwhitePlayLog{})
	types.RegistorLog(types.TyLogBlackwhiteShow, &BlackwhiteShowLog{})
	types.RegistorLog(types.TyLogBlackwhiteTimeout, &BlackwhiteTimeoutDoneLog{})
	types.RegistorLog(types.TyLogBlackwhiteDone, &BlackwhiteDoneLog{})
	types.RegistorLog(types.TyLogBlackwhiteLoopInfo, &BlackwhiteLoopInfoLog{})

	// init query rpc
	types.RegisterRPCQueryHandle(GetBlackwhiteRoundInfo, &convertor.QueryConvertor{ProtoObj: &ReqBlackwhiteRoundInfo{}})
	types.RegisterRPCQueryHandle(GetBlackwhiteByStatusAndAddr, &convertor.QueryConvertor{ProtoObj: &ReqBlackwhiteRoundList{}})
	types.RegisterRPCQueryHandle(GetBlackwhiteloopResult, &convertor.QueryConvertor{ProtoObj: &ReqLoopResult{}})
}

type BlackwhiteType struct {
	types.ExecTypeBase
}

func NewType() *BlackwhiteType {
	c := &BlackwhiteType{}
	c.SetChild(c)
	return c
}

func (b *BlackwhiteType) GetPayload() types.Message {
	return &BlackwhiteAction{}
}

func (b *BlackwhiteType) GetName() string {
	return BlackwhiteX
}

func (b *BlackwhiteType) GetLogMap() map[int64]*types.LogInfo {
	return nil
}

func (b *BlackwhiteType) GetTypeMap() map[string]int32 {
	return actionName
}

func (m BlackwhiteType) ActionName(tx *types.Transaction) string {
	var g BlackwhiteAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-Blackwhite-action-err"
	}
	if g.Ty == BlackwhiteActionCreate && g.GetCreate() != nil {
		return "BlackwhiteCreate"
	} else if g.Ty == BlackwhiteActionShow && g.GetShow() != nil {
		return "BlackwhiteShow"
	} else if g.Ty == BlackwhiteActionPlay && g.GetPlay() != nil {
		return "BlackwhitePlay"
	} else if g.Ty == BlackwhiteActionTimeoutDone && g.GetTimeoutDone() != nil {
		return "BlackwhiteTimeoutDone"
	}
	return "unkown"
}

func (blackwhite BlackwhiteType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action BlackwhiteAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (m BlackwhiteType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (m BlackwhiteType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	glog.Debug("Blackwhite.CreateTx", "action", action)
	var tx *types.Transaction
	return tx, nil
}

type BlackwhiteCreateLog struct {
}

func (l BlackwhiteCreateLog) Name() string {
	return "LogBlackwhiteCreate"
}

func (l BlackwhiteCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhitePlayLog struct {
}

func (l BlackwhitePlayLog) Name() string {
	return "LogBlackwhitePlay"
}

func (l BlackwhitePlayLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteShowLog struct {
}

func (l BlackwhiteShowLog) Name() string {
	return "LogBlackwhiteShow"
}

func (l BlackwhiteShowLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteTimeoutDoneLog struct {
}

func (l BlackwhiteTimeoutDoneLog) Name() string {
	return "LogBlackwhiteTimeoutDone"
}

func (l BlackwhiteTimeoutDoneLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteDoneLog struct {
}

func (l BlackwhiteDoneLog) Name() string {
	return "LogBlackwhiteDone"
}

func (l BlackwhiteDoneLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteLoopInfoLog struct {
}

func (l BlackwhiteLoopInfoLog) Name() string {
	return "LogBlackwhiteLoopInfo"
}

func (l BlackwhiteLoopInfoLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReplyLoopResults
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteRoundInfo struct {
}

func (t *BlackwhiteRoundInfo) Input(message json.RawMessage) ([]byte, error) {
	var req ReqBlackwhiteRoundInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteRoundInfo) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type BlackwhiteByStatusAndAddr struct {
}

func (t *BlackwhiteByStatusAndAddr) Input(message json.RawMessage) ([]byte, error) {
	var req ReqBlackwhiteRoundList
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteByStatusAndAddr) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type BlackwhiteloopResult struct {
}

func (t *BlackwhiteloopResult) Input(message json.RawMessage) ([]byte, error) {
	var req ReqLoopResult
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteloopResult) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
