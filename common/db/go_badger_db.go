// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
)

var blog = log.New("module", "db.gobadgerdb")

type GoBadgerDB struct {
	TransactionDB
	db *badger.DB
}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoBadgerDB(name, dir, cache)
	}
	registerDBCreator(goBadgerDBBackendStr, dbCreator, false)
}

func NewGoBadgerDB(name string, dir string, cache int) (*GoBadgerDB, error) {
	opts := badger.DefaultOptions
	opts.Dir = dir
	opts.ValueDir = dir
	if cache <= 128 {
		opts.ValueLogLoadingMode = options.FileIO
		//opts.MaxTableSize = int64(cache) << 18 // cache = 128, MaxTableSize = 32M
		opts.NumCompactors = 1
		opts.NumMemtables = 1
		opts.NumLevelZeroTables = 1
		opts.NumLevelZeroTablesStall = 2
		opts.TableLoadingMode = options.MemoryMap
		opts.ValueLogFileSize = 1 << 28 // 256M
	}

	db, err := badger.Open(opts)
	if err != nil {
		blog.Error("NewGoBadgerDB", "error", err)
		return nil, err
	}

	return &GoBadgerDB{db: db}, nil
}

func (db *GoBadgerDB) Get(key []byte) ([]byte, error) {
	var val []byte
	err := db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrNotFoundInDb
			} else {
				blog.Error("Get", "txn.Get.error", err)
				return err
			}
		}
		val, err = item.Value()
		if err != nil {
			blog.Error("Get", "item.Value.error", err)
			return err
		}

		// 兼容leveldb
		if val == nil {
			val = make([]byte, 0)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return val, nil
}

func (db *GoBadgerDB) Set(key []byte, value []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})

	if err != nil {
		blog.Error("Set", "error", err)
		return err
	}
	return nil
}

func (db *GoBadgerDB) SetSync(key []byte, value []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})

	if err != nil {
		blog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

func (db *GoBadgerDB) Delete(key []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		return err
	})

	if err != nil {
		blog.Error("Delete", "error", err)
		return err
	}
	return nil
}

func (db *GoBadgerDB) DeleteSync(key []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		return err
	})

	if err != nil {
		blog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

func (db *GoBadgerDB) DB() *badger.DB {
	return db.db
}

func (db *GoBadgerDB) Close() {
	db.db.Close()
}

func (db *GoBadgerDB) Print() {
	// TODO: Returns statistics of the underlying DB
	err := db.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			v, err := item.Value()
			if err != nil {
				return err
			}
			blog.Info("Print", "key", string(k), "value", string(v))
			//blog.Info("Print", "key", string(item.Key()))
		}
		return nil
	})
	if err != nil {
		blog.Error("Print", err)
	}
}

func (db *GoBadgerDB) Stats() map[string]string {
	//TODO
	return nil
}

func (db *GoBadgerDB) Iterator(start, end []byte, reverse bool) Iterator {
	txn := db.db.NewTransaction(false)
	opts := badger.DefaultIteratorOptions
	opts.Reverse = reverse
	it := txn.NewIterator(opts)
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	if reverse {
		it.Seek(end)
	} else {
		it.Seek(start)
	}
	return &goBadgerDBIt{it, itBase{start, end, reverse}, txn, nil}
}

func (db *GoBadgerDB) BatchGet(keys [][]byte) (value [][]byte, err error) {
	blog.Error("BatchGet", "Need to implement")
	return nil, nil
}

type goBadgerDBIt struct {
	*badger.Iterator
	itBase
	txn *badger.Txn
	err error
}

func (it *goBadgerDBIt) Next() bool {
	it.Iterator.Next()
	return it.Valid()
}

func (it *goBadgerDBIt) Rewind() bool {
	if it.reverse {
		it.Seek(it.end)
	} else {
		it.Seek(it.start)
	}
	return it.Valid()
}

func (it *goBadgerDBIt) Seek(key []byte) bool {
	it.Iterator.Seek(key)
	return it.Valid()
}

func (it *goBadgerDBIt) Close() {
	it.Iterator.Close()
	it.txn.Discard()
}

func (it *goBadgerDBIt) Valid() bool {
	return it.Iterator.Valid() && it.checkKey(it.Key())
}

func (it *goBadgerDBIt) Key() []byte {
	return it.Item().Key()
}

func (it *goBadgerDBIt) Value() []byte {
	value, err := it.Item().Value()
	if err != nil {
		it.err = err
	}
	return value
}

func (it *goBadgerDBIt) ValueCopy() []byte {
	value, err := it.Item().ValueCopy(nil)
	if err != nil {
		it.err = err
	}
	return value
}

func (it *goBadgerDBIt) Error() error {
	return it.err
}

type GoBadgerDBBatch struct {
	db    *GoBadgerDB
	batch *badger.Txn
	//wop   *opt.WriteOptions
	size int
}

func (db *GoBadgerDB) NewBatch(sync bool) Batch {
	batch := db.db.NewTransaction(true)
	return &GoBadgerDBBatch{db, batch, 0}
}

func (mBatch *GoBadgerDBBatch) Set(key, value []byte) {
	mBatch.batch.Set(key, value)
	mBatch.size += len(value)
}

func (mBatch *GoBadgerDBBatch) Delete(key []byte) {
	mBatch.batch.Delete(key)
	mBatch.size += 1
}

func (mBatch *GoBadgerDBBatch) Write() error {
	defer mBatch.batch.Discard()

	if err := mBatch.batch.Commit(nil); err != nil {
		blog.Error("Write", "error", err)
		return err
	}
	return nil
}

func (mBatch *GoBadgerDBBatch) ValueSize() int {
	return mBatch.size
}

func (mBatch *GoBadgerDBBatch) Reset() {
	if nil != mBatch.db && nil != mBatch.db.db {
		mBatch.batch = mBatch.db.db.NewTransaction(true)
	}
	mBatch.size = 0
}
