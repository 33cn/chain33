package table

import (
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

//Query 列表查询结构
type Query struct {
	table *Table
	kvdb  db.ReadOnlyListDB
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
	var p, k []byte
	if len(primaryKey) > 0 {
		row, err := query.table.GetData(primaryKey)
		if err != nil {
			return nil, err
		}
		p = query.table.indexPrefix(indexName)
		k, err = query.table.index(row, indexName)
		if err != nil {
			return nil, err
		}
	} else {
		p = query.table.indexPrefix(indexName)
	}
	p2 := commonPrefix(prefix, k)
	if len(p2) != len(prefix) {
		return nil, types.ErrNotFound
	}
	k = k[len(p2):]
	p = append(p, p2...)
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
	p2 := commonPrefix(prefix, primaryKey)
	if len(p2) != len(prefix) {
		return nil, types.ErrNotFound
	}
	primaryKey = primaryKey[len(p2):]
	p = append(p, p2...)
	values, err := query.kvdb.List(p, primaryKey, count, direction)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		primary, data, err := DecodeRow(value)
		if err != nil {
			return nil, err
		}
		row := query.table.meta.CreateRow()
		row.Primary = primary
		err = types.Decode(data, row.Data)
		if err != nil {
			return nil, err
		}
		rows = append(rows, &row)
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
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
