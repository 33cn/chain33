package types

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unicode"
)

type ExecutorType interface {
	//获取交易真正的to addr
	GetRealToAddr(tx *Transaction) string
	//给用户显示的from 和 to
	GetViewFromToAddr(tx *Transaction) (string, string)
	ActionName(tx *Transaction) string
	CreateTx(action string, message json.RawMessage) (*Transaction, error)
	Amount(tx *Transaction) (int64, error)
	DecodePayload(tx *Transaction) (interface{}, error)
	DecodePayloadValue(tx *Transaction) (string, reflect.Value, error)
}

type LogType interface {
	Name() string
	Decode([]byte) (interface{}, error)
}

type RPCQueryTypeConvert interface {
	JsonToProto(message json.RawMessage) ([]byte, error)
	ProtoToJson(reply *Message) (interface{}, error)
}

type FN_RPCQueryHandle func(param []byte) (Message, error)

//
type rpcTypeUtilItem struct {
	convertor RPCQueryTypeConvert
	handler   FN_RPCQueryHandle
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
	//尽可能的加载执行器
	//真正的权限控制在区块执行的时候做控制
	realname := GetRealExecName([]byte(exec))
	if exec, exist := executorMap[string(realname)]; exist {
		return exec
	}
	return nil
}

func RegistorLog(logTy int64, util LogType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := receiptLogMap[logTy]; exist {
		errMsg := fmt.Sprintf("DupLogType RegistorLog type existed", "logTy", logTy)
		panic(errMsg)
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

func registerRPCQueryHandle(funcName string, convertor RPCQueryTypeConvert, handler FN_RPCQueryHandle) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if _, exist := rpcTypeUtilMap[funcName]; exist {
		panic("DupRpcTypeUtil")
	} else {
		rpcTypeUtilMap[funcName] = &rpcTypeUtilItem{
			convertor: convertor,
			handler:   handler,
		}
	}
}

func RegisterRPCQueryHandle(funcName string, convertor RPCQueryTypeConvert) {
	registerRPCQueryHandle(funcName, convertor, nil)
}

// TODO: 后续将函数关联关系都在一处实现
//func RegisterRPCQueryHandleV2(funcName string, convertor RPCQueryTypeConvert, handler FN_RPCQueryHandle) {
//	registerRPCQueryHandle(funcName, convertor, handler)
//}

func LoadQueryType(funcName string) RPCQueryTypeConvert {
	if trans, ok := rpcTypeUtilMap[funcName]; ok {
		return trans.convertor
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

type ExecType interface {
	//write for executor
	GetPayload() Message
	//exec result of receipt log
	GetLogMap() map[int64]reflect.Type
	//actionType -> name map
	GetTypeMap() map[string]int32
	//action function list map
	GetFuncMap() map[string]reflect.Method
}

type ExecTypeGet interface {
	GetTy() int32
}

type ExecTypeBase struct {
	child ExecType
}

//用户看到的ToAddr
func (base *ExecTypeBase) GetRealToAddr(tx *Transaction) string {
	if !IsPara() {
		return tx.To
	}
	//平行链中的处理方式
	_, v, err := base.DecodePayloadValue(tx)
	if err != nil {
		return tx.To
	}
	payload := v.Interface()
	//四种assert的结构体
	if ato, ok := payload.(*AssetsTransferToExec); ok {
		return ato.GetTo()
	}
	if ato, ok := payload.(*AssetsTransfer); ok {
		return ato.GetTo()
	}
	if ato, ok := payload.(*AssetsWithdraw); ok {
		return ato.GetTo()
	}
	if ato, ok := payload.(*AssetsTransferToExec); ok {
		return ato.GetTo()
	}
	return tx.To
}

func (base *ExecTypeBase) Amount(tx *Transaction) (int64, error) {
	_, v, err := base.DecodePayloadValue(tx)
	if err != nil {
		return 0, err
	}
	payload := v.Interface()
	//四种assert的结构体
	if ato, ok := payload.(*AssetsTransferToExec); ok {
		return ato.GetAmount(), nil
	}
	if ato, ok := payload.(*AssetsTransfer); ok {
		return ato.GetAmount(), nil
	}
	if ato, ok := payload.(*AssetsWithdraw); ok {
		return ato.GetAmount(), nil
	}
	if ato, ok := payload.(*AssetsTransferToExec); ok {
		return ato.GetAmount(), nil
	}
	return 0, nil
}

//用户看到的FromAddr
func (base *ExecTypeBase) GetViewFromToAddr(tx *Transaction) (string, string) {
	return tx.From(), tx.To
}

func (base *ExecTypeBase) SetChild(child ExecType) {
	base.child = child
}

func (base *ExecTypeBase) DecodePayload(tx *Transaction) (interface{}, error) {
	payload := base.child.GetPayload()
	err := Decode(tx.GetPayload(), payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (base *ExecTypeBase) DecodePayloadValue(tx *Transaction) (string, reflect.Value, error) {
	action, err := base.DecodePayload(tx)
	if err != nil {
		return "", nilValue, err
	}
	name, ty, val := GetActionValue(action, base.child.GetFuncMap())
	if val.IsNil() {
		return "", nilValue, ErrActionNotSupport
	}
	typemap := base.child.GetTypeMap()
	//check types is ok
	if v, ok := typemap[name]; !ok || v != ty {
		return "", nilValue, ErrActionNotSupport
	}
	return name, val, nil
}

func (base *ExecTypeBase) ActionName(tx *Transaction) string {
	payload, err := base.DecodePayload(tx)
	if err != nil {
		return "unknown-err"
	}
	tm := base.child.GetTypeMap()
	if get, ok := payload.(ExecTypeGet); ok {
		ty := get.GetTy()
		for k, v := range tm {
			if v == ty {
				return lowcaseFirst(k)
			}
		}
	}
	return "unknown"
}

func lowcaseFirst(v string) string {
	if len(v) == 0 {
		return ""
	}
	change := []rune(v)
	if unicode.IsUpper(change[0]) {
		change[0] = unicode.ToLower(change[0])
		return string(change)
	}
	return v
}
