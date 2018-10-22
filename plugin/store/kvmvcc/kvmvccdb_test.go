package kvmvccdb

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"fmt"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	drivers "gitlab.33.cn/chain33/chain33/system/store"
	"gitlab.33.cn/chain33/chain33/types"
)

const MaxKeylenth int = 64

func newStoreCfg(dir string) *types.Store {
	return &types.Store{Name: "kvmvcc_test", Driver: "leveldb", DbPath: dir, DbCache: 100}
}

func newStoreCfgIter(dir string) (*types.Store, []byte) {
	return &types.Store{Name: "kvmvcc_test", Driver: "leveldb", DbPath: dir, DbCache: 100}, enableConfig()
}

func TestKvmvccdbNewClose(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
	assert.NotNil(t, store)

	store.Close()
}

func TestKvmvccdbSetGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
}

func TestKvmvccdbMemSet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
}

func TestKvmvccdbRollback(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
}

func TestKvmvccdbRollbackBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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

	//触发批量回滚
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
}

func enableConfig() []byte {
	data, _ := json.Marshal(&subConfig{EnableMVCCIter: true})
	return data
}

func TestIterateRangeByStateHash(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	store_cfg, sub := newStoreCfgIter(dir)
	store := New(store_cfg, sub).(*KVMVCCStore)
	assert.NotNil(t, store)

	execaddr := "0111vcBNSEA7fZhAdLJphDwQRQJa111"
	addr := "06htvcBNSEA7fZhAdLJphDwQRQJaHpy"
	addr1 := "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
	addr2 := "26htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
	addr3 := "36htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
	addr4 := "46htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"
	accCoin := account.NewCoinsAccount()

	account1 := &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addr1,
	}

	account2 := &types.Account{
		Balance: 900 * 1e8,
		Addr:    addr2,
	}

	account3 := &types.Account{
		Balance: 800 * 1e8,
		Addr:    addr3,
	}

	account4 := &types.Account{
		Balance: 700 * 1e8,
		Addr:    addr4,
	}
	set1 := accCoin.GetKVSet(account1)
	set2 := accCoin.GetKVSet(account2)
	set3 := accCoin.GetKVSet(account3)
	set4 := accCoin.GetKVSet(account4)

	set5 := accCoin.GetExecKVSet(execaddr, account4)

	fmt.Println("---test case1-1 ---")
	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{Key: set4[0].GetKey(), Value: set4[0].GetValue()})
	kv = append(kv, &types.KeyValue{Key: set3[0].GetKey(), Value: set3[0].GetValue()})
	kv = append(kv, &types.KeyValue{Key: set1[0].GetKey(), Value: set1[0].GetValue()})
	kv = append(kv, &types.KeyValue{Key: set2[0].GetKey(), Value: set2[0].GetValue()})
	kv = append(kv, &types.KeyValue{Key: set5[0].GetKey(), Value: set5[0].GetValue()})
	for i := 0; i < len(kv); i++ {
		fmt.Println("key:", string(kv[i].Key), "value:", string(kv[i].Value))
	}
	datas := &types.StoreSet{drivers.EmptyRoot[:], kv, 0}
	hash, err := store.MemSet(datas, true)
	assert.Nil(t, err)
	var kvset []*types.KeyValue
	req := &types.ReqHash{hash}
	hash1 := make([]byte, len(hash))
	copy(hash1, hash)
	store.Commit(req)

	resp := &types.ReplyGetTotalCoins{}
	resp.Count = 100000

	store.IterateRangeByStateHash(hash, []byte("mavl-coins-bty-"), []byte("mavl-coins-bty-exec"), true, resp.IterateRangeByStateHash)
	fmt.Println("resp.Num=", resp.Num)
	fmt.Println("resp.Amount=", resp.Amount)

	assert.Equal(t, int64(4), resp.Num)
	assert.Equal(t, int64(340000000000), resp.Amount)

	fmt.Println("---test case1-2 ---")
	for i := 1; i <= 10; i++ {
		kvset = nil

		s1 := fmt.Sprintf("%03d", 11-i)
		addrx := addr + s1
		account := &types.Account{
			Balance: ((1000 + int64(i)) * 1e8),
			Addr:    addrx,
		}
		set := accCoin.GetKVSet(account)
		fmt.Println("key:", string(set[0].GetKey()), "value:", set[0].GetValue())
		kvset = append(kvset, &types.KeyValue{set[0].GetKey(), set[0].GetValue()})
		datas1 := &types.StoreSet{hash1, kvset, datas.Height + int64(i)}
		hash1, err = store.MemSet(datas1, true)
		assert.Nil(t, err)
		req := &types.ReqHash{hash1}
		store.Commit(req)
	}

	resp = &types.ReplyGetTotalCoins{}
	resp.Count = 100000
	store.IterateRangeByStateHash(hash1, []byte("mavl-coins-bty-"), []byte("mavl-coins-bty-exec"), true, resp.IterateRangeByStateHash)
	fmt.Println("resp.Num=", resp.Num)
	fmt.Println("resp.Amount=", resp.Amount)
	assert.Equal(t, int64(14), resp.Num)
	assert.Equal(t, int64(1345500000000), resp.Amount)

	fmt.Println("---test case1-3 ---")

	resp = &types.ReplyGetTotalCoins{}
	resp.Count = 100000
	store.IterateRangeByStateHash(hash1, []byte("mavl-coins-bty-06htvcBNSEA7fZhAdLJphDwQRQJaHpy003"), []byte("mavl-coins-bty-exec"), true, resp.IterateRangeByStateHash)
	fmt.Println("resp.Num=", resp.Num)
	fmt.Println("resp.Amount=", resp.Amount)
	assert.Equal(t, int64(12), resp.Num)
	assert.Equal(t, int64(1143600000000), resp.Amount)

	fmt.Println("---test case1-4 ---")

	resp = &types.ReplyGetTotalCoins{}
	resp.Count = 2
	store.IterateRangeByStateHash(hash1, []byte("mavl-coins-bty-06htvcBNSEA7fZhAdLJphDwQRQJaHpy003"), []byte("mavl-coins-bty-exec"), true, resp.IterateRangeByStateHash)
	fmt.Println("resp.Num=", resp.Num)
	fmt.Println("resp.Amount=", resp.Amount)
	assert.Equal(t, int64(2), resp.Num)
	assert.Equal(t, int64(201500000000), resp.Amount)

	fmt.Println("---test case1-5 ---")

	resp = &types.ReplyGetTotalCoins{}
	resp.Count = 2
	store.IterateRangeByStateHash(hash1, []byte("mavl-coins-bty-"), []byte("mavl-coins-bty-exec"), true, resp.IterateRangeByStateHash)
	fmt.Println("resp.Num=", resp.Num)
	fmt.Println("resp.Amount=", resp.Amount)
	assert.Equal(t, int64(2), resp.Num)
	assert.Equal(t, int64(201900000000), resp.Amount)

	fmt.Println("---test case1-6 ---")

	resp = &types.ReplyGetTotalCoins{}
	resp.Count = 10000
	store.IterateRangeByStateHash(hash, []byte("mavl-coins-bty-"), []byte("mavl-coins-bty-exec"), true, resp.IterateRangeByStateHash)
	fmt.Println("resp.Num=", resp.Num)
	fmt.Println("resp.Amount=", resp.Amount)
	assert.Equal(t, int64(0), resp.Num)
	assert.Equal(t, int64(0), resp.Amount)
}

func GetRandomString(length int) string {
	return common.GetRandPrintString(20, length)
}

func BenchmarkGet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var keys [][]byte
	var hash = drivers.EmptyRoot[:]
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

func BenchmarkStoreGetKvs4N(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
	fmt.Println("kvmvcc BenchmarkStoreGetKvs4N cost time is", end.Sub(start), "num is", b.N)

	b.StopTimer()
}

func BenchmarkStoreGetKvsForNN(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
	fmt.Println("kvmvcc BenchmarkStoreGetKvsForNN cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}

func BenchmarkStoreGetKvsFor10000(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
	fmt.Println("kvmvcc BenchmarkStoreGetKvsFor10000 MemSet&Commit cost time is ", end1.Sub(start1), "blocks is", blocks)
	fmt.Println("kvmvcc BenchmarkStoreGetKvsFor10000 Get cost time is", end.Sub(start), "num is ", times, ",blocks is ", blocks)
	b.StopTimer()
}

func BenchmarkGetIter(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	store_cfg, sub := newStoreCfgIter(dir)
	store := New(store_cfg, sub).(*KVMVCCStore)
	assert.NotNil(b, store)

	var kv []*types.KeyValue
	var keys [][]byte
	var hash = drivers.EmptyRoot[:]
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

func BenchmarkSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
	assert.NotNil(b, store)
	b.Log(dir)

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
	fmt.Println("mpt BenchmarkSet cost time is", end.Sub(start), "num is", b.N)
}

//上一个用例，一次性插入多对kv；本用例每次插入30对kv，分多次插入，测试性能表现。
func BenchmarkStoreSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
	fmt.Println("kvmvcc BenchmarkSet cost time is", end.Sub(start), "num is", b.N)
}

func BenchmarkSetIter(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	store_cfg, sub := newStoreCfgIter(dir)
	store := New(store_cfg, sub).(*KVMVCCStore)
	assert.NotNil(b, store)
	b.Log(dir)

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
	fmt.Println("kvmvcc BenchmarkSet cost time is", end.Sub(start), "num is", b.N)
}

func isDirExists(path string) bool {
	fi, err := os.Stat(path)

	if err != nil {
		return os.IsExist(err)
	} else {
		return fi.IsDir()
	}

	panic("not reached")
}

//一次设定多对kv，测试一次的时间/多少对kv，来算平均一对kv的耗时。
func BenchmarkMemSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
	fmt.Println("kvmvcc BenchmarkMemSet cost time is", end.Sub(start), "num is", b.N)
}

//一次设定30对kv，设定N次，计算每次设定30对kv的耗时。
func BenchmarkStoreMemSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
		req := &types.ReqHash{
			hash}
		store.Rollback(req)
	}
	end := time.Now()
	fmt.Println("kvmvcc BenchmarkStoreMemSet cost time is", end.Sub(start), "num is", b.N)
}

func BenchmarkCommit(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
	req := &types.ReqHash{
		Hash: hash,
	}
	_, err = store.Commit(req)
	assert.NoError(b, err, "NoError")

	end := time.Now()
	fmt.Println("kvmvcc BenchmarkCommit cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}

func BenchmarkStoreCommit(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	var store_cfg = newStoreCfg(dir)
	store := New(store_cfg, nil).(*KVMVCCStore)
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
	fmt.Println("kvmvcc BenchmarkStoreCommit cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}

//一次设定多对kv，测试一次的时间/多少对kv，来算平均一对kv的耗时。
func BenchmarkIterMemSet(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	store_cfg, sub := newStoreCfgIter(dir)
	store := New(store_cfg, sub).(*KVMVCCStore)
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
	fmt.Println("kvmvcc BenchmarkMemSet cost time is", end.Sub(start), "num is", b.N)
}

func BenchmarkIterCommit(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	store_cfg, sub := newStoreCfgIter(dir)
	store := New(store_cfg, sub).(*KVMVCCStore)
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
	req := &types.ReqHash{
		Hash: hash,
	}
	_, err = store.Commit(req)
	assert.NoError(b, err, "NoError")

	end := time.Now()
	fmt.Println("kvmvcc BenchmarkCommit cost time is", end.Sub(start), "num is", b.N)
	b.StopTimer()
}
