package db

import (
	"bytes"
	"fmt"
	"path"

	log "github.com/inconshreveable/log15"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoLevelDB(name, dir, cache)
	}
	registerDBCreator(LevelDBBackendStr, dbCreator, false)
	registerDBCreator(GoLevelDBBackendStr, dbCreator, false)
}

type GoLevelDB struct {
	db *leveldb.DB
}

func NewGoLevelDB(name string, dir string, cache int) (*GoLevelDB, error) {
	dbPath := path.Join(dir, name+".db")
	if cache == 0 {
		cache = 64
	}
	handles := cache
	if handles < 16 {
		handles = 16
	}
	if cache < 16 {
		cache = 16
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

func (db *GoLevelDB) Get(key []byte) []byte {
	res, err := db.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil
		} else {
			fmt.Println(err)
		}
	}
	return res
}

func (db *GoLevelDB) Set(key []byte, value []byte) {
	err := db.db.Put(key, value, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func (db *GoLevelDB) SetSync(key []byte, value []byte) {
	err := db.db.Put(key, value, &opt.WriteOptions{Sync: true})
	if err != nil {
		fmt.Println(err)
	}
}

func (db *GoLevelDB) Delete(key []byte) {
	err := db.db.Delete(key, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func (db *GoLevelDB) DeleteSync(key []byte) {
	err := db.db.Delete(key, &opt.WriteOptions{Sync: true})
	if err != nil {
		fmt.Println(err)
	}
}

func (db *GoLevelDB) DB() *leveldb.DB {
	return db.db
}

func (db *GoLevelDB) Close() {
	db.db.Close()
}

func (db *GoLevelDB) Print() {
	str, _ := db.db.GetProperty("leveldb.stats")
	fmt.Printf("%v\n", str)

	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		fmt.Printf("[%X]:\t[%X]\n", key, value)
	}
}

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

func (db *GoLevelDB) Iterator(prefix []byte, reserve bool) Iterator {
	r := &util.Range{prefix, bytesPrefix(prefix)}
	it := db.db.NewIterator(r, nil)
	return &goLevelDBIt{it, reserve, prefix}
}

type goLevelDBIt struct {
	iterator.Iterator
	reserve bool
	prefix  []byte
}

func (dbit *goLevelDBIt) Close() {
	dbit.Release()
}

func (dbit *goLevelDBIt) Next() bool {
	if dbit.reserve {
		return dbit.Prev()
	}
	return dbit.Next()
}

func (dbit *goLevelDBIt) Rewind() bool {
	if dbit.reserve {
		return dbit.Last()
	}
	return dbit.First()
}

func (db *GoLevelDB) NewBatch(sync bool) Batch {
	batch := new(leveldb.Batch)
	wop := &opt.WriteOptions{Sync: sync}
	return &goLevelDBBatch{db, batch, wop}
}

//Helper for database api
/*
	PrefixScan(key []byte) [][]byte
	IteratorScan(Prefix []byte, key []byte, count int32, direction int32) [][]byte
	IteratorScanFromLast(key []byte, count int32, direction int32) [][]byte
	List(prefix, key []byte, count, direction int32) (values [][]byte)
*/
func (db *GoLevelDB) PrefixScan(key []byte) (txhashs [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(key), nil)
	for iter.Next() {
		value := iter.Value()
		//fmt.Printf("PrefixScan:%s\n", string(iter.Key()))
		value1 := make([]byte, len(value))
		copy(value1, value)
		txhashs = append(txhashs, value1)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return txhashs
}

func (db *GoLevelDB) List(prefix, key []byte, count, direction int32) (values [][]byte) {
	if len(key) == 0 {
		if direction == 1 {
			return db.IteratorScanFromFirst(prefix, count, direction)
		} else {
			return db.IteratorScanFromLast(prefix, count, direction)
		}
	}
	return db.IteratorScan(prefix, key, count, direction)
}

func (db *GoLevelDB) IteratorScan(Prefix []byte, key []byte, count int32, direction int32) (values [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(Prefix), nil)
	var i int32 = 0
	for ok := iter.Seek(key); ok; {
		if i == 0 && !bytes.Equal(key, iter.Key()) {
			log.Info("IteratorScan Equal ", "key", string(iter.Key()))
			return nil
		}
		if i != 0 {
			value := iter.Value()
			//log.Info("IteratorScan", "key", string(iter.Key()))
			value1 := make([]byte, len(value))
			copy(value1, value)
			values = append(values, value1)
		}
		if direction == 0 {
			ok = iter.Prev()
		} else {
			ok = iter.Next()
		}
		i++
		if i > count {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return values
}

func (db *GoLevelDB) IteratorScanFromFirst(key []byte, count int32, direction int32) (values [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(key), nil)
	var i int32 = 0
	for ok := iter.First(); ok; {
		value := iter.Value()
		value1 := make([]byte, len(value))
		copy(value1, value)
		values = append(values, value1)
		ok = iter.Next()
		i++
		if i == count {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return values
}

func (db *GoLevelDB) IteratorScanFromLast(key []byte, count int32, direction int32) (values [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(key), nil)
	var i int32 = 0
	for ok := iter.Last(); ok; {
		value := iter.Value()
		value1 := make([]byte, len(value))
		copy(value1, value)
		values = append(values, value1)
		ok = iter.Prev()
		i++
		if i == count {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return values
}

//--------------------------------------------------------------------------------

type goLevelDBBatch struct {
	db    *GoLevelDB
	batch *leveldb.Batch
	wop   *opt.WriteOptions
}

func (mBatch *goLevelDBBatch) Set(key, value []byte) {
	mBatch.batch.Put(key, value)
}

func (mBatch *goLevelDBBatch) Delete(key []byte) {
	mBatch.batch.Delete(key)
}

func (mBatch *goLevelDBBatch) Write() {
	err := mBatch.db.db.Write(mBatch.batch, mBatch.wop)
	if err != nil {
		fmt.Println(err)
	}
}
