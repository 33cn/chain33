// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"testing"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"

	_ "github.com/33cn/chain33/system/dapp/coins/types"
)

func TestMakeStringUpper(t *testing.T) {
	originStr := "abcdefg"
	destStr, err := MakeStringToUpper(originStr, 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, "Abcdefg", destStr)

	destStr, err = MakeStringToUpper(originStr, 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, "abCDefg", destStr)

	_, err = MakeStringToUpper(originStr, -1, 2)
	assert.Error(t, err)
}

func TestMakeStringLower(t *testing.T) {
	originStr := "ABCDEFG"
	destStr, err := MakeStringToLower(originStr, 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, "aBCDEFG", destStr)

	destStr, err = MakeStringToLower(originStr, 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, "ABcdEFG", destStr)

	_, err = MakeStringToLower(originStr, -1, 2)
	assert.Error(t, err)
}

func TestResetDatadir(t *testing.T) {
	cfg, _ := types.InitCfg("../cmd/chain33/chain33.toml")
	datadir := ResetDatadir(cfg, "$TEMP/hello")
	assert.Equal(t, datadir+"/datadir", cfg.BlockChain.DbPath)

	cfg, _ = types.InitCfg("../cmd/chain33/chain33.toml")
	datadir = ResetDatadir(cfg, "/TEMP/hello")
	assert.Equal(t, datadir+"/datadir", cfg.BlockChain.DbPath)

	cfg, _ = types.InitCfg("../cmd/chain33/chain33.toml")
	datadir = ResetDatadir(cfg, "~/hello")
	assert.Equal(t, datadir+"/datadir", cfg.BlockChain.DbPath)
}

func TestHexToPrivkey(t *testing.T) {
	key := HexToPrivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01")
	addr := address.PubKeyToAddress(key.PubKey().Bytes()).String()
	assert.Equal(t, addr, "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv")
}

func TestGetParaExecName(t *testing.T) {
	s := GetParaExecName("user.p.hello.", "world")
	assert.Equal(t, "user.p.hello.world", s)
	s = GetParaExecName("user.p.hello.", "user.p.2.world")
	assert.Equal(t, "user.p.2.world", s)
}

func TestUpperLower(t *testing.T) {
	out, err := MakeStringToUpper("hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "Hello", out)

	out, err = MakeStringToUpper("Hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "Hello", out)

	_, err = MakeStringToUpper("Hello", -1, 1)
	assert.NotNil(t, err)

	_, err = MakeStringToUpper("Hello", 1, -1)
	assert.NotNil(t, err)

	out, err = MakeStringToLower("hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "hello", out)

	out, err = MakeStringToLower("Hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "hello", out)

	_, err = MakeStringToLower("Hello", -1, 1)
	assert.NotNil(t, err)

	_, err = MakeStringToLower("Hello", 1, -1)
	assert.NotNil(t, err)
}

func TestGenTx(t *testing.T) {
	txs := GenNoneTxs(TestPrivkeyList[0], 2)
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, "none", string(txs[0].Execer))
	assert.Equal(t, "none", string(txs[1].Execer))

	txs = GenCoinsTxs(TestPrivkeyList[0], 2)
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, "coins", string(txs[0].Execer))
	assert.Equal(t, "coins", string(txs[1].Execer))

	txs = GenTxsTxHeigt(TestPrivkeyList[0], 2, 10)
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, "coins", string(txs[0].Execer))
	assert.Equal(t, "coins", string(txs[1].Execer))
	assert.Equal(t, types.TxHeightFlag+10+20, txs[0].Expire)
}

func TestGenBlock(t *testing.T) {
	block2 := CreateNoneBlock(TestPrivkeyList[0], 2)
	assert.Equal(t, 2, len(block2.Txs))

	block2 = CreateCoinsBlock(TestPrivkeyList[0], 2)
	assert.Equal(t, 2, len(block2.Txs))

	txs := GenNoneTxs(TestPrivkeyList[0], 2)
	newblock := CreateNewBlock(block2, txs)
	assert.Equal(t, newblock.Height, block2.Height+1)
	assert.Equal(t, newblock.ParentHash, block2.Hash())
}

func TestDelDupKey(t *testing.T) {
	kvs := []*types.KeyValue{
		{Key: []byte("hello"), Value: []byte("world")},
		{Key: []byte("hello1"), Value: []byte("world")},
		{Key: []byte("hello"), Value: []byte("world2")},
	}
	result := []*types.KeyValue{
		{Key: []byte("hello1"), Value: []byte("world")},
		{Key: []byte("hello"), Value: []byte("world2")},
	}
	kvs = DelDupKey(kvs)
	assert.Equal(t, kvs, result)

	kvs = []*types.KeyValue{
		{Key: []byte("hello1"), Value: []byte("world")},
		{Key: []byte("hello"), Value: []byte("world")},
		{Key: []byte("hello"), Value: []byte("world2")},
	}
	result = []*types.KeyValue{
		{Key: []byte("hello1"), Value: []byte("world")},
		{Key: []byte("hello"), Value: []byte("world2")},
	}
	kvs = DelDupKey(kvs)
	assert.Equal(t, kvs, result)
}

func TestDelDupTx(t *testing.T) {
	txs := GenNoneTxs(TestPrivkeyList[0], 2)
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, "none", string(txs[0].Execer))
	assert.Equal(t, "none", string(txs[1].Execer))
	result := txs
	txs = append(txs, txs...)
	txcache := make([]*types.TransactionCache, len(txs))
	for i := 0; i < len(txcache); i++ {
		txcache[i] = &types.TransactionCache{Transaction: txs[i]}
	}

	txcacheresult := make([]*types.TransactionCache, len(result))
	for i := 0; i < len(result); i++ {
		txcacheresult[i] = &types.TransactionCache{Transaction: result[i]}
		txcacheresult[i].Hash()
	}
	txcache = DelDupTx(txcache)
	assert.Equal(t, txcache, txcacheresult)
}

func TestDB(t *testing.T) {
	dir, db, kvdb := CreateTestDB()
	defer CloseTestDB(dir, db)
	err := kvdb.Set([]byte("a"), []byte("b"))
	assert.Nil(t, err)
	value, err := kvdb.Get([]byte("a"))
	assert.Nil(t, err)
	assert.Equal(t, value, []byte("b"))
}

func TestMockModule(t *testing.T) {
	q := queue.New("channel")
	client := q.Client()
	memKey := "mempool"
	mem := &MockModule{Key: memKey}
	mem.SetQueueClient(client)

	msg := client.NewMessage(memKey, types.EventTx, &types.Transaction{})
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	assert.Nil(t, err)
	reply, ok := resp.GetData().(*types.Reply)
	assert.Equal(t, ok, true)
	assert.Equal(t, reply.GetIsOk(), false)
	assert.Equal(t, reply.GetMsg(), []byte("mock mempool module not handle message 1"))
}
