package executor

import (
	"encoding/hex"
	"encoding/json"

	log "github.com/inconshreveable/log15"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", gt.BlackwhiteX)
var name string

func InitTypes() {
	name = types.ExecName(gt.BlackwhiteX)
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
	types.RegistorRpcType(gt.BlackwhiteCreateTx, &BlackwhiteCreateTxRPC{})
	types.RegistorRpcType(gt.BlackwhitePlayTx, &BlackwhitePlayTxRPC{})
	types.RegistorRpcType(gt.BlackwhiteShowTx, &BlackwhiteShowTxRPC{})
	types.RegistorRpcType(gt.BlackwhiteTimeoutDoneTx, &BlackwhiteTimeoutDoneTxRPC{})
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

func (blackwhite BlackwhiteType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action gt.BlackwhiteAction
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

func (t *BlackwhiteRoundInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req gt.ReqBlackwhiteRoundInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteRoundInfo) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type BlackwhiteByStatusAndAddr struct {
}

func (t *BlackwhiteByStatusAndAddr) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req gt.ReqBlackwhiteRoundList
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteByStatusAndAddr) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type BlackwhiteloopResult struct {
}

func (t *BlackwhiteloopResult) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req gt.ReqLoopResult
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteloopResult) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type BlackwhiteCreateTxRPC struct{}

func (t *BlackwhiteCreateTxRPC) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req gt.BlackwhiteCreateTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteCreateTxRPC) ProtoToJson(reply *types.Message) (interface{}, error) {
	if tx, ok := (*reply).(*types.Transaction); ok {
		data := types.Encode(tx)
		return hex.EncodeToString(data), nil
	}
	return nil, types.ErrTypeAsset
}

type BlackwhitePlayTxRPC struct {
}

func (t *BlackwhitePlayTxRPC) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req gt.BlackwhitePlayTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhitePlayTxRPC) ProtoToJson(reply *types.Message) (interface{}, error) {
	if tx, ok := (*reply).(*types.Transaction); ok {
		data := types.Encode(tx)
		return hex.EncodeToString(data), nil
	}
	return nil, types.ErrTypeAsset
}

type BlackwhiteShowTxRPC struct {
}

func (t *BlackwhiteShowTxRPC) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req gt.BlackwhiteShowTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteShowTxRPC) ProtoToJson(reply *types.Message) (interface{}, error) {
	if tx, ok := (*reply).(*types.Transaction); ok {
		data := types.Encode(tx)
		return hex.EncodeToString(data), nil
	}
	return nil, types.ErrTypeAsset
}

type BlackwhiteTimeoutDoneTxRPC struct {
}

func (t *BlackwhiteTimeoutDoneTxRPC) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req gt.BlackwhiteTimeoutDoneTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteTimeoutDoneTxRPC) ProtoToJson(reply *types.Message) (interface{}, error) {
	if tx, ok := (*reply).(*types.Transaction); ok {
		data := types.Encode(tx)
		return hex.EncodeToString(data), nil
	}
	return nil, types.ErrTypeAsset
}
