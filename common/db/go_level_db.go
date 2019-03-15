// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"path"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var llog = log.New("module", "db.goleveldb")

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoLevelDB(name, dir, cache)
	}
	registerDBCreator(levelDBBackendStr, dbCreator, false)
	registerDBCreator(goLevelDBBackendStr, dbCreator, false)
}

//GoLevelDB db
type GoLevelDB struct {
	BaseDB
	db *leveldb.DB
}

//NewGoLevelDB new
func NewGoLevelDB(name string, dir string, cache int) (*GoLevelDB, error) {
	dbPath := path.Join(dir, name+".db")
	if cache == 0 {
		cache = 64
	}
	handles := cache
	if handles < 16 {
		handles = 16
	}
	if cache < 4 {
		cache = 4
	}
	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(dbPath, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(dbPath, nil)
	}
	if err != nil {
		return nil, err
	}
	database := &GoLevelDB{db: db}
	return database, nil
}

//Get get
func (db *GoLevelDB) Get(key []byte) ([]byte, error) {
	res, err := db.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFoundInDb
		}
		llog.Error("Get", "error", err)
		return nil, err

	}
	return res, nil
}

//Set set
func (db *GoLevelDB) Set(key []byte, value []byte) error {
	err := db.db.Put(key, value, nil)
	if err != nil {
		llog.Error("Set", "error", err)
		return err
	}
	return nil
}

//SetSync 同步
func (db *GoLevelDB) SetSync(key []byte, value []byte) error {
	err := db.db.Put(key, value, &opt.WriteOptions{Sync: true})
	if err != nil {
		llog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

//Delete 删除
func (db *GoLevelDB) Delete(key []byte) error {
	err := db.db.Delete(key, nil)
	if err != nil {
		llog.Error("Delete", "error", err)
		return err
	}
	return nil
}

//DeleteSync 删除同步
func (db *GoLevelDB) DeleteSync(key []byte) error {
	err := db.db.Delete(key, &opt.WriteOptions{Sync: true})
	if err != nil {
		llog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

//DB db
func (db *GoLevelDB) DB() *leveldb.DB {
	return db.db
}

//Close 关闭
func (db *GoLevelDB) Close() {
	err := db.db.Close()
	if err != nil {
		llog.Error("Close", "error", err)
	}
}

//Print 打印
func (db *GoLevelDB) Print() {
	str, err := db.db.GetProperty("leveldb.stats")
	if err != nil {
		return
	}
	llog.Info("Print", "stats", str)

	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		//fmt.Printf("[%X]:\t[%X]\n", key, value)
		llog.Info("Print", "key", string(key), "value", string(value))
	}
}

//Stats ...
func (db *GoLevelDB) Stats() map[string]string {
	keys := []string{
		"leveldb.num-files-at-level{n}",
		"leveldb.stats",
		"leveldb.sstables",
		"leveldb.blockpool",
		"leveldb.cachedblock",
		"leveldb.openedtables",
		"leveldb.alivesnaps",
		"leveldb.aliveiters",
	}

	stats := make(map[string]string)
	for _, key := range keys {
		str, err := db.db.GetProperty(key)
		if err == nil {
			stats[key] = str
		}
	}
	return stats
}

//Iterator 迭代器
func (db *GoLevelDB) Iterator(start []byte, end []byte, reverse bool) Iterator {
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	r := &util.Range{Start: start, Limit: end}
	it := db.db.NewIterator(r, nil)
	return &goLevelDBIt{it, itBase{start, end, reverse}}
}

//BeginTx call panic when BeginTx not rewrite
func (db *GoLevelDB) BeginTx() (TxKV, error) {
	tx, err := db.db.OpenTransaction()
	if err != nil {
		return nil, err
	}
	return &goLevelDBTx{tx: tx}, nil
}

type goLevelDBIt struct {
	iterator.Iterator
	itBase
}

//Close 关闭
func (dbit *goLevelDBIt) Close() {
	dbit.Iterator.Release()
}

//Next next
func (dbit *goLevelDBIt) Next() bool {
	if dbit.reverse {
		return dbit.Iterator.Prev() && dbit.Valid()
	}
	return dbit.Iterator.Next() && dbit.Valid()
}

//Rewind ...
func (dbit *goLevelDBIt) Rewind() bool {
	if dbit.reverse {
		return dbit.Iterator.Last() && dbit.Valid()
	}
	return dbit.Iterator.First() && dbit.Valid()
}

func (dbit *goLevelDBIt) Value() []byte {
	return dbit.Iterator.Value()
}

func cloneByte(v []byte) []byte {
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *goLevelDBIt) ValueCopy() []byte {
	v := dbit.Iterator.Value()
	return cloneByte(v)
}

func (dbit *goLevelDBIt) Valid() bool {
	return dbit.Iterator.Valid() && dbit.checkKey(dbit.Key())
}

type goLevelDBBatch struct {
	db    *GoLevelDB
	batch *leveldb.Batch
	wop   *opt.WriteOptions
	size  int
	len   int
}

//NewBatch new
func (db *GoLevelDB) NewBatch(sync bool) Batch {
	batch := new(leveldb.Batch)
	wop := &opt.WriteOptions{Sync: sync}
	return &goLevelDBBatch{db, batch, wop, 0, 0}
}

func (mBatch *goLevelDBBatch) Set(key, value []byte) {
	mBatch.batch.Put(key, value)
	mBatch.size += len(key)
	mBatch.size += len(value)
	mBatch.len += len(value)
}

func (mBatch *goLevelDBBatch) Delete(key []byte) {
	mBatch.batch.Delete(key)
	mBatch.size += len(key)
	mBatch.len++
}

func (mBatch *goLevelDBBatch) Write() error {
	err := mBatch.db.db.Write(mBatch.batch, mBatch.wop)
	if err != nil {
		llog.Error("Write", "error", err)
		return err
	}
	return nil
}

func (mBatch *goLevelDBBatch) ValueSize() int {
	return mBatch.size
}

//ValueLen  batch数量
func (mBatch *goLevelDBBatch) ValueLen() int {
	return mBatch.len
}

func (mBatch *goLevelDBBatch) Reset() {
	mBatch.batch.Reset()
	mBatch.len = 0
	mBatch.size = 0
}

type goLevelDBTx struct {
	tx *leveldb.Transaction
}

func (db *goLevelDBTx) Commit() error {
	return db.tx.Commit()
}

func (db *goLevelDBTx) Rollback() {
	db.tx.Discard()
}

//Get get in transaction
func (db *goLevelDBTx) Get(key []byte) ([]byte, error) {
	res, err := db.tx.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFoundInDb
		}
		llog.Error("tx Get", "error", err)
		return nil, err
	}
	return res, nil
}

//Set set in transaction
func (db *goLevelDBTx) Set(key []byte, value []byte) error {
	err := db.tx.Put(key, value, nil)
	if err != nil {
		llog.Error("tx Set", "error", err)
		return err
	}
	return nil
}

//Iterator 迭代器 in transaction
func (db *goLevelDBTx) Iterator(start []byte, end []byte, reverse bool) Iterator {
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	r := &util.Range{Start: start, Limit: end}
	it := db.tx.NewIterator(r, nil)
	return &goLevelDBIt{it, itBase{start, end, reverse}}
}

//Begin call panic when Begin not rewrite
func (db *goLevelDBTx) Begin() {
	panic("Begin not impl")
}
