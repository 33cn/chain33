// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

// memdb迭代器测试
func TestGoMemDBIterator(t *testing.T) {
	dir, err := ioutil.TempDir("", "gomemdb")
	require.NoError(t, err)
	t.Log(dir)

	memdb, err := NewGoMemDB("gomemdb", dir, 128)
	require.NoError(t, err)
	defer memdb.Close()

	testDBIterator(t, memdb)
}

func TestGoMemDBIteratorDel(t *testing.T) {
	dir, err := ioutil.TempDir("", "gomemdb")
	require.NoError(t, err)
	t.Log(dir)

	memdb, err := NewGoMemDB("gomemdb", dir, 128)
	require.NoError(t, err)
	defer memdb.Close()

	testDBIteratorDel(t, memdb)
}

func TestGoMemDBBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "gomemdb")
	require.NoError(t, err)
	t.Log(dir)

	leveldb, err := NewGoMemDB("gomemdb", dir, 128)
	require.NoError(t, err)
	defer leveldb.Close()
	testBatch(t, leveldb)
}

// memdb边界测试
func TestGoMemDBBoundary(t *testing.T) {
	dir, err := ioutil.TempDir("", "gomemdb")
	require.NoError(t, err)
	t.Log(dir)

	memdb, err := NewGoMemDB("gomemdb", dir, 128)
	require.NoError(t, err)
	defer memdb.Close()

	testDBBoundary(t, memdb)
}

func BenchmarkRandomGoMemDBReadsWrites(b *testing.B) {
	b.StopTimer()

	numItems := int64(1000000)
	internal := map[int64]int64{}
	for i := 0; i < int(numItems); i++ {
		internal[int64(i)] = int64(0)
	}
	db, err := NewGoMemDB(fmt.Sprintf("test_%x", RandStr(12)), "", 1000)
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
			idxBytes := int642Bytes(int64(idx))
			valBytes := int642Bytes(int64(val))
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
			idxBytes := int642Bytes(int64(idx))
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
