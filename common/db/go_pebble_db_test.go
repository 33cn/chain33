// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/stretchr/testify/require"
)

// pebble迭代器测试
func TestPebbleDBIterator(t *testing.T) {
	dir, err := ioutil.TempDir("", "pebble")
	require.NoError(t, err)
	t.Log(dir)
	db, err := NewPebbleDB("pebble", dir, 128)
	require.NoError(t, err)
	defer db.Close()
	testDBIterator(t, db)
}

func TestPebbleDBIteratorAll(t *testing.T) {
	dir, err := ioutil.TempDir("", "pebble")
	require.NoError(t, err)
	t.Log(dir)
	db, err := NewPebbleDB("pebble", dir, 128)
	require.NoError(t, err)
	defer db.Close()
	testDBIteratorAllKey(t, db)
}

func TestPebbleDBIteratorReserverExample(t *testing.T) {
	dir, err := ioutil.TempDir("", "pebble")
	require.NoError(t, err)
	t.Log(dir)
	db, err := NewPebbleDB("pebble", dir, 128)
	require.NoError(t, err)
	defer db.Close()
	testDBIteratorReserverExample(t, db)
}

func TestPebbleDBIteratorDel(t *testing.T) {
	dir, err := ioutil.TempDir("", "pebble")
	require.NoError(t, err)
	t.Log(dir)

	db, err := NewPebbleDB("pebble", dir, 128)
	require.NoError(t, err)
	defer db.Close()

	testDBIteratorDel(t, db)
}

func TestPebbleDBBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "pebble")
	require.NoError(t, err)
	t.Log(dir)

	db, err := NewPebbleDB("pebble", dir, 128)
	require.NoError(t, err)
	defer db.Close()
	testBatch(t, db)
}

// pebble边界测试
func TestPebbleDBBoundary(t *testing.T) {
	dir, err := ioutil.TempDir("", "pebble")
	require.NoError(t, err)
	t.Log(dir)

	db, err := NewPebbleDB("pebble", dir, 128)
	require.NoError(t, err)
	defer db.Close()

	testDBBoundary(t, db)
}

func BenchmarkPebbleBatchWrites(b *testing.B) {
	dir, err := ioutil.TempDir("", "example")
	require.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewPebbleDB("pebble", dir, 100)
	require.Nil(b, err)
	defer db.Close()
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
			require.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	err = batch.Write()
	require.Nil(b, err)
	b.StopTimer()
}

func BenchmarkPebbleBatchWrites1k(b *testing.B) {
	benchmarkPebbleBatchWrites(b, 1)
}

func BenchmarkPebbleBatchWrites16k(b *testing.B) {
	benchmarkPebbleBatchWrites(b, 16)
}

func BenchmarkPebbleBatchWrites256k(b *testing.B) {
	benchmarkPebbleBatchWrites(b, 256)
}

func BenchmarkPebbleBatchWrites1M(b *testing.B) {
	benchmarkPebbleBatchWrites(b, 1024)
}

//func BenchmarkPebbleBatchWrites4M(b *testing.B) {
//	benchmarkPebbleBatchWrites(b, 1024*4)
//}

func benchmarkPebbleBatchWrites(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	require.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewPebbleDB("pebble", dir, 100)
	require.Nil(b, err)
	defer db.Close()
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
			require.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	if batch.ValueSize() > 0 {
		err = batch.Write()
		require.Nil(b, err)
	}
	b.StopTimer()
}

func BenchmarkPebbleRandomReads1K(b *testing.B) {
	benchmarkPebbleRandomReads(b, 1)
}

func BenchmarkPebbleRandomReads16K(b *testing.B) {
	benchmarkPebbleRandomReads(b, 16)
}

func BenchmarkPebbleRandomReads256K(b *testing.B) {
	benchmarkPebbleRandomReads(b, 256)
}

func BenchmarkPebbleRandomReads1M(b *testing.B) {
	benchmarkPebbleRandomReads(b, 1024)
}

func benchmarkPebbleRandomReads(b *testing.B, size int) {
	dir, err := ioutil.TempDir("", "example")
	require.Nil(b, err)
	defer os.RemoveAll(dir)
	db, err := NewPebbleDB("pebble", dir, 100)
	require.Nil(b, err)
	defer db.Close()
	batch := db.NewBatch(true)
	var keys [][]byte
	for i := 0; i < 32*1024/size; i++ {
		key := common.GetRandBytes(20, 64)
		value := common.GetRandBytes(size*1024, size*1024)
		batch.Set(key, value)
		keys = append(keys, key)
		if batch.ValueSize() > 1<<20 {
			err := batch.Write()
			require.Nil(b, err)
			batch = db.NewBatch(true)
		}
	}
	if batch.ValueSize() > 0 {
		err = batch.Write()
		require.Nil(b, err)
	}
	//开始rand 读取
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		index := RandInt() % len(keys)
		key := keys[index]
		_, err := db.Get(key)
		require.Nil(b, err)
	}
	b.StopTimer()
}
func BenchmarkPebbleRandomReadsWrites(b *testing.B) {
	numItems := int64(1000000)
	internal := map[int64]int64{}
	for i := 0; i < int(numItems); i++ {
		internal[int64(i)] = int64(0)
	}
	dir := fmt.Sprintf("test_%x", RandStr(12))
	defer os.RemoveAll(dir)
	db, err := NewPebbleDB("pebble", dir, 1000)
	if err != nil {
		b.Fatal(err.Error())
		return
	}
	defer db.Close()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Write someing
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
	b.StopTimer()
}

// pebble返回值测试
func TestPebbleDBResult(t *testing.T) {
	dir, err := ioutil.TempDir("", "pebble")
	require.NoError(t, err)
	t.Log(dir)

	db, err := NewPebbleDB("pebble", dir, 128)
	require.NoError(t, err)
	defer db.Close()

	testDBIteratorResult(t, db)
}
