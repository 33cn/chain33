// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type T struct {
	a int
	b *int
}

func (t *T) Query_Add(in *ReqNil) (Message, error) {
	result := t.a + *t.b
	return &Int64{Data: int64(result)}, nil
}

var qdata = NewQueryData("Query_")

func TestValue(t *testing.T) {
	b := 20
	data := &T{10, &b}
	qdata.Register("T", data)
	qdata.SetThis("T", reflect.ValueOf(data))

	reply, err := qdata.Call("T", "Add", &ReqNil{})
	assert.Nil(t, err)
	assert.Equal(t, reply.(*Int64).Data, int64(30))

	*data.b = 30
	reply, err = qdata.Call("T", "Add", &ReqNil{})
	assert.Nil(t, err)
	assert.Equal(t, reply.(*Int64).Data, int64(40))

	data.a = 30
	reply, err = qdata.Call("T", "Add", &ReqNil{})
	assert.Nil(t, err)
	assert.Equal(t, reply.(*Int64).Data, int64(60))

	data2 := &T{10, &b}
	qdata.SetThis("T", reflect.ValueOf(data2))
	reply, err = qdata.Call("T", "Add", &ReqNil{})
	assert.Nil(t, err)
	assert.Equal(t, reply.(*Int64).Data, int64(40))
}

func BenchmarkCallOrigin(b *testing.B) {
	bb := 20
	data := &T{10, &bb}
	result := int64(0)
	for i := 0; i < b.N; i++ {
		reply, _ := data.Query_Add(nil)
		result += reply.(*Int64).Data
	}
	assert.Equal(b, result, int64(b.N*30))
}

func init() {
	bb := 20
	data := &T{10, &bb}
	qdata.Register("TT", data)
	qdata.SetThis("TT", reflect.ValueOf(data))
}

func BenchmarkCallQueryData(b *testing.B) {
	result := int64(0)
	for i := 0; i < b.N; i++ {
		reply, _ := qdata.Call("TT", "Add", &ReqNil{})
		result += reply.(*Int64).Data
	}
	assert.Equal(b, result, int64(b.N*30))
}

func TestIsOK(t *testing.T) {
	data := make([]reflect.Value, 2)
	var err interface{}
	data[0] = reflect.ValueOf(&ReqNil{})
	data[1] = reflect.ValueOf(err)
	assert.Equal(t, reflect.Invalid, data[1].Kind())
	assert.Equal(t, true, IsNil(data[1]))
	assert.Equal(t, true, IsOK(data, 2))
}

func TestIsExported(t *testing.T) {
	assert.True(t, isExported("TestFunc"))
	assert.True(t, isExported("Query_Add"))
	assert.False(t, isExported("testFunc"))
	assert.False(t, isExported("_private"))
	assert.False(t, isExported(""))
}

func TestBuildFuncList(t *testing.T) {
	type Dapp_Action struct{}
	type Dapp_Query struct{}

	var list []interface{}
	list = append(list, (*Dapp_Action)(nil))
	list = append(list, (*Dapp_Query)(nil))
	result := buildFuncList(list)
	assert.True(t, result["GetAction"])
	assert.True(t, result["GetQuery"])
	assert.False(t, result["GetNope"])
}

func TestListActionMethod(t *testing.T) {
	b := 20
	data := &T{10, &b}
	methods := ListActionMethod(data, []interface{}{(*T)(nil)})
	assert.NotNil(t, methods["GetValue"])
	assert.NotNil(t, methods["Query_Add"])
}

func TestListType(t *testing.T) {
	tys := []interface{}{(*Int64)(nil), (*ReqNil)(nil)}
	result := ListType(tys)
	assert.NotNil(t, result["Int64"])
	assert.NotNil(t, result["ReqNil"])
	assert.Nil(t, result["NonExist"])
}

func TestListMethod(t *testing.T) {
	b := 20
	data := &T{10, &b}
	methods := ListMethod(data)
	assert.NotNil(t, methods["Query_Add"])
	assert.NotNil(t, methods["GetValue"])
}

func TestGetActionValue(t *testing.T) {
	// GetActionValue needs an execTypeGet interface, which provides GetTy()
	// Use a simple struct through the QueryData/reflect infrastructure
	q := NewQueryData("Query_")
	b := 20
	data := &T{10, &b}
	q.Register("T", data)
	q.SetThis("T", reflect.ValueOf(data))

	// Test through Call which uses GetActionValue internally
	reply, err := q.Call("T", "Add", &ReqNil{})
	assert.Nil(t, err)
	assert.Equal(t, reply.(*Int64).Data, int64(30))
}

func TestCallQueryFunc(t *testing.T) {
	b := 20
	data := &T{10, &b}
	methods := ListMethod(data)
	this := reflect.ValueOf(data)
	reply, err := CallQueryFunc(this, methods["Query_Add"], &ReqNil{})
	assert.Nil(t, err)
	assert.Equal(t, reply.(*Int64).Data, int64(30))
}

func TestBuildQueryType(t *testing.T) {
	b := 20
	data := &T{10, &b}
	methods := ListMethod(data)
	ms, tys := BuildQueryType("Query_", methods)
	assert.NotNil(t, ms["Add"])
	assert.NotNil(t, tys["Add"])
}

func TestIsNilVariants(t *testing.T) {
	var a *int
	assert.True(t, IsNil(nil))
	assert.True(t, IsNil(a))
	assert.False(t, IsNil(42))

	var b interface{} = (*int)(nil)
	assert.True(t, IsNil(b))

	assert.True(t, IsNilP(nil))
	assert.True(t, IsNilP(a))
	assert.False(t, IsNilP(42))
}

func TestQueryDataRegisterDup(t *testing.T) {
	q := NewQueryData("Query_")
	b := 20
	data := &T{10, &b}
	q.Register("dupKey", data)
	assert.Panics(t, func() {
		q.Register("dupKey", data)
	})
}

func TestQueryDataGetFuncNotFound(t *testing.T) {
	q := NewQueryData("Test_")
	_, err := q.GetFunc("nonexistent", "method")
	assert.Equal(t, ErrActionNotSupport, err)
}

func TestQueryDataGetTypeNotFound(t *testing.T) {
	q := NewQueryData("Test_")
	_, err := q.GetType("nonexistent", "method")
	assert.Equal(t, ErrActionNotSupport, err)
}

func TestCallWithErrorHandling(t *testing.T) {
	q := NewQueryData("Test_")
	_, err := q.Call("nonexistent", "method", &ReqNil{})
	assert.Equal(t, ErrActionNotSupport, err)
}

func TestIsOKEdgeCases(t *testing.T) {
	data := make([]reflect.Value, 1)
	data[0] = reflect.ValueOf(42)
	assert.False(t, IsOK(data, 2))

	data2 := make([]reflect.Value, 2)
	data2[0] = reflect.ValueOf(42)
	data2[1] = reflect.ValueOf("str")
	assert.True(t, IsOK(data2, 2))
}

func TestNewQueryData(t *testing.T) {
	q := NewQueryData("Prefix_")
	assert.NotNil(t, q)
	assert.Equal(t, "Prefix_", q.prefix)
	assert.NotNil(t, q.funcMap)
	assert.NotNil(t, q.typeMap)
	assert.NotNil(t, q.valueMap)
}
