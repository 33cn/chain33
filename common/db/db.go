// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package db 数据库操作底层接口定义以及实现包括：leveldb、
// memdb、mvcc、badgerdb、pegasus、ssdb
package db

import (
	"bytes"
	"fmt"

	"github.com/33cn/chain33/types"
	lru "github.com/hashicorp/golang-lru"
)

//ErrNotFoundInDb error
var ErrNotFoundInDb = types.ErrNotFound

//Lister 列表接口
type Lister interface {
	List(prefix, key []byte, count, direction int32) ([][]byte, error)
	PrefixCount(prefix []byte) int64
}

//TxKV transaction Key Value
type TxKV interface {
	KV
	IteratorDB
}

//KV kv
type KV interface {
	Get(key []byte) ([]byte, error)
	Set(key []byte, value []byte) (err error)
	Begin()
	Commit() error
	Rollback()
}

//KVDB kvdb
type KVDB interface {
	KV
	Lister
}

//DB db
type DB interface {
	KV
	IteratorDB
	SetSync([]byte, []byte) error
	Delete([]byte) error
	DeleteSync([]byte) error
	Close()
	NewBatch(sync bool) Batch
	BeginTx() (TxKV, error)
	// For debugging
	Print()
	Stats() map[string]string
	SetCacheSize(size int)
	GetCache() *lru.ARCCache
}

//KVDBList list
type KVDBList struct {
	DB
	list *ListHelper
}

//List 列表
func (l *KVDBList) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	vals := l.list.List(prefix, key, count, direction)
	if vals == nil {
		return nil, types.ErrNotFound
	}
	return vals, nil
}

//PrefixCount 前缀长度
func (l *KVDBList) PrefixCount(prefix []byte) int64 {
	return l.list.PrefixCount(prefix)
}

//NewKVDB new
func NewKVDB(db DB) KVDB {
	return &KVDBList{DB: db, list: NewListHelper(db)}
}

//Batch batch
type Batch interface {
	Set(key, value []byte)
	Delete(key []byte)
	Write() error
	ValueSize() int // size of data in the batch
	ValueLen() int  // amount of data in the batch
	Reset()         // Reset resets the batch for reuse
}

//IteratorSeeker ...
type IteratorSeeker interface {
	Rewind() bool
	Seek(key []byte) bool
	Next() bool
}

//Iterator 迭代器
type Iterator interface {
	IteratorSeeker
	Valid() bool
	Key() []byte
	Value() []byte
	ValueCopy() []byte
	Error() error
	Prefix() []byte
	IsReverse() bool
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

//Prefix 前缀
func (it *itBase) Prefix() []byte {
	return nil
}

func (it *itBase) IsReverse() bool {
	return it.reverse
}

//IteratorDB 迭代
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

//NewDB new
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

//BaseDB 交易缓存
type BaseDB struct {
	cache *lru.ARCCache
}

//GetCache 获取缓存
func (db *BaseDB) GetCache() *lru.ARCCache {
	return db.cache
}

//SetCacheSize 设置缓存大小
func (db *BaseDB) SetCacheSize(size int) {
	if db.cache != nil {
		return
	}
	var err error
	db.cache, err = lru.NewARC(size)
	if err != nil {
		panic(err)
	}
}

//Begin call panic when Begin not rewrite
func (db *BaseDB) Begin() {
	panic("Begin not impl")
}

//Commit call panic when Commit not rewrite
func (db *BaseDB) Commit() error {
	panic("Commit not impl")
}

//Rollback call panic when Rollback not rewrite
func (db *BaseDB) Rollback() {
	panic("Rollback not impl")
}

//BeginTx call panic when BeginTx not rewrite
func (db *BaseDB) BeginTx() (TxKV, error) {
	panic("BeginTx not impl")
}
