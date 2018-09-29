package kvmvccdb

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/common"
	drivers "gitlab.33.cn/chain33/chain33/system/store"
	"gitlab.33.cn/chain33/chain33/types"
)

var store_cfg0 = &types.Store{"kvmvcc", "leveldb", "/tmp/kvdb_test0", 100, false, false}
var store_cfg1 = &types.Store{"kvmvcc", "leveldb", "/tmp/kvdb_test1", 100, false, false}
var store_cfg2 = &types.Store{"kvmvcc", "leveldb", "/tmp/kvdb_test2", 100, false, false}
var store_cfg3 = &types.Store{"kvmvcc", "leveldb", "/tmp/kvdb_test3", 100, false, false}
var store_cfg4 = &types.Store{"kvmvcc", "leveldb", "/tmp/kvdb_test4", 100, false, false}
var store_cfg6 = &types.Store{"kvmvcc_test", "leveldb", "/tmp/mpt_test6", 100, false, false}
var store_cfg7 = &types.Store{"kvmvcc_test", "leveldb", "/tmp/mpt_test7", 100, false, false}
var store_cfg8 = &types.Store{"kvmvcc_test", "leveldb", "/tmp/mpt_test8", 100, false, false}
var store_cfg9 = &types.Store{"kvmvcc_test", "leveldb", "/tmp/mpt_test9", 100, false, false}

const MaxKeylenth int = 64

func TestKvmvccdbNewClose(t *testing.T) {
	os.RemoveAll(store_cfg0.DbPath)
	store := New(store_cfg0)
	assert.NotNil(t, store)

	store.Close()
	os.RemoveAll(store_cfg0.DbPath)
}

func TestKvmvccdbSetGet(t *testing.T) {
	os.RemoveAll(store_cfg1.DbPath)
	store := New(store_cfg1).(*KVMVCCStore)
	assert.NotNil(t, store)

	keys0 := [][]byte{[]byte("mk1"), []byte("mk2")}
	get0 := &types.StoreGet{drivers.EmptyRoot[:], keys0}
	values0 := store.Get(get0)
	//klog.Info("info", "info", values0)
	// Get exist key, result nil
	assert.Len(t, values0, 2)
	assert.Equal(t, []byte(nil), values0[0])
	assert.Equal(t, []byte(nil), values0[1])

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("k1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("k2"), []byte("v2")})
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.Set(datas, true)
	assert.Nil(t, err)
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

	get3 := &types.StoreGet{drivers.EmptyRoot[:], keys}
	values3 := store.Get(get3)
	assert.Len(t, values3, 1)

	os.RemoveAll(store_cfg1.DbPath)
}

func TestKvmvccdbMemSet(t *testing.T) {
	os.RemoveAll(store_cfg2.DbPath)
	store := New(store_cfg2).(*KVMVCCStore)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("mk1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("mk2"), []byte("v2")})
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.MemSet(datas, true)
	assert.Nil(t, err)
	keys := [][]byte{[]byte("mk1"), []byte("mk2")}
	get1 := &types.StoreGet{hash, keys}

	values := store.Get(get1)
	assert.Len(t, values, 2)
	assert.Nil(t, values[0])
	assert.Nil(t, values[1])

	actHash, _ := store.Commit(&types.ReqHash{hash})
	assert.Equal(t, hash, actHash)

	notExistHash, _ := store.Commit(&types.ReqHash{drivers.EmptyRoot[:]})
	assert.Nil(t, notExistHash)

	values = store.Get(get1)
	assert.Len(t, values, 2)
	assert.Equal(t, values[0], kv[0].Value)
	assert.Equal(t, values[1], kv[1].Value)

	os.RemoveAll(store_cfg2.DbPath)
}

func TestKvmvccdbRollback(t *testing.T) {
	os.RemoveAll(store_cfg3.DbPath)
	store := New(store_cfg3).(*KVMVCCStore)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("mk1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("mk2"), []byte("v2")})
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.MemSet(datas, true)
	assert.Nil(t, err)
	keys := [][]byte{[]byte("mk1"), []byte("mk2")}
	get1 := &types.StoreGet{hash, keys}
	values := store.Get(get1)
	assert.Len(t, values, 2)
	assert.Nil(t, values[0])
	assert.Nil(t, values[1])

	actHash, _ := store.Rollback(&types.ReqHash{hash})
	assert.Equal(t, hash, actHash)

	notExistHash, err := store.Rollback(&types.ReqHash{drivers.EmptyRoot[:]})
	assert.Nil(t, notExistHash)
	assert.Equal(t, types.ErrHashNotFound.Error(), err.Error())

	os.RemoveAll(store_cfg3.DbPath)
}

func TestKvmvccdbRollbackBatch(t *testing.T) {
	os.RemoveAll(store_cfg4.DbPath)
	store := New(store_cfg4).(*KVMVCCStore)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("mk1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("mk2"), []byte("v2")})
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.MemSet(datas, true)
	assert.Nil(t, err)
	var kvset []*types.KeyValue
	req := &types.ReqHash{hash}
	hash1 := make([]byte, len(hash))
	copy(hash1, hash)
	store.Commit(req)
	for i := 1; i <= 202; i++ {
		kvset = nil
		datas1 := &types.StoreSet{hash1, datas.KV, datas.Height + int64(i)}
		s1 := fmt.Sprintf("v1-%03d", datas.Height+int64(i))
		s2 := fmt.Sprintf("v2-%03d", datas.Height+int64(i))
		datas.KV[0].Value = []byte(s1)
		datas.KV[1].Value = []byte(s2)
		hash1 = calcHash(datas1)
		//zzh
		//klog.Debug("KVMVCCStore MemSet AddMVCC", "prestatehash", common.ToHex(datas.StateHash), "hash", common.ToHex(hash), "height", datas.Height)
		klog.Info("KVMVCCStore MemSet AddMVCC for 202", "prestatehash", common.ToHex(datas1.StateHash), "hash", common.ToHex(hash1), "height", datas1.Height)
		kvlist, err := store.mvcc.AddMVCC(datas1.KV, hash1, datas1.StateHash, datas1.Height)
		if err != nil {
			klog.Info("KVMVCCStore MemSet AddMVCC failed for 202, continue")
			continue
		}

		if len(kvlist) > 0 {
			kvset = append(kvset, kvlist...)
		}
		store.kvsetmap[string(hash1)] = kvset
		req := &types.ReqHash{hash1}
		store.Commit(req)
	}

	maxVersion, err := store.mvcc.GetMaxVersion()
	assert.Equal(t, err, nil)
	assert.Equal(t, int64(202), maxVersion)

	keys := [][]byte{[]byte("mk1"), []byte("mk2")}
	get1 := &types.StoreGet{hash, keys}
	values := store.Get(get1)
	assert.Len(t, values, 2)
	assert.Equal(t, []byte("v1"), values[0])
	assert.Equal(t, []byte("v2"), values[1])

	var kv2 []*types.KeyValue
	kv2 = append(kv2, &types.KeyValue{[]byte("mk1"), []byte("v11")})
	kv2 = append(kv2, &types.KeyValue{[]byte("mk2"), []byte("v22")})

	datas2 := &types.StoreSet{hash, kv2, 1}
	hash, err = store.MemSet(datas2, true)
	assert.Nil(t, err)
	req = &types.ReqHash{hash}
	store.Commit(req)

	maxVersion, err = store.mvcc.GetMaxVersion()
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(3), maxVersion)

	get2 := &types.StoreGet{hash, keys}
	values2 := store.Get(get2)
	assert.Len(t, values, 2)
	assert.Equal(t, values2[0], kv2[0].Value)
	assert.Equal(t, values2[1], kv2[1].Value)

	datas3 := &types.StoreSet{hash, kv2, 2}
	hash, err = store.MemSet(datas3, true)
	assert.Nil(t, err)
	req = &types.ReqHash{hash}
	store.Commit(req)

	maxVersion, err = store.mvcc.GetMaxVersion()
	assert.Equal(t, nil, err)
	assert.Equal(t, int64(2), maxVersion)

	os.RemoveAll(store_cfg4.DbPath)
}

func GetRandomString(lenth int) string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < lenth; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return string(result)
}

func BenchmarkGet(b *testing.B) {
	os.RemoveAll(store_cfg6.DbPath)
	store := New(store_cfg6).(*KVMVCCStore)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var keys [][]byte
	var hash []byte
	var err error
	hash = drivers.EmptyRoot[:]
	for i := 0; i < b.N; i++ {
		key := GetRandomString(MaxKeylenth)
		value := fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
		if i%10000 == 0 {
			datas := &types.StoreSet{hash, kv, 0}
			hash, err = store.Set(datas, true)
			assert.Nil(b, err)
			kv = nil
		}
	}
	if kv != nil {
		datas := &types.StoreSet{hash, kv, 0}
		hash, err = store.Set(datas, true)
		assert.Nil(b, err)
		kv = nil
	}
	assert.Nil(b, err)
	start := time.Now()
	b.ResetTimer()
	for _, key := range keys {
		getData := &types.StoreGet{
			hash,
			[][]byte{key}}
		store.Get(getData)
	}
	end := time.Now()
	fmt.Println("kvmvcc BenchmarkGet cost time is", end.Sub(start), "num is", b.N)
}

/*
func TestDBClose(t *testing.T) {
	err := os.RemoveAll(store_cfg0.DbPath)
	if err != nil {
		klog.Error("remove dbpath failed. ", "path", store_cfg0.DbPath, "err", err)
	}
	err = os.RemoveAll(store_cfg1.DbPath)
	if err != nil {
		klog.Error("remove dbpath failed. ", "path", store_cfg0.DbPath, "err", err)
	}
	err = os.RemoveAll(store_cfg2.DbPath)
	if err != nil {
		klog.Error("remove dbpath failed. ", "path", store_cfg0.DbPath, "err", err)
	}
	err = os.RemoveAll(store_cfg3.DbPath)
	if err != nil {
		klog.Error("remove dbpath failed. ", "path", store_cfg0.DbPath, "err", err)
	}

	assert.Equal(t, isDirExists(store_cfg0.DbPath), false)
	assert.Equal(t, isDirExists(store_cfg1.DbPath), false)
	assert.Equal(t, isDirExists(store_cfg2.DbPath), false)
	assert.Equal(t, isDirExists(store_cfg3.DbPath), false)
}
*/
func isDirExists(path string) bool {
	fi, err := os.Stat(path)

	if err != nil {
		return os.IsExist(err)
	} else {
		return fi.IsDir()
	}

	panic("not reached")
}
