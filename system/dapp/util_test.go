package dapp

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestKVCreator(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	creator := NewKVCreator(kvdb, false)
	creator.AddKVOnly([]byte("a"), []byte("b"))
	_, err := kvdb.Get([]byte("a"))
	assert.Equal(t, err, types.ErrNotFound)
	creator.Add([]byte("a"), []byte("b"))
	value, err := kvdb.Get([]byte("a"))
	assert.Equal(t, err, nil)
	assert.Equal(t, value, []byte("b"))

	creator = NewKVCreator(kvdb, true)
	creator.Add([]byte("a"), []byte("b"))
	creator.Add([]byte("a1"), []byte("b1"))
	creator.AddList([]*types.KeyValue{
		{Key: []byte("l1"), Value: []byte("vl1")},
		{Key: []byte("l2"), Value: []byte("vl2")},
	})
	creator.Add([]byte("c1"), nil)
	creator.Add([]byte("l2"), nil)
	assert.Equal(t, 5, len(creator.KVList()))
	log := creator.RollbackLog()
	kvs, err := creator.ParseRollback(log)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(kvs))
	assert.Equal(t, []byte("b"), kvs[0].Value)
	assert.Equal(t, []byte(nil), kvs[1].Value)
	assert.Equal(t, []byte("vl2"), kvs[4].Value)
}

func TestHeightIndexStr(t *testing.T) {
	assert.Equal(t, "000000000000100001", HeightIndexStr(1, 1))
}
