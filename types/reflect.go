package types

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
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
