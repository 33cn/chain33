package db

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

//localdb 不会往 maindb 写数据，他都是临时的数据，在内存中临时计算的一个数据库
func TestLocalDB(t *testing.T) {
	db1 := newGoMemDB(t)
	db := NewLocalDB(db1, false)

	db1.Set([]byte("key1"), []byte("value1"))
	v1, err := db.Get([]byte("key1"))
	assert.Nil(t, err)
	assert.Equal(t, v1, []byte("value1"))

	db.Set([]byte("key2"), []byte("value2"))
	data, err := db.List([]byte("key"), nil, 2, 1)
	assert.Nil(t, err)
	assert.Equal(t, data[0], []byte("value1"))
	assert.Equal(t, data[1], []byte("value2"))

	//test commit
	db.Begin()
	db.Set([]byte("key3"), []byte("value3"))
	data, err = db.List([]byte("key"), nil, 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, data[0], []byte("value1"))
	assert.Equal(t, data[1], []byte("value2"))
	assert.Equal(t, data[2], []byte("value3"))
	db.Commit()
	data, err = db.List([]byte("key"), nil, 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, data[0], []byte("value1"))
	assert.Equal(t, data[1], []byte("value2"))
	assert.Equal(t, data[2], []byte("value3"))

	//test rollback
	db.Begin()
	db.Set([]byte("key4"), []byte("value4"))
	data, err = db.List([]byte("key"), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, data[0], []byte("value4"))
	assert.Equal(t, data[1], []byte("value3"))
	assert.Equal(t, data[2], []byte("value2"))
	assert.Equal(t, data[3], []byte("value1"))
	db.Rollback()
	data, err = db.List([]byte("key"), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(data))
	assert.Equal(t, data[0], []byte("value3"))
	assert.Equal(t, data[1], []byte("value2"))
	assert.Equal(t, data[2], []byte("value1"))

	//test delete
	db1.Set([]byte("keyd1"), []byte("valued1"))
	err = db.Set([]byte("keyd1"), nil)
	assert.Nil(t, err)
	value, err := db.Get([]byte("keyd1"))
	assert.True(t, isdeleted(value))
	assert.True(t, isdeleted(nil))
	assert.Equal(t, types.ErrNotFound, err)

	//test list
	data, err = db.List([]byte("key"), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(data))
	assert.Equal(t, data[0], []byte("value3"))
	assert.Equal(t, data[1], []byte("value2"))
	assert.Equal(t, data[2], []byte("value1"))
}
