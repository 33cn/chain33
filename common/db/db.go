package db

import (
	"fmt"
)

type KVDB interface {
	Get(key []byte) (value []byte, err error)
	Set(key []byte, value []byte) (err error)
}

type DB interface {
	Get([]byte) []byte
	Set([]byte, []byte)
	SetSync([]byte, []byte)
	Delete([]byte)
	DeleteSync([]byte)
	Close()
	NewBatch(sync bool) Batch
	Iterator() Iterator
	// For debugging
	Print()
	Stats() map[string]string
}

type Batch interface {
	Set(key, value []byte)
	Delete(key []byte)
	Write()
}

type IteratorSeeker interface {
	// First moves the iterator to the first key/value pair. If the iterator
	// only contains one key/value pair then First and Last would moves
	// to the same key/value pair.
	// It returns whether such pair exist.
	First() bool

	// Last moves the iterator to the last key/value pair. If the iterator
	// only contains one key/value pair then First and Last would moves
	// to the same key/value pair.
	// It returns whether such pair exist.
	Last() bool

	// Seek moves the iterator to the first key/value pair whose key is greater
	// than or equal to the given key.
	// It returns whether such pair exist.
	//
	// It is safe to modify the contents of the argument after Seek returns.
	Seek(key []byte) bool

	// Next moves the iterator to the next key/value pair.
	// It returns whether the iterator is exhausted.
	Next() bool

	// Prev moves the iterator to the previous key/value pair.
	// It returns whether the iterator is exhausted.
	Prev() bool
}

type Iterator interface {
	IteratorSeeker
	Key() []byte
	Value() []byte
	Close()
}

//-----------------------------------------------------------------------------

const (
	LevelDBBackendStr    = "leveldb" // legacy, defaults to goleveldb.
	CLevelDBBackendStr   = "cleveldb"
	GoLevelDBBackendStr  = "goleveldb"
	MemDBBackendStr      = "memdb"
	GoBadgerDBBackendStr = "gobadgerdb"
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
