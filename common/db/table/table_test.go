// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package table

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func TestTransactinList(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(priv)
	err = table.Add(tx1)
	assert.Nil(t, err)
	tx2 := util.CreateNoneTx(priv)
	err = table.Add(tx2)
	assert.Nil(t, err)

	addr2, priv := util.Genaddress()
	tx3 := util.CreateNoneTx(priv)
	err = table.Add(tx3)
	assert.Nil(t, err)
	tx4 := util.CreateNoneTx(priv)
	err = table.Add(tx4)
	assert.Nil(t, err)
	//添加一个无效的类型
	err = table.Add(nil)
	assert.Equal(t, types.ErrTypeAsset, err)
	kvs, err := table.Save()
	assert.Nil(t, err)
	assert.Equal(t, len(kvs), 12)
	//save to database
	util.SaveKVList(ldb, kvs)
	//测试查询
	query := table.GetQuery(kvdb)

	rows, err := query.ListIndex("From", []byte(addr1), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	if bytes.Compare(tx1.Hash(), tx2.Hash()) > 0 {
		assert.Equal(t, true, proto.Equal(tx1, rows[0].Data))
		assert.Equal(t, true, proto.Equal(tx2, rows[1].Data))
	} else {
		assert.Equal(t, true, proto.Equal(tx1, rows[1].Data))
		assert.Equal(t, true, proto.Equal(tx2, rows[0].Data))
	}
	//prefix full
	rows, err = query.ListIndex("From", []byte(addr2), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	if bytes.Compare(tx3.Hash(), tx4.Hash()) > 0 {
		assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
		assert.Equal(t, true, proto.Equal(tx4, rows[1].Data))
	} else {
		assert.Equal(t, true, proto.Equal(tx3, rows[1].Data))
		assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	}
	//prefix part
	rows, err = query.ListIndex("From", []byte(addr2[0:10]), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	if bytes.Compare(tx3.Hash(), tx4.Hash()) > 0 {
		assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
		assert.Equal(t, true, proto.Equal(tx4, rows[1].Data))
	} else {
		assert.Equal(t, true, proto.Equal(tx3, rows[1].Data))
		assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	}
	//count
	rows, err = query.ListIndex("From", []byte(addr2[0:10]), nil, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	if bytes.Compare(tx3.Hash(), tx4.Hash()) > 0 {
		assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
	} else {
		assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	}
	primary := rows[0].Primary
	//primary
	rows, err = query.ListIndex("From", nil, primary, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	if bytes.Compare(tx3.Hash(), tx4.Hash()) > 0 {
		assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	} else {
		assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
	}
	//prefix + primary
	rows, err = query.ListIndex("From", []byte(addr2[0:10]), primary, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	if bytes.Compare(tx3.Hash(), tx4.Hash()) > 0 {
		assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	} else {
		assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
	}
	//List data
	rows, err = query.List("From", tx3, primary, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	if bytes.Compare(tx3.Hash(), tx4.Hash()) > 0 {
		assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	} else {
		assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
	}

	rows, err = query.ListIndex("From", []byte(addr1[0:10]), primary, 0, 0)
	assert.Equal(t, types.ErrNotFound, err)
	assert.Equal(t, 0, len(rows))
	//ListPrimary all
	rows, err = query.ListIndex("primary", nil, nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rows))

	//ListPrimary all
	rows, err = query.List("primary", nil, nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rows))

	row, err := query.ListOne("primary", nil, nil)
	assert.Nil(t, err)
	assert.Equal(t, row, rows[0])

	primary = rows[0].Primary
	rows, err = query.ListIndex("auto", primary, nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))

	rows, err = query.List("", rows[0].Data, nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))

	rows, err = query.ListIndex("", nil, primary, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(rows))
}

func TestTransactinListAuto(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix",
		Name:    "name",
		Primary: "",
		Index:   []string{"From", "To"},
	}
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(priv)
	err = table.Add(tx1)
	assert.Nil(t, err)
	tx2 := util.CreateNoneTx(priv)
	err = table.Add(tx2)
	assert.Nil(t, err)

	addr2, priv := util.Genaddress()
	tx3 := util.CreateNoneTx(priv)
	err = table.Add(tx3)
	assert.Nil(t, err)
	tx4 := util.CreateNoneTx(priv)
	err = table.Add(tx4)
	assert.Nil(t, err)
	//添加一个无效的类型
	err = table.Add(nil)
	assert.Equal(t, types.ErrTypeAsset, err)
	kvs, err := table.Save()
	assert.Nil(t, err)
	assert.Equal(t, len(kvs), 13)
	//save to database
	util.SaveKVList(ldb, kvs)
	//测试查询
	query := table.GetQuery(kvdb)

	rows, err := query.ListIndex("From", []byte(addr1), nil, 0, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, true, proto.Equal(tx1, rows[0].Data))
	assert.Equal(t, true, proto.Equal(tx2, rows[1].Data))
	//prefix full
	rows, err = query.ListIndex("From", []byte(addr2), nil, 0, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
	assert.Equal(t, true, proto.Equal(tx4, rows[1].Data))
	//prefix part
	rows, err = query.ListIndex("From", []byte(addr2[0:10]), nil, 0, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
	assert.Equal(t, true, proto.Equal(tx4, rows[1].Data))
	//count
	rows, err = query.ListIndex("From", []byte(addr2[0:10]), nil, 1, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(tx3, rows[0].Data))
	primary := rows[0].Primary
	//primary
	rows, err = query.ListIndex("From", nil, primary, 1, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	//prefix + primary
	rows, err = query.ListIndex("From", []byte(addr2[0:10]), primary, 0, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, true, proto.Equal(tx4, rows[0].Data))
	rows, err = query.ListIndex("From", []byte(addr1[0:10]), primary, 0, db.ListASC)
	assert.Equal(t, types.ErrNotFound, err)
	assert.Equal(t, 0, len(rows))
	//ListPrimary all
	rows, err = query.ListIndex("", nil, nil, 0, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rows))

	primary = rows[0].Primary
	rows, err = query.ListIndex("", primary, nil, 0, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(rows))

	rows, err = query.ListIndex("", nil, primary, 0, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(rows))
}

func mergeDup(kvs []*types.KeyValue) (kvset []*types.KeyValue) {
	maplist := make(map[string]*types.KeyValue)
	for _, kv := range kvs {
		if item, ok := maplist[string(kv.Key)]; ok {
			item.Value = kv.Value //更新item 的value
		} else {
			maplist[string(kv.Key)] = kv
			kvset = append(kvset, kv)
		}
	}
	return kvset
}

func TestRow(t *testing.T) {
	rowmeta := NewTransactionRow()
	row := rowmeta.CreateRow()
	_, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(priv)
	row.Data = tx1
	row.Primary = tx1.Hash()
	data, err := row.Encode()
	assert.Nil(t, err)
	primary, protodata, err := DecodeRow(data)
	assert.Nil(t, err)
	assert.Equal(t, primary, row.Primary)
	var tx types.Transaction
	err = types.Decode(protodata, &tx)
	assert.Nil(t, err)
	assert.Equal(t, proto.Equal(&tx, tx1), true)
}

func TestDel(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(priv)
	err = table.Add(tx1)
	assert.Nil(t, err)

	_, priv = util.Genaddress()
	tx2 := util.CreateNoneTx(priv)
	err = table.Add(tx2)
	assert.Nil(t, err)

	//删除掉一个
	err = table.Del(tx1.Hash())
	assert.Nil(t, err)

	//save 然后从列表中读取
	kvs, err := table.Save()
	assert.Nil(t, err)
	assert.Equal(t, 3, len(kvs))
	//save to database
	util.SaveKVList(ldb, kvs)
	//printKV(kvs)
	query := table.GetQuery(kvdb)
	rows, err := query.ListIndex("From", []byte(addr1[0:10]), nil, 0, 0)
	assert.Equal(t, types.ErrNotFound, err)
	assert.Equal(t, 0, len(rows))
}

func printAllKey(db db.DB) {
	it := db.Iterator(nil, nil, false)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		fmt.Println("db.allkey", string(it.Key()))
	}
}

func TestUpdate(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	_, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(priv)
	err = table.Add(tx1)
	assert.Nil(t, err)

	err = table.Update([]byte("hello"), tx1)
	assert.Equal(t, err, types.ErrInvalidParam)

	tx1.Signature = nil
	err = table.Update(tx1.Hash(), tx1)
	assert.Nil(t, err)
	kvs, err := table.Save()
	assert.Nil(t, err)
	assert.Equal(t, len(kvs), 3)
	//save to database
	util.SaveKVList(ldb, kvs)
	query := table.GetQuery(kvdb)
	rows, err := query.ListIndex("From", []byte(tx1.From()), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, rows[0].Data.(*types.Transaction).From(), tx1.From())
}

func TestReplace(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix-hello",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(priv)
	err = table.Add(tx1)
	assert.Nil(t, err)

	err = table.Add(tx1)
	assert.Equal(t, err, ErrDupPrimaryKey)

	//不改变hash，改变签名
	tx2 := *tx1
	tx2.Signature = nil
	err = table.Replace(&tx2)
	assert.Nil(t, err)
	//save 然后从列表中读取
	kvs, err := table.Save()
	assert.Nil(t, err)
	assert.Equal(t, 3, len(kvs))
	//save to database
	util.SaveKVList(ldb, kvs)
	query := table.GetQuery(kvdb)
	_, err = query.ListIndex("From", []byte(addr1[0:10]), nil, 0, 0)
	assert.Equal(t, err, types.ErrNotFound)

	rows, err := query.ListIndex("From", []byte(tx2.From()), nil, 0, 0)
	assert.Nil(t, err)
	assert.Equal(t, rows[0].Data.(*types.Transaction).From(), tx2.From())
}

type TransactionRow struct {
	*types.Transaction
}

func NewTransactionRow() *TransactionRow {
	return &TransactionRow{Transaction: &types.Transaction{}}
}

func (tx *TransactionRow) CreateRow() *Row {
	return &Row{Data: &types.Transaction{}}
}

func (tx *TransactionRow) SetPayload(data types.Message) error {
	if txdata, ok := data.(*types.Transaction); ok {
		tx.Transaction = txdata
		return nil
	}
	return types.ErrTypeAsset
}

func (tx *TransactionRow) Get(key string) ([]byte, error) {
	if key == "Hash" {
		return tx.Hash(), nil
	} else if key == "From" {
		return []byte(tx.From()), nil
	} else if key == "To" {
		return []byte(tx.To), nil
	}
	return nil, types.ErrNotFound
}
