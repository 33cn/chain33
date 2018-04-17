package db

import (
	log "github.com/inconshreveable/log15"
)

type ListHelper struct {
	db DB
}

var listlog = log.New("module", "db.ListHelper")

func NewListHelper(db DB) *ListHelper {
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

func (db *ListHelper) List(prefix, key []byte, count, direction int32) (values [][]byte) {
	if len(key) == 0 {
		if direction == 1 {
			return db.IteratorScanFromFirst(prefix, count)
		} else {
			return db.IteratorScanFromLast(prefix, count)
		}
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
		// blog.Debug("PrefixScan", "key", string(item.Key()), "value", value)
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
