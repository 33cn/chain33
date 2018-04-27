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

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir

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
		val, err := item.Value()
		if err != nil {
			return err
		}
		t.Log("The answer is: ", string(val))
		return nil
	})

	require.NoError(t, err)
}
