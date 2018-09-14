package dapp

import (
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type methodType struct {
	sync.Mutex // protects counters
	method     reflect.Method
	ArgType    reflect.Type
	ReplyType  reflect.Type
	numCalls   uint
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

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

func ListMethod(action interface{}, funclist []interface{}) map[string]reflect.Method {
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
	if len(rcvr) == 0 || rcvr[0].IsNil() || rcvr[0].Kind() != reflect.Interface {
		return "", 0, nilValue
	}
	sname := rcvr[0].Elem().Type().String()
	datas := strings.Split(sname, "_")
	if len(datas) != 2 {
		return "", 0, nilValue
	}
	funcname := "Get" + datas[1]
	if _, ok := funclist[funcname]; !ok {
		return "", 0, nilValue
	}
	val := funclist[funcname].Func.Call([]reflect.Value{value})
	if len(val) == 0 || val[0].IsNil() {
		return "", 0, nilValue
	}
	return datas[1], ty, val[0]
}
