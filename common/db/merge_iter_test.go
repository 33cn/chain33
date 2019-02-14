package db

import (
	"io/ioutil"
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

	list0 = it0.List(nil, nil, 100, 1)
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "1", string(list0[0]))
	assert.Equal(t, "2", string(list0[1]))

	//测试修改
	db1.Set([]byte("2"), []byte("12"))
	list0 = it0.List(nil, nil, 100, 0)
	for k, v := range list0 {
		println(k, string(v))
	}
	assert.Equal(t, 2, len(list0))
	assert.Equal(t, "12", string(list0[0]))
	assert.Equal(t, "1", string(list0[1]))
}
