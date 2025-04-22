// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"unicode"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/33cn/chain33/common/address"
	"github.com/golang/protobuf/proto"
)

func init() {
	rand.Seed(Now().UnixNano())
}

// LogType 结构体类型
type LogType interface {
	Name() string
	Decode([]byte) (interface{}, error)
	JSON([]byte) ([]byte, error)
}

type logInfoType struct {
	ty     int64
	execer []byte
}

func newLogType(execer []byte, ty int64) LogType {
	return &logInfoType{ty: ty, execer: execer}
}

func (l *logInfoType) Name() string {
	return GetLogName(l.execer, l.ty)
}

func (l *logInfoType) Decode(data []byte) (interface{}, error) {
	return DecodeLog(l.execer, l.ty, data)
}

func (l *logInfoType) JSON(data []byte) ([]byte, error) {
	d, err := l.Decode(data)
	if err != nil {
		return nil, err
	}
	if msg, ok := d.(Message); ok {
		return PBToJSON(msg)
	}
	jsdata, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return jsdata, nil
}

var executorMap = map[string]ExecutorType{}

// RegistorExecutor 注册执行器
func RegistorExecutor(exec string, util ExecutorType) {
	//tlog.Debug("rpc", "t", funcName, "t", util)
	if util.GetChild() == nil {
		panic("exec " + exec + " executorType child is nil")
	}
	if _, exist := executorMap[exec]; exist {
		panic("DupExecutorType")
	} else {
		executorMap[exec] = util
	}
}

// LoadExecutorType 加载执行器
func LoadExecutorType(execstr string) ExecutorType {
	//尽可能的加载执行器
	//真正的权限控制在区块执行的时候做控制
	realname := GetRealExecName([]byte(execstr))
	if exec, exist := executorMap[string(realname)]; exist {
		return exec
	}
	return nil
}

// CallExecNewTx 重构完成后删除
func CallExecNewTx(c *Chain33Config, execName, action string, param interface{}) ([]byte, error) {
	exec := LoadExecutorType(execName)
	if exec == nil {
		tlog.Error("callExecNewTx", "Error", "exec not found")
		return nil, ErrNotSupport
	}
	// param is interface{type, var-nil}, check with nil always fail
	if reflect.ValueOf(param).IsNil() {
		tlog.Error("callExecNewTx", "Error", "param in nil")
		return nil, ErrInvalidParam
	}
	jsonStr, err := json.Marshal(param)
	if err != nil {
		tlog.Error("callExecNewTx", "Error", err)
		return nil, err
	}
	tx, err := exec.CreateTx(action, json.RawMessage(jsonStr))
	if err != nil {
		tlog.Error("callExecNewTx", "Error", err)
		return nil, err
	}
	return FormatTxEncode(c, execName, tx)
}

// CallCreateTransaction 创建一个交易
func CallCreateTransaction(execName, action string, param Message) (*Transaction, error) {
	exec := LoadExecutorType(execName)
	if exec == nil {
		tlog.Error("CallCreateTx", "Error", "exec not found")
		return nil, ErrNotSupport
	}
	// param is interface{type, var-nil}, check with nil always fail
	if param == nil {
		tlog.Error("CallCreateTx", "Error", "param in nil")
		return nil, ErrInvalidParam
	}
	return exec.Create(action, param)
}

// CallCreateTx 构造交易信息
func CallCreateTx(c *Chain33Config, execName, action string, param Message) ([]byte, error) {
	tx, err := CallCreateTransaction(execName, action, param)
	if err != nil {
		return nil, err
	}
	return FormatTxEncode(c, execName, tx)
}

// CallCreateTxJSON create tx by json
func CallCreateTxJSON(c *Chain33Config, execName, action string, param json.RawMessage) ([]byte, error) {
	exec := LoadExecutorType(execName)
	if exec == nil {
		execer := GetParaExecName([]byte(execName))
		//找不到执行器，并且是user.xxx 的情况下
		if bytes.HasPrefix(execer, UserKey) {
			tx := &Transaction{Payload: param}
			return FormatTxEncode(c, execName, tx)
		}
		tlog.Error("CallCreateTxJSON", "Error", "exec not found")
		return nil, ErrExecNotFound
	}
	// param is interface{type, var-nil}, check with nil always fail
	if param == nil {
		tlog.Error("CallCreateTxJSON", "Error", "param in nil")
		return nil, ErrInvalidParam
	}
	tx, err := exec.CreateTx(action, param)
	if err != nil {
		tlog.Error("CallCreateTxJSON", "Error", err)
		return nil, err
	}
	return FormatTxEncode(c, execName, tx)
}

// CreateFormatTx 构造交易信息
func CreateFormatTx(c *Chain33Config, execName string, payload []byte) (*Transaction, error) {
	//填写nonce,execer,to, fee 等信息, 后面会增加一个修改transaction的函数，会加上execer fee 等的修改
	tx := &Transaction{Payload: payload}
	return FormatTx(c, execName, tx)
}

// FormatTx 格式化tx交易
func FormatTx(c *Chain33Config, execName string, tx *Transaction) (*Transaction, error) {
	//填写nonce,execer,to, fee 等信息, 后面会增加一个修改transaction的函数，会加上execer fee 等的修改
	tx.Nonce = rand.Int63()
	tx.ChainID = c.GetChainID()
	tx.Execer = []byte(execName)
	//平行链，所有的to地址都是合约地址
	if c.IsPara() || tx.To == "" {
		tx.To = address.ExecAddress(string(tx.Execer))
	}
	var err error
	if tx.Fee == 0 {
		tx.Fee, err = tx.GetRealFee(c.GetMinTxFeeRate())
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// FormatTxExt 根据输入参数格式化tx
func FormatTxExt(chainID int32, isPara bool, minFee int64, execName string, tx *Transaction) (*Transaction, error) {
	//填写nonce,execer,to, fee 等信息, 后面会增加一个修改transaction的函数，会加上execer fee 等的修改
	tx.Nonce = rand.Int63()
	tx.ChainID = chainID
	tx.Execer = []byte(execName)
	//平行链，所有的to地址都是合约地址
	if isPara || tx.To == "" {
		tx.To = address.ExecAddress(string(tx.Execer))
	}
	var err error
	if tx.Fee == 0 {
		tx.Fee, err = tx.GetRealFee(minFee)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

// FormatTxEncode 对交易信息编码成byte类型
func FormatTxEncode(c *Chain33Config, execName string, tx *Transaction) ([]byte, error) {
	tx, err := FormatTx(c, execName, tx)
	if err != nil {
		return nil, err
	}
	txbyte := Encode(tx)
	return txbyte, nil
}

// LoadLog 加载log类型
func LoadLog(execer []byte, ty int64) LogType {
	loginfo := getLogType(execer, ty)
	if loginfo.Name == "LogReserved" {
		return nil
	}
	return newLogType(execer, ty)
}

// GetLogName 通过反射,解析日志
func GetLogName(execer []byte, ty int64) string {
	t := getLogType(execer, ty)
	return t.Name
}

func getLogType(execer []byte, ty int64) *LogInfo {
	//加载执行器已经定义的log
	if execer != nil {
		ety := LoadExecutorType(string(execer))
		if ety != nil {
			logmap := ety.GetLogMap()
			if logty, ok := logmap[ty]; ok {
				return logty
			}
		}
	}
	//如果没有，那么用系统默认类型列表
	if logty, ok := SystemLog[ty]; ok {
		return logty
	}
	//否则就是默认类型
	return SystemLog[0]
}

// DecodeLog 解析log信息
func DecodeLog(execer []byte, ty int64, data []byte) (interface{}, error) {
	t := getLogType(execer, ty)
	if t.Name == "LogErr" || t.Name == "LogReserved" {
		msg := string(data)
		return msg, nil
	}
	pdata := reflect.New(t.Ty)
	if !pdata.CanInterface() {
		return nil, ErrDecode
	}
	msg, ok := pdata.Interface().(Message)
	if !ok {
		return nil, ErrDecode
	}
	err := Decode(data, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// ExecutorType  执行器接口
type ExecutorType interface {
	//获取交易真正的to addr
	GetRealToAddr(tx *Transaction) string
	GetCryptoDriver(ty int) (string, error)
	GetCryptoType(name string) (int, error)
	//给用户显示的from 和 to
	GetViewFromToAddr(tx *Transaction) (string, string)
	ActionName(tx *Transaction) string
	//新版本使用create接口，createTx 重构以后就废弃
	GetAction(action string) (Message, error)
	InitFuncList(list map[string]reflect.Method)
	Create(action string, message Message) (*Transaction, error)
	CreateTx(action string, message json.RawMessage) (*Transaction, error)
	CreateQuery(funcname string, message json.RawMessage) (Message, error)
	AssertCreate(createTx *CreateTx) (*Transaction, error)
	QueryToJSON(funcname string, message Message) ([]byte, error)
	Amount(tx *Transaction) (int64, error)
	DecodePayload(tx *Transaction) (Message, error)
	DecodePayloadValue(tx *Transaction) (string, reflect.Value, error)
	//write for executor
	GetPayload() Message
	GetChild() ExecutorType
	GetName() string
	//exec result of receipt log
	GetLogMap() map[int64]*LogInfo
	GetForks() *Forks
	IsFork(height int64, key string) bool
	//actionType -> name map
	GetTypeMap() map[string]int32
	//action function list map
	GetFuncMap() map[string]reflect.Method
	GetRPCFuncMap() map[string]reflect.Method
	GetExecFuncMap() map[string]reflect.Method
	CreateTransaction(action string, data Message) (*Transaction, error)
	// collect assets the tx deal with
	GetAssets(tx *Transaction) ([]*Asset, error)

	// about chain33Config
	GetConfig() *Chain33Config
	SetConfig(cfg *Chain33Config)
}

// ExecTypeGet  获取类型值
type execTypeGet interface {
	GetTy() int32
}

// ExecTypeBase  执行类型
type ExecTypeBase struct {
	child         ExecutorType
	childValue    reflect.Value
	actionFunList map[string]reflect.Method
	execFuncList  map[string]reflect.Method
	//actionListValueType    map[string]reflect.Type
	rpclist                map[string]reflect.Method
	queryMap               map[string]reflect.Type
	forks                  *Forks
	cfg                    *Chain33Config
	actionMsgDescriptor    protoreflect.MessageDescriptor
	fieldName              map[string]protoreflect.Name
	actionFieldDescriptors protoreflect.FieldDescriptors
}

// GetChild  获取子执行器
func (base *ExecTypeBase) GetChild() ExecutorType {
	return base.child
}

// SetChild  设置子执行器
func (base *ExecTypeBase) SetChild(child ExecutorType) {
	base.child = child
	base.childValue = reflect.ValueOf(child)
	base.rpclist = ListMethod(child)
	base.actionFunList = make(map[string]reflect.Method)
	base.fieldName = make(map[string]protoreflect.Name)
	base.forks = child.GetForks()

	action := child.GetPayload()
	if action == nil {
		return
	}
	base.actionFunList = ListMethod(action)
	actionDescriptor := proto.MessageV2(action).ProtoReflect().Descriptor()
	// 合约action不存在oneof类型, 不支持反射构造交易
	if actionDescriptor.Oneofs().Len() <= 0 {
		return
	}
	base.actionMsgDescriptor = actionDescriptor
	base.actionFieldDescriptors = actionDescriptor.Fields()

	// field名称是字段tag名称
	for i := 0; i < base.actionFieldDescriptors.Len(); i++ {
		field := base.actionFieldDescriptors.Get(i)
		base.setActionFieldName(field.Name())
	}

	//check type map is all in value type list
	typelist := base.child.GetTypeMap()
	for k := range typelist {
		fieldName := base.getActionFieldName(k)
		if base.actionFieldDescriptors.ByName(fieldName) == nil {
			panic("value type not found " + k)
		}
	}
}

// GetForks  获取fork信息
func (base *ExecTypeBase) GetForks() *Forks {
	return &Forks{}
}

// GetCryptoDriver  获取Crypto驱动
func (base *ExecTypeBase) GetCryptoDriver(ty int) (string, error) {
	return "", ErrNotSupport
}

// GetCryptoType  获取Crypto类型
func (base *ExecTypeBase) GetCryptoType(name string) (int, error) {
	return 0, ErrNotSupport
}

// InitFuncList  初始化函数列表
func (base *ExecTypeBase) InitFuncList(list map[string]reflect.Method) {
	base.execFuncList = list
	actionList := base.GetFuncMap()
	for k, v := range actionList {
		base.execFuncList[k] = v
	}
	//查询所有的Query_ 接口, 做一个函数名称 和 类型的映射
	_, base.queryMap = BuildQueryType("Query_", base.execFuncList)
}

// GetRPCFuncMap  获取rpc的接口列表
func (base *ExecTypeBase) GetRPCFuncMap() map[string]reflect.Method {
	return base.rpclist
}

// GetExecFuncMap  获取执行交易的接口列表
func (base *ExecTypeBase) GetExecFuncMap() map[string]reflect.Method {
	return base.execFuncList
}

// GetName  获取name
func (base *ExecTypeBase) GetName() string {
	return "typedriverbase"
}

// IsFork  是否fork高度
func (base *ExecTypeBase) IsFork(height int64, key string) bool {
	if base.GetForks() == nil {
		return false
	}
	return base.forks.IsFork(height, key)
}

// GetRealToAddr 用户看到的ToAddr
func (base *ExecTypeBase) GetRealToAddr(tx *Transaction) string {
	if !base.cfg.IsPara() {
		return tx.To
	}
	//平行链中的处理方式
	_, v, err := base.child.DecodePayloadValue(tx)
	if err != nil {
		return tx.To
	}
	payload := v.Interface()
	if to, ok := getTo(payload); ok {
		return to
	}
	return tx.To
}

// 三种assert的结构体,genesis 排除
func getTo(payload interface{}) (string, bool) {
	if ato, ok := payload.(*AssetsTransfer); ok {
		return ato.GetTo(), true
	}
	if ato, ok := payload.(*AssetsWithdraw); ok {
		return ato.GetTo(), true
	}
	if ato, ok := payload.(*AssetsTransferToExec); ok {
		return ato.GetTo(), true
	}
	return "", false
}

// IsAssetsTransfer 是否是资产转移相关的交易
func IsAssetsTransfer(payload interface{}) bool {
	if _, ok := payload.(*AssetsTransfer); ok {
		return true
	}
	if _, ok := payload.(*AssetsWithdraw); ok {
		return true
	}
	if _, ok := payload.(*AssetsTransferToExec); ok {
		return true
	}
	return false
}

// Amounter 转账金额
type Amounter interface {
	GetAmount() int64
}

// Amount 获取tx交易中的转账金额
func (base *ExecTypeBase) Amount(tx *Transaction) (int64, error) {
	_, v, err := base.child.DecodePayloadValue(tx)
	if err != nil {
		return 0, err
	}
	payload := v.Interface()
	//四种assert的结构体
	if ato, ok := payload.(Amounter); ok {
		return ato.GetAmount(), nil
	}
	return 0, nil
}

// GetViewFromToAddr 用户看到的FromAddr
func (base *ExecTypeBase) GetViewFromToAddr(tx *Transaction) (string, string) {
	return tx.From(), tx.To
}

// GetFuncMap 获取函数列表
func (base *ExecTypeBase) GetFuncMap() map[string]reflect.Method {
	return base.actionFunList
}

// DecodePayload 解析tx交易中的payload
func (base *ExecTypeBase) DecodePayload(tx *Transaction) (Message, error) {
	if base.child == nil {
		return nil, ErrActionNotSupport
	}
	payload := base.child.GetPayload()
	if payload == nil {
		return nil, ErrActionNotSupport
	}
	err := Decode(tx.GetPayload(), payload)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

// DecodePayloadValue 解析tx交易中的payload具体Value值
func (base *ExecTypeBase) DecodePayloadValue(tx *Transaction) (string, reflect.Value, error) {
	name, value, err := base.decodePayloadValue(tx)
	return name, value, err
}

func (base *ExecTypeBase) decodePayloadValue(tx *Transaction) (string, reflect.Value, error) {
	if base.child == nil {
		return "", nilValue, ErrActionNotSupport
	}
	action, err := base.child.DecodePayload(tx)
	if err != nil || action == nil {
		// 目前存证交易会解析失败, 这个错误不会影响
		tlog.Debug("DecodePayload", "err", err, "exec", string(tx.Execer))
		return "", nilValue, err
	}
	name, ty, val, err := GetActionValue(action, base.child.GetFuncMap())
	if err != nil {
		return "", nilValue, err
	}
	typemap := base.child.GetTypeMap()
	//check types is ok
	if v, ok := typemap[name]; !ok || v != ty {
		tlog.Error("GetTypeMap is not ok")
		return "", nilValue, ErrActionNotSupport
	}
	return name, val, nil
}

// ActionName 获取交易中payload的action name
func (base *ExecTypeBase) ActionName(tx *Transaction) string {
	payload, err := base.child.DecodePayload(tx)
	if err != nil {
		return "unknown-err"
	}
	tm := base.child.GetTypeMap()
	if get, ok := payload.(execTypeGet); ok {
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

// CreateQuery Query接口查询
func (base *ExecTypeBase) CreateQuery(funcname string, message json.RawMessage) (Message, error) {
	if _, ok := base.queryMap[funcname]; !ok {
		return nil, ErrActionNotSupport
	}
	ty := base.queryMap[funcname]
	p := reflect.New(ty.In(1).Elem())
	queryin := p.Interface()
	if in, ok := queryin.(proto.Message); ok {
		data, err := message.MarshalJSON()
		if err != nil {
			return nil, err
		}
		err = JSONToPB(data, in)
		if err != nil {
			return nil, err
		}
		return in, nil
	}
	return nil, ErrActionNotSupport
}

// QueryToJSON 转换成json格式
func (base *ExecTypeBase) QueryToJSON(funcname string, message Message) ([]byte, error) {
	if _, ok := base.queryMap[funcname]; !ok {
		return nil, ErrActionNotSupport
	}
	return PBToJSON(message)
}

func (base *ExecTypeBase) callRPC(method reflect.Method, action string, msg interface{}) (tx *Transaction, err error) {
	valueret := method.Func.Call([]reflect.Value{base.childValue, reflect.ValueOf(action), reflect.ValueOf(msg)})
	if len(valueret) != 2 {
		return nil, ErrMethodNotFound
	}
	if !valueret[0].CanInterface() {
		return nil, ErrMethodNotFound
	}
	if !valueret[1].CanInterface() {
		return nil, ErrMethodNotFound
	}
	r1 := valueret[0].Interface()
	if r1 != nil {
		if r, ok := r1.(*Transaction); ok {
			tx = r
		} else {
			return nil, ErrMethodReturnType
		}
	}
	//参数2
	r2 := valueret[1].Interface()
	err = nil
	if r2 != nil {
		if r, ok := r2.(error); ok {
			err = r
		} else {
			return nil, ErrMethodReturnType
		}
	}
	if tx == nil && err == nil {
		return nil, ErrActionNotSupport
	}
	return tx, err
}

// AssertCreate 构造assets资产交易
func (base *ExecTypeBase) AssertCreate(c *CreateTx) (*Transaction, error) {
	if c.ExecName != "" && !IsAllowExecName([]byte(c.ExecName), []byte(c.ExecName)) {
		tlog.Error("CreateTx", "Error", ErrExecNameNotMatch)
		return nil, ErrExecNameNotMatch
	}
	if c.Amount < 0 {
		return nil, ErrAmount
	}
	if c.IsWithdraw {
		p := &AssetsWithdraw{Cointoken: c.GetTokenSymbol(), Amount: c.Amount,
			Note: c.Note, ExecName: c.ExecName, To: c.To}
		return base.child.CreateTransaction("Withdraw", p)
	}
	if c.ExecName != "" {
		v := &AssetsTransferToExec{Cointoken: c.GetTokenSymbol(), Amount: c.Amount,
			Note: c.Note, ExecName: c.ExecName, To: c.To}
		return base.child.CreateTransaction("TransferToExec", v)
	}
	v := &AssetsTransfer{Cointoken: c.GetTokenSymbol(), Amount: c.Amount, Note: c.GetNote(), To: c.To}
	return base.child.CreateTransaction("Transfer", v)
}

// Create 构造tx交易
func (base *ExecTypeBase) Create(action string, msg Message) (*Transaction, error) {
	//先判断 FuncList 中有没有符合要求的函数 RPC_{action}
	if msg == nil {
		return nil, ErrInvalidParam
	}
	if action == "" {
		action = "Default_Process"
	}
	funclist := base.GetRPCFuncMap()
	if method, ok := funclist["RPC_"+action]; ok {
		return base.callRPC(method, action, msg)
	}

	typemap := base.child.GetTypeMap()
	fieldName := base.getActionFieldName(action)
	if _, ok := typemap[action]; ok && fieldName != "" {
		fieldDes := base.actionFieldDescriptors.ByName(fieldName)
		// 判断msg结构名称是否一致
		if fieldDes.Message().FullName() != proto.MessageV2(msg).ProtoReflect().Descriptor().FullName() {
			return nil, ErrInvalidParam
		}
		return base.CreateTransaction(action, msg)
	}
	tlog.Error(action + " ErrActionNotSupport")
	return nil, ErrActionNotSupport
}

// GetAction 获取action
func (base *ExecTypeBase) GetAction(action string) (Message, error) {
	typemap := base.child.GetTypeMap()
	fieldName := base.getActionFieldName(action)
	if _, ok := typemap[action]; ok && fieldName != "" {
		return dynamicpb.NewMessage(base.actionFieldDescriptors.ByName(fieldName).Message()), nil
	}
	tlog.Error(action + " ErrActionNotSupport")
	return nil, ErrActionNotSupport
}

// CreateTx 通过json rpc 创建交易
func (base *ExecTypeBase) CreateTx(action string, msg json.RawMessage) (*Transaction, error) {
	data, err := base.GetAction(action)
	if err != nil {
		return nil, err
	}
	b, err := msg.MarshalJSON()
	if err != nil {
		tlog.Error(action + " MarshalJSON  error")
		return nil, err
	}
	err = JSONToPB(b, data)
	if err != nil {
		tlog.Error(action + " jsontopb  error")
		return nil, err
	}
	return base.CreateTransaction(action, data)
}

func (base *ExecTypeBase) getActionFieldName(action string) protoreflect.Name {
	return base.fieldName[strings.ToLower(action)]
}

func (base *ExecTypeBase) setActionFieldName(name protoreflect.Name) {
	base.fieldName[strings.ToLower(string(name))] = name

}

// CreateTransaction 构造Transaction
func (base *ExecTypeBase) CreateTransaction(action string, data Message) (tx *Transaction, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrActionNotSupport
		}
	}()
	if base.actionFieldDescriptors == nil {
		return nil, ErrActionNotSupport
	}
	// 兼容protobuf不同版本msg, 统一使用反射类型
	if _, ok := data.(*dynamicpb.Message); !ok {
		fieldName := base.getActionFieldName(action)
		msg := dynamicpb.NewMessage(base.actionFieldDescriptors.ByName(fieldName).Message())
		proto.Merge(msg, data)
		data = msg
	}

	value := dynamicpb.NewMessage(base.actionMsgDescriptor)
	fieldName := base.getActionFieldName(action)
	value.Set(base.actionFieldDescriptors.ByName(fieldName), protoreflect.ValueOf(data))

	tyField := base.actionFieldDescriptors.ByName(base.getActionFieldName("ty"))
	if tyField == nil {
		tlog.Error("CreateTransaction ty filed not exist", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	if ty, ok := base.child.GetTypeMap()[action]; ok {
		value.Set(tyField, protoreflect.ValueOfInt32(ty))
		tx := &Transaction{
			Payload: Encode(value),
		}
		return tx, nil
	}
	return nil, ErrActionNotSupport
}

// GetAssets 获取资产信息
func (base *ExecTypeBase) GetAssets(tx *Transaction) ([]*Asset, error) {
	_, v, err := base.child.DecodePayloadValue(tx)
	if err != nil {
		return nil, err
	}
	payload := v.Interface()
	asset := &Asset{Exec: string(tx.Execer)}
	if a, ok := payload.(*AssetsTransfer); ok {
		asset.Symbol = a.Cointoken
	} else if a, ok := payload.(*AssetsWithdraw); ok {
		asset.Symbol = a.Cointoken
	} else if a, ok := payload.(*AssetsTransferToExec); ok {
		asset.Symbol = a.Cointoken
	} else {
		return nil, nil
	}
	amount, err := tx.Amount()
	if err != nil {
		return nil, nil
	}
	asset.Amount = amount
	return []*Asset{asset}, nil
}

// GetConfig ...
func (base *ExecTypeBase) GetConfig() *Chain33Config {
	return base.cfg
}

// SetConfig ...
func (base *ExecTypeBase) SetConfig(cfg *Chain33Config) {
	base.cfg = cfg
}
