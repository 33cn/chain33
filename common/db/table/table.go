// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//Package table 实现一个基于kv的关系型数据库的表格功能
package table

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/golang/protobuf/proto"
)

//设计结构:
/*
核心: 平衡
save:
数据保存:
tableprefix + tablename + Primary -> data

index:
tableprefix + tablemetaname + index + primary -> primary

read:
list by Primary -> 直接读出数据
list by index

根据index 先计算要读出的 primary list
从数据table读出数据（根据 primary key）

del:
利用 primaryKey + index 删除所有的 数据 和 索引
*/

//表关联设计
//指出是 添加 还是 删除 行
//primary key auto 的del 需要指定 primary key
const (
	None = iota
	Add
	Update
	Del
)

//meta key
const meta = sep + "m" + sep
const data = sep + "d" + sep

//RowMeta 定义行的操作
type RowMeta interface {
	CreateRow() *Row
	SetPayload(types.Message) error
	Get(key string) ([]byte, error)
}

//Row 行操作
type Row struct {
	Ty      int
	Primary []byte
	Data    types.Message
	old     types.Message
}

func encodeInt64(p int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, p)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeInt64(p []byte) (int64, error) {
	buf := bytes.NewBuffer(p)
	var i int64
	err := binary.Read(buf, binary.LittleEndian, &i)
	if err != nil {
		return 0, err
	}
	return i, nil
}

//Encode row
func (row *Row) Encode() ([]byte, error) {
	b, err := encodeInt64(int64(len(row.Primary)))
	if err != nil {
		return nil, err
	}
	b = append(b, row.Primary...)
	b = append(b, types.Encode(row.Data)...)
	return b, nil
}

//DecodeRow from data
func DecodeRow(data []byte) ([]byte, []byte, error) {
	if len(data) <= 8 {
		return nil, nil, types.ErrDecode
	}
	l, err := decodeInt64(data[:8])
	if err != nil {
		return nil, nil, err
	}
	if len(data) < int(l)+8 {
		return nil, nil, types.ErrDecode
	}
	return data[8 : int(l)+8], data[int(l)+8:], nil
}

//Table 定一个表格, 并且添加 primary key, index key
type Table struct {
	meta       RowMeta
	rows       []*Row
	rowmap     map[string]*Row
	kvdb       db.KV
	opt        *Option
	autoinc    *Count
	dataprefix string
	metaprefix string
}

//Option table 的选项
type Option struct {
	Prefix  string
	Name    string
	Primary string
	Join    bool
	Index   []string
}

const sep = "-"
const joinsep = "#"

//NewTable  新建一个表格
//primary 可以为: auto, 由系统自动创建
//index 可以为nil
func NewTable(rowmeta RowMeta, kvdb db.KV, opt *Option) (*Table, error) {
	if len(opt.Index) > 16 {
		return nil, ErrTooManyIndex
	}
	for _, index := range opt.Index {
		if strings.Contains(index, sep) || index == "primary" {
			return nil, ErrIndexKey
		}
		if !opt.Join && strings.Contains(index, joinsep) {
			return nil, ErrIndexKey
		}
	}
	if opt.Primary == "" {
		opt.Primary = "auto"
	}
	if _, err := getPrimaryKey(rowmeta, opt.Primary); err != nil {
		return nil, err
	}
	//不允许有 "-"
	if strings.Contains(opt.Name, sep) {
		return nil, ErrTablePrefixOrTableName
	}
	//非jointable 不允许 "#"
	if !opt.Join && strings.Contains(opt.Name, joinsep) {
		return nil, ErrTablePrefixOrTableName
	}
	dataprefix := opt.Prefix + sep + opt.Name + data
	metaprefix := opt.Prefix + sep + opt.Name + meta
	count := NewCount(opt.Prefix, opt.Name+sep+"autoinc"+sep, kvdb)
	return &Table{
		meta:       rowmeta,
		kvdb:       kvdb,
		rowmap:     make(map[string]*Row),
		opt:        opt,
		autoinc:    count,
		dataprefix: dataprefix,
		metaprefix: metaprefix}, nil
}

func getPrimaryKey(meta RowMeta, primary string) ([]byte, error) {
	if primary == "" {
		return nil, ErrEmptyPrimaryKey
	}
	if strings.Contains(primary, sep) {
		return nil, ErrPrimaryKey
	}
	if primary != "auto" {
		key, err := meta.Get(primary)
		return key, err
	}
	return nil, nil
}

func (table *Table) addRowCache(row *Row) {
	primary := string(row.Primary)
	if row.Ty == Del {
		delete(table.rowmap, primary)
	} else if row.Ty == Add || row.Ty == Update {
		table.rowmap[primary] = row
	}
	table.rows = append(table.rows, row)
}

func (table *Table) delRowCache(row *Row) {
	row.Ty = None
	primary := string(row.Primary)
	delete(table.rowmap, primary)
}

func (table *Table) mergeCache(rows []*Row, indexName string, indexValue []byte) ([]*Row, error) {
	replaced := make(map[string]bool)
	for i, row := range rows {
		if cacherow, ok := table.rowmap[string(row.Primary)]; ok {
			rows[i] = cacherow
			replaced[string(row.Primary)] = true
		}
	}
	//add not in db but in cache rows
	for _, row := range table.rowmap {
		if _, ok := replaced[string(row.Primary)]; ok {
			continue
		}
		v, err := table.index(row, indexName)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(v, indexValue) {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

func (table *Table) findRow(primary []byte) (*Row, bool, error) {
	if row, ok := table.rowmap[string(primary)]; ok {
		return row, true, nil
	}
	row, err := table.GetData(primary)
	return row, false, err
}

func (table *Table) hasIndex(name string) bool {
	for _, index := range table.opt.Index {
		if index == name {
			return true
		}
	}
	return false
}

func (table *Table) canGet(name string) bool {
	row := table.meta.CreateRow()
	err := table.meta.SetPayload(row.Data)
	if err != nil {
		return false
	}
	_, err = table.meta.Get(name)
	return err == nil
}

func (table *Table) checkIndex(data types.Message) error {
	err := table.meta.SetPayload(data)
	if err != nil {
		return err
	}
	if _, err := getPrimaryKey(table.meta, table.opt.Primary); err != nil {
		return err
	}
	for i := 0; i < len(table.opt.Index); i++ {
		_, err := table.meta.Get(table.opt.Index[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (table *Table) getPrimaryAuto() ([]byte, error) {
	i, err := table.autoinc.Inc()
	if err != nil {
		return nil, err
	}
	return []byte(pad(i)), nil
}

//primaryKey 获取主键
//1. auto 的情况下,只能自增。
//2. 没有auto的情况下从数据中取
func (table *Table) primaryKey(data types.Message) (primaryKey []byte, err error) {
	if table.opt.Primary == "auto" {
		primaryKey, err = table.getPrimaryAuto()
		if err != nil {
			return nil, err
		}
	} else {
		primaryKey, err = table.getPrimaryFromData(data)
	}
	return
}

func (table *Table) getPrimaryFromData(data types.Message) (primaryKey []byte, err error) {
	err = table.meta.SetPayload(data)
	if err != nil {
		return nil, err
	}
	primaryKey, err = getPrimaryKey(table.meta, table.opt.Primary)
	if err != nil {
		return nil, err
	}
	return
}

//ListIndex  list table index
func (table *Table) ListIndex(indexName string, prefix []byte, primaryKey []byte, count, direction int32) (rows []*Row, err error) {
	kvdb, ok := table.kvdb.(db.KVDB)
	if !ok {
		return nil, errors.New("list only support KVDB interface")
	}
	query := &Query{table: table, kvdb: kvdb}
	return query.ListIndex(indexName, prefix, primaryKey, count, direction)
}

//Replace 如果有重复的，那么替换
func (table *Table) Replace(data types.Message) error {
	if err := table.checkIndex(data); err != nil {
		return err
	}
	primaryKey, err := table.primaryKey(data)
	if err != nil {
		return err
	}
	//如果是auto的情况，一定是添加
	if table.opt.Primary == "auto" {
		table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Add})
		return nil
	}
	//如果没有找到行, 那么添加
	//TODO: 优化保存策略，不需要修改没有变化的index
	row, incache, err := table.findRow(primaryKey)
	if err == types.ErrNotFound {
		table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Add})
		return nil
	}
	//update or add
	if incache {
		row.Data = data
		return nil
	}
	//更新数据
	table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Update, old: row.Data})
	return nil
}

//Add 在表格中添加一行
func (table *Table) Add(data types.Message) error {
	if err := table.checkIndex(data); err != nil {
		return err
	}
	primaryKey, err := table.primaryKey(data)
	if err != nil {
		return err
	}
	//find in cache + db
	_, _, err = table.findRow(primaryKey)
	if err != types.ErrNotFound {
		return ErrDupPrimaryKey
	}
	//检查cache中是否有重复，有重复也返回错误
	table.addRowCache(&Row{Data: data, Primary: primaryKey, Ty: Add})
	return nil
}

//Update 更新数据库
func (table *Table) Update(primaryKey []byte, newdata types.Message) (err error) {
	if err := table.checkIndex(newdata); err != nil {
		return err
	}
	p1, err := table.getPrimaryFromData(newdata)
	if err != nil {
		return err
	}
	if !bytes.Equal(p1, primaryKey) {
		return types.ErrInvalidParam
	}
	row, incache, err := table.findRow(primaryKey)
	//查询发生错误
	if err != nil {
		return err
	}
	//update and add
	if incache {
		row.Data = newdata
		return nil
	}
	table.addRowCache(&Row{Data: newdata, Primary: primaryKey, Ty: Update, old: row.Data})
	return nil
}

//Del 在表格中删除一行(包括删除索引)
func (table *Table) Del(primaryKey []byte) error {
	row, incache, err := table.findRow(primaryKey)
	if err != nil {
		return err
	}
	if incache {
		rowty := row.Ty
		table.delRowCache(row)
		if rowty == Add {
			return nil
		}
	}
	//copy row
	delrow := *row
	delrow.Ty = Del
	table.addRowCache(&delrow)
	return nil
}

//DelRow 删除一行
func (table *Table) DelRow(data types.Message) error {
	primaryKey, err := table.primaryKey(data)
	if err != nil {
		return err
	}
	return table.Del(primaryKey)
}

//getDataKey data key 构造
func (table *Table) getDataKey(primaryKey []byte) []byte {
	return append([]byte(table.dataprefix), primaryKey...)
}

//GetIndexKey data key 构造
func (table *Table) getIndexKey(indexName string, index, primaryKey []byte) []byte {
	key := table.indexPrefix(indexName)
	key = append(key, index...)
	key = append(key, []byte(sep)...)
	key = append(key, primaryKey...)
	return key
}

func (table *Table) primaryPrefix() []byte {
	return []byte(table.dataprefix)
}

func (table *Table) indexPrefix(indexName string) []byte {
	key := append([]byte(table.metaprefix), []byte(indexName+sep)...)
	return key
}

func (table *Table) index(row *Row, indexName string) ([]byte, error) {
	err := table.meta.SetPayload(row.Data)
	if err != nil {
		return nil, err
	}
	return table.meta.Get(indexName)
}

func (table *Table) getData(primaryKey []byte) ([]byte, error) {
	key := table.getDataKey(primaryKey)
	value, err := table.kvdb.Get(key)
	if err != nil {
		return nil, err
	}
	return value, nil
}

//GetData 根据主键获取数据
func (table *Table) GetData(primaryKey []byte) (*Row, error) {
	value, err := table.getData(primaryKey)
	if err != nil {
		return nil, err
	}
	return table.getRow(value)
}

func (table *Table) getRow(value []byte) (*Row, error) {
	primary, data, err := DecodeRow(value)
	if err != nil {
		return nil, err
	}
	row := table.meta.CreateRow()
	row.Primary = primary
	err = types.Decode(data, row.Data)
	if err != nil {
		return nil, err
	}
	return row, nil
}

//Save 保存表格
func (table *Table) Save() (kvs []*types.KeyValue, err error) {
	for _, row := range table.rows {
		kvlist, err := table.saveRow(row)
		if err != nil {
			return nil, err
		}
		kvs = append(kvs, kvlist...)
	}
	kvlist, err := table.autoinc.Save()
	if err != nil {
		return nil, err
	}
	kvs = append(kvs, kvlist...)
	//del cache
	table.rowmap = make(map[string]*Row)
	table.rows = nil
	return util.DelDupKey(kvs), nil
}

func pad(i int64) string {
	return fmt.Sprintf("%020d", i)
}

func (table *Table) saveRow(row *Row) (kvs []*types.KeyValue, err error) {
	if row.Ty == Del {
		return table.delRow(row)
	} else if row.Ty == Add {
		return table.addRow(row)
	} else if row.Ty == Update {
		return table.updateRow(row)
	} else if row.Ty == None {
		return nil, nil
	}
	return nil, errors.New("save table unknow action")
}

func (table *Table) delRow(row *Row) (kvs []*types.KeyValue, err error) {
	if !table.opt.Join {
		deldata := &types.KeyValue{Key: table.getDataKey(row.Primary)}
		kvs = append(kvs, deldata)
	}
	for _, index := range table.opt.Index {
		indexkey, err := table.index(row, index)
		if err != nil {
			return nil, err
		}
		delindex := &types.KeyValue{Key: table.getIndexKey(index, indexkey, row.Primary)}
		kvs = append(kvs, delindex)
	}
	return kvs, nil
}

func (table *Table) addRow(row *Row) (kvs []*types.KeyValue, err error) {
	if !table.opt.Join {
		data, err := row.Encode()
		if err != nil {
			return nil, err
		}
		adddata := &types.KeyValue{Key: table.getDataKey(row.Primary), Value: data}
		kvs = append(kvs, adddata)
	}
	for _, index := range table.opt.Index {
		indexkey, err := table.index(row, index)
		if err != nil {
			return nil, err
		}
		addindex := &types.KeyValue{Key: table.getIndexKey(index, indexkey, row.Primary), Value: row.Primary}
		kvs = append(kvs, addindex)
	}
	return kvs, nil
}

func (table *Table) updateRow(row *Row) (kvs []*types.KeyValue, err error) {
	if proto.Equal(row.Data, row.old) {
		return nil, nil
	}
	if !table.opt.Join {
		data, err := row.Encode()
		if err != nil {
			return nil, err
		}
		adddata := &types.KeyValue{Key: table.getDataKey(row.Primary), Value: data}
		kvs = append(kvs, adddata)
	}
	oldrow := &Row{Data: row.old}
	for _, index := range table.opt.Index {
		indexkey, oldkey, ismodify, err := table.getModify(row, oldrow, index)
		if err != nil {
			return nil, err
		}
		if !ismodify {
			continue
		}
		//del old
		delindex := &types.KeyValue{Key: table.getIndexKey(index, oldkey, row.Primary)}
		kvs = append(kvs, delindex)
		//add new
		addindex := &types.KeyValue{Key: table.getIndexKey(index, indexkey, row.Primary), Value: row.Primary}
		kvs = append(kvs, addindex)
	}
	return kvs, nil
}

func (table *Table) getModify(row, oldrow *Row, index string) ([]byte, []byte, bool, error) {
	if oldrow.Data == nil {
		return nil, nil, false, ErrNilValue
	}
	indexkey, err := table.index(row, index)
	if err != nil {
		return nil, nil, false, err
	}
	oldkey, err := table.index(oldrow, index)
	if err != nil {
		return nil, nil, false, err
	}
	if bytes.Equal(indexkey, oldkey) {
		return indexkey, oldkey, false, nil
	}
	return indexkey, oldkey, true, nil
}

//GetQuery 获取查询结构(允许传入 kvdb 为nil)
func (table *Table) GetQuery(kvdb db.KVDB) *Query {
	if kvdb == nil {
		var ok bool
		kvdb, ok = table.kvdb.(db.KVDB)
		if !ok {
			return nil
		}
	}
	return &Query{table: table, kvdb: kvdb}
}

func (table *Table) getMeta() RowMeta {
	return table.meta
}

//GetMeta 获取meta
func (table *Table) GetMeta() RowMeta {
	return table.getMeta()
}

func (table *Table) getOpt() *Option {
	return table.opt
}
