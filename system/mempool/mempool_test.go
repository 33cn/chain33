// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/golang/protobuf/proto"

	"github.com/33cn/chain33/blockchain"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/limits"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/executor"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/store"
	_ "github.com/33cn/chain33/system/consensus/init"
	_ "github.com/33cn/chain33/system/crypto/init"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	_ "github.com/33cn/chain33/system/dapp/init"
	_ "github.com/33cn/chain33/system/store/init"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

//----------------------------- data for testing ---------------------------------
var (
	c, _       = crypto.New(types.GetSignName("", types.SECP256K1))
	hex        = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	a, _       = common.FromHex(hex)
	privKey, _ = c.PrivKeyFromBytes(a)
	random     *rand.Rand
	mainPriv   crypto.PrivKey
	toAddr     = address.PubKeyToAddress(privKey.PubKey().Bytes()).String()
	amount     = int64(1e8)
	v          = &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amount}}
	bigByte    = make([]byte, 99510)
	transfer   = &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
	tx1        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 2, To: toAddr}
	tx2        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr}
	tx3        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0, To: toAddr}
	tx4        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0, To: toAddr}
	tx5        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0, To: toAddr}
	tx6        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0, To: toAddr}
	tx7        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0, To: toAddr}
	tx8        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0, To: toAddr}
	tx9        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0, To: toAddr}
	tx10       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0, To: toAddr}
	tx11       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0, To: toAddr}
	tx12       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: toAddr}
	tx13       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, To: toAddr}
	tx14       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: "notaddress"}
	tx15       = &types.Transaction{Execer: []byte("user.write"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr}
	tx16       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000, Expire: 3, To: toAddr}
	tx17       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000, Expire: 4, To: toAddr}
	tx18       = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 4000000, Expire: 4, To: toAddr}

	bigTx1  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 100100000, Expire: 0, To: toAddr}
	bigTx2  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 100100000, Expire: 11, To: toAddr}
	bigTx3  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 11, To: toAddr}
	bigTx4  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 12, To: toAddr}
	bigTx5  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 13, To: toAddr}
	bigTx6  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 14, To: toAddr}
	bigTx7  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 15, To: toAddr}
	bigTx8  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 16, To: toAddr}
	bigTx9  = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 17, To: toAddr}
	bigTx10 = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 18, To: toAddr}
	bigTx11 = &types.Transaction{Execer: []byte("user.write"), Payload: bigByte, Fee: 1001000000, Expire: 19, To: toAddr}
)

//var privTo, _ = c.GenKey()
//var ad = address.PubKeyToAddress(privKey.PubKey().Bytes()).String()

var blk = &types.Block{
	Version:    1,
	ParentHash: []byte("parent hash"),
	TxHash:     []byte("tx hash"),
	Height:     2,
	BlockTime:  1,
	Txs:        []*types.Transaction{tx3, tx5},
}

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(types.Now().UnixNano()))
	queue.DisableLog()
	log.SetLogLevel("err") // 输出WARN(含)以下log
	mainPriv = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	tx1.Sign(types.SECP256K1, privKey)
	tx2.Sign(types.SECP256K1, privKey)
	tx3.Sign(types.SECP256K1, privKey)
	tx4.Sign(types.SECP256K1, privKey)
	tx5.Sign(types.SECP256K1, privKey)
	tx6.Sign(types.SECP256K1, privKey)
	tx7.Sign(types.SECP256K1, privKey)
	tx8.Sign(types.SECP256K1, privKey)
	tx9.Sign(types.SECP256K1, privKey)
	tx10.Sign(types.SECP256K1, privKey)
	tx11.Sign(types.SECP256K1, privKey)
	tx12.Sign(types.SECP256K1, privKey)
	tx13.Sign(types.SECP256K1, privKey)
	tx14.Sign(types.SECP256K1, privKey)
	tx15.Sign(types.SECP256K1, privKey)
	tx16.Sign(types.SECP256K1, privKey)
	tx17.Sign(types.SECP256K1, privKey)
	tx18.Sign(types.SECP256K1, privKey)
	bigTx1.Sign(types.SECP256K1, privKey)
	bigTx2.Sign(types.SECP256K1, privKey)
	bigTx3.Sign(types.SECP256K1, privKey)
	bigTx4.Sign(types.SECP256K1, privKey)
	bigTx5.Sign(types.SECP256K1, privKey)
	bigTx6.Sign(types.SECP256K1, privKey)
	bigTx7.Sign(types.SECP256K1, privKey)
	bigTx8.Sign(types.SECP256K1, privKey)
	bigTx9.Sign(types.SECP256K1, privKey)
	bigTx10.Sign(types.SECP256K1, privKey)
	bigTx11.Sign(types.SECP256K1, privKey)

}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func initEnv3() (queue.Queue, queue.Module, queue.Module, *Mempool) {
	var q = queue.New("channel")
	cfg, sub := types.InitCfg("../../cmd/chain33/chain33.test.toml")
	types.Init(cfg.Title, cfg)
	cfg.Consensus.Minerstart = false
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec, sub.Exec)
	exec.SetQueueClient(q.Client())

	types.SetMinFee(0)
	s := store.New(cfg.Store, sub.Store)
	s.SetQueueClient(q.Client())
	subConfig := SubConfig{cfg.Mempool.PoolCacheSize, cfg.Mempool.MinTxFee}
	mem := NewMempool(cfg.Mempool)
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.SetQueueClient(q.Client())
	mem.Wait()
	return q, chain, s, mem
}

func initEnv2(size int) (queue.Queue, *Mempool) {
	var q = queue.New("channel")
	cfg, _ := types.InitCfg("../../cmd/chain33/chain33.test.toml")
	types.Init(cfg.Title, cfg)
	blockchainProcess(q)
	execProcess(q)
	cfg.Mempool.PoolCacheSize = int64(size)
	subConfig := SubConfig{cfg.Mempool.PoolCacheSize, cfg.Mempool.MinTxFee}
	mem := NewMempool(cfg.Mempool)
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.SetQueueClient(q.Client())
	mem.setSync(true)
	mem.SetMinFee(0)
	mem.Wait()
	return q, mem
}

func initEnv(size int) (queue.Queue, *Mempool) {
	if size == 0 {
		size = 100
	}
	var q = queue.New("channel")
	cfg, _ := types.InitCfg("../../cmd/chain33/chain33.test.toml")
	types.Init(cfg.Title, cfg)
	blockchainProcess(q)
	execProcess(q)
	cfg.Mempool.PoolCacheSize = int64(size)
	subConfig := SubConfig{cfg.Mempool.PoolCacheSize, cfg.Mempool.MinTxFee}
	mem := NewMempool(cfg.Mempool)
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.SetQueueClient(q.Client())
	mem.setSync(true)
	mem.SetMinFee(types.GInt("MinFee"))
	mem.Wait()
	return q, mem
}

func initEnv4(size int) (queue.Queue, *Mempool) {
	if size == 0 {
		size = 100
	}
	var q = queue.New("channel")
	cfg, _ := types.InitCfg("testdata/chain33.test.toml")

	types.Init(cfg.Title, cfg)
	blockchainProcess(q)
	execProcess(q)
	cfg.Mempool.PoolCacheSize = int64(size)
	subConfig := SubConfig{cfg.Mempool.PoolCacheSize, cfg.Mempool.MinTxFee}
	mem := NewMempool(cfg.Mempool)
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.SetQueueClient(q.Client())
	mem.setSync(true)
	mem.SetMinFee(types.GInt("MinFee"))
	mem.Wait()
	return q, mem
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amount}}
	transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = rand.Int63()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func TestAddEmptyTx(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	msg := mem.client.NewMessage("mempool", types.EventTx, nil)
	mem.client.Send(msg, true)
	resp, err := mem.client.Wait(msg)
	if err != nil {
		t.Error(err)
		return
	}

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrEmptyTx.Error() {
		t.Error("TestAddEmptyTx failed")
	}
}

func TestAddTx(t *testing.T) {
	q, mem := initEnv(1)
	defer q.Close()
	defer mem.Close()
	msg := mem.client.NewMessage("mempool", types.EventTx, tx2)
	mem.client.Send(msg, true)
	mem.client.Wait(msg)
	if mem.Size() != 1 {
		t.Error("TestAddTx failed")
	}
}

func TestAddDuplicatedTx(t *testing.T) {
	q, mem := initEnv(100)
	defer q.Close()
	defer mem.Close()

	msg1 := mem.client.NewMessage("mempool", types.EventTx, tx2)
	err := mem.client.Send(msg1, true)
	if err != nil {
		t.Error(err)
		return
	}
	msg1, err = mem.client.Wait(msg1)
	if err != nil {
		t.Error(err)
		return
	}
	reply := msg1.GetData().(*types.Reply)
	err = checkReply(reply)
	if err != nil {
		t.Error(err)
		return
	}
	if mem.Size() != 1 {
		t.Error("TestAddDuplicatedTx failed", "size", mem.Size())
	}
	msg2 := mem.client.NewMessage("mempool", types.EventTx, tx2)
	mem.client.Send(msg2, true)
	mem.client.Wait(msg2)

	if mem.Size() != 1 {
		t.Error("TestAddDuplicatedTx failed", "size", mem.Size())
	}
}

func checkReply(reply *types.Reply) error {
	if !reply.GetIsOk() {
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func add4Tx(client queue.Client) error {
	msg1 := client.NewMessage("mempool", types.EventTx, tx1)
	msg2 := client.NewMessage("mempool", types.EventTx, tx2)
	msg3 := client.NewMessage("mempool", types.EventTx, tx3)
	msg4 := client.NewMessage("mempool", types.EventTx, tx4)
	client.Send(msg1, true)
	_, err := client.Wait(msg1)
	if err != nil {
		return err
	}

	client.Send(msg2, true)
	_, err = client.Wait(msg2)
	if err != nil {
		return err
	}

	client.Send(msg3, true)
	_, err = client.Wait(msg3)
	if err != nil {
		return err
	}

	client.Send(msg4, true)
	_, err = client.Wait(msg4)
	return err
}

func add4TxHash(client queue.Client) ([]string, error) {
	msg1 := client.NewMessage("mempool", types.EventTx, tx5)
	msg2 := client.NewMessage("mempool", types.EventTx, tx2)
	msg3 := client.NewMessage("mempool", types.EventTx, tx3)
	msg4 := client.NewMessage("mempool", types.EventTx, tx4)
	hashList := []string{string(tx5.Hash()), string(tx2.Hash()), string(tx3.Hash()), string(tx4.Hash())}
	client.Send(msg1, true)
	_, err := client.Wait(msg1)
	if err != nil {
		return nil, err
	}

	client.Send(msg2, true)
	_, err = client.Wait(msg2)
	if err != nil {
		return nil, err
	}

	client.Send(msg3, true)
	_, err = client.Wait(msg3)
	if err != nil {
		return nil, err
	}

	client.Send(msg4, true)
	_, err = client.Wait(msg4)
	if err != nil {
		return nil, err
	}
	return hashList, nil
}

func add10Tx(client queue.Client) error {
	err := add4Tx(client)
	if err != nil {
		return err
	}
	txs := []*types.Transaction{tx5, tx6, tx7, tx8, tx9, tx10}
	for _, tx := range txs {
		msg := client.NewMessage("mempool", types.EventTx, tx)
		client.Send(msg, true)
		_, err = client.Wait(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func TestGetTxList(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add tx
	hashes, err := add4TxHash(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	msg1 := mem.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 2, Hashes: nil})
	mem.client.Send(msg1, true)
	data1, err := mem.client.Wait(msg1)
	if err != nil {
		t.Error(err)
		return
	}
	txs1 := data1.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs1) != 2 {
		t.Error("get txlist number error")
	}

	var hashList [][]byte
	for i, tx := range txs1 {
		hashList = append(hashList, tx.Hash())
		if hashes[i] != string(tx.Hash()) {
			t.Error("gettxlist not in time order1")
		}
	}
	msg2 := mem.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 1, Hashes: hashList})
	mem.client.Send(msg2, true)
	data2, err := mem.client.Wait(msg2)
	if err != nil {
		t.Error(err)
		return
	}
	txs2 := data2.GetData().(*types.ReplyTxList).GetTxs()
	for i, tx := range txs2 {
		hashList = append(hashList, tx.Hash())
		if hashes[2+i] != string(tx.Hash()) {
			t.Error("gettxlist not in time order2")
		}
	}
OutsideLoop:
	for _, t1 := range txs1 {
		for _, t2 := range txs2 {
			if string(t1.Hash()) == string(t2.Hash()) {
				t.Error("TestGetTxList failed")
				break OutsideLoop
			}
		}
	}
}

func TestEventDelTxList(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add tx
	hashes, err := add4TxHash(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	hashBytes := [][]byte{[]byte(hashes[0]), []byte(hashes[1])}
	msg := mem.client.NewMessage("mempool", types.EventDelTxList, &types.TxHashList{Count: 2, Hashes: hashBytes})
	mem.client.Send(msg, true)
	_, err = mem.client.Wait(msg)
	if err != nil {
		t.Error(err)
		return
	}

	if mem.Size() != 2 {
		t.Error("TestEventDelTxList failed")
	}
}

func TestAddMoreTxThanPoolSize(t *testing.T) {
	q, mem := initEnv(4)
	defer q.Close()
	defer mem.Close()

	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	msg5 := mem.client.NewMessage("mempool", types.EventTx, tx5)
	mem.client.Send(msg5, true)
	mem.client.Wait(msg5)

	if mem.Size() != 4 || mem.cache.Exist(string(tx5.Hash())) {
		t.Error("TestAddMoreTxThanPoolSize failed", mem.Size(), mem.cache.Exist(string(tx5.Hash())))
	}
}

func TestAddMoreTxThanMaxAccountTx(t *testing.T) {
	q, mem := initEnv(4)
	mem.cfg.MaxTxNumPerAccount = 2
	defer q.Close()
	defer mem.Close()

	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}
	if mem.Size() != 2 {
		t.Error("TestAddMoreTxThanMaxAccountTx failed", "size", mem.Size())
	}
}

func TestRemoveTxOfBlock(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	blkDetail := &types.BlockDetail{Block: blk}
	msg5 := mem.client.NewMessage("mempool", types.EventAddBlock, blkDetail)
	mem.client.Send(msg5, false)

	msg := mem.client.NewMessage("mempool", types.EventGetMempoolSize, nil)
	mem.client.Send(msg, true)

	reply, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if reply.GetData().(*types.MempoolSize).Size != 3 {
		t.Error("TestGetMempoolSize failed")
	}
}

func TestAddBlockedTx(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	msg1 := mem.client.NewMessage("mempool", types.EventTx, tx3)
	err := mem.client.Send(msg1, true)
	assert.Nil(t, err)
	_, err = mem.client.Wait(msg1)
	assert.Nil(t, err)
	blkDetail := &types.BlockDetail{Block: blk}
	msg2 := mem.client.NewMessage("mempool", types.EventAddBlock, blkDetail)
	mem.client.Send(msg2, false)

	msg3 := mem.client.NewMessage("mempool", types.EventTx, tx3)
	err = mem.client.Send(msg3, true)
	assert.Nil(t, err)
	resp, err := mem.client.Wait(msg3)
	assert.Nil(t, err)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrDupTx.Error() {
		t.Error("TestAddBlockedTx failed")
	}
}

func TestDuplicateMempool(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add 10 txs
	err := add10Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}
	assert.Equal(t, mem.Size(), 10)
	msg := mem.client.NewMessage("mempool", types.EventGetMempool, nil)
	mem.client.Send(msg, true)

	reply, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if len(reply.GetData().(*types.ReplyTxList).GetTxs()) != 10 || mem.Size() != 10 {
		t.Error("TestDuplicateMempool failed")
	}
}

func TestGetLatestTx(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add 10 txs
	err := add10Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	msg11 := mem.client.NewMessage("mempool", types.EventTx, tx11)
	mem.client.Send(msg11, true)
	mem.client.Wait(msg11)

	msg := mem.client.NewMessage("mempool", types.EventGetLastMempool, nil)
	mem.client.Send(msg, true)

	reply, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if len(reply.GetData().(*types.ReplyTxList).GetTxs()) != 10 || mem.Size() != 11 {
		t.Error("TestGetLatestTx failed", len(reply.GetData().(*types.ReplyTxList).GetTxs()), mem.Size())
	}
}

func testProperFee(t *testing.T, client queue.Client, req *types.ReqProperFee, expectFee int64) int64 {
	msg := client.NewMessage("mempool", types.EventGetProperFee, req)
	client.Send(msg, true)
	reply, err := client.Wait(msg)
	if err != nil {
		t.Error(err)
		return 0
	}
	fee := reply.GetData().(*types.ReplyProperFee).GetProperFee()
	assert.Equal(t, expectFee, fee)
	return fee
}

func TestGetProperFee(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()
	defer func() {
		mem.cfg.IsLevelFee = false
	}()

	// add 10 txs
	err := add10Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}
	maxTxNum := types.GetP(mem.Height()).MaxTxNumber
	maxSize := types.MaxBlockSize
	msg11 := mem.client.NewMessage("mempool", types.EventTx, tx11)
	mem.client.Send(msg11, true)
	mem.client.Wait(msg11)

	baseFee := testProperFee(t, mem.client, nil, mem.cfg.MinTxFee)
	mem.cfg.IsLevelFee = true
	testProperFee(t, mem.client, nil, baseFee)
	testProperFee(t, mem.client, &types.ReqProperFee{}, baseFee)
	//more than 1/2 max num
	testProperFee(t, mem.client, &types.ReqProperFee{TxCount: int32(maxTxNum / 2)}, 100*baseFee)
	//more than 1/10 max num
	testProperFee(t, mem.client, &types.ReqProperFee{TxCount: int32(maxTxNum / 10)}, 10*baseFee)
	//more than 1/20 max size
	testProperFee(t, mem.client, &types.ReqProperFee{TxCount: 1, TxSize: int32(maxSize / 20)}, 100*baseFee)
	//more than 1/100 max size
	testProperFee(t, mem.client, &types.ReqProperFee{TxCount: 1, TxSize: int32(maxSize / 100)}, 10*baseFee)
}

func TestCheckLowFee(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	mem.SetMinFee(1000)
	msg := mem.client.NewMessage("mempool", types.EventTx, tx13)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrTxFeeTooLow.Error() {
		t.Error("TestCheckLowFee failed")
	}
}

func TestCheckSignature(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// make wrong signature
	tx12.Signature.Signature = tx12.Signature.Signature[5:]

	msg := mem.client.NewMessage("mempool", types.EventTx, tx12)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrSign.Error() {
		t.Error("TestCheckSignature failed", string(resp.GetData().(*types.Reply).GetMsg()))
	}
}

func TestCheckExpire1(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()
	mem.setHeader(&types.Header{Height: 50, BlockTime: 1e9 + 1})
	ctx1 := *tx1
	msg := mem.client.NewMessage("mempool", types.EventTx, &ctx1)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrTxExpire.Error() {
		t.Error("TestCheckExpire failed", string(resp.GetData().(*types.Reply).GetMsg()))
	}
}

func TestCheckExpire2(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add tx
	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}
	mem.setHeader(&types.Header{Height: 50, BlockTime: 1e9 + 1})
	msg := mem.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 100})
	mem.client.Send(msg, true)
	data, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	txs := data.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs) != 3 {
		t.Error("TestCheckExpire failed", len(txs))
	}
}

func TestCheckExpire3(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add tx
	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}
	mem.setHeader(&types.Header{Height: 50, BlockTime: 1e9 + 1})
	assert.Equal(t, mem.Size(), 4)
	mem.removeExpired()
	assert.Equal(t, mem.Size(), 3)
}

func TestWrongToAddr(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	msg := mem.client.NewMessage("mempool", types.EventTx, tx14)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrInvalidAddress.Error() {
		t.Error("TestWrongToAddr failed")
	}
}

func TestExecToAddrNotMatch(t *testing.T) {
	q, chain, s, mem := initEnv3()
	defer q.Close()
	defer mem.Close()
	defer chain.Close()
	defer s.Close()

	msg := mem.client.NewMessage("mempool", types.EventTx, tx15)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)
	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrToAddrNotSameToExecAddr.Error() {
		t.Error("TestExecToAddrNotMatch failed", string(resp.GetData().(*types.Reply).GetMsg()))
	}
}

func TestGetAddrTxs(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add tx
	_, err := add4TxHash(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	ad := address.PubKeyToAddress(privKey.PubKey().Bytes()).String()
	addrs := []string{ad}
	msg := mem.client.NewMessage("mempool", types.EventGetAddrTxs, &types.ReqAddrs{Addrs: addrs})
	mem.client.Send(msg, true)
	data, err := mem.client.Wait(msg)
	if err != nil {
		t.Error(err)
		return
	}
	txsFact := data.GetData().(*types.TransactionDetails).Txs
	txsExpect := mem.GetAccTxs(&types.ReqAddrs{Addrs: addrs}).Txs
	if len(txsExpect) != len(txsFact) {
		t.Error("TestGetAddrTxs failed", "length not match")
	}
	same := 0
	for _, i := range txsExpect {
		for _, j := range txsFact {
			if j.Tx == i.Tx {
				same++
				break
			}
		}
	}
	if same != len(txsExpect) {
		t.Error("TestGetAddrTxs failed", same)
	}
}

func TestDelBlock(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()
	delBlock := blk
	var blockDetail = &types.BlockDetail{Block: delBlock}

	mem.setHeader(&types.Header{Height: 2, BlockTime: 1e9 + 1})
	msg1 := mem.client.NewMessage("mempool", types.EventDelBlock, blockDetail)
	mem.client.Send(msg1, true)

	msg2 := mem.client.NewMessage("mempool", types.EventGetMempoolSize, nil)
	mem.client.Send(msg2, true)

	reply, err := mem.client.Wait(msg2)

	if err != nil {
		t.Error(err)
		return
	}
	size := reply.GetData().(*types.MempoolSize).Size
	if size != 2 {
		t.Error("TestDelBlock failed")
	}
}

func TestAddTxGroup(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()
	toAddr := "1PjMi9yGTjA9bbqUZa1Sj7dAUKyLA8KqE1"

	//copytx
	crouptx1 := types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: toAddr}
	crouptx2 := types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, To: toAddr}
	crouptx3 := types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr}
	crouptx4 := types.Transaction{Execer: []byte("user.write"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr}

	txGroup, _ := types.CreateTxGroup([]*types.Transaction{&crouptx1, &crouptx2, &crouptx3, &crouptx4}, types.GInt("MinFee"))

	for i := range txGroup.Txs {
		err := txGroup.SignN(i, types.SECP256K1, mainPriv)
		if err != nil {
			t.Error("TestAddTxGroup SignNfailed ", err.Error())
		}
	}
	tx := txGroup.Tx()

	msg := mem.client.NewMessage("mempool", types.EventTx, tx)
	mem.client.Send(msg, true)
	resp, err := mem.client.Wait(msg)
	if err != nil {
		t.Error("TestAddTxGroup failed", err.Error())
	}
	reply := resp.GetData().(*types.Reply)
	if !reply.GetIsOk() {
		t.Error("TestAddTxGroup failed", string(reply.GetMsg()))
	}
}

func TestLevelFeeBigByte(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()
	defer func() {
		mem.cfg.IsLevelFee = false
	}()
	mem.cfg.IsLevelFee = true
	mem.SetMinFee(100000)
	msg0 := mem.client.NewMessage("mempool", types.EventTx, tx1)
	mem.client.Send(msg0, true)
	resp0, _ := mem.client.Wait(msg0)
	if string(resp0.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(resp0.GetData().(*types.Reply).GetMsg()))
	}

	msg00 := mem.client.NewMessage("mempool", types.EventTx, tx17)
	mem.client.Send(msg00, true)
	resp00, _ := mem.client.Wait(msg00)
	if string(resp00.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(resp00.GetData().(*types.Reply).GetMsg()))
	}

	msgBig1 := mem.client.NewMessage("mempool", types.EventTx, bigTx1)
	mem.client.Send(msgBig1, true)
	respBig1, _ := mem.client.Wait(msgBig1)
	if string(respBig1.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(respBig1.GetData().(*types.Reply).GetMsg()))
	}

	msgBig2 := mem.client.NewMessage("mempool", types.EventTx, bigTx2)
	mem.client.Send(msgBig2, true)
	respBig2, _ := mem.client.Wait(msgBig2)
	if string(respBig2.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(respBig2.GetData().(*types.Reply).GetMsg()))
	}

	msgBig3 := mem.client.NewMessage("mempool", types.EventTx, bigTx3)
	mem.client.Send(msgBig3, true)
	respBig3, _ := mem.client.Wait(msgBig3)
	if string(respBig3.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(respBig3.GetData().(*types.Reply).GetMsg()))
	}

	//test low fee , feeRate = 10 * minfee
	msg2 := mem.client.NewMessage("mempool", types.EventTx, tx16)
	mem.client.Send(msg2, true)
	resp2, _ := mem.client.Wait(msg2)
	if string(resp2.GetData().(*types.Reply).GetMsg()) != types.ErrTxFeeTooLow.Error() {
		t.Error(string(resp2.GetData().(*types.Reply).GetMsg()))
	}

	//test high fee , feeRate = 10 * minfee
	msg3 := mem.client.NewMessage("mempool", types.EventTx, tx6)
	mem.client.Send(msg3, true)
	resp3, _ := mem.client.Wait(msg3)
	if string(resp3.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(resp3.GetData().(*types.Reply).GetMsg()))
	}

	//test group high fee , feeRate = 10 * minfee
	txGroup, err := types.CreateTxGroup([]*types.Transaction{bigTx4, bigTx5, bigTx6, bigTx7, bigTx8, bigTx9, bigTx10, bigTx11}, types.GInt("MinFee"))
	if err != nil {
		t.Error("CreateTxGroup err ", err.Error())
	}
	for i := range txGroup.Txs {
		err := txGroup.SignN(i, types.SECP256K1, mainPriv)
		if err != nil {
			t.Error("TestAddTxGroup SignNfailed ", err.Error())
		}
	}
	bigtxGroup := txGroup.Tx()

	msgBigG := mem.client.NewMessage("mempool", types.EventTx, bigtxGroup)
	mem.client.Send(msgBigG, true)
	respBigG, _ := mem.client.Wait(msgBigG)
	if string(respBigG.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(respBigG.GetData().(*types.Reply).GetMsg()))
	}

	//test low fee , feeRate = 100 * minfee
	msg4 := mem.client.NewMessage("mempool", types.EventTx, tx18)
	mem.client.Send(msg4, true)
	resp4, _ := mem.client.Wait(msg4)
	if string(resp4.GetData().(*types.Reply).GetMsg()) != types.ErrTxFeeTooLow.Error() {
		t.Error(string(resp4.GetData().(*types.Reply).GetMsg()))
	}

	//test high fee , feeRate = 100 * minfee
	msg5 := mem.client.NewMessage("mempool", types.EventTx, tx8)
	mem.client.Send(msg5, true)
	resp5, _ := mem.client.Wait(msg5)
	if string(resp5.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(resp5.GetData().(*types.Reply).GetMsg()))
	}
}
func TestLevelFeeTxNum(t *testing.T) {
	q, mem := initEnv4(0)
	defer q.Close()
	defer mem.Close()
	defer func() {
		mem.cfg.IsLevelFee = false
	}()
	mem.cfg.IsLevelFee = true
	mem.SetMinFee(100000)

	//test low fee , feeRate = 10 * minfee
	msg1 := mem.client.NewMessage("mempool", types.EventTx, tx16)
	mem.client.Send(msg1, true)
	resp1, _ := mem.client.Wait(msg1)
	if string(resp1.GetData().(*types.Reply).GetMsg()) != types.ErrTxFeeTooLow.Error() {
		t.Error(string(resp1.GetData().(*types.Reply).GetMsg()))
	}

	//test high fee , feeRate = 10 * minfee
	msg2 := mem.client.NewMessage("mempool", types.EventTx, tx6)
	mem.client.Send(msg2, true)
	resp2, _ := mem.client.Wait(msg2)
	if string(resp2.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(resp2.GetData().(*types.Reply).GetMsg()))
	}

	//test high fee , feeRate = 10 * minfee
	msg3 := mem.client.NewMessage("mempool", types.EventTx, tx7)
	mem.client.Send(msg3, true)
	resp3, _ := mem.client.Wait(msg3)
	if string(resp3.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(resp3.GetData().(*types.Reply).GetMsg()))
	}

	//test low fee , feeRate = 100 * minfee
	msg4 := mem.client.NewMessage("mempool", types.EventTx, tx18)
	mem.client.Send(msg4, true)
	resp4, _ := mem.client.Wait(msg4)
	if string(resp4.GetData().(*types.Reply).GetMsg()) != types.ErrTxFeeTooLow.Error() {
		t.Error(string(resp4.GetData().(*types.Reply).GetMsg()))
	}

	//test high fee , feeRate = 100 * minfee
	msg5 := mem.client.NewMessage("mempool", types.EventTx, tx8)
	mem.client.Send(msg5, true)
	resp5, _ := mem.client.Wait(msg5)
	if string(resp5.GetData().(*types.Reply).GetMsg()) != "" {
		t.Error(string(resp5.GetData().(*types.Reply).GetMsg()))
	}
}

func TestSimpleQueue_TotalFee(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()
	txa := &types.Transaction{Payload: []byte("123"), Fee: 100000}
	mem.cache.Push(txa)

	txb := &types.Transaction{Payload: []byte("1234"), Fee: 100000}
	mem.cache.Push(txb)

	var sumFee int64
	mem.cache.Walk(mem.cache.Size(), func(it *Item) bool {
		sumFee += it.Value.Fee
		return true
	})
	assert.Equal(t, sumFee, mem.cache.TotalFee())
	assert.Equal(t, sumFee, int64(200000))

	mem.cache.Remove(string(txb.Hash()))

	var sumFee2 int64
	mem.cache.Walk(mem.cache.Size(), func(it *Item) bool {
		sumFee2 += it.Value.Fee
		return true
	})
	assert.Equal(t, sumFee2, mem.cache.TotalFee())
	assert.Equal(t, sumFee2, int64(100000))
}

func TestSimpleQueue_TotalByte(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()
	txa := &types.Transaction{Payload: []byte("123"), Fee: 100000}
	mem.cache.Push(txa)

	txb := &types.Transaction{Payload: []byte("1234"), Fee: 100000}
	mem.cache.Push(txb)

	var sumByte int64
	mem.cache.Walk(mem.cache.Size(), func(it *Item) bool {
		sumByte += int64(proto.Size(it.Value))
		return true
	})
	assert.Equal(t, sumByte, mem.cache.TotalByte())
	assert.Equal(t, sumByte, int64(19))

	mem.cache.Remove(string(txb.Hash()))

	var sumByte2 int64
	mem.cache.Walk(mem.cache.Size(), func(it *Item) bool {
		sumByte2 += int64(proto.Size(it.Value))
		return true
	})
	assert.Equal(t, sumByte2, mem.cache.TotalByte())
	assert.Equal(t, sumByte2, int64(9))
}

func BenchmarkMempool(b *testing.B) {
	q, mem := initEnv(10240)
	defer q.Close()
	defer mem.Close()
	maxTxNumPerAccount = 100000
	for i := 0; i < b.N; i++ {
		to, _ := genaddress()
		tx := createTx(mainPriv, to, 10000)
		msg := mem.client.NewMessage("mempool", types.EventTx, tx)
		err := mem.client.Send(msg, true)
		if err != nil {
			println(err)
		}
	}
	to0, _ := genaddress()
	tx0 := createTx(mainPriv, to0, 10000)
	msg := mem.client.NewMessage("mempool", types.EventTx, tx0)
	mem.client.Send(msg, true)
	mem.client.Wait(msg)
	println(mem.Size() == b.N+1)
}

func blockchainProcess(q queue.Queue) {
	dup := make(map[string]bool)
	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			if msg.Ty == types.EventGetLastHeader {
				msg.Reply(client.NewMessage("", types.EventHeader, &types.Header{Height: 1, BlockTime: 1}))
			} else if msg.Ty == types.EventIsSync {
				msg.Reply(client.NewMessage("", types.EventReplyIsSync, &types.IsCaughtUp{Iscaughtup: true}))
			} else if msg.Ty == types.EventTxHashList {
				txs := msg.Data.(*types.TxHashList)
				var hashlist [][]byte
				for _, hash := range txs.Hashes {
					if dup[string(hash)] {
						hashlist = append(hashlist, hash)
						continue
					}
					dup[string(hash)] = true
				}
				msg.Reply(client.NewMessage("consensus", types.EventTxHashListReply, &types.TxHashList{Hashes: hashlist}))
			}
		}
	}()
}

func execProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("execs")
		for msg := range client.Recv() {
			if msg.Ty == types.EventCheckTx {
				datas := msg.GetData().(*types.ExecTxList)
				result := &types.ReceiptCheckTxList{}
				for i := 0; i < len(datas.Txs); i++ {
					result.Errs = append(result.Errs, "")
				}
				msg.Reply(client.NewMessage("", types.EventReceiptCheckTx, result))
			}
		}
	}()
}
func TestTx(t *testing.T) {
	subConfig := SubConfig{10240, 10000}
	cache := newCache(10240, 10, 10240)
	cache.SetQueueCache(NewSimpleQueue(subConfig))
	tx := &types.Transaction{Execer: []byte("user.write"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr}

	var replyTxList types.ReplyTxList
	var sHastList types.ReqTxHashList
	var hastList types.ReqTxHashList
	for i := 1; i <= 10240; i++ {
		tx.Expire = int64(i)
		cache.Push(tx)
		sHastList.Hashes = append(sHastList.Hashes, types.CalcTxShortHash(tx.Hash()))
		hastList.Hashes = append(hastList.Hashes, string(tx.Hash()))
	}

	for i := 1; i <= 1600; i++ {
		Tx := cache.GetSHashTxCache(sHastList.Hashes[i])
		if Tx == nil {
			panic("TestTx:GetSHashTxCache is nil")
		}
		replyTxList.Txs = append(replyTxList.Txs, Tx)
	}

	for i := 1; i <= 1600; i++ {
		Tx := cache.getTxByHash(hastList.Hashes[i])
		if Tx == nil {
			panic("TestTx:getTxByHash is nil")
		}
		replyTxList.Txs = append(replyTxList.Txs, Tx)
	}
}

func TestEventTxListByHash(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	// add tx
	hashes, err := add4TxHash(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}
	//通过交易hash获取交易信息
	reqTxHashList := types.ReqTxHashList{
		Hashes:      hashes,
		IsShortHash: false,
	}
	msg1 := mem.client.NewMessage("mempool", types.EventTxListByHash, &reqTxHashList)
	mem.client.Send(msg1, true)
	data1, err := mem.client.Wait(msg1)
	if err != nil {
		t.Error(err)
		return
	}

	txs1 := data1.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs1) != 4 {
		t.Error("TestEventTxListByHash:get txlist number error")
	}

	for i, tx := range txs1 {
		if hashes[i] != string(tx.Hash()) {
			t.Error("TestEventTxListByHash:hash mismatch")
		}
	}

	//通过短hash获取tx交易
	var shashes []string
	for _, hash := range hashes {
		shashes = append(shashes, types.CalcTxShortHash([]byte(hash)))
	}
	reqTxHashList.Hashes = shashes
	reqTxHashList.IsShortHash = true

	msg2 := mem.client.NewMessage("mempool", types.EventTxListByHash, &reqTxHashList)
	mem.client.Send(msg2, true)
	data2, err := mem.client.Wait(msg2)
	if err != nil {
		t.Error(err)
		return
	}
	txs2 := data2.GetData().(*types.ReplyTxList).GetTxs()
	for i, tx := range txs2 {
		if hashes[i] != string(tx.Hash()) {
			t.Error("TestEventTxListByHash:shash mismatch")
		}
	}
}
