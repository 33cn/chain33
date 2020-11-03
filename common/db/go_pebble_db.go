package db

import (
	"bytes"
	"path"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/bloom"
)

var pebbleLog = log15.New("module", "db.pebble")

// PebbleDB db
type PebbleDB struct {
	BaseDB
	db *pebble.DB
}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewPebbleDB(name, dir, cache)
	}
	registerDBCreator(pebbleDBBackendStr, dbCreator, false)
}

// NewPebbleDB new
func NewPebbleDB(name string, dir string, cache int) (*PebbleDB, error) {
	dbPath := path.Join(dir, name+".db")
	db, err := pebble.Open(dbPath, &pebble.Options{
		BytesPerSync: 4 << 20,
		Levels:       []pebble.LevelOptions{{FilterPolicy: bloom.FilterPolicy(10)}},
	})
	if err != nil {
		return nil, err
	}
	return &PebbleDB{db: db}, nil
}

//Get get
func (db *PebbleDB) Get(key []byte) ([]byte, error) {
	res, closer, err := db.db.Get(key)
	if err != nil {
		if err == pebble.ErrNotFound {
			return nil, ErrNotFoundInDb
		}
		pebbleLog.Error("Get", "error", err)
		return nil, err
	}
	if err = closer.Close(); err != nil {
		return nil, err
	}
	return res, nil
}

//Set set
func (db *PebbleDB) Set(key []byte, value []byte) error {
	err := db.db.Set(key, value, nil)
	if err != nil {
		pebbleLog.Error("Set", "error", err)
		return err
	}
	return nil
}

//SetSync 同步set
func (db *PebbleDB) SetSync(key []byte, value []byte) error {
	err := db.db.Set(key, value, &pebble.WriteOptions{Sync: true})
	if err != nil {
		pebbleLog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

//Delete 删除
func (db *PebbleDB) Delete(key []byte) error {
	err := db.db.Delete(key, nil)
	if err != nil {
		pebbleLog.Error("Delete", "error", err)
		return err
	}
	return nil
}

//DeleteSync 同步删除
func (db *PebbleDB) DeleteSync(key []byte) error {
	err := db.db.Delete(key, &pebble.WriteOptions{Sync: true})
	if err != nil {
		pebbleLog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

//DB db
func (db *PebbleDB) DB() *pebble.DB {
	return db.db
}

//Close 关闭
func (db *PebbleDB) Close() {
	err := db.db.Close()
	if err != nil {
		pebbleLog.Error("Close", "error", err)
	}
}

//Print 打印
func (db *PebbleDB) Print() {}

//Stats ...
func (db *PebbleDB) Stats() map[string]string { return nil }

//Iterator 迭代器
func (db *PebbleDB) Iterator(start []byte, end []byte, reverse bool) Iterator {
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	it := db.db.NewIter(&pebble.IterOptions{
		LowerBound: start,
		UpperBound: end,
	})

	return &pebbleIt{it, itBase{start, end, reverse}, true}
}

// CompactRange ...
func (db *PebbleDB) CompactRange(start, limit []byte) error {
	return db.db.Compact(start, limit)
}

type pebbleIt struct {
	*pebble.Iterator
	itBase
	first bool
}

// Seek seek
/*
TODO：
	关于seek, seek方法将iterator定位到大于等于key的一个位置，如果存在这样一个key则返true，否则返回false。
	在leveldb中，seek返回false时，可以继续调用Prev()方法向前遍历。
	pebble中seek方法分为SeekGE(key)和SeekLT(key)，且当这两个方法返回false时，无法再进一步调用Next()或者Prev()方法向后或者向前遍历。
	因此在封装 *pebbleIt.Next() 方法时作了特殊处理。
*/
func (it *pebbleIt) Seek(key []byte) bool {
	return it.Iterator.SeekGE(key)
}

//Close 关闭
func (it *pebbleIt) Close() {
	_ = it.Iterator.Close()
}

//Next next
func (it *pebbleIt) Next() bool {
	if it.reverse {
		//TODO：
		//为兼容leveldb和系统接口作的特殊处理
		if it.first {
			it.first = false
			if !it.Valid() {
				return it.Iterator.Last() && it.Valid()
			}
		}
		return it.Iterator.Prev() && it.Valid()
	}
	return it.Iterator.Next() && it.Valid()
}

//Rewind ...
func (it *pebbleIt) Rewind() bool {
	if it.reverse {
		return it.Iterator.Last() && it.Valid()
	}
	return it.Iterator.First() && it.Valid()
}

// Value value
func (it *pebbleIt) Value() []byte {
	return it.Iterator.Value()
}

// ValueCopy copy
func (it *pebbleIt) ValueCopy() []byte {
	v := it.Iterator.Value()
	return cloneByte(v)
}

// Valid valid
func (it *pebbleIt) Valid() bool {
	return it.Iterator.Valid() && it.checkKey(it.Key())
}

type pebbleBatch struct {
	db    *PebbleDB
	batch *pebble.Batch
	wop   *pebble.WriteOptions
	size  int
	len   int
}

//NewBatch new
func (db *PebbleDB) NewBatch(sync bool) Batch {
	batch := &pebbleBatch{
		db:    db,
		batch: db.db.NewBatch(),
		wop:   &pebble.WriteOptions{Sync: sync},
		size:  0,
		len:   0,
	}
	return batch
}

// Set batch set
func (pb *pebbleBatch) Set(key, value []byte) {
	_ = pb.batch.Set(key, value, pb.wop)
	pb.size += len(key)
	pb.size += len(value)
	pb.len += len(value)
}

// Delete batch delete
func (pb *pebbleBatch) Delete(key []byte) {
	_ = pb.batch.Delete(key, pb.wop)
	pb.size += len(key)
	pb.len++
}

// Write batch write
func (pb *pebbleBatch) Write() error {
	err := pb.batch.Commit(pb.wop)
	if err != nil {
		pebbleLog.Error("Write", "error", err)
		return err
	}
	return nil
}

// ValueSize size of batch
func (pb *pebbleBatch) ValueSize() int {
	return pb.size
}

// ValueLen size of batch value
func (pb *pebbleBatch) ValueLen() int {
	return pb.len
}

// Reset resets the batch
func (pb *pebbleBatch) Reset() {
	pb.batch.Reset()
	pb.len = 0
	pb.size = 0
}

// UpdateWriteSync ...
func (pb *pebbleBatch) UpdateWriteSync(sync bool) {
	pb.wop.Sync = sync
}
