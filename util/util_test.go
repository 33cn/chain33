// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	qmocks "github.com/33cn/chain33/queue/mocks"
	_ "github.com/33cn/chain33/system/crypto/secp256k1"
	_ "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
		{Key: []byte("hello"), Value: []byte("world2")},
		{Key: []byte("hello1"), Value: []byte("world")},
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

func BenchmarkDelDupKey(b *testing.B) {
	var kvs []*types.KeyValue
	for i := 0; i < 1000; i++ {
		key := common.GetRandBytes(20, 40)
		value := common.GetRandBytes(40, 60)
		kvs = append(kvs, &types.KeyValue{Key: key, Value: value})
		if i%10 == 0 {
			kvs = append(kvs, &types.KeyValue{Key: key, Value: value})
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testkv := make([]*types.KeyValue, len(kvs))
		copy(testkv, kvs)
		newkvs := DelDupKey(testkv)
		if newkvs[len(newkvs)-1] == nil {
			assert.NotNil(b, newkvs[len(newkvs)-1])
		}
	}
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

func TestJSONPrint(t *testing.T) {
	JSONPrint(t, &types.Reply{})
}

type testClient struct {
	qmocks.Client
}

var gid int64

func (t *testClient) NewMessage(topic string, ty int64, data interface{}) *queue.Message {
	id := atomic.AddInt64(&gid, 1)
	return queue.NewMessage(id, topic, ty, data)
}

func (t *testClient) Wait(in *queue.Message) (*queue.Message, error) {
	switch in.Ty {
	case types.EventTxHashList:
		return &queue.Message{Data: &types.TxHashList{}}, nil
	case types.EventExecTxList:
		return &queue.Message{Data: &types.Receipts{Receipts: []*types.Receipt{{Ty: 2}, {Ty: types.ExecErr}}}}, nil
	case types.EventStoreMemSet:
		return &queue.Message{Data: &types.ReplyHash{}}, nil
	case types.EventStoreRollback:
		return &queue.Message{Data: &types.ReplyHash{}}, nil
	case types.EventStoreCommit:
		return &queue.Message{Data: &types.ReplyHash{}}, nil
	case types.EventCheckBlock:
		return &queue.Message{Data: &types.Reply{IsOk: true}}, nil
	}

	return &queue.Message{}, nil
}

func TestExecBlock(t *testing.T) {
	client := &testClient{}
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	var txs []*types.Transaction
	addr, priv := Genaddress()
	tx := CreateCoinsTx(priv, addr, types.Coin)
	tx.Sign(types.SECP256K1, priv)
	txs = append(txs, tx)
	_, _, err := ExecBlock(client, nil, &types.Block{Txs: txs}, false, true, false)
	assert.NoError(t, err)
}

func TestExecBlockUpgrade(t *testing.T) {
	client := &testClient{}
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	var txs []*types.Transaction
	addr, priv := Genaddress()
	tx := CreateCoinsTx(priv, addr, types.Coin)
	tx.Sign(types.SECP256K1, priv)
	txs = append(txs, tx)
	err := ExecBlockUpgrade(client, nil, &types.Block{Txs: txs}, false)
	assert.NoError(t, err)
}

func TestExecAndCheckBlock(t *testing.T) {
	client := &testClient{}
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	_, err := ExecAndCheckBlock(client, &types.Block{}, nil, 0)
	assert.NoError(t, err)
	_, err = ExecAndCheckBlock2(client, &types.Block{}, nil, nil)
	assert.NoError(t, err)
}

func TestCheckBlock(t *testing.T) {
	client := &testClient{}
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	err := CheckBlock(client, nil)
	assert.NoError(t, err)
}

func TestExecKVSetRollback(t *testing.T) {
	client := &testClient{}
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	err := ExecKVSetRollback(client, nil)
	assert.NoError(t, err)
}

func TestCheckDupTx(t *testing.T) {
	client := &testClient{}
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	var txs []*types.Transaction
	addr, priv := Genaddress()
	tx := CreateCoinsTx(priv, addr, types.Coin)
	tx.Sign(types.SECP256K1, priv)
	txs = append(txs, tx)
	_, err := CheckDupTx(client, txs, 1)
	assert.NoError(t, err)
}

func TestReportErrEventToFront(t *testing.T) {
	logger := log.New("test")
	client := &testClient{}
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	ReportErrEventToFront(logger, client, "from", "to", errors.New("test"))
}
