// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/33cn/chain33/util"
)

func TestReplaceTarget(t *testing.T) {
	fileName := "../config/template/executor/${CLASSNAME}.go.tmp"
	bcontent, err := util.ReadFile(fileName)
	assert.NoError(t, err)
	t.Log(string(bcontent))
}
