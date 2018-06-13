package mavl

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"
)

var store_cfg0 = &types.Store{"mavl_test", "leveldb", "/tmp/mavl_test0", 100}
var store_cfg1 = &types.Store{"mavl_test", "leveldb", "/tmp/mavl_test1", 100}
var store_cfg2 = &types.Store{"mavl_test", "leveldb", "/tmp/mavl_test2", 100}
var store_cfg3 = &types.Store{"mavl_test", "leveldb", "/tmp/mavl_test3", 100}
var store_cfg4 = &types.Store{"mavl_test", "leveldb", "/tmp/mavl_test4", 100}

func TestKvdbNewClose(t *testing.T) {
	os.RemoveAll(store_cfg0.DbPath)
	store := New(store_cfg0)
	assert.NotNil(t, store)

	store.Close()
}

func TestKvddbSetGet(t *testing.T) {
	os.RemoveAll(store_cfg1.DbPath)
	store := New(store_cfg1)
	assert.NotNil(t, store)

	keys0 := [][]byte{[]byte("mk1"), []byte("mk2")}
	get0 := &types.StoreGet{[]byte("1st"), keys0}
	values0 := store.Get(get0)
	mlog.Info("info", "info", values0)
	// Get exist key, result nil
	assert.Len(t, values0, 2)
	assert.Equal(t, []byte(nil), values0[0])
	assert.Equal(t, []byte(nil), values0[1])

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("k1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("k2"), []byte("v2")})
	datas := &types.StoreSet{
		[]byte("1st"),
		kv,
	}
	hash := store.Set(datas, true)

	keys := [][]byte{[]byte("k1"), []byte("k2")}
	get1 := &types.StoreGet{hash, keys}

	values := store.Get(get1)
	assert.Len(t, values, 2)
	assert.Equal(t, []byte("v1"), values[0])
	assert.Equal(t, []byte("v2"), values[1])

	keys = [][]byte{[]byte("k1")}
	get2 := &types.StoreGet{hash, keys}
	values2 := store.Get(get2)
	assert.Len(t, values2, 1)
	assert.Equal(t, []byte("v1"), values2[0])

	get3 := &types.StoreGet{[]byte("1st"), keys}
	values3 := store.Get(get3)
	assert.Len(t, values3, 1)
	assert.Equal(t, []byte(nil), values3[0])
}

func TestKvdbMemSet(t *testing.T) {
	os.RemoveAll(store_cfg2.DbPath)
	store := New(store_cfg2)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("mk1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("mk2"), []byte("v2")})
	datas := &types.StoreSet{
		[]byte("1st"),
		kv,
	}
	hash := store.MemSet(datas, true)

	keys := [][]byte{[]byte("mk1"), []byte("mk2")}
	get1 := &types.StoreGet{hash, keys}

	values := store.Get(get1)
	assert.Len(t, values, 2)

	actHash, _ := store.Commit(&types.ReqHash{hash})
	assert.Equal(t, hash, actHash)

	notExistHash, _ := store.Commit(&types.ReqHash{[]byte("1st")})
	assert.Nil(t, notExistHash)
}

func TestKvdbRollback(t *testing.T) {
	os.RemoveAll(store_cfg3.DbPath)
	store := New(store_cfg3)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("mk1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("mk2"), []byte("v2")})
	datas := &types.StoreSet{
		[]byte("1st"),
		kv,
	}
	hash := store.MemSet(datas, true)

	keys := [][]byte{[]byte("mk1"), []byte("mk2")}
	get1 := &types.StoreGet{hash, keys}
	values := store.Get(get1)
	assert.Len(t, values, 2)

	actHash := store.Rollback(&types.ReqHash{hash})
	assert.Equal(t, hash, actHash)

	notExistHash := store.Rollback(&types.ReqHash{[]byte("1st")})
	assert.Nil(t, notExistHash)
}

var checkKVResult []*types.KeyValue

func checkKV(k, v []byte) bool {
	checkKVResult = append(checkKVResult,
		&types.KeyValue{k, v})
	mlog.Debug("checkKV", "key", string(k), "value", string(v))
	return false
}
func TestKvdbIterate(t *testing.T) {
	os.RemoveAll(store_cfg4.DbPath)
	store := New(store_cfg4)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("mk1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("mk2"), []byte("v2")})
	datas := &types.StoreSet{
		[]byte("1st"),
		kv,
	}
	hash := store.Set(datas, true)

	store.IterateRangeByStateHash(hash, []byte("mk1"), []byte("mk3"), true, checkKV)
	assert.Len(t, checkKVResult, 2)
	assert.Equal(t, []byte("v1"), checkKVResult[0].Value)
	assert.Equal(t, []byte("v2"), checkKVResult[1].Value)

}
