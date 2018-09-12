package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types/jsonpb"
)

func TestAllowExecName(t *testing.T) {
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

	isok = IsAllowExecName([]byte("evm"), []byte("user.evm.evm2"))
	assert.Equal(t, isok, true)

	//两层的情况也可以自动识别
	isok = IsAllowExecName([]byte("evm"), []byte("user.p.guodun.user.evm.xxx"))
	assert.Equal(t, isok, true)

	//三层不支持
	isok = IsAllowExecName([]byte("evm"), []byte("user.p.guodun.user.evm.user.hello"))
	assert.Equal(t, isok, true)

	isok = IsAllowExecName([]byte("user.p.evm.user.hello"), []byte("user.p.guodun.user.p.evm.user.hello"))
	assert.Equal(t, isok, true)
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

	encode2 := &jsonpb.Marshaler{EmitDefaults: false}
	s, err = encode2.MarshalToString(r)
	assert.Nil(t, err)
	assert.Equal(t, s, `{}`)
}
