package db

import (
    "fmt"
    "github.com/dgraph-io/badger"
)

func init() {
    dbCreator := func(name string, dir string, cache int) (DB, error) {
        return NewGoBadgerDB(name, dir, cache)
    }
    registerDBCreator(GoBadgerDBBackendStr, dbCreator, false)
}

type GoBadgerDB struct {
    db *badger.DB
}


func NewGoBadgerDB(name string, dir string, cache int) (*GoBadgerDB, error) {
    opts := badger.DefaultOptions
    opts.Dir = dir
    opts.ValueDir = dir
    //opts.ValueLogLoadingMode = options.FileIO
    //opts.MaxTableSize = cache << 21

    db, err := badger.Open(opts)
    if err != nil {
        return nil, err
    }

    return &GoBadgerDB{db: db}, nil
}

func (db *GoBadgerDB) Get(key []byte) []byte {
    var val []byte
    err := db.db.View(func(txn *badger.Txn) error {
        item, err := txn.Get(key)
        if err != nil {
            return nil
        }
        val, err = item.Value()
        if err != nil {
            return nil
        }
        return nil
    })

    if err != nil {
        fmt.Println(err)
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
        fmt.Println(err)
    }
}

func (db *GoBadgerDB) SetSync(key []byte, value []byte) {
    err := db.db.Update(func(txn *badger.Txn) error {
        err := txn.Set(key, value)
            return err
    })

    if err != nil {
        fmt.Println(err)
    }
}

func (db *GoBadgerDB) Delete(key []byte) {
    err := db.db.Update(func(txn *badger.Txn) error {
        err := txn.Delete(key)
            return err
    })

    if err != nil {
        fmt.Println(err)
    }
}

func (db *GoBadgerDB) DeleteSync(key []byte) {
    err := db.db.Update(func(txn *badger.Txn) error {
        err := txn.Delete(key)
            return err
    })

    if err != nil {
        fmt.Println(err)
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
            fmt.Printf("key=%s, value=%s\n", k, v)
        }
        return nil
    })
    if err != nil {
        fmt.Println(err)
    }
}

func (db *GoBadgerDB) Stats() map[string]string {
    //TODO
    return nil
}

func (db *GoBadgerDB) Iterator() Iterator {
    return nil
    /*
    txn := db.db.NewTransaction(true)
    return txn.NewIterator(badger.DefaultIteratorOptions)
    */
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
            /*
            k := item.Key()
            v, err := item.Value()
            if err != nil {
                return err
            }
            fmt.Printf("key=%s, value=%s\n", k, v)
            */
            value, err := item.ValueCopy(nil)
            if err != nil {
                return err
            }
            txhashs = append(txhashs, value)
        }
        return nil
    })
    if err != nil {
        fmt.Println(err)
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
            /*
            k := item.Key()
            v, err := item.Value()
            if err != nil {
                return err
            }
            fmt.Printf("key=%s, value=%s\n", k, v)
            */
            value, err := item.ValueCopy(nil)
            if err != nil {
                return err
            }
            values = append(values, value)
            i++
            if i == count {
                break
            }
        }
        return nil
    })

    if err != nil {
        fmt.Println(err)
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
            /*
            k := item.Key()
            v, err := item.Value()
            if err != nil {
                return err
            }
            fmt.Printf("key=%s, value=%s\n", k, v)
            */
            value, err := item.ValueCopy(nil)
            if err != nil {
                return err
            }
            values = append(values, value)
            i++
            if i == count {
                break
            }
        }
        return nil
    })

    if err != nil {
        fmt.Println(err)
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
        for it.Seek(key); it.ValidForPrefix(key); it.Next() {
            item := it.Item()
            /*
            k := item.Key()
            v, err := item.Value()
            if err != nil {
                return err
            }
            fmt.Printf("key=%s, value=%s\n", k, v)
            */
            value, err := item.ValueCopy(nil)
            if err != nil {
                return err
            }
            values = append(values, value)
            i++
            if i == count {
                break
            }
        }
        return nil
    })

    if err != nil {
        fmt.Println(err)
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
        fmt.Println(err)
    }
}

