// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	proto "github.com/golang/protobuf/proto"
)

func buildFuncList(funclist []interface{}) map[string]bool {
	list := make(map[string]bool)
	for i := 0; i < len(funclist); i++ {
		tyname := reflect.TypeOf(funclist[i]).Elem().Name()
		datas := strings.Split(tyname, "_")
		if len(datas) != 2 {
			continue
		}
		list["Get"+datas[1]] = true
	}
	return list
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

func ListActionMethod(action interface{}, funclist []interface{}) map[string]reflect.Method {
	typ := reflect.TypeOf(action)
	flist := buildFuncList(funclist)
	methods := make(map[string]reflect.Method)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		//mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" || !isExported(mname) {
			continue
		}
		if mname == "GetValue" {
			methods[mname] = method
			continue
		}
		if flist[mname] {
			methods[mname] = method
		}
	}
	return methods
}

func ListType(tys []interface{}) map[string]reflect.Type {
	typelist := make(map[string]reflect.Type)
	for _, ty := range tys {
		typ := reflect.TypeOf(ty).Elem()
		typelist[typ.Name()] = typ
	}
	return typelist
}

func ListMethod(action interface{}) map[string]reflect.Method {
	typ := reflect.TypeOf(action)
	return ListMethodByType(typ)
}

func ListMethodByType(typ reflect.Type) map[string]reflect.Method {
	methods := make(map[string]reflect.Method)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		//mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" || !isExported(mname) {
			continue
		}
		methods[mname] = method
	}
	return methods
}

type ExecutorAction interface {
	GetTy() int32
}

var nilValue = reflect.ValueOf(nil)

func GetActionValue(action interface{}, funclist map[string]reflect.Method) (string, int32, reflect.Value) {
	var ty int32
	if a, ok := action.(ExecutorAction); ok {
		ty = a.GetTy()
	}
	value := reflect.ValueOf(action)
	if _, ok := funclist["GetValue"]; !ok {
		return "", 0, nilValue
	}
	rcvr := funclist["GetValue"].Func.Call([]reflect.Value{value})
	if !IsOK(rcvr, 1) || IsNil(rcvr[0]) {
		return "", 0, nilValue
	}
	if rcvr[0].Kind() != reflect.Ptr && rcvr[0].Kind() != reflect.Interface {
		return "", 0, nilValue
	}
	elem := rcvr[0].Elem()
	if IsNil(elem) {
		return "", 0, nilValue
	}
	sname := elem.Type().String()
	datas := strings.Split(sname, "_")
	if len(datas) != 2 {
		return "", 0, nilValue
	}
	funcname := "Get" + datas[1]
	if _, ok := funclist[funcname]; !ok {
		return "", 0, nilValue
	}
	val := funclist[funcname].Func.Call([]reflect.Value{value})
	if !IsOK(val, 1) || IsNil(val[0]) {
		return "", 0, nilValue
	}
	return datas[1], ty, val[0]
}

func IsOK(list []reflect.Value, n int) bool {
	if len(list) != n {
		return false
	}
	for i := 0; i < len(list); i++ {
		if !IsNil(list[i]) && !list[i].CanInterface() {
			return false
		}
	}
	return true
}

func CallQueryFunc(this reflect.Value, f reflect.Method, in Message) (reply Message, err error) {
	valueret := f.Func.Call([]reflect.Value{this, reflect.ValueOf(in)})
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
		if r, ok := r1.(Message); ok {
			reply = r
		} else {
			return nil, ErrMethodReturnType
		}
	}
	//参数2
	r2 := valueret[1].Interface()
	if r2 != nil {
		if r, ok := r2.(error); ok {
			err = r
		} else {
			return nil, ErrMethodReturnType
		}
	}
	if reply == nil && err == nil {
		return nil, ErrActionNotSupport
	}
	return reply, err
}

func BuildQueryType(prefix string, methods map[string]reflect.Method) (map[string]reflect.Method, map[string]reflect.Type) {
	tys := make(map[string]reflect.Type)
	ms := make(map[string]reflect.Method)
	for name, method := range methods {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		ty := method.Type
		if ty.NumIn() != 2 {
			continue
		}
		paramIn := ty.In(1)
		if paramIn.Kind() != reflect.Ptr {
			continue
		}
		p := reflect.New(ty.In(1).Elem())
		queryin := p.Interface()
		if _, ok := queryin.(proto.Message); !ok {
			continue
		}
		if ty.NumOut() != 2 {
			continue
		}
		if !ty.Out(0).AssignableTo(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
			continue
		}
		if !ty.Out(1).AssignableTo(reflect.TypeOf((*error)(nil)).Elem()) {
			continue
		}
		name = name[len(prefix):]
		tys[name] = ty
		ms[name] = method
	}
	return ms, tys
}

type QueryData struct {
	sync.RWMutex
	prefix   string
	funcMap  map[string]map[string]reflect.Method
	typeMap  map[string]map[string]reflect.Type
	valueMap map[string]reflect.Value
}

func NewQueryData(prefix string) *QueryData {
	data := &QueryData{
		prefix:   prefix,
		funcMap:  make(map[string]map[string]reflect.Method),
		typeMap:  make(map[string]map[string]reflect.Type),
		valueMap: make(map[string]reflect.Value),
	}
	return data
}

func (q *QueryData) Register(key string, obj interface{}) {
	if _, existed := q.funcMap[key]; existed {
		panic("QueryData reg dup")
	}
	q.funcMap[key], q.typeMap[key] = BuildQueryType(q.prefix, ListMethod(obj))
}

func (q *QueryData) SetThis(key string, this reflect.Value) {
	q.Lock()
	defer q.Unlock()
	q.valueMap[key] = this
}

func (q *QueryData) getThis(key string) (reflect.Value, bool) {
	q.RLock()
	defer q.RUnlock()
	v, ok := q.valueMap[key]
	return v, ok
}

func (q *QueryData) GetFunc(driver, name string) (reflect.Method, error) {
	funclist, ok := q.funcMap[driver]
	if !ok {
		return reflect.Method{}, ErrActionNotSupport
	}
	if f, ok := funclist[name]; ok {
		return f, nil
	}
	return reflect.Method{}, ErrActionNotSupport
}

func (q *QueryData) GetType(driver, name string) (reflect.Type, error) {
	typelist, ok := q.typeMap[driver]
	if !ok {
		return nil, ErrActionNotSupport
	}
	if t, ok := typelist[name]; ok {
		return t, nil
	}
	return nil, ErrActionNotSupport
}

func (q *QueryData) Decode(driver, name string, in []byte) (reply Message, err error) {
	ty, err := q.GetType(driver, name)
	if err != nil {
		return nil, err
	}
	p := reflect.New(ty.In(1).Elem())
	queryin := p.Interface()
	if paramIn, ok := queryin.(proto.Message); ok {
		err = Decode(in, paramIn)
		return paramIn, err
	}
	return nil, ErrActionNotSupport
}

func (q *QueryData) DecodeJson(driver, name string, in json.Marshaler) (reply Message, err error) {
	ty, err := q.GetType(driver, name)
	if err != nil {
		return nil, err
	}
	p := reflect.New(ty.In(1).Elem())
	queryin := p.Interface()
	if paramIn, ok := queryin.(proto.Message); ok {
		data, err := in.MarshalJSON()
		if err != nil {
			return nil, err
		}
		err = JsonToPB(data, paramIn)
		return paramIn, err
	}
	return nil, ErrActionNotSupport
}

func (q *QueryData) Call(driver, name string, in Message) (reply Message, err error) {
	defer func() {
		return
		if r := recover(); r != nil {
			tlog.Error("query data call error", "driver", driver, "name", name, "param", in, "msg", r)
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
			reply = nil
		}
	}()
	f, err := q.GetFunc(driver, name)
	if err != nil {
		return nil, err
	}
	m, ok := q.getThis(driver)
	if !ok {
		return nil, ErrQueryThistIsNotSet
	}
	return CallQueryFunc(m, f, in)
}

//判断所有的空值
func IsNil(a interface{}) (ok bool) {
	defer func() {
		if e := recover(); e != nil {
			panic(e)
			ok = false
		}
	}()
	if v, ok := a.(reflect.Value); ok {
		if !v.IsValid() {
			return true
		}
		return v.IsNil()
	}
	return a == nil || reflect.ValueOf(a).IsNil()
}

//空指针或者接口
func IsNilP(a interface{}) bool {
	if a == nil {
		return true
	}
	var v reflect.Value
	if val, ok := a.(reflect.Value); ok {
		v = val
	} else {
		v = reflect.ValueOf(a)
	}
	if !v.IsValid() {
		return true
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		return v.IsNil()
	}
	return false
}
