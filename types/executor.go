package types

import "encoding/json"

type ExecutorType interface {
	//获取交易真正的to addr
	GetRealToAddr(tx *Transaction) string
	//给用户显示的from 和 to
	GetViewFromToAddr(tx *Transaction) (string, string)
	ActionName(tx *Transaction) string
	CreateTx(action string, message json.RawMessage) (*Transaction, error)
	Amount(tx *Transaction) (int64, error)
	DecodePayload(tx *Transaction) (interface{}, error)
}

type LogType interface {
	Name() string
	Decode([]byte) (interface{}, error)
}

type RpcQueryConverte interface {
	JsonToProto(message json.RawMessage) ([]byte, error)
	ProtoToJson(reply *Message) (interface{}, error)
}

type FN_RPCQueryHandle func(param []byte) (Message, error)

//
type rpcTypeUtilItem struct {
	conv    RpcQueryConverte
	handler FN_RPCQueryHandle
}

var executorMap = map[string]ExecutorType{}
var receiptLogMap = map[int64]LogType{}
var rpcTypeUtilMap = map[string]*rpcTypeUtilItem{}

func RegistorExecutor(exec string, util ExecutorType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := executorMap[exec]; exist {
		panic("DupExecutorType")
	} else {
		executorMap[exec] = util
	}
}

func LoadExecutor(exec string) ExecutorType {
	if exec, exist := executorMap[exec]; exist {
		return exec
	}
	return nil
}

func RegistorLog(logTy int64, util LogType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := receiptLogMap[logTy]; exist {
		panic("DupLogType")
	} else {
		receiptLogMap[logTy] = util
	}
}

func LoadLog(ty int64) LogType {
	if log, exist := receiptLogMap[ty]; exist {
		return log
	}
	return nil
}

func registorRpcType(funcName string, util RpcQueryConverte, handler FN_RPCQueryHandle) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := rpcTypeUtilMap[funcName]; exist {
		panic("DupRpcTypeUtil")
	} else {
		rpcTypeUtilMap[funcName] = &rpcTypeUtilItem{
			conv:    util,
			handler: handler,
		}
	}
}

func RegistorRpcType(funcName string, util RpcQueryConverte) {
	registorRpcType(funcName, util, nil)
}

func RegistorRpcTypeV2(funcName string, util RpcQueryConverte, handler FN_RPCQueryHandle) {
	registorRpcType(funcName, util, handler)
}

func LoadQueryType(funcName string) RpcQueryConverte {
	if trans, ok := rpcTypeUtilMap[funcName]; ok {
		return trans.conv
	}
	return nil
}

// 处理已经注册的RPC Query响应处理过程
func ProcessRPCQuery(funcName string, param []byte) (Message, error) {
	if item, ok := rpcTypeUtilMap[funcName]; ok {
		if item.handler != nil {
			return item.handler(param)
		} else {
			return nil, ErrEmpty
		}
	}
	return nil, ErrNotFound
}

type ExecTypeBase struct {
}

//用户看到的ToAddr
func (base ExecTypeBase) GetRealToAddr(tx *Transaction) string {
	return tx.To
}

//用户看到的FromAddr
func (base ExecTypeBase) GetViewFromToAddr(tx *Transaction) (string, string) {
	return tx.From(), tx.To
}
