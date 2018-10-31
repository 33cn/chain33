package types

import (
	"encoding/json"

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
	types.RegisterDappFork(BlackwhiteX, "ForkBlackWhiteV2", 900000)
	types.RegisterDappFork(BlackwhiteX, "Enable", 850000)
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
	return logInfo
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

func (m BlackwhiteType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (m BlackwhiteType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	glog.Debug("Blackwhite.CreateTx", "action", action)
	var tx *types.Transaction
	return tx, nil
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
