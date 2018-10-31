package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogErr(t *testing.T) {
	var errlog LogErr = []byte("hello world")
	logty := LoadLog(nil, TyLogErr)
	assert.NotNil(t, logty)
	assert.Equal(t, logty.Name(), "LogErr")
	result, err := logty.Decode([]byte("hello world"))
	assert.Nil(t, err)
	assert.Equal(t, LogErr(result.(string)), errlog)

	//json test
	data, err := logty.Json([]byte("hello world"))
	assert.Nil(t, err)
	assert.Equal(t, string(data), `"hello world"`)
}
