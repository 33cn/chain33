package db

import (
	"testing"
	"io/ioutil"
	"code.aliyun.com/chain33/chain33/common/db"
	"github.com/dgraph-io/badger"
)

func TestDB(t *testing.T) {
	dir , err := ioutil.TempDir("", "badger")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(dir)

	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir

	db, err := badger.Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	err = db.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("key1"), []byte("hello"))
  		return err
	})
}

func TestBadger(t *testing.T) {
	dir , err := ioutil.TempDir("", "badger")
        if err != nil {
                t.Fatal(err)
        }
        t.Log(dir)

	badgerdb, err := db.NewGoBadgerDB("gobagderdb", dir, 16)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("test Set")
	badgerdb.Set([]byte("aaaaaa/1"), []byte("aaaaa"))
	badgerdb.Set([]byte("my_key/1"), []byte("1"))
	badgerdb.Set([]byte("my_key/2"), []byte("2"))
	badgerdb.Set([]byte("my_key/3"), []byte("3"))
	badgerdb.Set([]byte("my_key/4"), []byte("4"))
	badgerdb.Set([]byte("zzzzzz/1"), []byte("zzzzzz"))

	t.Log("test Get")
	v := badgerdb.Get([]byte("aaaaaa/1"))
	t.Log(string(v))

	t.Log("test PrefixScan")
	list := badgerdb.PrefixScan(nil)
	for _, v = range list {
		t.Log(string(v))
	}

	t.Log("test IteratorScanFromFirst")
	list = badgerdb.IteratorScanFromFirst([]byte("my"), 2, 0)
	for _, v = range list {
		t.Log(string(v))
	}

	t.Log("test IteratorScanFromLast")
	list = badgerdb.IteratorScanFromLast([]byte("my"), 100, 0)
	for _, v = range list {
		t.Log(string(v))
	}
}
