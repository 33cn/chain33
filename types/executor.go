package types

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"time"
	"unicode"

	"gitlab.33.cn/chain33/chain33/types"
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
	GetValueTypeMap() map[string]reflect.Type
	//action function list map
	GetFuncMap() map[string]reflect.Method
	GetRPCFuncMap() map[string]reflect.Method
}

type ExecTypeGet interface {
	GetTy() int32
}

type ExecTypeBase struct {
	child               ExecType
	childValue          reflect.Value
	actionFunList       map[string]reflect.Method
	actionListValueType map[string]reflect.Type
	rpclist             map[string]reflect.Method
}

func (base *ExecTypeBase) SetChild(child ExecType) {
	base.child = child
	base.childValue = reflect.ValueOf(child)
	base.rpclist = ListMethod(child)
	action := child.GetPayload()
	base.actionFunList = ListMethod(action)
	retval := base.actionFunList["XXX_OneofFuncs"].Func.Call([]reflect.Value{reflect.ValueOf(action)})
	if len(retval) != 4 {
		panic("err XXX_OneofFuncs")
	}
	list := ListType(retval[3].Interface().([]interface{}))
	base.actionListValueType = make(map[string]reflect.Type)
	for k, v := range list {
		data := strings.Split(k, "_")
		if len(data) != 2 {
			panic("name create " + k)
		}
		base.actionListValueType["Value_"+data[1]] = v
		field := v.Field(0)
		base.actionListValueType[field.Name] = field.Type
	}
	//check type map is all in value type list
	typelist := base.child.GetTypeMap()
	for k := range typelist {
		if _, ok := base.actionListValueType[k]; !ok {
			panic("value type not found " + k)
		}
	}
}

func (base *ExecTypeBase) GetRPCFuncMap() map[string]reflect.Method {
	return base.rpclist
}

func (base *ExecTypeBase) GetValueTypeMap() map[string]reflect.Type {
	return base.actionListValueType
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

func (base *ExecTypeBase) GetFuncMap() map[string]reflect.Method {
	return base.actionFunList
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

func (base *ExecTypeBase) callRPC(method reflect.Method, action string, msg json.RawMessage) (tx *Transaction, err error) {
	valueret := method.Func.Call([]reflect.Value{base.childValue, reflect.ValueOf(action), reflect.ValueOf(msg)})
	if len(valueret) != 2 {
		return nil, types.ErrMethodNotFound
	}
	r1 := valueret[0].Interface()
	if r1 != nil {
		if r, ok := r1.(*types.Transaction); ok {
			tx = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	//参数2
	r2 := valueret[1].Interface()
	err = nil
	if r2 != nil {
		if r, ok := r2.(error); ok {
			err = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	if tx == nil && err == nil {
		return nil, types.ErrActionNotSupport
	}
	return tx, err
}

func (base *ExecTypeBase) createRPC(action string, message json.RawMessage) (*Transaction, error) {
	err := json.Unmarshal(message, &param)
	if err != nil {
		tlog.Error("CreateTx", "Error", err)
		return nil, ErrInputPara
	}
	if param.ExecName != "" && !IsAllowExecName([]byte(param.ExecName), []byte(param.ExecName)) {
		tlog.Error("CreateTx", "Error", ErrExecNameNotMatch)
		return nil, ErrExecNameNotMatch
	}

	//to地址要么是普通用户地址，要么就是执行器地址，不能为空
	if param.To == "" {
		return nil, ErrAddrNotExist
	}

	var tx *Transaction
	if param.Amount < 0 {
		return nil, ErrAmount
	}
	if param.IsToken {
		return nil, ErrNotSupport
	} else {
		tx = base.CreateCoinsTransfer(&param)
	}

	tx.Fee, err = tx.GetRealFee(MinFee)
	if err != nil {
		return nil, err
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	return tx, nil

}

func (base *ExecTypeBase) CreateTx(action string, message json.RawMessage) (*Transaction, error) {
	//先判断 FuncList 中有没有符合要求的函数 RPC_{action}
	funclist := base.GetRPCFuncMap()
	if method, ok := funclist["RPC_"+action]; ok {
		return base.callRPC(method, action, msg)
	}
	typemap := base.child.GetTypeMap()
	if _, ok := typemap[action]; ok {
		return base.createRPC(action, msg)
	}
	return nil, types.ErrActionNotSupport
}

func (base *ExecTypeBase) CreateCoinsTransfer(param *CreateTx) *Transaction {
	return nil
	/*
		transfer := base.child.GetPayload()
		to := ""
		if IsPara() {
			to = param.GetTo()
		}
			if !param.IsWithdraw {
				if param.ExecName != "" {
					v := &cty.CoinsAction_TransferToExec{TransferToExec: &AssetsTransferToExec{
						Amount: param.Amount, Note: param.GetNote(), ExecName: param.GetExecName(), To: to}}
					transfer.Value = v
					transfer.Ty = cty.CoinsActionTransferToExec
				} else {
					v := &cty.CoinsAction_Transfer{Transfer: &AssetsTransfer{
						Amount: param.Amount, Note: param.GetNote(), To: to}}
					transfer.Value = v
					transfer.Ty = cty.CoinsActionTransfer
				}
			} else {
				v := &cty.CoinsAction_Withdraw{Withdraw: &AssetsWithdraw{
					Amount: param.Amount, Note: param.GetNote(), ExecName: param.GetExecName(), To: to}}
				transfer.Value = v
				transfer.Ty = cty.CoinsActionWithdraw
			}
			if IsPara() {
				return &Transaction{Execer: []byte(nameX), Payload: Encode(transfer), To: address.ExecAddress(nameX)}
			}
			return &Transaction{Execer: []byte(nameX), Payload: Encode(transfer), To: param.GetTo()}
	*/
}
