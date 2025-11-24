package db

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

// localdb 不会往 maindb 写数据，他都是临时的数据，在内存中临时计算的一个数据库
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

func TestLocalDBList(t *testing.T) {

	db, dir := newGoLevelDB(t)
	defer os.RemoveAll(dir)
	ldb := NewLocalDB(db, false)

	for i := 0; i < 10; i++ {
		db.Set([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("%d", i)))
	}

	//交易开始执行
	ldb.Begin()
	for i := 5; i < 10; i++ {
		ldb.Set([]byte(fmt.Sprintf("key%d", i)), nil)
	}
	//交易执行中
	testList := func() {
		values, err := ldb.List([]byte("key"), nil, 20, ListASC)
		require.Nil(t, err)
		require.Equal(t, 5, len(values))
		for i := 0; i < 5; i++ {
			require.Equal(t, []byte(fmt.Sprintf("%d", i)), values[i])
		}
		values, err = ldb.List([]byte("key"), nil, 20, ListDESC)
		require.Nil(t, err)
		require.Equal(t, 5, len(values))
		for i := 0; i < 5; i++ {
			require.Equal(t, []byte(fmt.Sprintf("%d", 4-i)), values[i])
		}
	}
	testList()
	ldb.Commit()

	// 交易执行结束
	testList()

	ldb.Begin()
	for i := 0; i < 5; i++ {
		ldb.Set([]byte(fmt.Sprintf("key%d", i)), nil)
	}
	for i := 5; i < 10; i++ {
		ldb.Set([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("%d", i-5)))
	}

	testList()
	ldb.Commit()
	testList()
}
