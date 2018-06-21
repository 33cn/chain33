package db

import (
	"bytes"
	"encoding/hex"

	log "github.com/inconshreveable/log15"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gopkg.in/couchbase/gocb.v1"
)

var clog = log.New("module", "db.gocouchbase")

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoCouchBase(name, dir, cache)
	}

	registerDBCreator(goCouchBaseBackendStr, dbCreator, false)
}

type GoCouchBase struct {
	TransactionDB
	db      *leveldb.DB
	cluster *gocb.Cluster
	bucket  *gocb.Bucket
	items   []gocb.BulkOp
}

func NewGoCouchBase(name string, dir string, cache int) (*GoCouchBase, error) {
	cl, err := gocb.Connect("couchbase://localhost")
	if err != nil {
		clog.Error("Connect err", "error", err)
	}
	err = cl.Authenticate(gocb.PasswordAuthenticator{
		Username: "Administrator",
		Password: "test123",
	})

	if err != nil {
		clog.Error("Auth err", "error", err)
	}
	bucket, err := cl.OpenBucket("default", "")
	if err != nil {
		clog.Error("Open bucket err", "error", err)
	}

	database := &GoCouchBase{cluster: cl, bucket: bucket, items: nil}
	return database, nil
}

func (db *GoCouchBase) Get(key []byte) (val []byte, err error) {
	_, err = db.bucket.Get(hex.EncodeToString(key), &val)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFoundInDb
		} else {
			clog.Error("Get", "error", err)
			return nil, err
		}
	}
	return val, nil
}

func (db *GoCouchBase) Set(key []byte, value []byte) error {
	db.items = append(db.items, &gocb.UpsertOp{Key: hex.EncodeToString(key), Value: value})
	return nil
}

func (db *GoCouchBase) SetSync(key []byte, value []byte) error {
	err := db.db.Put(key, value, &opt.WriteOptions{Sync: true})
	if err != nil {
		clog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

func (db *GoCouchBase) Delete(key []byte) error {
	err := db.db.Delete(key, nil)
	if err != nil {
		clog.Error("Delete", "error", err)
		return err
	}
	return nil
}

func (db *GoCouchBase) DeleteSync(key []byte) error {
	err := db.db.Delete(key, &opt.WriteOptions{Sync: true})
	if err != nil {
		clog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

func (db *GoCouchBase) DB() *leveldb.DB {
	return db.db
}

func (db *GoCouchBase) Close() {
	db.db.Close()
}

func (db *GoCouchBase) Print() {
	str, _ := db.db.GetProperty("leveldb.stats")
	clog.Info("Print", "stats", str)

	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		//fmt.Printf("[%X]:\t[%X]\n", key, value)
		clog.Info("Print", "key", string(key), "value", string(value))
	}
}

func (db *GoCouchBase) Stats() map[string]string {
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

func (db *GoCouchBase) Iterator(prefix []byte, reserve bool) Iterator {
	r := &util.Range{prefix, bytesPrefix(prefix)}
	it := db.db.NewIterator(r, nil)
	return &GoCouchBaseIt{it, reserve, prefix}
}

type GoCouchBaseIt struct {
	iterator.Iterator
	reserve bool
	prefix  []byte
}

func (dbit *GoCouchBaseIt) Close() {
	dbit.Iterator.Release()
}

func (dbit *GoCouchBaseIt) Next() bool {
	if dbit.reserve {
		return dbit.Iterator.Prev() && dbit.Valid()
	}
	return dbit.Iterator.Next() && dbit.Valid()
}

func (dbit *GoCouchBaseIt) Rewind() bool {
	if dbit.reserve {
		return dbit.Iterator.Last() && dbit.Valid()
	}
	return dbit.Iterator.First() && dbit.Valid()
}

func (dbit *GoCouchBaseIt) Value() []byte {
	return dbit.Iterator.Value()
}

func (dbit *GoCouchBaseIt) ValueCopy() []byte {
	v := dbit.Iterator.Value()
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *GoCouchBaseIt) Valid() bool {
	return dbit.Iterator.Valid() && bytes.HasPrefix(dbit.Key(), dbit.prefix)
}

type GoCouchBaseBatch struct {
	bucket *gocb.Bucket
	items  []gocb.BulkOp
}

func (db *GoCouchBase) NewBatch(sync bool) Batch {
	bucket := db.bucket
	items := db.items
	return &GoCouchBaseBatch{bucket, items}
}

func (mBatch *GoCouchBaseBatch) Set(key, value []byte) {
	mBatch.items = append(mBatch.items, &gocb.UpsertOp{Key: hex.EncodeToString(key), Value: value})
}

func (mBatch *GoCouchBaseBatch) Delete(key []byte) {
	for i := 0; i < len(mBatch.items); i++ {
		k := mBatch.items[i].(*gocb.UpsertOp).Key
		if k == hex.EncodeToString(key) {
			mBatch.items[i].(*gocb.UpsertOp).Key = ""
			mBatch.items[i].(*gocb.UpsertOp).Value = nil
		}
	}
}

func (mBatch *GoCouchBaseBatch) Write() error {
	err := mBatch.bucket.Do(mBatch.items)
	return err
}
