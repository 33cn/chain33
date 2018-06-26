package db

import (
	"bytes"
	"encoding/hex"

	log "github.com/inconshreveable/log15"

	// "github.com/syndtr/goleveldb/leveldb/opt"
	"fmt"

	"github.com/couchbase/gocb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strings"
)

var (
	clog           = log.New("module", "db.gocouchbase")
	BATCH_MAP_SIZE = 10240
)

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoCouchBase(name, dir, cache)
	}

	registerDBCreator(goCouchBaseBackendStr, dbCreator, false)
}

type GoCouchBase struct {
	TransactionDB
	// db      *leveldb.DB
	bucket *gocb.Bucket
	items  []gocb.BulkOp
	// batch  map[string][]byte
}

func NewGoCouchBase(name string, dir string, cache int) (*GoCouchBase, error) {
	cluster, err := gocb.Connect("couchbase://localhost")
	if err != nil {
		clog.Error("Connect err", "error", err)
		return nil, err
	}

	cluster.Authenticate(gocb.PasswordAuthenticator{
		Username: "Administrator",
		Password: "test123",
	})

	bucket, err := cluster.OpenBucket("default", "")
	if err != nil {
		fmt.Println("Get Pool err", "error", err)
	}

	return &GoCouchBase{bucket: bucket}, nil
}

func (db *GoCouchBase) Get(key []byte) (val []byte, err error) {
	_, err = db.bucket.Get(hex.EncodeToString(key), &val)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
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
	err := db.Set(key, value)
	if err != nil {
		clog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

func (db *GoCouchBase) Delete(key []byte) error {
	for _, op := range db.items {
		k := op.(*gocb.UpsertOp).Key
		if k == hex.EncodeToString(key) {
			op = nil
		}
	}
	return nil
}

func (db *GoCouchBase) DeleteSync(key []byte) error {
	err := db.Delete(key)
	if err != nil {
		clog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

func (db *GoCouchBase) Close() {
	db.bucket.Close()
}

func (db *GoCouchBase) Print() {

}

func (db *GoCouchBase) Stats() map[string]string {
	stats := make(map[string]string)
	return stats
}

func (db *GoCouchBase) Iterator(prefix []byte, reserve bool) Iterator {
	r := &util.Range{prefix, bytesPrefix(prefix)}

	view := gocb.NewViewQuery("dev_view", "all_keys")
	view.IdRange(hex.EncodeToString(r.Start), hex.EncodeToString(r.Limit))
	res, err := db.bucket.ExecuteViewQuery(view)
	if err != nil {
		clog.Error("View err", "error", err)
		return &GoCouchBaseIt{}
	}
	var keys []string
	var val []byte
	for res.Next(&val) {
		keys = append(keys, hex.EncodeToString(val))
	}

	var v []byte
	var mVal map[string][]byte
	for _, k := range keys {

		db.bucket.Get(k, &v)
		mVal[k] = v
	}
	return &GoCouchBaseIt{keys: keys, mVal: mVal, prefix: prefix, reserve: reserve}
}

type GoCouchBaseIt struct {
	iterator.Iterator
	mVal    map[string][]byte
	prefix  []byte
	reserve bool
	keys    []string
	index   int64
}

func (dbit *GoCouchBaseIt) Close() {
	dbit.mVal = nil
	dbit.prefix = nil
	dbit.reserve = false
	dbit.index = 0
}

func (dbit *GoCouchBaseIt) Next() bool {
	if dbit.reserve {
		dbit.index--
	} else {
		dbit.index++
	}
	if dbit.mVal[dbit.keys[dbit.index]] != nil {
		return true
	} else {
		return false
	}
}

func (dbit *GoCouchBaseIt) Rewind() bool {
	if dbit.reserve {
		dbit.index = int64(len(dbit.keys))
	} else {
		dbit.index = 0
	}

	return dbit.Valid()
}

func (dbit *GoCouchBaseIt) Value() []byte {
	return dbit.mVal[dbit.keys[dbit.index]]
}

func (dbit *GoCouchBaseIt) ValueCopy() []byte {
	v := dbit.mVal[dbit.keys[dbit.index]]
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *GoCouchBaseIt) Valid() bool {
	val, _ := hex.DecodeString(dbit.keys[dbit.index])
	return dbit.mVal[dbit.keys[dbit.index]] != nil && bytes.Contains(val, dbit.prefix)
}

type GoCouchBaseBatch struct {
	bucket   *gocb.Bucket
	batchMap []gocb.BulkOp
}

func (db *GoCouchBase) NewBatch(sync bool) Batch {
	bucket := db.bucket
	batch := db.items
	return &GoCouchBaseBatch{bucket, batch}
}

func (mBatch *GoCouchBaseBatch) Set(key, value []byte) {
	mBatch.batchMap = append(mBatch.batchMap, &gocb.UpsertOp{Key: hex.EncodeToString(key), Value: value})
}

func (mBatch *GoCouchBaseBatch) Delete(key []byte) {
	for _, op := range mBatch.batchMap {
		k := op.(*gocb.UpsertOp).Key
		if k == hex.EncodeToString(key) {
			op = nil
		}
	}
}

func (mBatch *GoCouchBaseBatch) Write() error {
	err := mBatch.bucket.Do(mBatch.batchMap)
	if err != nil {
		clog.Error("Write err", "error", err)
	}
	return nil
}
