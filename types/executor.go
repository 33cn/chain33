package types

import "encoding/json"


type ExecutorType interface {
	// name 是 executor name 构造时用
	//func Name()
	ActionName(transaction *Transaction) string
	NewTx(action string, message json.RawMessage) (*Transaction, error)
}

type LogType interface {
	Name()string
	Decode([]byte) (interface{}, error)
}

type RpcTypeQuery interface {
	Input(message json.RawMessage) ([]byte, error)
	Output(interface{}) (interface{}, error)
}

var executorMap = map[string]ExecutorType{}
var receiptLogMap = map[int64]LogType{}

func RegistorExecutor(funcName string, util ExecutorType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := executorMap[funcName]; exist {
		panic("DupExecutorType")
	} else {
		executorMap[funcName] = util
	}
}

func LoadExecutor(exec string) ExecutorType {
	if exec, exist := executorMap[exec]; exist {
		return exec
	}
	return nil
}

func RegistorLog(funcName int64, util LogType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := receiptLogMap[funcName]; exist {
		panic("DupLogType")
	} else {
		receiptLogMap[funcName] = util
	}
}

func LoadLog(ty int64) LogType {
	if log, exist := receiptLogMap[ty]; exist {
		return log
	}
	return nil
}

func registorRpcType(funcName string, util RpcTypeQuery) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := RpcTypeUtilMap[funcName]; exist {
		panic("DupRpcTypeUtil")
	} else {
		RpcTypeUtilMap[funcName] = util
	}
}

func RegistorRpcType(funcName string, util RpcTypeQuery) {
	registorRpcType(funcName, util)
}

func LoadQueryType(funcName string) RpcTypeQuery {
	if trans, ok := RpcTypeUtilMap[funcName]; ok {
		return trans
	}
	return nil
}

var RpcTypeUtilMap = map[string]RpcTypeQuery{}
