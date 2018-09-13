package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
