package db

import (
	"bytes"

	log "github.com/inconshreveable/log15"
)

type ListHelper struct {
	db IteratorDB
}

var listlog = log.New("module", "db.ListHelper")

func NewListHelper(db IteratorDB) *ListHelper {
	return &ListHelper{db}
}

func (db *ListHelper) PrefixScan(prefix []byte) (values [][]byte) {
	it := db.db.Iterator(prefix, false)
	defer it.Close()

	for it.Rewind(); it.Valid(); it.Next() {
		value := it.ValueCopy()
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			values = nil
			return
		}
		//blog.Debug("PrefixScan", "key", string(item.Key()), "value", string(value))
		values = append(values, value)
	}
	return
}

const (
	ListDESC = int32(0)
	ListASC  = int32(1)
	ListSeek = int32(2)
)

func (db *ListHelper) List(prefix, key []byte, count, direction int32) (values [][]byte) {
	if len(key) == 0 {
		if direction == ListASC {
			return db.IteratorScanFromFirst(prefix, count)
		} else {
			return db.IteratorScanFromLast(prefix, count)
		}
	}
	if count == 1 && direction == ListSeek {
		it := db.db.Iterator(prefix, true)
		defer it.Close()
		it.Seek(key)
		//判断是否相等
		if !bytes.Equal(key, it.Key()) {
			it.Next()
			if !it.Valid() {
				return nil
			}
		}
		return [][]byte{cloneByte(it.Key()), cloneByte(it.Value())}
	}
	return db.IteratorScan(prefix, key, count, direction)
}

func (db *ListHelper) IteratorScan(prefix []byte, key []byte, count int32, direction int32) (values [][]byte) {
	var reserse = false
	if direction == 0 {
		reserse = true
	}
	it := db.db.Iterator(prefix, reserse)
	defer it.Close()

	var i int32
	it.Seek(key)
	if !it.Valid() {
		listlog.Error("PrefixScan it.Value()", "error", it.Error())
		values = nil
		return
	}
	for it.Next(); it.Valid(); it.Next() {
		value := it.ValueCopy()
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			values = nil
			return
		}
		// blog.Debug("PrefixScan", "key", string(item.Key()), "value", value)
		values = append(values, value)
		i++
		if i == count {
			break
		}
	}
	return
}

func (db *ListHelper) IteratorScanFromFirst(prefix []byte, count int32) (values [][]byte) {
	it := db.db.Iterator(prefix, false)
	defer it.Close()

	var i int32
	for it.Rewind(); it.Valid(); it.Next() {
		value := it.ValueCopy()
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			values = nil
			return
		}
		//listlog.Debug("PrefixScan", "key", string(it.Key()), "value", value)
		values = append(values, value)
		i++
		if i == count {
			break
		}
	}
	return
}

func (db *ListHelper) IteratorScanFromLast(prefix []byte, count int32) (values [][]byte) {
	it := db.db.Iterator(prefix, true)
	defer it.Close()

	var i int32
	for it.Rewind(); it.Valid(); it.Next() {
		value := it.ValueCopy()
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			values = nil
			return
		}
		// blog.Debug("PrefixScan", "key", string(item.Key()), "value", value)
		values = append(values, value)
		i++
		if i == count {
			break
		}
	}
	return
}

func (db *ListHelper) PrefixCount(prefix []byte) (count int64) {
	it := db.db.Iterator(prefix, true)
	defer it.Close()

	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			listlog.Error("PrefixCount it.Value()", "error", it.Error())
			count = 0
			return
		}

		count++
	}
	return
}

func (db *ListHelper) IteratorCallback(start []byte, end []byte, count int32, direction int32, fn func(key, value []byte) bool) {
	reserse := direction == 0
	it := db.db.Iterator(start, reserse)
	defer it.Close()
	var i int32
	for it.Rewind(); it.Valid(); it.Next() {
		value := it.Value()
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			return
		}
		key := it.Key()
		//判断key 和 end 的关系
		if end != nil {
			cmp := bytes.Compare(key, end)
			if !reserse && cmp > 0 {
				break
			}
			if reserse && cmp < 0 {
				break
			}
		}
		if fn(cloneByte(key), cloneByte(value)) {
			break
		}
		//count 到数目了
		i++
		if i == count {
			break
		}
	}
}
