// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"testing"
	"time"

	comdb "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var random = rand.New(rand.NewSource(time.Now().UnixNano()))

const (
	strChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" // 62 characters
)

func RandInt() int {
	return random.Int()
}

func RandStr(length int) string {
	chars := []byte{}
MAIN_LOOP:
	for {
		val := random.Int63()
		for i := 0; i < 10; i++ {
			v := int(val & 0x3f) // rightmost 6 bits
			if v >= 62 {         // only 62 characters in strChars
				val >>= 6
				continue
			} else {
				chars = append(chars, strChars[v])
				if len(chars) == length {
					break MAIN_LOOP
				}
				val >>= 6
			}
		}
	}

	return string(chars)
}

func Int642Bytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func Bytes2Int64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

/*
// leveldb迭代器测试
func TestGoLevelDBIterator(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)
	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	ComTestDBIterator(t, leveldb)
}

func TestGoLevelDBIteratorAll(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)
	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	testDBIteratorAllKey(t, leveldb)
}

func TestGoLevelDBIteratorReserverExample(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)
	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	testDBIteratorReserverExample(t, leveldb)
}

func TestGoLevelDBIteratorDel(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)

	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()

	ComTestDBIteratorDel(t, leveldb)
}

func TestLevelDBBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)

	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	ComTestBatch(t, leveldb)
}

func TestLevelDBTransaction(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)

	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	testTransaction(t, leveldb)
}

// leveldb边界测试
func TestGoLevelDBBoundary(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)

	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()

	ComTestDBBoundary(t, leveldb)
}

func BenchmarkBatchWrites(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir + ".db")
	os.RemoveAll(dir + ".db")
	db, err := NewGoLevelDB(dir, "", 100)
	assert.Nil(b, err)
	batch := db.NewBatch(true)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := string(common.GetRandBytes(20, 64))
		value := fmt.Sprintf("v%d", i)
		b.StartTimer()
		batch.Set([]byte(key), []byte(value))
		if i > 0 && i%10000 == 0 {
			err := batch.Write()
			assert.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	err = batch.Write()
	assert.Nil(b, err)
}

func BenchmarkBatchWrites1M(b *testing.B) {
	benchmarkBatchWrites(b, 1024)
}

func BenchmarkBatchWrites1k(b *testing.B) {
	benchmarkBatchWrites(b, 1)
}

func BenchmarkBatchWrites16k(b *testing.B) {
	benchmarkBatchWrites(b, 16)
}

func benchmarkBatchWrites(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir + ".db")
	os.RemoveAll(dir + ".db")
	db, err := NewGoLevelDB(dir, "", 100)
	assert.Nil(b, err)
	batch := db.NewBatch(true)
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := common.GetRandBytes(20, 64)
		value := common.GetRandBytes(size*1024, size*1024)
		b.StartTimer()
		batch.Set(key, value)
		if i > 0 && i%1000 == 0 {
			err := batch.Write()
			assert.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	err = batch.Write()
	assert.Nil(b, err)
}

func BenchmarkBatchWrites32k(b *testing.B) {
	benchmarkBatchWrites(b, 32)
}

func BenchmarkRandomReadsWrites1K(b *testing.B) {
	benchmarkRandomReadsWrites(b, 1)
}

func BenchmarkRandomReadsWrites16K(b *testing.B) {
	benchmarkRandomReadsWrites(b, 16)
}

func BenchmarkRandomReadsWrites32K(b *testing.B) {
	benchmarkRandomReadsWrites(b, 32)
}

func benchmarkRandomReadsWrites(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir + ".db")
	os.RemoveAll(dir + ".db")
	db, err := NewGoLevelDB(dir, "", 100)
	assert.Nil(b, err)
	batch := db.NewBatch(true)
	var keys [][]byte
	for i := 0; i < 100000; i++ {
		key := common.GetRandBytes(20, 64)
		value := common.GetRandBytes(size*1024, size*1024)
		batch.Set(key, value)
		keys = append(keys, key)
		if i > 0 && i%1000 == 0 {
			err := batch.Write()
			assert.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	err = batch.Write()
	assert.Nil(b, err)
	//开始rand 读取
	db.Close()
	db, err = NewGoLevelDB(dir, "", 1)
	assert.Nil(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[RandInt()%len(keys)]
		_, err := db.Get(key)
		assert.Nil(b, err)
	}
}
func BenchmarkRandomReadsWrites(b *testing.B) {
	b.StopTimer()
	numItems := int64(1000000)
	internal := map[int64]int64{}
	for i := 0; i < int(numItems); i++ {
		internal[int64(i)] = int64(0)
	}
	dir := fmt.Sprintf("test_%x", RandStr(12))
	defer os.RemoveAll(dir + ".db")
	os.RemoveAll(dir + ".db")
	db, err := NewGoLevelDB(dir, "", 1000)
	if err != nil {
		b.Fatal(err.Error())
		return
	}
	fmt.Println("ok, starting")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		// Write something
		{
			idx := (int64(RandInt()) % numItems)
			internal[idx]++
			val := internal[idx]
			idxBytes := int642Bytes(idx)
			valBytes := int642Bytes(val)
			//fmt.Printf("Set %X -> %X\n", idxBytes, valBytes)
			db.Set(
				idxBytes,
				valBytes,
			)
		}
		// Read something
		{
			idx := (int64(RandInt()) % numItems)
			val := internal[idx]
			idxBytes := int642Bytes(idx)
			valBytes, _ := db.Get(idxBytes)
			//fmt.Printf("Get %X -> %X\n", idxBytes, valBytes)
			if val == 0 {
				if !bytes.Equal(valBytes, nil) {
					b.Errorf("Expected %v for %v, got %X",
						nil, idx, valBytes)
					break
				}
			} else {
				if len(valBytes) != 8 {
					b.Errorf("Expected length 8 for %v, got %X",
						idx, valBytes)
					break
				}
				valGot := bytes2Int64(valBytes)
				if val != valGot {
					b.Errorf("Expected %v for %v, got %v",
						val, idx, valGot)
					break
				}
			}
		}
	}

	db.Close()
}

func int642Bytes(i int64) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(i))
	return buf
}

func bytes2Int64(buf []byte) int64 {
	return int64(binary.BigEndian.Uint64(buf))
}

// leveldb返回值测试
func TestGoLevelDBResult(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)

	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()

	testDBIteratorResult(t, leveldb)
}

func testDBIteratorAllKey(t *testing.T, db comdb.DB) {
	var datas = [][]byte{
		[]byte("aa0"), []byte("aa1"), []byte("bb0"), []byte("bb1"), []byte("cc0"), []byte("cc1"),
	}
	for _, v := range datas {
		db.Set(v, v)
	}
	//一次遍历
	it := db.Iterator(nil, types.EmptyValue, false)
	i := 0
	for it.Rewind(); it.Valid(); it.Next() {
		assert.Equal(t, it.Key(), datas[i])
		db.Delete(it.Key())
		i++
		if i == 2 {
			break
		}
	}
	it.Close()
	//从第3个开始遍历
	it = db.Iterator([]byte("aa1"), types.EmptyValue, false)
	i = 2
	for it.Rewind(); it.Valid(); it.Next() {
		assert.Equal(t, it.Key(), datas[i])
		db.Delete(it.Key())
		i++
		if i == 4 {
			break
		}
	}
	it.Close()
	//从第5个开始遍历
	it = db.Iterator([]byte("bb1"), types.EmptyValue, false)
	i = 4
	for it.Rewind(); it.Valid(); it.Next() {
		assert.Equal(t, it.Key(), datas[i])
		db.Delete(it.Key())
		i++
		if i == 6 {
			break
		}
	}
	it.Close()
}

func testDBIteratorReserverExample(t *testing.T, db comdb.DB) {
	var datas = [][]byte{
		[]byte("aa0"), []byte("aa1"), []byte("bb0"), []byte("bb1"), []byte("cc0"), []byte("cc1"),
	}
	for _, v := range datas {
		db.Set(v, v)
	}
	// 从尾部到头一次遍历
	it := db.Iterator(nil, types.EmptyValue, true)
	i := 5
	for it.Rewind(); it.Valid(); it.Next() {
		assert.Equal(t, it.Key(), datas[i])
		//fmt.Println(i, string(it.Key()))
		i--
	}
	it.Close()
	assert.Equal(t, i, -1)

	// 从bb0开始从后到前遍历,end需要填入bb0的下一个，才可以遍历到bb0
	it = db.Iterator(nil, []byte("bb1"), true)
	i = 2
	for it.Rewind(); it.Valid(); it.Next() {
		assert.Equal(t, it.Key(), datas[i])
		//fmt.Println(i, string(it.Key()))
		i--
	}
	it.Close()
	assert.Equal(t, i, -1)

	// 反向前缀查找
	it = db.Iterator([]byte("bb"), nil, true)
	i = 3
	for it.Rewind(); it.Valid(); it.Next() {
		assert.Equal(t, it.Key(), datas[i])
		// fmt.Println(string(it.Key()))
		i--
	}
	it.Close()
	assert.Equal(t, i, 1)
}

func testTransaction(t *testing.T, db comdb.DB) {
	tx, err := db.BeginTx()
	assert.Nil(t, err)
	tx.Set([]byte("hello1"), []byte("world1"))
	value, err := tx.Get([]byte("hello1"))
	assert.Nil(t, err)
	assert.Equal(t, "world1", string(value))
	tx.Rollback()
	value, err = db.Get([]byte("hello1"))
	assert.Equal(t, types.ErrNotFound, err)
	assert.Equal(t, []byte(nil), value)

	tx, err = db.BeginTx()
	assert.Nil(t, err)
	tx.Set([]byte("hello2"), []byte("world2"))
	value, err = tx.Get([]byte("hello2"))
	assert.Nil(t, err)
	assert.Equal(t, "world2", string(value))
	err = tx.Commit()
	assert.Nil(t, err)
	value, err = db.Get([]byte("hello2"))
	assert.Nil(t, err)
	assert.Equal(t, "world2", string(value))
}

// 返回值测试
func testDBIteratorResult(t *testing.T, db comdb.DB) {
	t.Log("test Set")
	db.Set([]byte("aaaaaa/1"), []byte("aaaaaa/1"))
	db.Set([]byte("my_key/1"), []byte("my_value/1"))
	db.Set([]byte("my_key/2"), []byte("my_value/2"))
	db.Set([]byte("my_key/3"), []byte("my_value/3"))
	db.Set([]byte("my_key/4"), []byte("my_value/4"))
	db.Set([]byte("my"), []byte("my"))
	db.Set([]byte("my_"), []byte("my_"))
	db.Set([]byte("zzzzzz/1"), []byte("zzzzzz/1"))
	b, err := hex.DecodeString("ff")
	require.NoError(t, err)
	db.Set(b, []byte("0xff"))

	t.Log("test Get")
	v, _ := db.Get([]byte("aaaaaa/1"))
	require.Equal(t, string(v), "aaaaaa/1")
	//test list:
	it0 := comdb.NewListHelper(db)
	list0 := it0.List([]byte("my_key"), nil, 100, comdb.ListASC|comdb.ListKeyOnly)
	require.Equal(t, list0, [][]byte{[]byte("my_key/1"), []byte("my_key/2"), []byte("my_key/3"), []byte("my_key/4")})

	it1 := comdb.NewListHelper(db)
	list1 := it1.List([]byte("my_key"), nil, 100, comdb.ListASC)
	require.Equal(t, list1, [][]byte{[]byte("my_value/1"), []byte("my_value/2"), []byte("my_value/3"), []byte("my_value/4")})

	it2 := comdb.NewListHelper(db)
	list2 := it2.List([]byte("my_key"), nil, 100, comdb.ListASC|comdb.ListWithKey)
	require.Equal(t, 4, len(list2))
	for i, v := range list2 {
		var kv types.KeyValue
		err = types.Decode(v, &kv)
		require.Equal(t, nil, err)
		require.Equal(t, fmt.Sprintf("my_key/%d", i+1), string(kv.Key))
		require.Equal(t, fmt.Sprintf("my_value/%d", i+1), string(kv.Value))
	}
}
*/
// 迭代测试
func ComTestDBIterator(t *testing.T, db comdb.DB) {
	t.Log("test Set")
	db.Set([]byte("aaaaaa/1"), []byte("aaaaaa/1"))
	db.Set([]byte("my_key/1"), []byte("my_key/1"))
	db.Set([]byte("my_key/2"), []byte("my_key/2"))
	db.Set([]byte("my_key/3"), []byte("my_key/3"))
	db.Set([]byte("my_key/4"), []byte("my_key/4"))
	db.Set([]byte("my"), []byte("my"))
	db.Set([]byte("my_"), []byte("my_"))
	db.Set([]byte("zzzzzz/1"), []byte("zzzzzz/1"))
	b, err := hex.DecodeString("ff")
	require.NoError(t, err)
	db.Set(b, []byte("0xff"))

	t.Log("test Get")
	v, _ := db.Get([]byte("aaaaaa/1"))
	require.Equal(t, string(v), "aaaaaa/1")
	//test list:
	it0 := comdb.NewListHelper(db)
	list0 := it0.List(nil, nil, 100, 1)
	for _, v = range list0 {
		t.Log("list0", string(v))
	}
	t.Log("test PrefixScan")
	it := comdb.NewListHelper(db)
	list := it.PrefixScan(nil)
	for _, v = range list {
		t.Log("list:", string(v))
	}
	assert.Equal(t, list0, list)
	require.Equal(t, list, [][]byte{[]byte("aaaaaa/1"), []byte("my"), []byte("my_"), []byte("my_key/1"), []byte("my_key/2"), []byte("my_key/3"), []byte("my_key/4"), []byte("zzzzzz/1"), []byte("0xff")})
	t.Log("test IteratorScanFromFirst")
	list = it.IteratorScanFromFirst([]byte("my"), 2, comdb.ListASC)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my"), []byte("my_")})

	t.Log("test IteratorScanFromLast")
	list = it.IteratorScanFromLast([]byte("my"), 100, comdb.ListDESC)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/4"), []byte("my_key/3"), []byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})

	t.Log("test IteratorScan 1")
	list = it.IteratorScan([]byte("my"), []byte("my_key/3"), 100, comdb.ListASC)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/4")})

	t.Log("test IteratorScan 0")
	list = it.IteratorScan([]byte("my"), []byte("my_key/3"), 100, comdb.ListDESC)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})
}

func ComTestDBBoundary(t *testing.T, db comdb.DB) {
	a, _ := hex.DecodeString("0f")
	c, _ := hex.DecodeString("0fff")
	b, _ := hex.DecodeString("ff")
	d, _ := hex.DecodeString("ffff")
	db.Set(a, []byte("0x0f"))
	db.Set(c, []byte("0x0fff"))
	db.Set(b, []byte("0xff"))
	db.Set(d, []byte("0xffff"))

	var v []byte
	_ = v
	it := comdb.NewListHelper(db)

	// f为prefix
	t.Log("PrefixScan")
	list := it.PrefixScan(a)
	require.Equal(t, list, [][]byte{[]byte("0x0f"), []byte("0x0fff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(a, 2, comdb.ListASC)
	require.Equal(t, list, [][]byte{[]byte("0x0f"), []byte("0x0fff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(a, 100, comdb.ListDESC)
	require.Equal(t, list, [][]byte{[]byte("0x0fff"), []byte("0x0f")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(a, a, 100, comdb.ListASC)
	require.Equal(t, list, [][]byte{[]byte("0x0fff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(a, a, 100, comdb.ListDESC)
	require.Equal(t, list, [][]byte(nil))

	// ff为prefix
	t.Log("PrefixScan")
	list = it.PrefixScan(b)
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(b, 2, comdb.ListASC)
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(b, 100, comdb.ListDESC)
	require.Equal(t, list, [][]byte{[]byte("0xffff"), []byte("0xff")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(b, b, 100, comdb.ListASC)
	require.Equal(t, list, [][]byte{[]byte("0xffff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(b, d, 100, comdb.ListDESC)
	require.Equal(t, list, [][]byte{[]byte("0xff")})
}

func ComTestDBIteratorDel(t *testing.T, db comdb.DB) {
	for i := 0; i < 1000; i++ {
		k := []byte(fmt.Sprintf("my_key/%010d", i))
		v := []byte(fmt.Sprintf("my_value/%010d", i))
		db.Set(k, v)
	}

	prefix := []byte("my")
	it := db.Iterator(prefix, nil, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		t.Log(string(it.Key()), "*********", string(it.Value()))
		batch := db.NewBatch(true)
		batch.Delete(it.Key())
		batch.Write()
	}
}

func ComTestBatch(t *testing.T, db comdb.DB) {
	batch := db.NewBatch(false)
	batch.Set([]byte("hello"), []byte("world"))
	err := batch.Write()
	assert.Nil(t, err)

	batch = db.NewBatch(false)
	v, err := db.Get([]byte("hello"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("world"))

	//set and del
	batch.Set([]byte("hello1"), []byte("world"))
	batch.Set([]byte("hello2"), []byte("world"))
	batch.Set([]byte("hello3"), []byte("world"))
	batch.Set([]byte("hello4"), []byte("world"))
	batch.Set([]byte("hello5"), []byte("world"))
	batch.Delete([]byte("hello1"))
	err = batch.Write()
	assert.Nil(t, err)
	v, err = db.Get([]byte("hello1"))
	assert.Equal(t, err, types.ErrNotFound)
	assert.Nil(t, v)
}
