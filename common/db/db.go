package db

import (
	"bytes"
	"errors"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
	"gitlab.33.cn/chain33/chain33/types"
)

var ErrNotFoundInDb = errors.New("ErrNotFoundInDb")

type Lister interface {
	List(prefix, key []byte, count, direction int32) ([][]byte, error)
	PrefixCount(prefix []byte) int64
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
	IteratorDB
	SetSync([]byte, []byte) error
	Delete([]byte) error
	DeleteSync([]byte) error
	Close()
	NewBatch(sync bool) Batch
	// For debugging
	Print()
	Stats() map[string]string
	SetCacheSize(size int)
	GetCache() *lru.ARCCache
}

type KVDBList struct {
	DB
	list *ListHelper
}

func (l *KVDBList) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	vals := l.list.List(prefix, key, count, direction)
	if vals == nil {
		return nil, types.ErrNotFound
	}
	return vals, nil
}

func (l *KVDBList) PrefixCount(prefix []byte) int64 {
	return l.list.PrefixCount(prefix)
}

func NewKVDB(db DB) KVDB {
	return &KVDBList{DB: db, list: NewListHelper(db)}
}

type Batch interface {
	Set(key, value []byte)
	Delete(key []byte)
	Write() error
	ValueSize() int // amount of data in the batch
	Reset()         // Reset resets the batch for reuse
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
	Prefix() []byte
	Close()
}

type itBase struct {
	start   []byte
	end     []byte
	reverse bool
}

func (it *itBase) checkKey(key []byte) bool {
	//key must in start and end
	var startok = true
	var endok = true
	if it.start != nil {
		startok = bytes.Compare(key, it.start) >= 0
	}
	if it.end != nil {
		endok = bytes.Compare(key, it.end) <= 0
	}
	ok := startok && endok
	return ok
}

func (it *itBase) Prefix() []byte {
	return nil
}

type IteratorDB interface {
	Iterator(start []byte, end []byte, reserver bool) Iterator
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

const (
	levelDBBackendStr     = "leveldb" // legacy, defaults to goleveldb.
	goLevelDBBackendStr   = "goleveldb"
	memDBBackendStr       = "memdb"
	goBadgerDBBackendStr  = "gobadgerdb"
	ssDBBackendStr        = "ssdb"
	goPegasusDbBackendStr = "pegasus"
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
