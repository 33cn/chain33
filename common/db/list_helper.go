// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"fmt"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

// ListHelper ...
type ListHelper struct {
	db IteratorDB
}

var listlog = log.New("module", "db.ListHelper")

// NewListHelper new
func NewListHelper(db IteratorDB) *ListHelper {
	return &ListHelper{db: db}
}

// PrefixScan 前缀
func (db *ListHelper) PrefixScan(prefix []byte) [][]byte {
	it := db.db.Iterator(prefix, nil, false)
	defer it.Close()

	resutls := newCollector(0)
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			return nil
		}
		resutls.collect(it)
		//blog.Debug("PrefixScan", "key", string(item.Key()), "value", string(value))
	}
	return resutls.result()
}

// const
const (
	// direction 位模式指定direction 参数
	// 000， <- 位从低位开始数
	// 0位：direction  ListDESC ListASC
	// 1位：next key， 目前是一个特殊用途， 因为其他的都是返回value， 这个模式同时返回key， value。 不和其他位组合
	// 2位: 为1返回时，返回 key+value，默认只返回value. 返回的是 types.Encode(types.KeyValue)
	// 3位: 为1返回时，返回 key， 默认只返回value. 和 2位不可以同时设置为1
	ListDESC    = int32(0) // 0
	ListASC     = int32(1) // 1
	ListSeek    = int32(2) // 10
	ListWithKey = int32(4) // 01xx
	ListKeyOnly = int32(8) // 10xx
)

// List 列表
func (db *ListHelper) List(prefix, key []byte, count, direction int32) [][]byte {
	if len(key) != 0 && count == 1 && direction == ListSeek {
		return db.nextKeyValue(prefix, key, count, direction)
	}

	if len(key) == 0 {
		if isASC(direction) {
			return db.IteratorScanFromFirst(prefix, count, direction)
		}
		return db.IteratorScanFromLast(prefix, count, direction)
	}
	return db.IteratorScan(prefix, key, count, direction)
}

// IteratorScan 迭代
func (db *ListHelper) IteratorScan(prefix []byte, key []byte, count int32, direction int32) [][]byte {
	reverse := isReverse(direction)
	it := db.db.Iterator(prefix, nil, reverse)
	defer it.Close()
	results := newCollector(direction)

	var i int32
	it.Seek(key)
	if !it.Valid() {
		listlog.Error("PrefixScan it.Value()", "error", it.Error())
		return nil
	}
	// key存在时, 需要指向下一个
	if bytes.Equal(it.Key(), key) {
		it.Next()
	}
	for ; it.Valid(); it.Next() {
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			return nil
		}
		if isdeleted(it.Value()) {
			continue
		}
		results.collect(it)
		i++
		if i == count {
			break
		}
	}
	return results.result()
}

func (db *ListHelper) iteratorScan(prefix []byte, count int32, reverse bool, direction int32) [][]byte {
	it := db.db.Iterator(prefix, nil, reverse)
	defer it.Close()
	results := newCollector(direction)
	var i int32
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			return nil
		}
		if isdeleted(it.Value()) {
			continue
		}
		results.collect(it)
		i++
		if i == count {
			break
		}
	}
	return results.result()
}

// IteratorScanFromFirst 从头迭代
func (db *ListHelper) IteratorScanFromFirst(prefix []byte, count int32, direction int32) (values [][]byte) {
	return db.iteratorScan(prefix, count, false, direction)
}

// IteratorScanFromLast 从尾迭代
func (db *ListHelper) IteratorScanFromLast(prefix []byte, count int32, direction int32) (values [][]byte) {
	return db.iteratorScan(prefix, count, true, direction)
}

func isdeleted(d []byte) bool {
	return len(d) == 0
}

// PrefixCount 前缀数量
func (db *ListHelper) PrefixCount(prefix []byte) (count int64) {
	it := db.db.Iterator(prefix, nil, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			listlog.Error("PrefixCount it.Value()", "error", it.Error())
			count = 0
			return
		}
		if isdeleted(it.Value()) {
			continue
		}
		count++
	}
	return
}

// IteratorCallback 迭代回滚
func (db *ListHelper) IteratorCallback(start []byte, end []byte, count int32, direction int32, fn func(key, value []byte) bool) {
	reserse := isReverse(direction)
	it := db.db.Iterator(start, end, reserse)
	defer it.Close()
	var i int32
	for it.Rewind(); it.Valid(); it.Next() {
		value := it.Value()
		if it.Error() != nil {
			listlog.Error("PrefixScan it.Value()", "error", it.Error())
			return
		}
		if isdeleted(it.Value()) {
			continue
		}
		key := it.Key()
		//判断key 和 end 的关系
		if end != nil {
			cmp := bytes.Compare(key, end)
			if !reserse && cmp > 0 {
				fmt.Println("break1")
				break
			}
			if reserse && cmp < 0 {
				fmt.Println("break2")
				break
			}
		}
		if fn(cloneByte(key), cloneByte(value)) {
			fmt.Println("break3")
			break
		}
		//count 到数目了
		i++
		if i == count {
			fmt.Println("break4")
			break
		}
	}
}

func isASC(direction int32) bool {
	return direction&ListASC == ListASC
}

func isReverse(direction int32) bool {
	return !isASC(direction)
}

// nextKeyValue List 时, count 为 1, deriction 为 ListSeek, key 非空， 取key 的下一个KV
func (db *ListHelper) nextKeyValue(prefix, key []byte, count, direction int32) (values [][]byte) {
	it := db.db.Iterator(prefix, nil, true)
	defer it.Close()
	it.Seek(key)
	//判断是已经删除的key
	for it.Valid() && isdeleted(it.Value()) {
		it.Next()
	}

	if !it.Valid() {
		return nil
	}
	return [][]byte{cloneByte(it.Key()), cloneByte(it.Value())}
}

// collector 辅助收集list 的数据
type collector struct {
	results   [][]byte
	direction int32
}

func newCollector(direction int32) *collector {
	r := make([][]byte, 0)
	return &collector{results: r, direction: direction}
}

func (c *collector) collect(it Iterator) {
	//blog.Debug("collect", "key", string(item.Key()), "value", value)
	if c.direction&ListKeyOnly != 0 {
		c.results = append(c.results, cloneByte(it.Key()))
	} else if c.direction&ListWithKey != 0 {
		v := types.KeyValue{Key: cloneByte(it.Key()), Value: cloneByte(it.Value())}
		c.results = append(c.results, types.Encode(&v))
		// c.results = append(c.results, Key: cloneByte(it.Key()), Value: cloneByte(it.Value()))
	} else {
		c.results = append(c.results, cloneByte(it.Value()))
	}
}

func (c *collector) result() [][]byte {
	if len(c.results) == 0 {
		return nil
	}
	return c.results
}
