package db

import (
	"fmt"
	"path"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

func init() {
	dbCreator := func(name string, dir string) (DB, error) {
		return NewGoLevelDB(name, dir)
	}
	registerDBCreator(LevelDBBackendStr, dbCreator, false)
	registerDBCreator(GoLevelDBBackendStr, dbCreator, false)
}

type GoLevelDB struct {
	db *leveldb.DB
}

func NewGoLevelDB(name string, dir string) (*GoLevelDB, error) {
	dbPath := path.Join(dir, name+".db")
	cache := 128
	handles := 128
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

func (db *GoLevelDB) Iterator() Iterator {
	return db.db.NewIterator(nil, nil)
}

func (db *GoLevelDB) NewBatch(sync bool) Batch {
	batch := new(leveldb.Batch)
	wop := &opt.WriteOptions{Sync: sync}
	return &goLevelDBBatch{db, batch, wop}
}

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
func (db *GoLevelDB) IteratorScan(key []byte, count int32, direction int32) (values [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix([]byte("Tx:")), nil)
	var i int32 = 0
	for ok := iter.Seek(key); ok; {
		value := iter.Value()
		//fmt.Printf("PrefixScan:%v\n", value)
		value1 := make([]byte, len(value))
		copy(value1, value)
		values = append(values, value1)
		if direction == 0 {
			ok = iter.Prev()
		} else {
			ok = iter.Next()
		}
		i++
		if i >= count {
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
		//fmt.Printf("IteratorScanFromLast:%v\n", string(iter.Key()))
		value1 := make([]byte, len(value))
		copy(value1, value)
		values = append(values, value1)
		ok = iter.Prev()
		i++
		if i >= count {
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
