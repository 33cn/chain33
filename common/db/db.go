package db

import (
	"errors"
	"fmt"
)

var ErrNotFoundInDb = errors.New("ErrNotFoundInDb")

type KVDB interface {
	Get(key []byte) (value []byte, err error)
	Set(key []byte, value []byte) (err error)
	List(prefix, key []byte, count, direction int32) (values [][]byte, err error)
}

type DB interface {
	Get([]byte) ([]byte, error)
	Set([]byte, []byte) error
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
	cLevelDBBackendStr   = "cleveldb"
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

func NewDB(name string, backend string, dir string, cache int) DB {
	dbCreator, ok := backends[backend]
	if !ok {
		fmt.Printf("Error initializing DB: %v\n", backend)
		panic("initializing DB error")
	}
	db, err := dbCreator(name, dir, cache)
	if err != nil {
		fmt.Printf("Error initializing DB: %v\n", err)
		panic("initializing DB error")
	}
	return db
}
