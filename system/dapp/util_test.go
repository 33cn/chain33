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
	creator := NewKVCreator(kvdb, []byte("prefix-"), nil)
	creator.AddKVOnly([]byte("a"), []byte("b"))
	_, err := kvdb.Get([]byte("prefix-a"))
	assert.Equal(t, err, types.ErrNotFound)
	creator.Add([]byte("a"), []byte("b"))
	value, err := kvdb.Get([]byte("prefix-a"))
	assert.Equal(t, err, nil)
	assert.Equal(t, value, []byte("b"))

	creator = NewKVCreator(kvdb, []byte("prefix-"), []byte("rollback"))
	creator.Add([]byte("a"), []byte("b"))
	creator.Add([]byte("a1"), []byte("b1"))
	creator.AddNoPrefix([]byte("np"), []byte("np-value"))
	creator.AddList([]*types.KeyValue{
		{Key: []byte("l1"), Value: []byte("vl1")},
		{Key: []byte("l2"), Value: []byte("vl2")},
	})
	creator.AddListNoPrefix([]*types.KeyValue{
		{Key: []byte("l1"), Value: []byte("vl1")},
		{Key: []byte("l2"), Value: []byte("vl2")},
	})
	creator.Add([]byte("c1"), nil)
	creator.Add([]byte("l2"), nil)
	creator.AddRollbackKV()
	assert.Equal(t, 9, len(creator.KVList()))
	util.SaveKVList(ldb, creator.KVList())
	kvs, err := creator.GetRollbackKVList()
	assert.Nil(t, err)
	assert.Equal(t, 8, len(kvs))
	assert.Equal(t, []byte("b"), kvs[7].Value)
	assert.Equal(t, []byte(nil), kvs[6].Value)
	assert.Equal(t, []byte(nil), kvs[5].Value)
	assert.Equal(t, []byte(nil), kvs[4].Value)
	assert.Equal(t, []byte(nil), kvs[3].Value)
	assert.Equal(t, []byte(nil), kvs[2].Value)
	assert.Equal(t, []byte(nil), kvs[1].Value)
	assert.Equal(t, []byte("vl2"), kvs[0].Value)
	//current: a = b
	//set data:
	//a -> b (a -> b)
	//a1 -> b1 (a1 -> nil)
	//l1 -> vl1 (l1 -> nil)
	//l2 -> vl2 (l2->nil)
	//c1 -> nil (ignore)
	//l2 -> nil (l2 -> vl2)
	//rollback 的过程实际上是 set 的逆过程，就像时间倒流一样
	//save rollback kvs
	_, err = creator.Get([]byte("np"))
	assert.Equal(t, types.ErrNotFound, err)
	v, _ := creator.GetNoPrefix([]byte("np"))
	assert.Equal(t, []byte("np-value"), v)
	util.SaveKVList(ldb, kvs)
	v, _ = kvdb.Get([]byte("prefix-a"))
	assert.Equal(t, []byte("b"), v)
	v, _ = creator.Get([]byte("a"))
	assert.Equal(t, []byte("b"), v)
	_, err = creator.Get([]byte("a1"))
	assert.Equal(t, types.ErrNotFound, err)
	_, err = creator.Get([]byte("l1"))
	assert.Equal(t, types.ErrNotFound, err)
	_, err = creator.Get([]byte("l2"))
	assert.Equal(t, types.ErrNotFound, err)
	_, err = creator.Get([]byte("c1"))
	assert.Equal(t, types.ErrNotFound, err)

	creator = NewKVCreator(kvdb, []byte("prefix-"), nil)
	creator.AddKVListOnly([]*types.KeyValue{{Key: []byte("k"), Value: []byte("v")}})
	creator.DelRollbackKV()
	creator.AddToLogs(nil)
}

func TestHeightIndexStr(t *testing.T) {
	assert.Equal(t, "000000000000100001", HeightIndexStr(1, 1))
}
