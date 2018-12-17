// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

//Query 列表查询结构
type Query struct {
	table *Table
	kvdb  db.KVDB
}

//ListIndex 根据索引查询列表
//index 用哪个index
//prefix 必须要符合的前缀, 可以为空
//primaryKey 开始查询的位置(不包含数据本身)
//count 最多取的数量
//direction 方向
func (query *Query) ListIndex(indexName string, prefix []byte, primaryKey []byte, count, direction int32) (rows []*Row, err error) {
	if indexName == "" {
		return query.ListPrimary(prefix, primaryKey, count, direction)
	}
	p := query.table.indexPrefix(indexName)
	var k []byte
	if len(primaryKey) > 0 {
		row, err := query.table.GetData(primaryKey)
		if err != nil {
			return nil, err
		}
		key, err := query.table.index(row, indexName)
		if err != nil {
			return nil, err
		}
		//如果存在prefix
		if prefix != nil {
			p2 := commonPrefix(prefix, key)
			if len(p2) != len(prefix) {
				return nil, types.ErrNotFound
			}
			p = append(p, p2...)
		}
		k = query.table.getIndexKey(indexName, key, row.Primary)
	} else {
		//这个情况下 k == nil
		p = append(p, prefix...)
	}
	values, err := query.kvdb.List(p, k, count, direction)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		row, err := query.table.GetData(value)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

//ListPrimary list primary data
func (query *Query) ListPrimary(prefix []byte, primaryKey []byte, count, direction int32) (rows []*Row, err error) {
	p := query.table.primaryPrefix()
	var k []byte
	if primaryKey != nil {
		if prefix != nil {
			p2 := commonPrefix(prefix, primaryKey)
			if len(p2) != len(prefix) {
				return nil, types.ErrNotFound
			}
			p = append(p, p2...)
		}
		k = append(p, primaryKey...)
	} else {
		p = append(p, prefix...)
	}
	values, err := query.kvdb.List(p, k, count, direction)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		row, err := query.table.getRow(value)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func commonPrefix(key1, key2 []byte) []byte {
	l1 := len(key1)
	l2 := len(key2)
	l := min(l1, l2)
	for i := 0; i < l; i++ {
		if key1[i] != key2[i] {
			return key1[:i]
		}
	}
	return key1[0:l]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
