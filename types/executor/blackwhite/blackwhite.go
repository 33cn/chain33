package blackwhite

import (
	"encoding/json"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
	//"time"
	//"math/rand"
	//"gitlab.33.cn/chain33/chain33/common/address"
)

// status
const (
	BlackwhiteStatusReady = iota
	BlackwhiteStatusCancel
	BlackwhiteStatusPlay
	BlackwhiteStatusTimeoutDone
	BlackwhiteStatusDone
)

const (
	GetBlackwhiteRoundInfo = "GetBlackwhiteRoundInfo"
)

const name = types.BlackwhiteX

var glog = log.New("module", name)

func Init() {
	// init executor type
	types.RegistorExecutor(name, &BlackwhiteType{})

	// init log
	types.RegistorLog(types.TyLogBlackwhiteCreate, &BlackwhiteCreateLog{})
	types.RegistorLog(types.TyLogBlackwhiteCancel, &BlackwhiteCancelLog{})
	types.RegistorLog(types.TyLogBlackwhitePlay, &BlackwhitePlayLog{})
	types.RegistorLog(types.TyLogBlackwhiteTimeoutDone, &BlackwhiteTimeoutDoneLog{})

	// init query rpc
	types.RegistorRpcType(GetBlackwhiteRoundInfo, &BlackwhiteRoundInfo{})
}

type BlackwhiteType struct {
	types.ExecTypeBase
}

func (m BlackwhiteType) ActionName(tx *types.Transaction) string {
	var g types.BlackwhiteAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-Blackwhite-action-err"
	}
	if g.Ty == types.BlackwhiteActionCreate && g.GetCreate() != nil {
		return "BlackwhiteCreate"
	} else if g.Ty == types.BlackwhiteActionCancel && g.GetCancel() != nil {
		return "BlackwhiteCancel"
	} else if g.Ty == types.BlackwhiteActionPlay && g.GetPlay() != nil {
		return "BlackwhitePlay"
	} else if g.Ty == types.BlackwhiteActionTimeoutDone && g.GetTimeoutDone() != nil {
		return "BlackwhiteTimeoutDone"
	}
	return "unkown"
}

func (m BlackwhiteType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (m BlackwhiteType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

type BlackwhitePlayLog struct {
}

func (l BlackwhitePlayLog) Name() string {
	return "LogBlackwhitePlay"
}

func (l BlackwhitePlayLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptBlackwhite
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
	var logTmp types.ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteCancelLog struct {
}

func (l BlackwhiteCancelLog) Name() string {
	return "LogBlackwhiteCancel"
}

func (l BlackwhiteCancelLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteCreateLog struct {
}

func (l BlackwhiteCreateLog) Name() string {
	return "LogBlackwhiteCreate"
}

func (l BlackwhiteCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptBlackwhite
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteRoundInfo struct {
}

func (t *BlackwhiteRoundInfo) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqBlackwhiteRoundInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteRoundInfo) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
