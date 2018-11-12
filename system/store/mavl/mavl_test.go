// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"io/ioutil"
	"os"
	"testing"

	"fmt"
	"time"

	"github.com/33cn/chain33/common"
	drivers "github.com/33cn/chain33/system/store"
	mavldb "github.com/33cn/chain33/system/store/mavl/db"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

const MaxKeylenth int = 64

func newStoreCfg(dir string) *types.Store {
	return &types.Store{Name: "mavl_test", Driver: "leveldb", DbPath: dir, DbCache: 100}
}

func TestKvdbNewClose(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(t, store)

	store.Close()
}

func TestKvddbSetGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(t, store)

	keys0 := [][]byte{[]byte("mk1"), []byte("mk2")}
	get0 := &types.StoreGet{drivers.EmptyRoot[:], keys0}
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
	assert.Equal(t, []byte(nil), values3[0])
}

func TestKvdbMemSet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
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

	actHash, _ := store.Commit(&types.ReqHash{hash})
	assert.Equal(t, hash, actHash)

	notExistHash, _ := store.Commit(&types.ReqHash{drivers.EmptyRoot[:]})
	assert.Nil(t, notExistHash)
}

func TestKvdbRollback(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
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

	actHash, _ := store.Rollback(&types.ReqHash{hash})
	assert.Equal(t, hash, actHash)

	notExistHash, _ := store.Rollback(&types.ReqHash{drivers.EmptyRoot[:]})
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
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("mk1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("mk2"), []byte("v2")})
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.Set(datas, true)
	assert.Nil(t, err)
	store.IterateRangeByStateHash(hash, []byte("mk1"), []byte("mk3"), true, checkKV)
	assert.Len(t, checkKVResult, 2)
	assert.Equal(t, []byte("v1"), checkKVResult[0].Value)
	assert.Equal(t, []byte("v2"), checkKVResult[1].Value)

}

func GetRandomString(length int) string {
	return common.GetRandPrintString(20, length)
}

func TestKvdbIterateTimes(t *testing.T) {
	checkKVResult = checkKVResult[:0]
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(t, store)

	var kv []*types.KeyValue
	var key string
	var value string

	for i := 0; i < 1000; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.Set(datas, true)
	assert.Nil(t, err)
	start := time.Now()
	store.IterateRangeByStateHash(hash, nil, nil, true, checkKV)
	end := time.Now()
	fmt.Println("mavl cost time is", end.Sub(start))
	assert.Len(t, checkKVResult, 1000)
}

func BenchmarkGet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)
	mavldb.EnableMavlPrefix(true)
	defer mavldb.EnableMavlPrefix(false)
	var kv []*types.KeyValue
	var keys [][]byte
	var hash = drivers.EmptyRoot[:]
	fmt.Println("N = ", b.N)
	for i := 0; i < b.N; i++ {
		key := GetRandomString(MaxKeylenth)
		value := fmt.Sprintf("%s%d", key, i)
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

	start := time.Now()
	b.ResetTimer()
	for _, key := range keys {
		getData := &types.StoreGet{
			hash,
			[][]byte{key}}
		store.Get(getData)
	}
	end := time.Now()
	fmt.Println("mavl BenchmarkGet cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}

//这个用例测试Store.Get接口，一次调用会返回一组kvs(30对kv)；前一个用例每次查询一个kv。
func BenchmarkStoreGetKvs4N(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	kvnum := 30
	for i := 0; i < kvnum; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.Set(datas, true)
	assert.Nil(b, err)
	getData := &types.StoreGet{
		hash,
		keys}

	start := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		values := store.Get(getData)
		assert.Len(b, values, kvnum)
	}
	end := time.Now()
	fmt.Println("mavl BenchmarkStoreGetKvs4N cost time is", end.Sub(start), "num is", b.N)

	b.StopTimer()
}

//这个用例测试Store.Get接口，一次调用会返回一组kvs(30对kv)，数据构造模拟真实情况,N条数据、N次查询。
func BenchmarkStoreGetKvsForNN(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	for i := 0; i < 30; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}

	var hashes [][]byte
	for i := 0; i < b.N; i++ {
		datas.Height = int64(i)
		value = fmt.Sprintf("vv%d", i)
		for j := 0; j < 10; j++ {
			datas.KV[j].Value = []byte(value)
		}
		hash, err := store.MemSet(datas, true)
		assert.Nil(b, err)
		req := &types.ReqHash{
			Hash: hash,
		}
		_, err = store.Commit(req)
		assert.NoError(b, err, "NoError")
		datas.StateHash = hash
		hashes = append(hashes, hash)
	}

	start := time.Now()
	b.ResetTimer()

	getData := &types.StoreGet{
		hashes[0],
		keys}

	for i := 0; i < b.N; i++ {
		getData.StateHash = hashes[i]
		store.Get(getData)
	}
	end := time.Now()
	fmt.Println("mavl BenchmarkStoreGetKvsForNN cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}

//这个用例测试Store.Get接口，一次调用会返回一组kvs(30对kv)，数据构造模拟真实情况，预置10000条数据，重复调用10000次。
func BenchmarkStoreGetKvsFor10000(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	for i := 0; i < 30; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}

	var hashes [][]byte
	blocks := 10000
	times := 10000
	start1 := time.Now()
	for i := 0; i < blocks; i++ {
		datas.Height = int64(i)
		value = fmt.Sprintf("vv%d", i)
		for j := 0; j < 30; j++ {
			datas.KV[j].Value = []byte(value)
		}
		hash, err := store.MemSet(datas, true)
		assert.Nil(b, err)
		req := &types.ReqHash{
			Hash: hash,
		}
		_, err = store.Commit(req)
		assert.NoError(b, err, "NoError")
		datas.StateHash = hash
		hashes = append(hashes, hash)
	}
	end1 := time.Now()

	start := time.Now()
	b.ResetTimer()

	getData := &types.StoreGet{
		hashes[0],
		keys}

	for i := 0; i < times; i++ {
		getData.StateHash = hashes[i]
		store.Get(getData)
	}
	end := time.Now()
	fmt.Println("mavl BenchmarkStoreGetKvsFor10000 MemSet&Commit cost time is ", end1.Sub(start1), "blocks is", blocks)
	fmt.Println("mavl BenchmarkStoreGetKvsFor10000 Get cost time is", end.Sub(start), "num is ", times, ",blocks is ", blocks)
	b.StopTimer()
}

func BenchmarkSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)
	mavldb.EnableMavlPrefix(true)
	defer mavldb.EnableMavlPrefix(false)
	var kv []*types.KeyValue
	var keys [][]byte
	var hash = drivers.EmptyRoot[:]
	start := time.Now()
	for i := 0; i < b.N; i++ {
		key := GetRandomString(MaxKeylenth)
		value := fmt.Sprintf("%s%d", key, i)
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
	end := time.Now()
	fmt.Println("mavl BenchmarkSet cost time is", end.Sub(start), "num is", b.N)
}

//这个用例测试Store.Set接口，一次调用保存一组kvs（30对）到数据库中。
func BenchmarkStoreSetKvs(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	for i := 0; i < 30; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	start := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash, err := store.Set(datas, true)
		assert.Nil(b, err)
		assert.NotNil(b, hash)
	}
	end := time.Now()
	fmt.Println("mavl BenchmarkSet cost time is", end.Sub(start), "num is", b.N)
}

func BenchmarkMemSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)
	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	for i := 0; i < b.N; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	start := time.Now()
	b.ResetTimer()
	hash, err := store.MemSet(datas, true)
	assert.Nil(b, err)
	assert.NotNil(b, hash)
	end := time.Now()
	fmt.Println("mavl BenchmarkMemSet cost time is", end.Sub(start), "num is", b.N)
}

//这个用例测试Store.MemSet接口，一次调用保存一组kvs（30对）到数据库中。
func BenchmarkStoreMemSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	for i := 0; i < 30; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	start := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hash, err := store.MemSet(datas, true)
		assert.Nil(b, err)
		assert.NotNil(b, hash)
	}
	end := time.Now()
	fmt.Println("mavl BenchmarkMemSet cost time is", end.Sub(start), "num is", b.N)
}

func BenchmarkCommit(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	for i := 0; i < b.N; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}
	hash, err := store.MemSet(datas, true)
	assert.Nil(b, err)
	req := &types.ReqHash{
		Hash: hash,
	}

	start := time.Now()
	b.ResetTimer()
	_, err = store.Commit(req)
	assert.NoError(b, err, "NoError")
	end := time.Now()
	fmt.Println("mavl BenchmarkCommit cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}

//模拟真实的数据提交操作，数据之间的关系也保持正确（hash计算），统计的时间包括MemSet和Commit，可以减去之前用例中测试出来的MemSet的时间来估算Commit耗时
func BenchmarkStoreCommit(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*Store)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var key string
	var value string
	var keys [][]byte

	for i := 0; i < 30; i++ {
		key = GetRandomString(MaxKeylenth)
		value = fmt.Sprintf("v%d", i)
		keys = append(keys, []byte(string(key)))
		kv = append(kv, &types.KeyValue{[]byte(string(key)), []byte(string(value))})
	}
	datas := &types.StoreSet{
		drivers.EmptyRoot[:],
		kv,
		0}

	start := time.Now()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		datas.Height = int64(i)
		value = fmt.Sprintf("vv%d", i)
		for j := 0; j < 10; j++ {
			datas.KV[j].Value = []byte(value)
		}
		hash, err := store.MemSet(datas, true)
		assert.Nil(b, err)
		req := &types.ReqHash{
			Hash: hash,
		}
		_, err = store.Commit(req)
		assert.NoError(b, err, "NoError")
		datas.StateHash = hash
	}
	end := time.Now()
	fmt.Println("mavl BenchmarkCommit cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}
