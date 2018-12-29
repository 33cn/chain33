// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"sync"

	"sort"
	"strings"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var mlog = log.New("module", "db.memdb")

// memdb 应该无需区分同步与异步操作

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoMemDB(name, dir, cache)
	}
	registerDBCreator(memDBBackendStr, dbCreator, false)
}

//GoMemDB db
type GoMemDB struct {
	TransactionDB
	db   map[string][]byte
	lock sync.RWMutex
}

//NewGoMemDB new
func NewGoMemDB(name string, dir string, cache int) (*GoMemDB, error) {
	// memdb 不需要创建文件，后续考虑增加缓存数目
	return &GoMemDB{
		db: make(map[string][]byte),
	}, nil
}

//CopyBytes 复制字节
func CopyBytes(b []byte) (copiedBytes []byte) {
	/* 兼容leveldb
	if b == nil {
		return nil
	}
	*/
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return copiedBytes
}

//Get get
func (db *GoMemDB) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return CopyBytes(entry), nil
	}
	return nil, ErrNotFoundInDb
}

//Set set
func (db *GoMemDB) Set(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	//debug.PrintStack()
	//println("--", string(key)[0:4], common.ToHex(key))
	db.db[string(key)] = CopyBytes(value)
	if db.db[string(key)] == nil {
		mlog.Error("Set", "error have no mem")
	}
	return nil
}

//SetSync 设置同步
func (db *GoMemDB) SetSync(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()
	//debug.PrintStack()
	//println("--", string(key)[0:4], common.ToHex(key))
	db.db[string(key)] = CopyBytes(value)
	if db.db[string(key)] == nil {
		mlog.Error("Set", "error have no mem")
	}
	return nil
}

//Delete 删除
func (db *GoMemDB) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

//DeleteSync 删除同步
func (db *GoMemDB) DeleteSync(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
	return nil
}

//DB db
func (db *GoMemDB) DB() map[string][]byte {
	return db.db
}

//Close 关闭
func (db *GoMemDB) Close() {

}

//Print 打印
func (db *GoMemDB) Print() {
	for key, value := range db.db {
		mlog.Info("Print", "key", key, "value", string(value))
	}
}

//Stats ...
func (db *GoMemDB) Stats() map[string]string {
	//TODO
	return nil
}

//Iterator 迭代器
func (db *GoMemDB) Iterator(start []byte, end []byte, reverse bool) Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	base := itBase{start, end, reverse}

	var keys []string
	for k := range db.db {
		if base.checkKey([]byte(k)) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var index int
	return &goMemDBIt{base, index, keys, db}
}

//BatchGet 批量获取
func (db *GoMemDB) BatchGet(keys [][]byte) (value [][]byte, err error) {
	mlog.Error("BatchGet", "Need to implement")
	return nil, nil
}

type goMemDBIt struct {
	itBase
	index   int      // 记录当前索引
	keys    []string // 记录所有keys值
	goMemDb *GoMemDB
}

//Seek 查找
func (dbit *goMemDBIt) Seek(key []byte) bool { //指向当前的index值
	for i, k := range dbit.keys {
		if 0 == strings.Compare(k, string(key)) {
			dbit.index = i
			return true
		}
	}
	return false
}

//Close 关闭
func (dbit *goMemDBIt) Close() {
	dbit.goMemDb.Close()
}

//Next next
func (dbit *goMemDBIt) Next() bool {
	if dbit.reverse { // 反向
		dbit.index-- //将当前key值指向前一个
		return dbit.Valid()
	}
	// 正向
	dbit.index++ //将当前key值指向后一个
	return dbit.Valid()

}

//Rewind ...
func (dbit *goMemDBIt) Rewind() bool {
	if dbit.reverse { // 反向
		if (len(dbit.keys) > 0) && dbit.Valid() {
			dbit.index = len(dbit.keys) - 1 // 将当前key值指向最后一个
			return true
		}
		return false
	}
	// 正向
	if dbit.Valid() {
		dbit.index = 0 // 将当前key值指向第一个
		return true
	}
	return false
}

//Key key
func (dbit *goMemDBIt) Key() []byte {
	return []byte(dbit.keys[dbit.index])
}

//Value value
func (dbit *goMemDBIt) Value() []byte {
	value, _ := dbit.goMemDb.Get([]byte(dbit.keys[dbit.index]))
	return value
}

func (dbit *goMemDBIt) ValueCopy() []byte {
	v, _ := dbit.goMemDb.Get([]byte(dbit.keys[dbit.index]))
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *goMemDBIt) Valid() bool {

	if (dbit.goMemDb == nil) && (len(dbit.keys) == 0) {
		return false
	}

	if len(dbit.keys) > dbit.index && dbit.index >= 0 {
		return true
	}
	return false

}

func (dbit *goMemDBIt) Error() error {
	return nil
}

type kv struct{ k, v []byte }
type memBatch struct {
	db     *GoMemDB
	writes []kv
	size   int
	len    int
}

//NewBatch new
func (db *GoMemDB) NewBatch(sync bool) Batch {
	return &memBatch{db: db}
}

func (b *memBatch) Set(key, value []byte) {
	//println("-b-", string(key)[0:4], common.ToHex(key))
	b.writes = append(b.writes, kv{CopyBytes(key), CopyBytes(value)})
	b.size += len(value)
	b.size += len(key)
	b.len += len(value)
}

func (b *memBatch) Delete(key []byte) {
	b.writes = append(b.writes, kv{CopyBytes(key), CopyBytes(nil)})
	b.size += len(key)
	b.len++
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		if kv.v == nil {
			//println("[d]", string(kv.k))
			delete(b.db.db, string(kv.k))
		} else {
			//println("[i]", string(kv.k))
			b.db.db[string(kv.k)] = kv.v
		}
	}
	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

//ValueLen  batch数量
func (b *memBatch) ValueLen() int {
	return b.len
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
	b.len = 0
}
