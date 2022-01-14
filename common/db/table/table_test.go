// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package table

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

func testTxTable(t *testing.T) (string, db.DB, db.KVDB, *Table) {
	dir, ldb, kvdb := util.CreateTestDB()
	opt := &Option{
		Prefix:  "prefix",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	return dir, ldb, kvdb, table
}

func TestTableListPrimary(t *testing.T) {

	dir, ldb, _, table := testTxTable(t)
	defer util.CloseTestDB(dir, ldb)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	addr1, priv1 := util.Genaddress()
	_, priv2 := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv1)
	assert.Nil(t, table.Add(tx1))
	tx2 := util.CreateNoneTx(cfg, priv2)
	assert.Nil(t, table.Add(tx2))
	kvs, err := table.Save()
	assert.Nil(t, err)
	for i := 0; i < len(kvs); i++ {
		fmt.Println(string(kvs[i].Key), ":", hex.EncodeToString(kvs[i].Value))
	}
	//save to database
	util.SaveKVList(ldb, kvs)
	// get smaller key
	hashsmall := []byte(hex.EncodeToString(tx1.Hash()))
	hashbig := []byte(hex.EncodeToString(tx2.Hash()))
	if bytes.Compare(hashsmall, hashbig) > 0 {
		hashsmall, hashbig = hashbig, hashsmall
	}
	// List: select * from table limit 10 asc
	rows, err := table.ListIndex("primary", nil, nil, 10, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, hashsmall, rows[0].Primary)
	row1 := NewTransactionRow()
	row1.SetPayload(rows[0].Data)
	row2 := NewTransactionRow()
	row2.SetPayload(rows[1].Data)
	fmt.Println(row1)
	fmt.Println(row2)
	// List: select * from table where primary > hashsmall and primary like prefix% order by asc
	//读到hashbig, 并且要符合前缀，显然不会符合
	fmt.Println(string(hashsmall[:20]))
	row, err := table.ListIndex("primary", hashsmall[:20], hashsmall, 10, db.ListASC)
	assert.Equal(t, types.ErrNotFound, err)
	if err == nil {
		row3 := NewTransactionRow()
		row3.SetPayload(row[0].Data)
		fmt.Println(row3)
	}
	t.Skip()
	// List: select * from table where primary < hashsmall and primary like prefix% order by desc
	//读不到任何数据, 并且要符合前缀，显然不会符合
	row, err = table.ListIndex("primary", hashsmall[:20], hashsmall, 10, db.ListDESC)
	assert.Equal(t, types.ErrNotFound, err)
	t.Log(row)

	// List: select * from table limit 10 asc
	rows, err = table.ListIndex("primary", nil, nil, 10, db.ListASC)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))
	assert.Equal(t, hashsmall, rows[0].Primary)

	// List: select * from table where primary < hashsmall order by desc
	//从hashsmall 开始往小的方向读
	_, err = table.ListIndex("primary", nil, hashsmall, 10, db.ListDESC)
	assert.Equal(t, types.ErrNotFound, err)

	// List: select * from table where primary like prefix% limit 10 order by asc
	// 注意: 这里的prefix 只是数据的前缀
	rows, _ = table.ListIndex("primary", hashsmall[:20], nil, 10, db.ListASC)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, hashsmall, rows[0].Primary)

	// List: select * from table where primary > hashsmall order by asc
	//从hashsmall 开始往大的方向读
	rows, _ = table.ListIndex("primary", nil, hashsmall, 10, db.ListASC)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, hashbig, rows[0].Primary)
	// List index
	rows, err = table.ListIndex("From", nil, nil, 10, db.ListDESC)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(rows))

	// List with prefix
	rows, _ = table.ListIndex("From", []byte(addr1[:20]), nil, 10, db.ListDESC)
	assert.Equal(t, 1, len(rows))
	assert.Equal(t, tx1.Hash(), rows[0].Primary)

	// List with primary and prefix
	_, err = table.ListIndex("From", []byte(addr1)[:20], tx1.Hash(), 10, db.ListASC)
	assert.Equal(t, types.ErrNotFound, err)
}

func TestTransactinList(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx1)
	assert.Nil(t, err)
	tx2 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx2)
	assert.Nil(t, err)

	addr2, priv := util.Genaddress()
	tx3 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx3)
	assert.Nil(t, err)
	tx4 := util.CreateNoneTx(cfg, priv)
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
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx1)
	assert.Nil(t, err)
	tx2 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx2)
	assert.Nil(t, err)

	addr2, priv := util.Genaddress()
	tx3 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx3)
	assert.Nil(t, err)
	tx4 := util.CreateNoneTx(cfg, priv)
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

func TestRow(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	rowmeta := NewTransactionRow()
	row := rowmeta.CreateRow()
	_, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv)
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
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx1)
	assert.Nil(t, err)

	_, priv = util.Genaddress()
	tx2 := util.CreateNoneTx(cfg, priv)
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

func TestUpdate(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	_, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv)
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

func TestReplaceTwice(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	opt := &Option{
		Prefix:  "prefix-hello",
		Name:    "name",
		Primary: "Hash",
		Index:   []string{"From", "To"},
	}
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx1)
	assert.Nil(t, err)

	//修改sign的内容不改变hash，但是会改变from 值，这个时候，应该是from 做了修改，而不是产生两条记录
	tx2 := types.Clone(tx1).(*types.Transaction)
	tx2.Signature.Pubkey[16] = tx2.Signature.Pubkey[16] + 1
	assert.Equal(t, tx1.From(), addr1)
	assert.Equal(t, tx1.Hash(), tx2.Hash())

	err = table.Replace(tx2)
	assert.Nil(t, err)

	//修改sign的内容不改变hash，但是会改变from 值，这个时候，应该是from 做了修改，而不是产生两条记录
	tx3 := types.Clone(tx1).(*types.Transaction)
	tx3.Signature.Pubkey[10] = tx3.Signature.Pubkey[10] + 1
	assert.Equal(t, tx1.From(), addr1)
	assert.Equal(t, tx1.Hash(), tx2.Hash())
	assert.Equal(t, tx1.Hash(), tx3.Hash())

	err = table.Replace(tx3)
	assert.Nil(t, err)
	//save 然后从列表中读取
	kvs, err := table.Save()
	assert.Nil(t, err)
	util.SaveKVList(ldb, kvs)
	query := table.GetQuery(kvdb)
	_, err = query.ListIndex("From", []byte(tx2.From()), nil, 0, 0)
	assert.Equal(t, err, types.ErrNotFound)
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
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	table, err := NewTable(NewTransactionRow(), kvdb, opt)
	assert.Nil(t, err)
	addr1, priv := util.Genaddress()
	tx1 := util.CreateNoneTx(cfg, priv)
	err = table.Add(tx1)
	assert.Nil(t, err)

	err = table.Add(tx1)
	assert.Equal(t, err, ErrDupPrimaryKey)

	//不改变hash，改变签名
	tx2 := types.CloneTx(tx1)
	tx2.Signature = nil
	err = table.Replace(tx2)
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
	result, err := tx.get(key)
	return result, err
}

func (tx *TransactionRow) get(key string) ([]byte, error) {
	if key == "Hash" {
		return []byte(hex.EncodeToString(tx.Hash())), nil
	} else if key == "From" {
		return []byte(tx.From()), nil
	} else if key == "To" {
		return []byte(tx.To), nil
	}
	return nil, types.ErrNotFound
}

func (tx *TransactionRow) String() string {
	hash, _ := tx.get("Hash")
	from, _ := tx.get("From")
	to, _ := tx.get("To")
	s := ""
	s += "Hash:" + string(hash) + "\n"
	s += "From:" + string(from) + "\n"
	s += "To:" + string(to)
	return s
}
