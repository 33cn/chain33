package types

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"unicode"

	proto "github.com/golang/protobuf/proto"
	"github.com/33cn/chain33/common/address"
)

func init() {
	rand.Seed(Now().UnixNano())
}

type LogType interface {
	Name() string
	Decode([]byte) (interface{}, error)
	Json([]byte) ([]byte, error)
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

func (l *logInfoType) Json(data []byte) ([]byte, error) {
	d, err := l.Decode(data)
	if err != nil {
		return nil, err
	}
	if msg, ok := d.(Message); ok {
		return PBToJson(msg)
	}
	jsdata, err := json.Marshal(d)
	if err != nil {
		return nil, err
	}
	return jsdata, nil
}

var executorMap = map[string]ExecutorType{}

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

func LoadExecutorType(execstr string) ExecutorType {
	//尽可能的加载执行器
	//真正的权限控制在区块执行的时候做控制
	realname := GetRealExecName([]byte(execstr))
	if exec, exist := executorMap[string(realname)]; exist {
		return exec
	}
	return nil
}

// 重构完成后删除
func CallExecNewTx(execName, action string, param interface{}) ([]byte, error) {
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
	return FormatTxEncode(execName, tx)
}

func CallCreateTx(execName, action string, param Message) ([]byte, error) {
	exec := LoadExecutorType(execName)
	if exec == nil {
		tlog.Error("callExecNewTx", "Error", "exec not found")
		return nil, ErrNotSupport
	}
	// param is interface{type, var-nil}, check with nil always fail
	if param == nil {
		tlog.Error("callExecNewTx", "Error", "param in nil")
		return nil, ErrInvalidParam
	}
	tx, err := exec.Create(action, param)
	if err != nil {
		tlog.Error("callExecNewTx", "Error", err)
		return nil, err
	}
	return FormatTxEncode(execName, tx)
}

func CreateFormatTx(execName string, payload []byte) (*Transaction, error) {
	//填写nonce,execer,to, fee 等信息, 后面会增加一个修改transaction的函数，会加上execer fee 等的修改
	tx := &Transaction{Payload: payload}
	return FormatTx(execName, tx)
}

func FormatTx(execName string, tx *Transaction) (*Transaction, error) {
	//填写nonce,execer,to, fee 等信息, 后面会增加一个修改transaction的函数，会加上execer fee 等的修改
	tx.Nonce = rand.Int63()
	tx.Execer = []byte(execName)
	//平行链，所有的to地址都是合约地址
	if IsPara() || tx.To == "" {
		tx.To = address.ExecAddress(string(tx.Execer))
	}
	var err error
	if tx.Fee == 0 {
		tx.Fee, err = tx.GetRealFee(GInt("MinFee"))
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

func FormatTxEncode(execName string, tx *Transaction) ([]byte, error) {
	tx, err := FormatTx(execName, tx)
	if err != nil {
		return nil, err
	}
	txbyte := Encode(tx)
	return txbyte, nil
}

func LoadLog(execer []byte, ty int64) LogType {
	loginfo := getLogType(execer, ty)
	if loginfo.Name == "LogReserved" {
		return nil
	}
	return newLogType(execer, ty)
}

//通过反射,解析日志
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
	QueryToJson(funcname string, message Message) ([]byte, error)
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
	GetValueTypeMap() map[string]reflect.Type
	//action function list map
	GetFuncMap() map[string]reflect.Method
	GetRPCFuncMap() map[string]reflect.Method
	GetExecFuncMap() map[string]reflect.Method
	CreateTransaction(action string, data Message) (*Transaction, error)
	// collect assets the tx deal with
	GetAssets(tx *Transaction) ([]*Asset, error)
}

type ExecTypeGet interface {
	GetTy() int32
}

type ExecTypeBase struct {
	child               ExecutorType
	childValue          reflect.Value
	actionFunList       map[string]reflect.Method
	execFuncList        map[string]reflect.Method
	actionListValueType map[string]reflect.Type
	rpclist             map[string]reflect.Method
	queryMap            map[string]reflect.Type
	forks               *Forks
}

func (base *ExecTypeBase) GetChild() ExecutorType {
	return base.child
}

func (base *ExecTypeBase) SetChild(child ExecutorType) {
	base.child = child
	base.childValue = reflect.ValueOf(child)
	base.rpclist = ListMethod(child)
	base.actionListValueType = make(map[string]reflect.Type)
	base.actionFunList = make(map[string]reflect.Method)
	base.forks = child.GetForks()

	action := child.GetPayload()
	if action == nil {
		return
	}
	base.actionFunList = ListMethod(action)
	if _, ok := base.actionFunList["XXX_OneofFuncs"]; !ok {
		return
	}
	retval := base.actionFunList["XXX_OneofFuncs"].Func.Call([]reflect.Value{reflect.ValueOf(action)})
	if len(retval) != 4 {
		panic("err XXX_OneofFuncs")
	}
	list := ListType(retval[3].Interface().([]interface{}))

	for k, v := range list {
		data := strings.Split(k, "_")
		if len(data) != 2 {
			panic("name create " + k)
		}
		base.actionListValueType["Value_"+data[1]] = v
		field := v.Field(0)
		base.actionListValueType[field.Name] = field.Type.Elem()
		_, ok := v.FieldByName(data[1])
		if !ok {
			panic("no filed " + k)
		}
	}
	//check type map is all in value type list
	typelist := base.child.GetTypeMap()
	for k := range typelist {
		if _, ok := base.actionListValueType[k]; !ok {
			panic("value type not found " + k)
		}
		if _, ok := base.actionListValueType["Value_"+k]; !ok {
			panic("value type not found " + k)
		}
	}
}

func (base *ExecTypeBase) GetForks() *Forks {
	return &Forks{}
}

func (base *ExecTypeBase) GetCryptoDriver(ty int) (string, error) {
	return "", ErrNotSupport
}

func (base *ExecTypeBase) GetCryptoType(name string) (int, error) {
	return 0, ErrNotSupport
}

func (base *ExecTypeBase) InitFuncList(list map[string]reflect.Method) {
	base.execFuncList = list
	actionList := base.GetFuncMap()
	for k, v := range actionList {
		base.execFuncList[k] = v
	}
	//查询所有的Query_ 接口, 做一个函数名称 和 类型的映射
	_, base.queryMap = BuildQueryType("Query_", base.execFuncList)
}

func (base *ExecTypeBase) GetRPCFuncMap() map[string]reflect.Method {
	return base.rpclist
}

func (base *ExecTypeBase) GetExecFuncMap() map[string]reflect.Method {
	return base.execFuncList
}

func (base *ExecTypeBase) GetName() string {
	return "typedriverbase"
}

func (base *ExecTypeBase) IsFork(height int64, key string) bool {
	if base.GetForks == nil {
		return false
	}
	return base.forks.IsFork(GetTitle(), height, key)
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
	if to, ok := getTo(payload); ok {
		return to
	}
	return tx.To
}

//三种assert的结构体,genesis 排除
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

type Amounter interface {
	GetAmount() int64
}

func (base *ExecTypeBase) Amount(tx *Transaction) (int64, error) {
	_, v, err := base.DecodePayloadValue(tx)
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

//用户看到的FromAddr
func (base *ExecTypeBase) GetViewFromToAddr(tx *Transaction) (string, string) {
	return tx.From(), tx.To
}

func (base *ExecTypeBase) GetFuncMap() map[string]reflect.Method {
	return base.actionFunList
}

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
	if IsNilP(payload) {
		return nil, ErrDecode
	}
	return payload, nil
}

func (base *ExecTypeBase) DecodePayloadValue(tx *Transaction) (string, reflect.Value, error) {
	if base.child == nil {
		return "", nilValue, ErrActionNotSupport
	}
	action, err := base.child.DecodePayload(tx)
	if err != nil || action == nil {
		tlog.Error("DecodePayload", "err", err, "exec", string(tx.Execer))
		return "", nilValue, err
	}
	name, ty, val := GetActionValue(action, base.child.GetFuncMap())
	if IsNil(val) {
		tlog.Error("GetActionValue is nil")
		return "", nilValue, ErrActionNotSupport
	}
	typemap := base.child.GetTypeMap()
	//check types is ok
	if v, ok := typemap[name]; !ok || v != ty {
		tlog.Error("GetTypeMap is not ok")
		return "", nilValue, ErrActionNotSupport
	}
	return name, val, nil
}

func (base *ExecTypeBase) ActionName(tx *Transaction) string {
	payload, err := base.child.DecodePayload(tx)
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
		err = JsonToPB(data, in)
		if err != nil {
			return nil, err
		}
		return in, nil
	}
	return nil, ErrActionNotSupport
}

func (base *ExecTypeBase) QueryToJson(funcname string, message Message) ([]byte, error) {
	if _, ok := base.queryMap[funcname]; !ok {
		return nil, ErrActionNotSupport
	}
	return PBToJson(message)
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
	if _, ok := msg.(Message); !ok {
		return nil, ErrInvalidParam
	}
	typemap := base.child.GetTypeMap()
	if _, ok := typemap[action]; ok {
		ty1 := base.actionListValueType[action]
		ty2 := reflect.TypeOf(msg).Elem()
		if ty1 != ty2 {
			return nil, ErrInvalidParam
		}
		return base.CreateTransaction(action, msg.(Message))
	}
	tlog.Error(action + " ErrActionNotSupport")
	return nil, ErrActionNotSupport
}

func (base *ExecTypeBase) GetAction(action string) (Message, error) {
	typemap := base.child.GetTypeMap()
	if _, ok := typemap[action]; ok {
		tyvalue := reflect.New(base.actionListValueType[action])
		if !tyvalue.CanInterface() {
			tlog.Error(action + " tyvalue.CanInterface error")
			return nil, ErrActionNotSupport
		}
		data, ok := tyvalue.Interface().(Message)
		if !ok {
			tlog.Error(action + " tyvalue is not Message")
			return nil, ErrActionNotSupport
		}
		return data, nil
	}
	tlog.Error(action + " ErrActionNotSupport")
	return nil, ErrActionNotSupport
}

//重构完成后删除
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
	err = JsonToPB(b, data)
	if err != nil {
		tlog.Error(action + " jsontopb  error")
		return nil, err
	}
	return base.CreateTransaction(action, data)
}

func (base *ExecTypeBase) CreateTransaction(action string, data Message) (tx *Transaction, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = ErrActionNotSupport
		}
	}()
	value := base.child.GetPayload()
	v := reflect.New(base.actionListValueType["Value_"+action])
	vn := reflect.Indirect(v)
	if vn.Kind() != reflect.Struct {
		tlog.Error("CreateTransaction vn not struct kind", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field := vn.FieldByName(action)
	if !field.IsValid() || !field.CanSet() {
		tlog.Error("CreateTransaction vn filed can't set", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field.Set(reflect.ValueOf(data))
	value2 := reflect.Indirect(reflect.ValueOf(value))
	if value2.Kind() != reflect.Struct {
		tlog.Error("CreateTransaction value2 not struct kind", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field = value2.FieldByName("Value")
	if !field.IsValid() || !field.CanSet() {
		tlog.Error("CreateTransaction value filed can't set", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	field.Set(v)

	field = value2.FieldByName("Ty")
	if !field.IsValid() || !field.CanSet() {
		tlog.Error("CreateTransaction ty filed can't set", "exectutor", base.child.GetName(), "action", action)
		return nil, ErrActionNotSupport
	}
	tymap := base.child.GetTypeMap()
	if tyid, ok := tymap[action]; ok {
		field.Set(reflect.ValueOf(tyid))
		return &Transaction{Payload: Encode(value)}, nil
	}
	return nil, ErrActionNotSupport
}

func (base *ExecTypeBase) GetAssets(tx *Transaction) ([]*Asset, error) {
	_, v, err := base.DecodePayloadValue(tx)
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
	return []*Asset{asset}, nil
}
