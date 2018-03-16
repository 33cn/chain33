package db

import (
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
	log "github.com/inconshreveable/log15"
)

var blog = log.New("module", "db.gobadgerdb")

type GoBadgerDB struct {
	db *badger.DB
}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoBadgerDB(name, dir, cache)
	}
	registerDBCreator(GoBadgerDBBackendStr, dbCreator, false)
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

func (db *GoBadgerDB) Get(key []byte) []byte {
	var val []byte
	err := db.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			blog.Error("Get", "txn.Get.error", err)
			return nil
		}
		val, err = item.Value()
		if err != nil {
			blog.Error("Get", "item.Value.error", err)
			return nil
		}
		return nil
	})

	if err != nil {
		blog.Error("Get", "error", err)
		return nil
	}
	return val
}

func (db *GoBadgerDB) Set(key []byte, value []byte) {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})

	if err != nil {
		blog.Error("Set", "error", err)
	}
}

func (db *GoBadgerDB) SetSync(key []byte, value []byte) {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Set(key, value)
		return err
	})

	if err != nil {
		blog.Error("SetSync", "error", err)
	}
}

func (db *GoBadgerDB) Delete(key []byte) {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		return err
	})

	if err != nil {
		blog.Error("Delete", "error", err)
	}
}

func (db *GoBadgerDB) DeleteSync(key []byte) {
	err := db.db.Update(func(txn *badger.Txn) error {
		err := txn.Delete(key)
		return err
	})

	if err != nil {
		blog.Error("DeleteSync", "error", err)
	}
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

func (db *GoBadgerDB) Iterator() Iterator {
	txn := db.db.NewTransaction(false)
	it := txn.NewIterator(badger.DefaultIteratorOptions)
	return &goBadgerDBIt{txn, it}
}

type goBadgerDBIt struct {
	txn *badger.Txn
	it  *badger.Iterator
}

func (it *goBadgerDBIt) Close() {
	it.it.Close()
	it.txn.Discard()
}

func (it *goBadgerDBIt) Key() []byte {
	return it.it.Item().Key()
}

func (it *goBadgerDBIt) Value() []byte {
	value, _ := it.it.Item().Value()
	return value
}

func (db *GoBadgerDB) NewBatch(sync bool) Batch {
	batch := db.db.NewTransaction(true)
	return &GoBadgerDBBatch{db, batch}
}

func (db *GoBadgerDB) PrefixScan(key []byte) (txhashs [][]byte) {
	err := db.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(key); it.ValidForPrefix(key); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("PrefixScan", "key", string(item.Key()), "value", value)
			txhashs = append(txhashs, value)
		}
		return nil
	})
	if err != nil {
		blog.Error("PrefixScan", "error", err)
		txhashs = txhashs[:]
	}
	return
}

func (db *GoBadgerDB) List(prefix, key []byte, count, direction int32) (values [][]byte) {
	if len(key) == 0 {
		if direction == 1 {
			return db.IteratorScanFromFirst(prefix, count, direction)
		} else {
			return db.IteratorScanFromLast(prefix, count, direction)
		}
	}
	return db.IteratorScan(prefix, key, count, direction)
}

func (db *GoBadgerDB) IteratorScan(Prefix []byte, key []byte, count int32, direction int32) (values [][]byte) {
	err := db.db.View(func(txn *badger.Txn) error {
		var i int32 = 0
		opts := badger.DefaultIteratorOptions
		if direction == 0 {
			opts.Reverse = true
		}
		it := txn.NewIterator(opts)
		defer it.Close()
		it.Seek(Prefix)
		for it.Seek(key); it.ValidForPrefix(Prefix); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("IteratorScan", "key", string(item.Key()), "value", value)
			values = append(values, value)
			i++
			if i == count {
				break
			}
		}
		return nil
	})

	if err != nil {
		blog.Error("IteratorScan", "error", err)
		values = values[:]
	}
	return
}

func (db *GoBadgerDB) IteratorScanFromFirst(key []byte, count int32, direction int32) (values [][]byte) {
	err := db.db.View(func(txn *badger.Txn) error {
		var i int32 = 0
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(key); it.ValidForPrefix(key); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("IteratorScanFromFirst", "key", string(item.Key()), "value", value)
			values = append(values, value)
			i++
			if i == count {
				break
			}
		}
		return nil
	})

	if err != nil {
		blog.Error("IteratorScanFromFirst", "error", err)
		values = values[:]
	}
	return
}

func (db *GoBadgerDB) IteratorScanFromLast(key []byte, count int32, direction int32) (values [][]byte) {
	err := db.db.View(func(txn *badger.Txn) error {
		var i int32 = 0
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := append(key, 0xFF)
		for it.Seek(prefix); it.ValidForPrefix(key); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("IteratorScanFromLast", "key", string(item.Key()), "value", value)
			values = append(values, value)
			i++
			if i == count {
				break
			}
		}
		return nil
	})

	if err != nil {
		blog.Error("IteratorScanFromLast", "error", err)
		values = values[:]
	}
	return
}

//--------------------------------------------------------------------------------

type GoBadgerDBBatch struct {
	db    *GoBadgerDB
	batch *badger.Txn
	//wop   *opt.WriteOptions
}

func (mBatch *GoBadgerDBBatch) Set(key, value []byte) {
	mBatch.batch.Set(key, value)
}

func (mBatch *GoBadgerDBBatch) Delete(key []byte) {
	mBatch.batch.Delete(key)
}

func (mBatch *GoBadgerDBBatch) Write() {
	defer mBatch.batch.Discard()

	if err := mBatch.batch.Commit(nil); err != nil {
		blog.Error("Write", "error", err)
	}
}
