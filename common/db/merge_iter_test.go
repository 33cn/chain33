package db

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newGoMemDB(t *testing.T) DB {
	dir, err := ioutil.TempDir("", "gomemdb")
	require.NoError(t, err)
	memdb, err := NewGoMemDB("gomemdb", dir, 128)
	require.NoError(t, err)
	return memdb
}

func TestMergeIter(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db1.Set([]byte("1"), []byte("1"))
	db2.Set([]byte("2"), []byte("2"))

	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2})
	it0 := NewListHelper(db)
	list0 := it0.List(nil, nil, 100, 0)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "2", string(list0[0]))
	assert.Equal(t, "1", string(list0[1]))
	/*
		list0 = it0.List(nil, nil, 100, 1)
		assert.Equal(t, 2, len(list0))
		assert.Equal(t, "1", string(list0[0]))
		assert.Equal(t, "2", string(list0[1]))
	*/
}

func newGoLevelDB(t *testing.T) (DB, string) {
	dir, err := ioutil.TempDir("", "goleveldb")
	assert.Nil(t, err)
	db, err := NewGoLevelDB("test", dir, 16)
	assert.Nil(t, err)
	return db, dir
}

func newGoBadgerDB(t *testing.T) (DB, string) {
	dir, err := ioutil.TempDir("", "badgerdb")
	assert.Nil(t, err)
	db, err := NewGoBadgerDB("test", dir, 16)
	assert.Nil(t, err)
	return db, dir
}

func TestMergeIterSeek1(t *testing.T) {
	db1 := newGoMemDB(t)
	db1.Set([]byte("1"), []byte("1"))

	it0 := NewListHelper(db1)
	list0 := it0.List(nil, []byte("2"), 1, ListSeek)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "1", string(list0[0]))
}

func TestMergeIterSeek(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db3, dir := newGoLevelDB(t)
	defer os.RemoveAll(dir) // clean up

	db1.Set([]byte("1"), []byte("1"))
	db2.Set([]byte("3"), []byte("3"))
	db3.Set([]byte("5"), []byte("5"))
	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2, db3})
	it0 := NewListHelper(db)
	list0 := it0.List(nil, []byte("2"), 1, ListSeek)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "1", string(list0[1]))

	list0 = it0.List(nil, []byte("3"), 1, ListSeek)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "3", string(list0[1]))
}

func TestMergeIterSeekPrefix(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db3, dir := newGoLevelDB(t)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	db1.Set([]byte("key1"), []byte("1"))
	db2.Set([]byte("key3"), []byte("3"))
	db3.Set([]byte("key5"), []byte("5"))
	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2, db3})
	it0 := NewListHelper(db)
	list0 := it0.List([]byte("key"), []byte("key2"), 1, ListSeek)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "1", string(list0[1]))

	list0 = it0.List([]byte("key"), []byte("key3"), 1, ListSeek)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "3", string(list0[1]))

	list0 = it0.List([]byte("key"), []byte("key6"), 1, ListSeek)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "5", string(list0[1]))
}

func TestMergeIterDup1(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db1.Set([]byte("1"), []byte("1"))
	db2.Set([]byte("2"), []byte("2"))

	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2})
	it0 := NewListHelper(db)
	//测试修改
	db1.Set([]byte("2"), []byte("12"))
	list0 := it0.List(nil, nil, 100, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "12", string(list0[0]))
	assert.Equal(t, "1", string(list0[1]))
}

func TestMergeIterDup2(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db1.Set([]byte("key-1"), []byte("db1-key-1"))
	db1.Set([]byte("key-3"), []byte("db1-key-3"))
	db2.Set([]byte("key-2"), []byte("db2-key-2"))

	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2})
	it0 := NewListHelper(db)
	//测试修改
	db2.Set([]byte("key-3"), []byte("db2-key-3"))
	list0 := it0.List(nil, nil, 100, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 3, len(list0))
	assert.Equal(t, "db1-key-3", string(list0[0]))
	assert.Equal(t, "db2-key-2", string(list0[1]))
	assert.Equal(t, "db1-key-1", string(list0[2]))

	list0 = it0.List(nil, nil, 100, 1)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 3, len(list0))
	assert.Equal(t, "db1-key-1", string(list0[0]))
	assert.Equal(t, "db2-key-2", string(list0[1]))
	assert.Equal(t, "db1-key-3", string(list0[2]))
}

func TestMergeIterDup3(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db1.Set([]byte("key-1"), []byte("db1-key-1"))
	db1.Set([]byte("key-3"), []byte("db1-key-3"))
	db2.Set([]byte("key-2"), []byte("db2-key-2"))

	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2})
	it0 := NewListHelper(db)
	//测试修改
	db1.Set([]byte("key-2"), []byte("db1-key-2"))
	list0 := it0.List(nil, nil, 100, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 3, len(list0))
	assert.Equal(t, "db1-key-3", string(list0[0]))
	assert.Equal(t, "db1-key-2", string(list0[1]))
	assert.Equal(t, "db1-key-1", string(list0[2]))

	list0 = it0.List(nil, nil, 100, 1)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 3, len(list0))
	assert.Equal(t, "db1-key-1", string(list0[0]))
	assert.Equal(t, "db1-key-2", string(list0[1]))
	assert.Equal(t, "db1-key-3", string(list0[2]))
}

func TestMergeIter3(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db3 := newGoMemDB(t)
	db3.Set([]byte("key-1"), []byte("db3-key-1"))
	db3.Set([]byte("key-2"), []byte("db3-key-2"))
	db3.Set([]byte("key-3"), []byte("db3-key-3"))

	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2, db3})
	it0 := NewListHelper(db)
	list0 := it0.List([]byte("key-"), nil, 0, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 3, len(list0))
	assert.Equal(t, "db3-key-3", string(list0[0]))
	assert.Equal(t, "db3-key-2", string(list0[1]))
	assert.Equal(t, "db3-key-1", string(list0[2]))
}

func TestMergeIter1(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db3 := newGoMemDB(t)
	db1.Set([]byte("key-1"), []byte("db1-key-1"))
	db1.Set([]byte("key-2"), []byte("db1-key-2"))
	db1.Set([]byte("key-3"), []byte("db1-key-3"))

	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2, db3})
	it0 := NewListHelper(db)
	list0 := it0.List(nil, nil, 100, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 3, len(list0))
	assert.Equal(t, "db1-key-3", string(list0[0]))
	assert.Equal(t, "db1-key-2", string(list0[1]))
	assert.Equal(t, "db1-key-1", string(list0[2]))
}

func TestMergeIterSearch(t *testing.T) {
	db1 := newGoMemDB(t)
	db2 := newGoMemDB(t)
	db1.Set([]byte("key-1"), []byte("db1-key-1"))
	db1.Set([]byte("key-2"), []byte("db1-key-2"))
	db2.Set([]byte("key-2"), []byte("db2-key-2"))
	db2.Set([]byte("key-3"), []byte("db2-key-3"))
	db2.Set([]byte("key-4"), []byte("db2-key-4"))

	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2})
	it0 := NewListHelper(db)
	list0 := it0.List([]byte("key-"), []byte("key-2"), 100, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 1, len(list0))
	assert.Equal(t, "db1-key-1", string(list0[0]))

	list0 = it0.List([]byte("key-"), []byte("key-2"), 100, 1)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "db2-key-3", string(list0[0]))
	assert.Equal(t, "db2-key-4", string(list0[1]))
}

func TestIterSearch(t *testing.T) {
	db1 := newGoMemDB(t)
	defer db1.Close()
	db1.Set([]byte("key-1"), []byte("db1-key-1"))
	db1.Set([]byte("key-2"), []byte("db2-key-2"))
	db1.Set([]byte("key-2"), []byte("db1-key-2"))
	db1.Set([]byte("key-3"), []byte("db2-key-3"))
	db1.Set([]byte("key-4"), []byte("db2-key-4"))

	it0 := NewListHelper(db1)
	list0 := it0.List([]byte("key-"), []byte("key-2"), 100, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 1, len(list0))
	assert.Equal(t, "db1-key-1", string(list0[0]))

	list0 = it0.List([]byte("key-"), []byte("key-2"), 100, 1)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "db2-key-3", string(list0[0]))
	assert.Equal(t, "db2-key-4", string(list0[1]))
}

func TestMergeIterListIssue1211(t *testing.T) {
	db1 := newGoMemDB(t)
	db2, dir := newGoLevelDB(t)
	defer os.RemoveAll(dir)
	db := NewMergedIteratorDB([]IteratorDB{db1, db2})
	it := NewListHelper(db)

	for i := 6; i <= 9; i++ {
		db1.Set([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("%d", i)))
	}
	for i := 0; i < 6; i++ {
		db2.Set([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("%d", i)))
	}

	it2 := db.Iterator(nil, nil, false)
	for i := 0; i < 10; i++ {
		it2.Next()
		assert.Equal(t, []byte(fmt.Sprintf("key%d", i)), it2.Key())
	}

	values := it.List([]byte("key"), []byte("key5"), 5, ListDESC)
	assert.Equal(t, 5, len(values))
	for i, val := range values {
		assert.Equal(t, []byte(fmt.Sprintf("%d", 4-i)), val)
	}
}

func TestMergeIterList(t *testing.T) {
	levelDB, dir := newGoLevelDB(t)
	testMergeIterList(t, newGoMemDB(t), newGoMemDB(t), levelDB)
	_ = os.RemoveAll(dir)
	badgerDB, dir := newGoBadgerDB(t)
	testMergeIterList(t, newGoMemDB(t), newGoMemDB(t), badgerDB)
	_ = os.RemoveAll(dir)
	levelDB, dir1 := newGoLevelDB(t)
	badgerDB, dir2 := newGoBadgerDB(t)
	testMergeIterList(t, badgerDB, levelDB, newGoMemDB(t))
	_ = os.RemoveAll(dir1)
	_ = os.RemoveAll(dir2)
}

func testMergeIterList(t *testing.T, db1, db2, db3 DB) {

	for i := 0; i < 10; i++ {
		db3.Set([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("%d", i)))
	}
	//合并以后:
	db := NewMergedIteratorDB([]IteratorDB{db1, db2, db3})
	it := NewListHelper(db)

	//key9 ~ key1
	listAll := func(totalCount int, direction int32) [][]byte {
		var values [][]byte
		var primary []byte
		for i := 0; i < 3; i++ {
			data := it.List([]byte("key"), primary, 4, direction)
			values = append(values, data...)
			primary = []byte(fmt.Sprintf("key%s", data[len(data)-1]))
		}
		assert.Equal(t, totalCount, len(values))
		return values
	}

	values := listAll(10, ListDESC)
	for i, val := range values {
		assert.Equal(t, []byte(fmt.Sprintf("%d", 9-i)), val)
	}
	values = listAll(10, ListASC)
	for i, val := range values {
		assert.Equal(t, []byte(fmt.Sprintf("%d", i)), val)
	}

	// db2数据覆盖
	db2.Set([]byte("key3"), []byte("33"))
	values = listAll(10, ListDESC)
	for i, val := range values {
		value := []byte(fmt.Sprintf("%d", 9-i))
		if i == 6 {
			value = []byte("33")
		}
		assert.Equal(t, value, val)
	}
	values = listAll(10, ListASC)
	for i, val := range values {
		value := []byte(fmt.Sprintf("%d", i))
		if i == 3 {
			value = []byte("33")
		}
		assert.Equal(t, value, val)
	}

	// db1数据覆盖
	db1.Set([]byte("key3"), []byte("333"))
	db1.Set([]byte("key5"), []byte("555"))
	values = listAll(10, ListDESC)
	for i, val := range values {
		value := []byte(fmt.Sprintf("%d", 9-i))
		if i == 4 {
			value = []byte("555")
		}
		if i == 6 {
			value = []byte("333")
		}
		assert.Equal(t, value, val)
	}
	values = listAll(10, ListASC)
	for i, val := range values {
		value := []byte(fmt.Sprintf("%d", i))
		if i == 5 {
			value = []byte("555")
		}
		if i == 3 {
			value = []byte("333")
		}
		assert.Equal(t, value, val)
	}

	// 新增key
	db1.Set([]byte("key91"), []byte("10"))
	db2.Set([]byte("key92"), []byte("11"))
	values = listAll(12, ListDESC)
	for i, val := range values {
		value := []byte(fmt.Sprintf("%d", 11-i))
		if i == 6 {
			value = []byte("555")
		}
		if i == 8 {
			value = []byte("333")
		}
		assert.Equal(t, value, val)
	}
	values = listAll(12, ListASC)
	for i, val := range values {
		value := []byte(fmt.Sprintf("%d", i))
		if i == 5 {
			value = []byte("555")
		}
		if i == 3 {
			value = []byte("333")
		}
		assert.Equal(t, value, val)
	}
}
