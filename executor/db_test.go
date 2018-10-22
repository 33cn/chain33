package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
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
	old := types.ForkV22ExecRollback
	types.ForkV22ExecRollback = 10
	defer func() {
		types.ForkV22ExecRollback = old
	}()
	db := newStateDbForTest(types.ForkV22ExecRollback - 1)

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
	db = newStateDbForTest(types.ForkV22ExecRollback)
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
