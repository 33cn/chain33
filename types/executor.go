package types

import "encoding/json"

type ExecutorType interface {
	// name 是 executor name 构造时用
	//func Name()
	ActionName(tx *Transaction) string
	NewTx(action string, message json.RawMessage) (*Transaction, error)
	Amount(tx *Transaction) (int64, error)
}

type LogType interface {
	Name() string
	Decode([]byte) (interface{}, error)
}

type RpcQueryType interface {
	Input(message json.RawMessage) ([]byte, error)
	Output(interface{}) (interface{}, error)
}

var executorMap = map[string]ExecutorType{}
var receiptLogMap = map[int64]LogType{}
var rpcTypeUtilMap = map[string]RpcQueryType{}

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

func registorRpcType(funcName string, util RpcQueryType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := rpcTypeUtilMap[funcName]; exist {
		panic("DupRpcTypeUtil")
	} else {
		rpcTypeUtilMap[funcName] = util
	}
}

func RegistorRpcType(funcName string, util RpcQueryType) {
	registorRpcType(funcName, util)
}

func LoadQueryType(funcName string) RpcQueryType {
	if trans, ok := rpcTypeUtilMap[funcName]; ok {
		return trans
	}
	return nil
}
