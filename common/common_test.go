// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandString(t *testing.T) {
	str := GetRandString(10)
	t.Log(str)
	assert.Len(t, str, 10)
}

func TestRandStringLen(t *testing.T) {
	str := GetRandBytes(10, 20)
	t.Log(string(str))
	if len(str) < 10 || len(str) > 20 {
		t.Error("rand str len")
	}
}

func TestGetRandPrintString(t *testing.T) {
	str := GetRandPrintString(10, 20)
	t.Log(str)
	if len(str) < 10 || len(str) > 20 {
		t.Error("rand str len")
	}
}
