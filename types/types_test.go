package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types/jsonpb"
)

func TestAllowExecName(t *testing.T) {
	//allow exec list
	old := AllowUserExec
	defer func() {
		AllowUserExec = old
	}()
	AllowUserExec = nil
	AllowUserExec = append(AllowUserExec, []byte("coins"))
	isok := IsAllowExecName([]byte("a"), []byte("a"))
	assert.Equal(t, isok, false)

	isok = IsAllowExecName([]byte("coins"), []byte("coins"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.coins"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.coinsx"))
	assert.Equal(t, isok, false)

	isok = IsAllowExecName([]byte("coins"), []byte("user.coins.evm2"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.p.guodun.coins.evm2"))
	assert.Equal(t, isok, false)

	isok = IsAllowExecName([]byte("coins"), []byte("user.p.guodun.coins"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("coins"), []byte("user.p.guodun.user.coins"))
	assert.Equal(t, isok, true)
}

func BenchmarkExecName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ExecName("hello")
	}
}

func BenchmarkG(b *testing.B) {
	for i := 0; i < b.N; i++ {
		G("TestNet")
	}
}

func BenchmarkS(b *testing.B) {
	for i := 0; i < b.N; i++ {
		S("helloword", true)
	}
}
func TestJsonNoName(t *testing.T) {
	flag := int32(1)
	params := struct {
		Flag int32
	}{
		Flag: flag,
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, string(data), "{\"Flag\":1}")
}

func TestNil(t *testing.T) {
	v := reflect.ValueOf(nil)
	assert.Equal(t, v.IsValid(), false)
}

func TestProtoToJson(t *testing.T) {
	r := &Reply{}
	b, err := json.Marshal(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{}`))

	encode := &jsonpb.Marshaler{EmitDefaults: true}
	s, err := encode.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{"isOk":false,"msg":null}`)
	var dr Reply
	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":null}`, &dr)
	assert.Nil(t, err)
	assert.Nil(t, dr.Msg)
	encode2 := &jsonpb.Marshaler{EmitDefaults: false}
	s, err = encode2.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{}`)

	r = &Reply{Msg: []byte("OK")}
	b, err = json.Marshal(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{"msg":"T0s="}`))

	encode = &jsonpb.Marshaler{EmitDefaults: true}
	s, err = encode.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{"isOk":false,"msg":"0x4f4b"}`)

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":"0x4f4b"}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, dr.Msg, []byte("OK"))

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":"4f4b"}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, dr.Msg, []byte("OK"))

	r = &Reply{Msg: []byte{}}
	b, err = json.Marshal(r)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte(`{}`))

	encode = &jsonpb.Marshaler{EmitDefaults: true}
	s, err = encode.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{"isOk":false,"msg":""}`)

	err = jsonpb.UnmarshalString(`{"isOk":false,"msg":""}`, &dr)
	assert.Nil(t, err)
	assert.Equal(t, dr.Msg, []byte{})
}

func TestHex(t *testing.T) {
	s := "0x4f4b"
	b, err := common.FromHex(s)
	assert.Nil(t, err)
	assert.Equal(t, b, []byte("OK"))
}

func TestGetLogName(t *testing.T) {
	name := GetLogName([]byte("xxx"), 0)
	assert.Equal(t, "LogReserved", name)
	assert.Equal(t, "LogErr", GetLogName([]byte("coins"), 1))
	assert.Equal(t, "LogFee", GetLogName([]byte("token"), 2))
	assert.Equal(t, "LogReserved", GetLogName([]byte("xxxx"), 100))
}

func TestDecodeLog(t *testing.T) {
	data, _ := common.FromHex("0x0a2b10c0c599b78c1d2222314c6d7952616a4e44686f735042746259586d694c466b5174623833673948795565122b1080ab8db78c1d2222314c6d7952616a4e44686f735042746259586d694c466b5174623833673948795565")
	l, err := DecodeLog([]byte("xxx"), 2, data)
	assert.Nil(t, err)
	j, err := json.Marshal(l)
	assert.Nil(t, err)
	assert.Equal(t, "{\"prev\":{\"balance\":999769400000,\"addr\":\"1LmyRajNDhosPBtbYXmiLFkQtb83g9HyUe\"},\"current\":{\"balance\":999769200000,\"addr\":\"1LmyRajNDhosPBtbYXmiLFkQtb83g9HyUe\"}}", string(j))
}

func TestGetRealExecName(t *testing.T) {
	a := []struct {
		key     string
		realkey string
	}{
		{"coins", "coins"},
		{"user.p.coins", "user.p.coins"},
		{"user.p.guodun.coins", "coins"},
		{"user.evm.hash", "evm"},
		{"user.p.para.evm.hash", "evm.hash"},
		{"user.p.para.user.evm.hash", "evm"},
		{"user.p.para.", "user.p.para."},
	}
	for _, v := range a {
		assert.Equal(t, string(GetRealExecName([]byte(v.key))), v.realkey)
	}
}
