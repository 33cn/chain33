package rpc

import (
	"encoding/hex"
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/types/convertor"

	log "github.com/inconshreveable/log15"
	bw "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", bw.BlackwhiteX)
var name string
var jrpc = &Jrpc{}
var grpc = &Grpc{}

func InitRPC(s pluginmgr.RPCServer) {
	cli := channelClient{}
	cli.Init(s.GetQueueClient())
	jrpc.cli = cli
	grpc.channelClient = cli
	s.JRPC().RegisterName(bw.JRPCName, jrpc)
	bw.RegisterBlackwhiteServer(s.GRPC(), grpc)
}

func Init(s pluginmgr.RPCServer) {
	name = bw.BlackwhiteX
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
	types.RegisterRPCQueryHandle(bw.GetBlackwhiteRoundInfo, &convertor.QueryConvertor{ProtoObj: &bw.ReqBlackwhiteRoundInfo{}})
	types.RegisterRPCQueryHandle(bw.GetBlackwhiteByStatusAndAddr, &convertor.QueryConvertor{ProtoObj: &bw.ReqBlackwhiteRoundList{}})
	types.RegisterRPCQueryHandle(bw.GetBlackwhiteloopResult, &convertor.QueryConvertor{ProtoObj: &bw.ReqLoopResult{}})

	InitRPC(s)
}

type BlackwhiteType struct {
	types.ExecTypeBase
}

func (m BlackwhiteType) ActionName(tx *types.Transaction) string {
	var g bw.BlackwhiteAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-Blackwhite-action-err"
	}
	if g.Ty == bw.BlackwhiteActionCreate && g.GetCreate() != nil {
		return "BlackwhiteCreate"
	} else if g.Ty == bw.BlackwhiteActionShow && g.GetShow() != nil {
		return "BlackwhiteShow"
	} else if g.Ty == bw.BlackwhiteActionPlay && g.GetPlay() != nil {
		return "BlackwhitePlay"
	} else if g.Ty == bw.BlackwhiteActionTimeoutDone && g.GetTimeoutDone() != nil {
		return "BlackwhiteTimeoutDone"
	}
	return "unkown"
}

func (blackwhite BlackwhiteType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action bw.BlackwhiteAction
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
	var logTmp bw.ReceiptBlackwhite
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
	var logTmp bw.ReceiptBlackwhite
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
	var logTmp bw.ReceiptBlackwhite
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
	var logTmp bw.ReceiptBlackwhite
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
	var logTmp bw.ReceiptBlackwhite
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
	var logTmp bw.ReplyLoopResults
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type BlackwhiteRoundInfo struct {
}

func (t *BlackwhiteRoundInfo) Input(message json.RawMessage) ([]byte, error) {
	var req bw.ReqBlackwhiteRoundInfo
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
	var req bw.ReqBlackwhiteRoundList
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
	var req bw.ReqLoopResult
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteloopResult) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type BlackwhiteCreateTxRPC struct{}

func (t *BlackwhiteCreateTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhiteCreateTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteCreateTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type BlackwhitePlayTxRPC struct {
}

func (t *BlackwhitePlayTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhitePlayTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhitePlayTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type BlackwhiteShowTxRPC struct {
}

func (t *BlackwhiteShowTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhiteShowTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteShowTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type BlackwhiteTimeoutDoneTxRPC struct {
}

func (t *BlackwhiteTimeoutDoneTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhiteTimeoutDoneTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteTimeoutDoneTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}
