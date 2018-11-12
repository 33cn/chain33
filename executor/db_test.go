// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"testing"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func newStateDbForTest(height int64) db.KV {
	return NewStateDB(nil, nil, nil, &StateDBOption{Height: height})
}
func TestStateDBGet(t *testing.T) {
	db := newStateDbForTest(0)
	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err := db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	err = db.Set([]byte("k1"), []byte("v11"))
	assert.Nil(t, err)
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v11"))
}

func TestStateDBTxGetOld(t *testing.T) {
	title := types.GetTitle()
	types.Init("chain33", nil)
	defer types.Init(title, nil)
	db := newStateDbForTest(types.GetFork("ForkExecRollback") - 1)

	db.Begin()
	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err := db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	db.Rollback()
	v, err = db.Get([]byte("k1"))
	assert.Equal(t, err, types.ErrNotFound)
	assert.Equal(t, v, []byte(nil))

	db.Begin()
	err = db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	db.Commit()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	err = db.Set([]byte("k1"), []byte("v11"))
	assert.Nil(t, err)

	db.Begin()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	//fork 之前有bug，这里读到了脏数据
	assert.Equal(t, v, []byte("v1"))

	db.Begin()
	db.Rollback()
	db.Commit()

	//新版本
	db = newStateDbForTest(types.GetFork("ForkExecRollback"))
	db.Begin()
	err = db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	db.Commit()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	err = db.Set([]byte("k1"), []byte("v11"))
	assert.Nil(t, err)

	db.Begin()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	//fork 之前有bug，这里读到了脏数据
	assert.Equal(t, v, []byte("v11"))
}
