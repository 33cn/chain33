// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/33cn/chain33/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// leveldb迭代器测试
func TestGoLevelDBIterator(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)
	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	testDBIterator(t, leveldb)
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

	testDBIteratorDel(t, leveldb)
}

func TestLevelDBBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	require.NoError(t, err)
	t.Log(dir)

	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	testBatch(t, leveldb)
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

	testDBBoundary(t, leveldb)
}

func BenchmarkLevelDBBatchWrites(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewGoLevelDB("goleveldb", dir, 100)
	assert.Nil(b, err)
	batch := db.NewBatch(true)
	b.ResetTimer()
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
	b.StopTimer()
}

func BenchmarkLevelDBBatchWrites1k(b *testing.B) {
	benchmarkBatchWrites(b, 1)
}

func BenchmarkLevelDBBatchWrites16k(b *testing.B) {
	benchmarkBatchWrites(b, 16)
}

func BenchmarkLevelDBBatchWrites256k(b *testing.B) {
	benchmarkBatchWrites(b, 256)
}

func BenchmarkLevelDBBatchWrites1M(b *testing.B) {
	benchmarkBatchWrites(b, 1024)
}

//func BenchmarkLevelDBBatchWrites4M(b *testing.B) {
//	benchmarkBatchWrites(b, 1024*4)
//}

func benchmarkBatchWrites(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewGoLevelDB("goleveldb", dir, 100)
	assert.Nil(b, err)
	batch := db.NewBatch(true)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := common.GetRandBytes(20, 64)
		value := common.GetRandBytes(size*1024, size*1024)
		b.StartTimer()
		batch.Set(key, value)
		if i > 0 && i%100 == 0 {
			err := batch.Write()
			assert.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	err = batch.Write()
	assert.Nil(b, err)
	b.StopTimer()
}

func BenchmarkLevelDBRandomReads1K(b *testing.B) {
	benchmarkLevelDBRandomReads(b, 1)
}

func BenchmarkLevelDBRandomReads16K(b *testing.B) {
	benchmarkLevelDBRandomReads(b, 16)
}

func BenchmarkLevelDBRandomReads256K(b *testing.B) {
	benchmarkLevelDBRandomReads(b, 256)
}

func BenchmarkLevelDBRandomReads1M(b *testing.B) {
	benchmarkLevelDBRandomReads(b, 1024)
}

func benchmarkLevelDBRandomReads(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewGoLevelDB("goleveldb", dir, 100)
	assert.Nil(b, err)
	batch := db.NewBatch(true)
	var keys [][]byte
	for i := 0; i < 32*1024/size; i++ {
		key := common.GetRandBytes(20, 64)
		value := common.GetRandBytes(size*1024, size*1024)
		batch.Set(key, value)
		keys = append(keys, key)
		if batch.ValueSize() > 1<<20 {
			err := batch.Write()
			assert.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	if batch.ValueSize() > 0 {
		err = batch.Write()
		assert.Nil(b, err)
	}

	//开始rand 读取
	db.Close()
	db, err = NewGoLevelDB("goleveldb", dir, 1)
	assert.Nil(b, err)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := RandInt() % len(keys)
		key := keys[index]
		_, err := db.Get(key)
		assert.Nil(b, err)
	}
	b.StopTimer()
}

func BenchmarkLevelDBRandomReadsWrites(b *testing.B) {
	numItems := int64(1000000)
	internal := map[int64]int64{}
	for i := 0; i < int(numItems); i++ {
		internal[int64(i)] = int64(0)
	}
	dir := fmt.Sprintf("test_%x", RandStr(12))
	defer os.RemoveAll(dir)
	db, err := NewGoLevelDB("goleveldb", dir, 1000)
	if err != nil {
		b.Fatal(err.Error())
		return
	}
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
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	b.StopTimer()
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
