package db

type ListHelper struct {
	db DB
}

func NewListHelper(db DB) *ListHelper {
	return &ListHelper{db}
}

func (db *ListHelper) PrefixScan(key []byte) (txhashs [][]byte) {
	db.db.Iterator(key, false)



	err := db.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(key); it.ValidForPrefix(key); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("PrefixScan", "key", string(item.Key()), "value", value)
			txhashs = append(txhashs, value)
		}
		return nil
	})
	if err != nil {
		blog.Error("PrefixScan", "error", err)
		txhashs = txhashs[:]
	}
	return
}

func (db *ListHelper) List(prefix, key []byte, count, direction int32) (values [][]byte) {
	if len(key) == 0 {
		if direction == 1 {
			return db.IteratorScanFromFirst(prefix, count, direction)
		} else {
			return db.IteratorScanFromLast(prefix, count, direction)
		}
	}
	return db.IteratorScan(prefix, key, count, direction)
}

func (db *ListHelper) IteratorScan(Prefix []byte, key []byte, count int32, direction int32) (values [][]byte) {
	err := db.db.View(func(txn *badger.Txn) error {
		var i int32 = 0
		opts := badger.DefaultIteratorOptions
		if direction == 0 {
			opts.Reverse = true
		}
		it := txn.NewIterator(opts)
		defer it.Close()
		it.Seek(Prefix)
		for it.Seek(key); it.ValidForPrefix(Prefix); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("IteratorScan", "key", string(item.Key()), "value", value)
			values = append(values, value)
			i++
			if i == count {
				break
			}
		}
		return nil
	})

	if err != nil {
		blog.Error("IteratorScan", "error", err)
		values = values[:]
	}
	return
}

func (db *ListHelper) IteratorScanFromFirst(key []byte, count int32, direction int32) (values [][]byte) {
	err := db.db.View(func(txn *badger.Txn) error {
		var i int32 = 0
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(key); it.ValidForPrefix(key); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("IteratorScanFromFirst", "key", string(item.Key()), "value", value)
			values = append(values, value)
			i++
			if i == count {
				break
			}
		}
		return nil
	})

	if err != nil {
		blog.Error("IteratorScanFromFirst", "error", err)
		values = values[:]
	}
	return
}

func (db *ListHelper) IteratorScanFromLast(key []byte, count int32, direction int32) (values [][]byte) {
	err := db.db.View(func(txn *badger.Txn) error {
		var i int32 = 0
		opts := badger.DefaultIteratorOptions
		opts.Reverse = true
		it := txn.NewIterator(opts)
		defer it.Close()
		prefix := append(key, 0xFF)
		for it.Seek(prefix); it.ValidForPrefix(key); it.Next() {
			item := it.Item()
			value, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			// blog.Debug("IteratorScanFromLast", "key", string(item.Key()), "value", value)
			values = append(values, value)
			i++
			if i == count {
				break
			}
		}
		return nil
	})

	if err != nil {
		blog.Error("IteratorScanFromLast", "error", err)
		values = values[:]
	}
	return
}
--------------------------------------------

func (db *GoLevelDB) PrefixScan(key []byte) (txhashs [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(key), nil)
	for iter.Next() {
		value := iter.Value()
		//fmt.Printf("PrefixScan:%s\n", string(iter.Key()))
		value1 := make([]byte, len(value))
		copy(value1, value)
		txhashs = append(txhashs, value1)
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return txhashs
}

func (db *GoLevelDB) List(prefix, key []byte, count, direction int32) (values [][]byte) {
	if len(key) == 0 {
		if direction == 1 {
			return db.IteratorScanFromFirst(prefix, count, direction)
		} else {
			return db.IteratorScanFromLast(prefix, count, direction)
		}
	}
	return db.IteratorScan(prefix, key, count, direction)
}

func (db *GoLevelDB) IteratorScan(Prefix []byte, key []byte, count int32, direction int32) (values [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(Prefix), nil)
	var i int32 = 0
	for ok := iter.Seek(key); ok; {
		if i == 0 && !bytes.Equal(key, iter.Key()) {
			log.Info("IteratorScan Equal ", "key", string(iter.Key()))
			return nil
		}
		if i != 0 {
			value := iter.Value()
			//log.Info("IteratorScan", "key", string(iter.Key()))
			value1 := make([]byte, len(value))
			copy(value1, value)
			values = append(values, value1)
		}
		if direction == 0 {
			ok = iter.Prev()
		} else {
			ok = iter.Next()
		}
		i++
		if i > count {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return values
}

func (db *GoLevelDB) IteratorScanFromFirst(key []byte, count int32, direction int32) (values [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(key), nil)
	var i int32 = 0
	for ok := iter.First(); ok; {
		value := iter.Value()
		value1 := make([]byte, len(value))
		copy(value1, value)
		values = append(values, value1)
		ok = iter.Next()
		i++
		if i == count {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return values
}

func (db *GoLevelDB) IteratorScanFromLast(key []byte, count int32, direction int32) (values [][]byte) {
	iter := db.db.NewIterator(util.BytesPrefix(key), nil)
	var i int32 = 0
	for ok := iter.Last(); ok; {
		value := iter.Value()
		value1 := make([]byte, len(value))
		copy(value1, value)
		values = append(values, value1)
		ok = iter.Prev()
		i++
		if i == count {
			break
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		return nil
	}
	return values
}
