// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"testing"

	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestCount(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	count := NewCount("prefix", "name#hello", kvdb)
	count.Inc()
	count.Dec()
	count.Inc()
	i, err := count.Get()
	assert.Nil(t, err)
	assert.Equal(t, i, int64(1))
	kvs, err := count.Save()
	assert.Nil(t, err)
	util.SaveKVList(ldb, kvs)

	count = NewCount("prefix", "name#hello", kvdb)
	i, err = count.Get()
	assert.Nil(t, err)
	assert.Equal(t, i, int64(1))

	count.Set(2)
	i, err = count.Get()
	assert.Nil(t, err)
	assert.Equal(t, i, int64(2))
}
