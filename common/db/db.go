package db

import (
	"errors"
	"fmt"
<<<<<<< HEAD
=======

>>>>>>> develop
	lru "github.com/hashicorp/golang-lru"
)

var ErrNotFoundInDb = errors.New("ErrNotFoundInDb")

type Lister interface {
	List(prefix, key []byte, count, direction int32) ([][]byte, error)
	PrefixCount(prefix []byte) int64
	//AddrTxsCount(key []byte) int64
}

type KV interface {
	Get(key []byte) ([]byte, error)
	BatchGet(keys [][]byte) (values [][]byte, err error)
	Set(key []byte, value []byte) (err error)
	Begin()
	Rollback()
	Commit()
}

type KVDB interface {
	KV
	Lister
}

type DB interface {
	KV
	SetSync([]byte, []byte) error
	Delete([]byte) error
	DeleteSync([]byte) error
	Close()
	NewBatch(sync bool) Batch
	//迭代prefix 范围的所有key value, 支持正反顺序迭代
	Iterator(prefix []byte, reserver bool) Iterator
	// For debugging
	Print()
	Stats() map[string]string
	SetCacheSize(size int)
	GetCache() *lru.ARCCache
}

type Batch interface {
	Set(key, value []byte)
	Delete(key []byte)
	Write() error
}

type IteratorSeeker interface {
	Rewind() bool
	Seek(key []byte) bool
	Next() bool
}

type Iterator interface {
	IteratorSeeker
	Valid() bool
	Key() []byte
	Value() []byte
	ValueCopy() []byte
	Error() error
	Close()
}

func bytesPrefix(prefix []byte) []byte {
	var limit []byte
	for i := len(prefix) - 1; i >= 0; i-- {
		c := prefix[i]
		if c < 0xff {
			limit = make([]byte, i+1)
			copy(limit, prefix)
			limit[i] = c + 1
			break
		}
	}
	return limit
}

//-----------------------------------------------------------------------------

const (
	levelDBBackendStr    = "leveldb" // legacy, defaults to goleveldb.
	goLevelDBBackendStr  = "goleveldb"
	memDBBackendStr      = "memdb"
	goBadgerDBBackendStr = "gobadgerdb"
)

type dbCreator func(name string, dir string, cache int) (DB, error)

var backends = map[string]dbCreator{}

func registerDBCreator(backend string, creator dbCreator, force bool) {
	_, ok := backends[backend]
	if !force && ok {
		return
	}
	backends[backend] = creator
}

func NewDB(name string, backend string, dir string, cache int32) DB {
	dbCreator, ok := backends[backend]
	if !ok {
		fmt.Printf("Error initializing DB: %v\n", backend)
		panic("initializing DB error")
	}
	db, err := dbCreator(name, dir, int(cache))
	if err != nil {
		fmt.Printf("Error initializing DB: %v\n", err)
		panic("initializing DB error")
	}
	return db
}

type TransactionDB struct {
	cache *lru.ARCCache
}

func (db *TransactionDB) Begin() {

}

func (db *TransactionDB) Rollback() {

}

func (db *TransactionDB) Commit() {

}

func (db *TransactionDB) GetCache() *lru.ARCCache {
	return db.cache
}

func (db *TransactionDB) SetCacheSize(size int) {
	if db.cache != nil {
		return
	}
	db.cache, _ = lru.NewARC(size)
}
