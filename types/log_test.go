// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	data, err := logty.JSON([]byte("hello world"))
	assert.Nil(t, err)
	assert.Equal(t, string(data), `"hello world"`)
}
