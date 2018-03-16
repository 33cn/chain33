package db

import (
	"github.com/dgraph-io/badger"
	"io/ioutil"
	"testing"
	"github.com/stretchr/testify/require"
)

func TestBadger(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	t.Log(dir)

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir

	db, err := badger.Open(opts)
	require.NoError(t, err)
	defer db.Close()

	err = db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("key1"), []byte("hello"))
		return err
	})
	require.NoError(t, err)

	err = db.View(func(txn *badger.Txn) error {
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

func TestBadgerDB(t *testing.T) {
	dir, err := ioutil.TempDir("", "badger")
	require.NoError(t, err)
	t.Log(dir)

	badgerdb, err := NewGoBadgerDB("gobagderdb", dir, 16)
	require.NoError(t, err)
	defer badgerdb.Close()

	t.Log("test Set")
	badgerdb.Set([]byte("aaaaaa/1"), []byte("aaaaaa/1"))
	badgerdb.Set([]byte("my_key/1"), []byte("my_key/1"))
	badgerdb.Set([]byte("my_key/2"), []byte("my_key/2"))
	badgerdb.Set([]byte("my_key/3"), []byte("my_key/3"))
	badgerdb.Set([]byte("my_key/4"), []byte("my_key/4"))
	badgerdb.Set([]byte("my"), []byte("my"))
	badgerdb.Set([]byte("my_"), []byte("my_"))
	badgerdb.Set([]byte("zzzzzz/1"), []byte("zzzzzz/1"))

	t.Log("test Get")
	v := badgerdb.Get([]byte("aaaaaa/1"))
	require.Equal(t, string(v), "aaaaaa/1")

	t.Log("test PrefixScan")
	list := badgerdb.PrefixScan(nil)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("aaaaaa/1"), []byte("my"), []byte("my_"), []byte("my_key/1"), []byte("my_key/2"), []byte("my_key/3"), []byte("my_key/4"), []byte("zzzzzz/1")})

	t.Log("test IteratorScanFromFirst")
	list = badgerdb.IteratorScanFromFirst([]byte("my"), 2, 1)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my"), []byte("my_")})

	t.Log("test IteratorScanFromLast")
	list = badgerdb.IteratorScanFromLast([]byte("my"), 100, 1)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/4"), []byte("my_key/3"), []byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})

	t.Log("test IteratorScan 1")
	list = badgerdb.IteratorScan([]byte("my"), []byte("my_key/3"), 100, 1)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/3"), []byte("my_key/4")})

	t.Log("test IteratorScan 0")
	list = badgerdb.IteratorScan([]byte("my"), []byte("my_key/3"), 100, 0)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/3"), []byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})
}
