package executor

import (
	"encoding/json"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
	gt "gitlab.33.cn/chain33/chain33/pluginmanager/plugins/blackwhite/types"
)


var glog = log.New("module", types.BlackwhiteX)
var name string

// TODO: 需要在插件初始化时，调用
func InitTypes() {
	name = types.ExecName(types.BlackwhiteX)
	// init executor type
	types.RegistorExecutor(name, &BlackwhiteType{})

	// init log
	types.RegistorLog(types.TyLogBlackwhiteCreate, &BlackwhiteCreateLog{})
	types.RegistorLog(types.TyLogBlackwhitePlay, &BlackwhitePlayLog{})
	types.RegistorLog(types.TyLogBlackwhiteShow, &BlackwhiteShowLog{})
	types.RegistorLog(types.TyLogBlackwhiteTimeout, &BlackwhiteTimeoutDoneLog{})
	types.RegistorLog(types.TyLogBlackwhiteDone, &BlackwhiteDoneLog{})
	types.RegistorLog(types.TyLogBlackwhiteLoopInfo, &BlackwhiteLoopInfoLog{})

	// init query rpc
	types.RegistorRpcType(gt.GetBlackwhiteRoundInfo, &BlackwhiteRoundInfo{})
	types.RegistorRpcType(gt.GetBlackwhiteByStatusAndAddr, &BlackwhiteByStatusAndAddr{})
	types.RegistorRpcType(gt.GetBlackwhiteloopResult, &BlackwhiteloopResult{})
}

type BlackwhiteType struct {
	types.ExecTypeBase
}

func (m BlackwhiteType) ActionName(tx *types.Transaction) string {
	var g gt.BlackwhiteAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-Blackwhite-action-err"
	}
	if g.Ty == gt.BlackwhiteActionCreate && g.GetCreate() != nil {
		return "BlackwhiteCreate"
	} else if g.Ty == gt.BlackwhiteActionShow && g.GetShow() != nil {
		return "BlackwhiteShow"
	} else if g.Ty == gt.BlackwhiteActionPlay && g.GetPlay() != nil {
		return "BlackwhitePlay"
	} else if g.Ty == gt.BlackwhiteActionTimeoutDone && g.GetTimeoutDone() != nil {
		return "BlackwhiteTimeoutDone"
	}
	return "unkown"
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
	var logTmp gt.ReceiptBlackwhite
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
	var logTmp gt.ReceiptBlackwhite
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
	var logTmp gt.ReceiptBlackwhite
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
	var logTmp gt.ReceiptBlackwhite
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
	var logTmp gt.ReceiptBlackwhite
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
	var logTmp gt.ReplyLoopResults
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteRoundInfo struct {
}

func (t *BlackwhiteRoundInfo) Input(message json.RawMessage) ([]byte, error) {
	var req gt.ReqBlackwhiteRoundInfo
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
	var req gt.ReqBlackwhiteRoundList
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
	var req gt.ReqLoopResult
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteloopResult) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
