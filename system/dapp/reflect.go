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

// suitableMethods returns suitable Rpc methods of typ, it will report
// error using log if reportErr is true.
func listMethods(typ reflect.Type) map[string]reflect.Method {
	methods := make(map[string]reflect.Method)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		//mtype := method.Type
		mname := method.Name
		// Method must be exported.
		if method.PkgPath != "" {
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

func GetActionValue(action interface{}) (string, int32, reflect.Value) {
	var ty int32
	if a, ok := action.(ExecutorAction); ok {
		ty = a.GetTy()
	}
	value := reflect.ValueOf(action)
	getValue := value.MethodByName("GetValue")
	if getValue.IsNil() {
		return "", 0, nilValue
	}
	rcvr := getValue.Call([]reflect.Value{})
	if len(rcvr) == 0 || rcvr[0].IsNil() {
		return "", 0, nilValue
	}
	sname := reflect.Indirect(reflect.ValueOf(rcvr[0].Interface())).Type().Name()
	datas := strings.Split(sname, "_")
	if len(datas) != 2 {
		return "", 0, nilValue
	}
	get := value.MethodByName("Get" + datas[1])
	if get.IsNil() {
		return "", 0, nilValue
	}
	val := get.Call([]reflect.Value{})
	if len(val) == 0 || val[0].IsNil() {
		return "", 0, nilValue
	}
	return datas[1], ty, val[0]
}
