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
