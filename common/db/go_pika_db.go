package db

import (
	"bytes"
	"encoding/hex"
	log "github.com/inconshreveable/log15"
	"strings"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/hoisie/redis"
	"github.com/pkg/errors"
)

var (
	plog = log.New("module", "db.gocouchbase")
)

type RedisError string

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoPikaDB(name, dir, cache)
	}

	registerDBCreator(goPikaDbBackendStr, dbCreator, false)
}

type GoPikaDB struct {
	TransactionDB
	client   *redis.Client
	batch    map[string][]byte
	batchDel []string
}

func NewGoPikaDB(name string, dir string, cache int) (*GoPikaDB, error) {
	var client redis.Client
	client.Addr = dir
	batch := make(map[string][]byte)
	return &GoPikaDB{client: &client, batch: batch}, nil
}

func (db *GoPikaDB) Get(key []byte) (val []byte, err error) {
	val, err = db.client.Get(hex.EncodeToString(key))
	if err != nil {
		return nil, ErrNotFoundInDb
	}
	return val, nil
}

func (db *GoPikaDB) BatchGet(keys [][]byte) (value [][]byte, err error) {
	for _, key := range keys {
		val, _ := db.Get(key)
		value = append(value, val)

	}
	if value != nil {
		return value, nil
	} else {
		return nil, errors.New("All keys get failed.")
	}

}

func (db *GoPikaDB) Set(key []byte, value []byte) error {
	db.batch[hex.EncodeToString(key)] = value
	return nil
}

func (db *GoPikaDB) SetSync(key []byte, value []byte) error {
	db.Set(key, value)
	return nil
}

func (db *GoPikaDB) Delete(key []byte) error {
	db.batchDel = append(db.batchDel, hex.EncodeToString(key))
	return nil
}

func (db *GoPikaDB) DeleteSync(key []byte) error {
	err := db.Delete(key)
	if err != nil {
		plog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

func (db *GoPikaDB) Close() {
	db.client = nil
	db.batch = nil
}

func (db *GoPikaDB) Print() {
}

func (db *GoPikaDB) Stats() map[string]string {
	stats := make(map[string]string)
	return stats
}

func (db *GoPikaDB) Iterator(prefix []byte, reserve bool) Iterator {
	keys, err := db.client.Keys(hex.EncodeToString(prefix) + "*")
	if err != nil {
		plog.Error("Get keys err", "error", err)
	}

	var mVal map[string][]byte
	mVal = make(map[string][]byte)
	for _, k := range keys {
		v, _ := db.client.Get(k)
		mVal[k] = v
	}
	return &GoPikaIt{keys: keys, mVal: mVal, prefix: prefix, reserve: reserve}
}

type GoPikaIt struct {
	iterator.Iterator
	mVal    map[string][]byte
	prefix  []byte
	reserve bool
	keys    []string
	index   int
}

func (dbit *GoPikaIt) Close() {
	dbit.mVal = nil
	dbit.prefix = nil
	dbit.reserve = false
	dbit.index = 0
}

func (dbit *GoPikaIt) Next() bool {
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

func (dbit *GoPikaIt) Rewind() bool {
	if dbit.reserve {
		dbit.index = (len(dbit.keys)) - 1
	} else {
		dbit.index = 0
	}
	return dbit.Valid()
}

func (dbit *GoPikaIt) Value() []byte {
	return dbit.mVal[dbit.keys[dbit.index]]
}

func (dbit *GoPikaIt) ValueCopy() []byte {
	v := dbit.mVal[dbit.keys[dbit.index]]
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *GoPikaIt) Valid() bool {
	if dbit.index <= 0 || int(dbit.index) >= len(dbit.keys) {
		return false
	}
	val, _ := hex.DecodeString(dbit.keys[dbit.index])
	return dbit.mVal[dbit.keys[dbit.index]] != nil && bytes.Contains(val, dbit.prefix)
}

func (dbit *GoPikaIt) Error() error {
	return nil
}

func (dbit *GoPikaIt) Seek(key []byte) bool {
	keyStr := string(key)
	pos := 0
	for i, v := range dbit.keys {
		if i < dbit.index {
			continue
		}
		if strings.Compare(keyStr, v) < 0 {
			continue
		} else {
			pos = i
			break
		}
	}

	tmp := dbit.index
	dbit.index = pos
	if dbit.Valid() {
		return true
	} else {
		dbit.index = tmp
		return false
	}
}

type GoPikdBatch struct {
	db *GoPikaDB
}

func (db *GoPikaDB) NewBatch(sync bool) Batch {
	return &GoPikdBatch{db}
}

func (mBatch *GoPikdBatch) Set(key, value []byte) {
	mBatch.db.batch[hex.EncodeToString(key)] = value
}

func (mBatch *GoPikdBatch) Delete(key []byte) {
	mBatch.db.batchDel = append(mBatch.db.batchDel, hex.EncodeToString(key))
}

func (mBatch *GoPikdBatch) Write() error {
	err := mBatch.db.client.Mset(mBatch.db.batch)
	if err != nil {
		return err
	}

	for _, key := range mBatch.db.batchDel {
		mBatch.db.client.Del(key)
	}
	mBatch.db.batch = nil
	mBatch.db.batch = make(map[string][]byte)
	mBatch.db.batchDel = nil
	return nil
}
