package executor

import "reflect"

var actionFunList = make(map[string]reflect.Method)
var executorFunList = make(map[string]reflect.Method)
