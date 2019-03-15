package executor_test

import (
	"testing"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/executor"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestLocalDBGet(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	db := executor.NewLocalDB(mock33.GetClient())
	defer db.(*executor.LocalDB).Close()
	testDBGet(t, db)
}

func TestLocalDBEnable(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	db := executor.NewLocalDB(mock33.GetClient())
	ldb := db.(*executor.LocalDB)
	defer ldb.Close()
	_, err := ldb.Get([]byte("hello"))
	assert.Equal(t, err, types.ErrNotFound)
	ldb.DisableRead()
	_, err = ldb.Get([]byte("hello"))

	assert.Equal(t, err, types.ErrDisableRead)
	_, err = ldb.List(nil, nil, 0, 0)
	assert.Equal(t, err, types.ErrDisableRead)
	ldb.EnableRead()
	_, err = ldb.Get([]byte("hello"))
	assert.Equal(t, err, types.ErrNotFound)
	_, err = ldb.List(nil, nil, 0, 0)
	assert.Equal(t, err, nil)
	ldb.DisableWrite()
	err = ldb.Set([]byte("hello"), nil)
	assert.Equal(t, err, types.ErrDisableWrite)
	ldb.EnableWrite()
	err = ldb.Set([]byte("hello"), nil)
	assert.Equal(t, err, nil)

}

func BenchmarkLocalDBGet(b *testing.B) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	db := executor.NewLocalDB(mock33.GetClient())
	defer db.(*executor.LocalDB).Close()

	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(b, err)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		v, err := db.Get([]byte("k1"))
		assert.Nil(b, err)
		assert.Equal(b, v, []byte("v1"))
	}
}

func TestLocalDBTxGet(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	db := executor.NewLocalDB(mock33.GetClient())
	testTxGet(t, db)
}

func testDBGet(t *testing.T, db dbm.KV) {
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

func testTxGet(t *testing.T, db dbm.KV) {
	//新版本
	db.Begin()
	err := db.Set([]byte("k1"), []byte("v1"))
	assert.Nil(t, err)
	v, err := db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	db.Commit()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v1"))

	//在非transaction中set，直接set成功，不能rollback
	err = db.Set([]byte("k1"), []byte("v11"))
	assert.Nil(t, err)

	db.Begin()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v11"))

	err = db.Set([]byte("k1"), []byte("v12"))
	assert.Nil(t, err)
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v12"))

	db.Rollback()
	v, err = db.Get([]byte("k1"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v11"))
}

func TestLocalDB(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	db := executor.NewLocalDB(mock33.GetClient())
	defer db.(*executor.LocalDB).Close()
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

	//beigin and rollback not imp
	db.Begin()
	err = db.Set([]byte("k2"), []byte("v2"))
	assert.Nil(t, err)
	db.Rollback()
	_, err = db.Get([]byte("k2"))
	assert.Equal(t, err, types.ErrNotFound)
	err = db.Set([]byte("k2"), []byte("v2"))
	assert.Nil(t, err)
	//get
	v, err = db.Get([]byte("k2"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v2"))
	//list
	values, err := db.List([]byte("k"), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, string(values[0]), "v2")
	assert.Equal(t, string(values[1]), "v11")
	err = db.Commit()
	assert.Nil(t, err)
	//get
	v, err = db.Get([]byte("k2"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("v2"))
	//list
	values, err = db.List([]byte("k"), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, len(values), 2)
	assert.Equal(t, string(values[0]), "v2")
	assert.Equal(t, string(values[1]), "v11")
}
