package table

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
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

//指出是 添加 还是 删除 行
//primary key auto 的del 需要指定 primary key
const (
	None = iota
	Add
	Del
)

//meta key
const meta = "#m#"
const data = "#d#"

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
	kvdb       db.ReadOnlyDB
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
	Index   []string
}

//NewTable  新建一个表格
//primary 可以为: auto, 由系统自动创建
//index 可以为nil
func NewTable(rowmeta RowMeta, kvdb db.ReadOnlyDB, opt *Option) (*Table, error) {
	if len(opt.Index) > 16 {
		return nil, ErrTooManyIndex
	}
	for _, index := range opt.Index {
		if strings.Contains(index, "#") {
			return nil, ErrIndexKey
		}
	}
	if _, err := getPrimaryKey(rowmeta, opt.Primary); err != nil {
		return nil, err
	}
	//不允许有#
	if strings.Contains(opt.Prefix, "#") || strings.Contains(opt.Name, "#") {
		return nil, ErrTablePrefixOrTableName
	}
	dataprefix := opt.Prefix + "#" + opt.Name + data
	metaprefix := opt.Prefix + "#" + opt.Name + meta
	count := NewCount(opt.Prefix, opt.Name+"#autoinc#", kvdb)
	return &Table{
		meta:       rowmeta,
		kvdb:       kvdb,
		opt:        opt,
		autoinc:    count,
		dataprefix: dataprefix,
		metaprefix: metaprefix}, nil
}

func getPrimaryKey(meta RowMeta, primary string) ([]byte, error) {
	if primary == "" {
		return nil, ErrEmptyPrimaryKey
	}
	if strings.Contains(primary, "#") {
		return nil, ErrPrimaryKey
	}
	if primary != "auto" {
		key, err := meta.Get(primary)
		return key, err
	}
	return nil, nil
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

//Add 在表格中添加一行
func (table *Table) Add(data types.Message) error {
	if err := table.checkIndex(data); err != nil {
		return err
	}
	var err error
	var primaryKey []byte
	if table.opt.Primary == "auto" {
		primaryKey, err = table.getPrimaryAuto()
	} else {
		err = table.meta.SetPayload(data)
		if err != nil {
			return err
		}
		primaryKey, err = getPrimaryKey(table.meta, table.opt.Primary)
		if err != nil {
			return err
		}
	}
	table.rows = append(table.rows, &Row{Data: data, Primary: primaryKey, Ty: Add})
	return nil
}

//Del 在表格中删除一行(包括删除索引)
func (table *Table) Del(primaryKey []byte, data types.Message) error {
	if err := table.checkIndex(data); err != nil {
		return err
	}
	table.rows = append(table.rows, &Row{Data: data, Primary: primaryKey, Ty: Add})
	return nil
}

//DelByPrimary 在表格中删除一行(包括删除索引)
func (table *Table) DelByPrimary(primaryKey []byte) error {
	row, err := table.GetData(primaryKey)
	if err != nil {
		return err
	}
	if err := table.checkIndex(row.Data); err != nil {
		return err
	}
	table.rows = append(table.rows, &Row{Data: row.Data, Primary: primaryKey, Ty: Add})
	return nil
}

//getDataKey data key 构造
func (table *Table) getDataKey(primaryKey []byte) []byte {
	return append([]byte(table.dataprefix), primaryKey...)
}

//GetIndexKey data key 构造
func (table *Table) getIndexKey(indexName string, index, primaryKey []byte) []byte {
	key := table.indexPrefix(indexName)
	key = append(key, index...)
	key = append(key, []byte("#")...)
	key = append(key, primaryKey...)
	return key
}

func (table *Table) primaryPrefix() []byte {
	return []byte(table.dataprefix)
}

func (table *Table) indexPrefix(indexName string) []byte {
	key := append([]byte(table.metaprefix), []byte(indexName+"#")...)
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
	return kvs, nil
}

func pad(i int64) string {
	return fmt.Sprintf("%020d", i)
}

func (table *Table) saveRow(row *Row) (kvs []*types.KeyValue, err error) {
	if row.Ty == Del {
		return table.delRow(row)
	}
	return table.addRow(row)
}

func (table *Table) delRow(row *Row) (kvs []*types.KeyValue, err error) {
	deldata := &types.KeyValue{Key: table.getDataKey(row.Primary)}
	kvs = append(kvs, deldata)
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
	data, err := row.Encode()
	if err != nil {
		return nil, err
	}
	adddata := &types.KeyValue{Key: table.getDataKey(row.Primary), Value: data}
	kvs = append(kvs, adddata)
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

//GetQuery 获取查询结构
func (table *Table) GetQuery(kvdb db.ReadOnlyListDB) *Query {
	return &Query{table: table, kvdb: kvdb}
}
