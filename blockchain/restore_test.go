// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"testing"

	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestNeedReExec(t *testing.T) {
	defer func() {
		r := recover()
		assert.Equal(t, r, "not support upgrade store to greater than 2.0.0")
	}()
	chain := &BlockChain{}
	testcase := []*types.UpgradeMeta{
		{Starting: true},
		{Starting: false, Version: "1.0.0"},
		{Starting: false, Version: "2.0.0"},
		{Starting: false, Version: "3.0.0"},
	}
	result := []bool{
		true,
		false,
		true,
		false,
	}
	for i, c := range testcase {
		if i == 3 {
			version.SetStoreDBVersion("3.0.0")
		}
		res := chain.needReExec(c)
		assert.Equal(t, res, result[i])
	}
}
