// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"io/ioutil"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/require"
)

// badgerdb迭代器测试
func TestGoBadgerDBIterator(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	t.Log(dir)

	badgerdb, err := NewGoBadgerDB("gobadgerdb", dir, 128)
	require.NoError(t, err)
	defer badgerdb.Close()

	testDBIterator(t, badgerdb)
}

func TestGoBadgerDBIteratorDel(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	t.Log(dir)

	badgerdb, err := NewGoBadgerDB("gobadgerdb", dir, 128)
	require.NoError(t, err)
	defer badgerdb.Close()

	testDBIteratorDel(t, badgerdb)
}

// badgerdb边界测试
func TestGoBadgerDBBoundary(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	t.Log(dir)

	badgerdb, err := NewGoBadgerDB("gobadgerdb", dir, 128)
	require.NoError(t, err)
	defer badgerdb.Close()

	testDBBoundary(t, badgerdb)
}

func TestBadgerDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	t.Log(dir)

	opts := badger.DefaultOptions(dir)
	badgerdb, err := badger.Open(opts)
	require.NoError(t, err)
	defer badgerdb.Close()

	err = badgerdb.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("key1"), []byte("hello"))
		return err
	})
	require.NoError(t, err)

	err = badgerdb.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("key1"))
		if err != nil {
			return err
		}
		val, err := item.ValueCopy(nil)
		if err != nil {
			return err
		}
		t.Log("The answer is: ", string(val))
		return nil
	})

	require.NoError(t, err)
}

func TestBadgerBatchDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger121")
	require.NoError(t, err)
	//t.Log(dir)

	badgerdb, err := NewGoBadgerDB("gobadgerdb", dir, 128)
	require.NoError(t, err)
	batch := badgerdb.NewBatch(true)

	batch.Set([]byte("123"), []byte("121"))
	batch.Set([]byte("124"), []byte("111"))
	batch.Set([]byte("125"), []byte("111"))
	batch.Write()

	batch.Reset()

	batch.Set([]byte("126"), []byte("111"))
	batch.Write()

	value, err := badgerdb.Get([]byte("123"))
	require.NoError(t, err)
	require.Equal(t, string(value), "121")
	value, err = badgerdb.Get([]byte("126"))
	require.NoError(t, err)
	require.Equal(t, string(value), "111")
}

func TestBadgerBatch(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger1212")
	require.NoError(t, err)
	t.Log(dir)

	db, err := NewGoBadgerDB("gobadgerdb", dir, 128)
	require.NoError(t, err)
	defer db.Close()
	testBatch(t, db)
}
