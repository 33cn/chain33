package blockchain

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/common/db/table"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

var (
	chainParaTxPrefix  = []byte("CHAIN-paratx")
	chainBodyPrefix    = []byte("CHAIN-body")
	chainHeaderPrefix  = []byte("CHAIN-header")
	chainReceiptPrefix = []byte("CHAIN-receipt")
)

func calcHeightHashKey(height int64, hash []byte) []byte {
	return append([]byte(fmt.Sprintf("%012d", height)), hash...)
}
func calcHeightParaKey(height int64) []byte {
	return []byte(fmt.Sprintf("%012d", height))
}

func calcHeightTitleKey(height int64, title string) []byte {
	return append([]byte(fmt.Sprintf("%012d", height)), []byte(title)...)
}

/*
table  body
data:  block body
index: hash
*/
var bodyOpt = &table.Option{
	Prefix:  "CHAIN-body",
	Name:    "body",
	Primary: "heighthash",
	Index:   []string{"hash"},
}

//NewBodyTable 新建表
func NewBodyTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewBodyRow()
	table, err := table.NewTable(rowmeta, kvdb, bodyOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//BodyRow table meta 结构
type BodyRow struct {
	*types.BlockBody
}

//NewBodyRow 新建一个meta 结构
func NewBodyRow() *BodyRow {
	return &BodyRow{BlockBody: &types.BlockBody{}}
}

//CreateRow 新建数据行
func (body *BodyRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.BlockBody{}}
}

//SetPayload 设置数据
func (body *BodyRow) SetPayload(data types.Message) error {
	if blockbody, ok := data.(*types.BlockBody); ok {
		body.BlockBody = blockbody
		return nil
	}
	return types.ErrTypeAsset
}

//Get 获取索引对应的key值
func (body *BodyRow) Get(key string) ([]byte, error) {
	if key == "heighthash" {
		return calcHeightHashKey(body.Height, body.Hash), nil
	} else if key == "hash" {
		return body.Hash, nil
	}
	return nil, types.ErrNotFound
}

//saveBlockBodyTable 保存block body
func saveBlockBodyTable(db dbm.DB, body *types.BlockBody) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewBodyTable(kvdb)

	err := table.Replace(body)
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//通过指定的index获取对应的blockbody
//通过高度获取：height+hash；indexName="",prefix=nil,primaryKey=calcHeightHashKey
//通过index获取：hash; indexName="hash",prefix=BodyRow.Get(indexName),primaryKey=nil
func getBodyByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte) (*types.BlockBody, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewBodyTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, 0, dbm.ListASC)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		panic("getBodyByIndex")
	}
	body, ok := rows[0].Data.(*types.BlockBody)
	if !ok {
		return nil, types.ErrDecode
	}
	return body, nil
}

//delBlockBodyTable 删除block Body
func delBlockBodyTable(db dbm.DB, height int64, hash []byte) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewBodyTable(kvdb)

	err := table.Del(calcHeightHashKey(height, hash))
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

/*
table  header
data:  block header
primary:heighthash
index: hash
*/
var headerOpt = &table.Option{
	Prefix:  "CHAIN-header",
	Name:    "header",
	Primary: "heighthash",
	Index:   []string{"hash"},
}

//HeaderRow table meta 结构
type HeaderRow struct {
	*types.Header
}

//NewHeaderRow 新建一个meta 结构
func NewHeaderRow() *HeaderRow {
	return &HeaderRow{Header: &types.Header{}}
}

//NewHeaderTable 新建表
func NewHeaderTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewHeaderRow()
	table, err := table.NewTable(rowmeta, kvdb, headerOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//CreateRow 新建数据行
func (header *HeaderRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.Header{}}
}

//SetPayload 设置数据
func (header *HeaderRow) SetPayload(data types.Message) error {
	if blockheader, ok := data.(*types.Header); ok {
		header.Header = blockheader
		return nil
	}
	return types.ErrTypeAsset
}

//Get 获取索引对应的key值
func (header *HeaderRow) Get(key string) ([]byte, error) {
	if key == "heighthash" {
		return calcHeightHashKey(header.Height, header.Hash), nil
	} else if key == "hash" {
		return header.Hash, nil
	}
	return nil, types.ErrNotFound
}

//saveHeaderTable 保存block header
func saveHeaderTable(db dbm.DB, header *types.Header) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewHeaderTable(kvdb)

	err := table.Replace(header)
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//通过指定的index获取对应的blockheader
//通过高度获取：height+hash；indexName="",prefix=nil,primaryKey=calcHeightHashKey
//通过index获取：hash; indexName="hash",prefix=HeaderRow.Get(indexName),primaryKey=nil
func getHeaderByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte) (*types.Header, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewHeaderTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, 0, dbm.ListASC)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		panic("getHeaderByIndex")
	}
	header, ok := rows[0].Data.(*types.Header)
	if !ok {
		return nil, types.ErrDecode
	}
	return header, nil
}

/*
table  paratx
data:  types.HeightPara
primary:heighttitle
index: height,title
*/
var paratxOpt = &table.Option{
	Prefix:  "CHAIN-paratx",
	Name:    "paratx",
	Primary: "heighttitle",
	Index:   []string{"height", "title"},
}

//ParaTxRow table meta 结构
type ParaTxRow struct {
	*types.HeightPara
}

//NewParaTxRow 新建一个meta 结构
func NewParaTxRow() *ParaTxRow {
	return &ParaTxRow{HeightPara: &types.HeightPara{}}
}

//NewParaTxTable 新建表
func NewParaTxTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewParaTxRow()
	table, err := table.NewTable(rowmeta, kvdb, paratxOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//CreateRow 新建数据行
func (paratx *ParaTxRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.HeightPara{}}
}

//SetPayload 设置数据
func (paratx *ParaTxRow) SetPayload(data types.Message) error {
	if heightPara, ok := data.(*types.HeightPara); ok {
		paratx.HeightPara = heightPara
		return nil
	}
	return types.ErrTypeAsset
}

//Get 获取索引对应的key值
func (paratx *ParaTxRow) Get(key string) ([]byte, error) {
	if key == "heighttitle" {
		return calcHeightTitleKey(paratx.Height, paratx.Title), nil
	} else if key == "height" {
		return calcHeightParaKey(paratx.Height), nil
	} else if key == "title" {
		return []byte(paratx.Title), nil
	}
	return nil, types.ErrNotFound
}

//saveParaTxTable 保存平行链标识
func saveParaTxTable(cfg *types.Chain33Config, db dbm.DB, height int64, hash []byte, txs []*types.Transaction) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewParaTxTable(kvdb)
	if !cfg.IsFork(height, "ForkRootHash") {
		for _, tx := range txs {
			exec := string(tx.Execer)
			if types.IsParaExecName(exec) {
				if title, ok := types.GetParaExecTitleName(exec); ok {
					var paratx = &types.HeightPara{Height: height, Title: title, Hash: hash}
					err := table.Replace(paratx)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	} else {
		//交易分类排序之后需要存储各个链的子roothash
		_, childhashs := merkle.CalcMultiLayerMerkleInfo(cfg, height, txs)
		for i, childhash := range childhashs {
			var paratx = &types.HeightPara{
				Height:         height,
				Hash:           hash,
				Title:          childhash.Title,
				ChildHash:      childhash.ChildHash,
				StartIndex:     childhash.StartIndex,
				ChildHashIndex: uint32(i),
				TxCount:        childhash.GetTxCount(),
			}
			err := table.Replace(paratx)
			if err != nil {
				return nil, err
			}
		}

	}
	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//delParaTxTable 删除平行链标识
//删除本高度对应的所有平行链的标识
func delParaTxTable(db dbm.DB, height int64) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewParaTxTable(kvdb)
	rows, _ := table.ListIndex("height", calcHeightParaKey(height), nil, 0, dbm.ListASC)
	for _, row := range rows {
		err := table.Del(row.Primary)
		if err != nil {
			return nil, err
		}
	}
	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//通过指定的index获取对应的ParaTx信息
//通过height高度获取：indexName="height",prefix=HeaderRow.Get(indexName)
//通过title获取;     indexName="title",prefix=HeaderRow.Get(indexName)
func getParaTxByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte, count, direction int32) (*types.HeightParas, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewParaTxTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, count, direction)
	if err != nil {
		return nil, err
	}
	var rep types.HeightParas
	for _, row := range rows {
		r, ok := row.Data.(*types.HeightPara)
		if !ok {
			return nil, types.ErrDecode
		}
		rep.Items = append(rep.Items, r)
	}
	return &rep, nil
}

/*
table  receipt
data:  block receipt
index: hash
*/
var receiptOpt = &table.Option{
	Prefix:  "CHAIN-receipt",
	Name:    "receipt",
	Primary: "heighthash",
	Index:   []string{"hash"},
}

//NewReceiptTable 新建表
func NewReceiptTable(kvdb dbm.KV) *table.Table {
	rowmeta := NewReceiptRow()
	table, err := table.NewTable(rowmeta, kvdb, receiptOpt)
	if err != nil {
		panic(err)
	}
	return table
}

//ReceiptRow table meta 结构
type ReceiptRow struct {
	*types.BlockReceipt
}

//NewReceiptRow 新建一个meta 结构
func NewReceiptRow() *ReceiptRow {
	return &ReceiptRow{BlockReceipt: &types.BlockReceipt{}}
}

//CreateRow 新建数据行
func (recpt *ReceiptRow) CreateRow() *table.Row {
	return &table.Row{Data: &types.BlockReceipt{}}
}

//SetPayload 设置数据
func (recpt *ReceiptRow) SetPayload(data types.Message) error {
	if blockReceipt, ok := data.(*types.BlockReceipt); ok {
		recpt.BlockReceipt = blockReceipt
		return nil
	}
	return types.ErrTypeAsset
}

//Get 获取索引对应的key值
func (recpt *ReceiptRow) Get(key string) ([]byte, error) {
	if key == "heighthash" {
		return calcHeightHashKey(recpt.Height, recpt.Hash), nil
	} else if key == "hash" {
		return recpt.Hash, nil
	}
	return nil, types.ErrNotFound
}

//saveBlockReceiptTable 保存block receipt
func saveBlockReceiptTable(db dbm.DB, recpt *types.BlockReceipt) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewReceiptTable(kvdb)

	err := table.Replace(recpt)
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}

//通过指定的index获取对应的blockReceipt
//通过高度获取：height+hash；indexName="",prefix=nil,primaryKey=calcHeightHashKey
//通过index获取：hash; indexName="hash",prefix=ReceiptRow.Get(indexName),primaryKey=nil
func getReceiptByIndex(db dbm.DB, indexName string, prefix []byte, primaryKey []byte) (*types.BlockReceipt, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewReceiptTable(kvdb)

	rows, err := table.ListIndex(indexName, prefix, primaryKey, 0, dbm.ListASC)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		panic("getReceiptByIndex")
	}
	recpt, ok := rows[0].Data.(*types.BlockReceipt)
	if !ok {
		return nil, types.ErrDecode
	}
	return recpt, nil
}

//delBlockReceiptTable 删除block receipt
func delBlockReceiptTable(db dbm.DB, height int64, hash []byte) ([]*types.KeyValue, error) {
	kvdb := dbm.NewKVDB(db)
	table := NewReceiptTable(kvdb)

	err := table.Del(calcHeightHashKey(height, hash))
	if err != nil {
		return nil, err
	}

	kvs, err := table.Save()
	if err != nil {
		return nil, err
	}
	return kvs, nil
}
