// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/stretchr/testify/assert"
)

func TestInitReduceLocaldb(t *testing.T) {

	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	//cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore

	// for test initReduceLocaldb
	flagHeight := int64(0)
	endHeight := int64(80000)
	flag := int64(0)
	if flag == 0 {
		if endHeight > flagHeight {
			chain.walkOver(flagHeight, endHeight, false,
				func(batch dbm.Batch, height int64) {
					batch.Set([]byte(fmt.Sprintf("key-%d", height)), []byte(fmt.Sprintf("value-%d", height)))
				},
				func(batch dbm.Batch, height int64) {
					batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
				})
			// CompactRange执行将会阻塞仅仅做一次压缩
			chainlog.Info("reduceLocaldb start compact db")
			blockStore.db.CompactRange(nil, nil)
			chainlog.Info("reduceLocaldb end compact db")
		}
		blockStore.saveReduceLocaldbFlag()
	}

	flag, err = blockStore.loadFlag(types.FlagReduceLocaldb)
	assert.NoError(t, err)
	assert.Equal(t, flag, int64(1))

	flagHeight, err = blockStore.loadFlag(types.ReduceLocaldbHeight)
	assert.NoError(t, err)
	assert.Equal(t, flagHeight, endHeight)

}

func TestInitReduceLocaldb1(t *testing.T) {

	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	//cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore

	// for test initReduceLocaldb
	flagHeight := int64(0)
	endHeight := int64(80000)
	flag := int64(0)
	if flag == 0 {
		defer func() {
			if r := recover(); r != nil {
				flag, err = blockStore.loadFlag(types.FlagReduceLocaldb)
				assert.NoError(t, err)
				assert.Equal(t, flag, int64(0))

				flagHeight, err = blockStore.loadFlag(types.ReduceLocaldbHeight)
				assert.NoError(t, err)
				assert.NotEqual(t, flagHeight, endHeight)
				return
			}
		}()

		if endHeight > flagHeight {
			chain.walkOver(flagHeight, endHeight, false,
				func(batch dbm.Batch, height int64) {
					batch.Set([]byte(fmt.Sprintf("key-%d", height)), []byte(fmt.Sprintf("value-%d", height)))
				},
				func(batch dbm.Batch, height int64) {
					if height == endHeight {
						panic("for test")
					}
					batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
				})
			// CompactRange执行将会阻塞仅仅做一次压缩
			chainlog.Info("reduceLocaldb start compact db")
			blockStore.db.CompactRange(nil, nil)
			chainlog.Info("reduceLocaldb end compact db")
		}
		blockStore.saveReduceLocaldbFlag()
	}
}

func TestReduceBody(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore

	// generate blockdetail
	txs := util.GenCoinsTxs(cfg, util.HexToPrivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01"), 10)
	block := &types.Block{Txs: txs}
	block.MainHash = block.Hash(cfg)
	block.Height = 0
	blockdetail := &types.BlockDetail{
		Block: block,
		Receipts: []*types.ReceiptData{
			{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}},
			{Ty: 1, Logs: []*types.ReceiptLog{{Ty: 111, Log: []byte("111")}, {Ty: 1111, Log: []byte("1111")}}},
			{Ty: 2, Logs: []*types.ReceiptLog{{Ty: 222, Log: []byte("222")}, {Ty: 2222, Log: []byte("2222")}}},
			{Ty: 3, Logs: []*types.ReceiptLog{{Ty: 333, Log: []byte("333")}, {Ty: 3333, Log: []byte("3333")}}},
		},
		KV: []*types.KeyValue{{Key: []byte("000"), Value: []byte("000")}, {Key: []byte("111"), Value: []byte("111")}},
	}

	// save blockdetail
	newbatch := blockStore.NewBatch(true)
	_, err = blockStore.SaveBlock(newbatch, blockdetail, 0)
	assert.NoError(t, err)
	newbatch.Write()

	// reduceBody
	newbatch = blockStore.NewBatch(true)
	chain.reduceBody(newbatch, 0)
	newbatch.Write()

	// check
	body, err := blockStore.LoadBlockBody(0)
	assert.NoError(t, err)
	for _, recep := range body.Receipts {
		for _, log := range recep.Logs {
			assert.Nil(t, log.Log)
		}
	}
}

func TestReduceBodyInit(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore

	// generate blockdetail
	txs := util.GenCoinsTxs(cfg, util.HexToPrivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01"), 10)
	block := &types.Block{Txs: txs}
	block.MainHash = block.Hash(cfg)
	block.Height = 0
	blockdetail := &types.BlockDetail{
		Block: block,
		Receipts: []*types.ReceiptData{
			{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}},
			{Ty: 1, Logs: []*types.ReceiptLog{{Ty: 111, Log: []byte("111")}, {Ty: 1111, Log: []byte("1111")}}},
			{Ty: 2, Logs: []*types.ReceiptLog{{Ty: 222, Log: []byte("222")}, {Ty: 2222, Log: []byte("2222")}}},
			{Ty: 3, Logs: []*types.ReceiptLog{{Ty: 333, Log: []byte("333")}, {Ty: 3333, Log: []byte("3333")}}},
		},
		KV: []*types.KeyValue{{Key: []byte("000"), Value: []byte("000")}, {Key: []byte("111"), Value: []byte("111")}},
	}

	// save blockdetail
	newbatch := blockStore.NewBatch(true)
	_, err = blockStore.SaveBlock(newbatch, blockdetail, 0)
	assert.NoError(t, err)
	newbatch.Write()

	// save tx TxResult
	newbatch = blockStore.NewBatch(true)
	for index, tx := range txs {
		var txresult types.TxResult
		txresult.Height = block.Height
		txresult.Index = int32(index)
		txresult.Tx = tx
		txresult.Receiptdate = &types.ReceiptData{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}}
		txresult.Blocktime = 3123131231
		txresult.ActionName = tx.ActionName()
		newbatch.Set(cfg.CalcTxKey(tx.Hash()), cfg.CalcTxKeyValue(&txresult))
	}
	newbatch.Write()

	// reduceBodyInit
	cfg.S("reduceLocaldb", true)
	newbatch = blockStore.NewBatch(true)
	chain.reduceBodyInit(newbatch, 0)
	newbatch.Write()

	// check
	// 1 body
	body, err := blockStore.LoadBlockBody(0)
	assert.NoError(t, err)
	for _, recep := range body.Receipts {
		for _, log := range recep.Logs {
			assert.Nil(t, log.Log)
		}
	}
	// 2 tx
	for _, tx := range txs {
		hash := tx.Hash()
		_, err := blockStore.db.Get(hash)
		assert.Error(t, err, types.ErrNotFound)
		v, err := blockStore.db.Get(cfg.CalcTxKey(hash))
		assert.NoError(t, err)
		var txresult types.TxResult
		err = types.Decode(v, &txresult)
		assert.NoError(t, err)
		assert.Nil(t, txresult.Receiptdate)
	}
}

func TestReduceReceipts(t *testing.T) {
	receipts := []*types.ReceiptData{
		{Ty: 0, Logs: []*types.ReceiptLog{{Ty: 0, Log: []byte("000")}, {Ty: 0, Log: []byte("0000")}}},
		{Ty: 1, Logs: []*types.ReceiptLog{{Ty: 111, Log: []byte("111")}, {Ty: 1111, Log: []byte("1111")}}},
		{Ty: 2, Logs: []*types.ReceiptLog{{Ty: 222, Log: []byte("222")}, {Ty: 2222, Log: []byte("2222")}}},
		{Ty: 3, Logs: []*types.ReceiptLog{{Ty: 333, Log: []byte("333")}, {Ty: 3333, Log: []byte("3333")}}},
	}
	reduceReceipts(receipts)

	// check
	for _, recpt := range receipts {
		for _, log := range recpt.Logs {
			assert.Nil(t, log.Log)
		}
	}
}

func TestFIFO(t *testing.T) {
	fifo10 := NewFIFO(10)

	for i := 0; i < 11; i++ {
		fifo10.Add(i, []byte(fmt.Sprintf("value-%d", i)))
	}
	// check Contains Get
	ok := fifo10.Contains(0)
	assert.Equal(t, ok, false)
	value, ok := fifo10.Get(0)
	assert.Equal(t, value, nil)
	assert.Equal(t, ok, false)
	for i := 1; i < 11; i++ {
		ok := fifo10.Contains(i)
		assert.Equal(t, ok, true)
		value, ok := fifo10.Get(i)
		assert.Equal(t, ok, true)
		assert.Equal(t, value.([]byte), []byte(fmt.Sprintf("value-%d", i)))
	}
	// check Remove
	ok = fifo10.Remove(10)
	assert.Equal(t, ok, true)
	value, ok = fifo10.Get(10)
	assert.Equal(t, value, nil)
	assert.Equal(t, ok, false)

	ok = fifo10.Remove(11)
	assert.Equal(t, ok, false)

	// test for size = 0
	fifo0 := NewFIFO(0)

	fifo0.Add(0, []byte(fmt.Sprintf("value-%d", 0)))
	value, ok = fifo0.Get(0)
	assert.Equal(t, ok, true)
	assert.Equal(t, value.([]byte), []byte(fmt.Sprintf("value-%d", 0)))

	fifo0.Add(1, []byte(fmt.Sprintf("value-%d", 1)))
	value, ok = fifo0.Get(0)
	assert.Equal(t, ok, false)

	value, ok = fifo0.Get(1)
	assert.Equal(t, ok, true)
	assert.Equal(t, value.([]byte), []byte(fmt.Sprintf("value-%d", 1)))

	// remove 0  fasle
	ok = fifo0.Remove(0)
	assert.Equal(t, ok, false)

	// remove 1 true
	ok = fifo0.Remove(1)
	assert.Equal(t, ok, true)
	// Get 1 false
	_, ok = fifo0.Get(1)
	assert.Equal(t, ok, false)
}
