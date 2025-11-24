// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/33cn/chain33/system/address/eth"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	"github.com/stretchr/testify/assert"

	nty "github.com/33cn/chain33/system/dapp/none/types"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/mock"

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
	"github.com/stretchr/testify/require"
)

// ----------------------------- data for testing ---------------------------------
var (
	c, _       = crypto.Load(types.GetSignName("", types.SECP256K1), -1)
	hexPirv    = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	a, _       = common.FromHex(hexPirv)
	privKey, _ = c.PrivKeyFromBytes(a)
	random     *rand.Rand
	mainPriv   crypto.PrivKey
	toAddr     = address.PubKeyToAddr(address.DefaultID, privKey.PubKey().Bytes())
	amount     = int64(1e8)
	v          = &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amount}}
	bigByte    = make([]byte, 99510)
	transfer   = &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
	tx1        = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 4, To: toAddr}
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
	cr, err := crypto.Load(types.GetSignName("", types.SECP256K1), -1)
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
	cfg := types.NewChain33Config(types.ReadFile("../../cmd/chain33/chain33.test.toml"))
	mcfg := cfg.GetModuleConfig()
	var q = queue.New("channel")
	q.SetConfig(cfg)
	mcfg.Consensus.Minerstart = false
	chain := blockchain.New(cfg)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg)
	exec.SetQueueClient(q.Client())

	cfg.SetMinFee(0)
	s := store.New(cfg)
	s.SetQueueClient(q.Client())
	subConfig := SubConfig{mcfg.Mempool.PoolCacheSize, mcfg.Mempool.MinTxFeeRate}
	mem := NewMempool(mcfg.Mempool)
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.SetQueueClient(q.Client())
	mem.Wait()
	return q, chain, s, mem
}

func initEnv(size int) (queue.Queue, *Mempool) {
	if size == 0 {
		size = 100
	}
	cfg := types.NewChain33Config(types.MergeCfg(types.ReadFile("../../cmd/chain33/chain33.test.toml"), types.ReadFile("../../cmd/chain33/chain33.fork.toml")))
	mcfg := cfg.GetModuleConfig()
	var q = queue.New("channel")
	q.SetConfig(cfg)
	blockchainProcess(q)
	rpcProcess(q)
	execProcess(q)
	mcfg.Mempool.PoolCacheSize = int64(size)
	subConfig := SubConfig{mcfg.Mempool.PoolCacheSize, mcfg.Mempool.MinTxFeeRate}
	mem := NewMempool(mcfg.Mempool)
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.SetQueueClient(q.Client())
	mem.setSync(true)
	mem.SetMinFee(cfg.GetMinTxFeeRate())
	mem.Wait()
	return q, mem
}

func initEnv4(size int) (queue.Queue, *Mempool) {
	if size == 0 {
		size = 100
	}
	cfg := types.NewChain33Config(types.MergeCfg(types.ReadFile("../../cmd/chain33/chain33.fork.toml"), types.ReadFile("testdata/chain33.test.toml")))
	mcfg := cfg.GetModuleConfig()
	var q = queue.New("channel")
	q.SetConfig(cfg)
	blockchainProcess(q)
	execProcess(q)
	mcfg.Mempool.PoolCacheSize = int64(size)
	subConfig := SubConfig{mcfg.Mempool.PoolCacheSize, mcfg.Mempool.MinTxFeeRate}
	mem := NewMempool(mcfg.Mempool)
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.SetQueueClient(q.Client())
	mem.setSync(true)
	mem.SetMinFee(cfg.GetMinTxFeeRate())
	mem.Wait()
	return q, mem
}

func createTx(cfg *types.Chain33Config, priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amount}}
	transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = rand.Int63()
	tx.ChainID = cfg.GetChainID()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.Load(types.GetSignName("", types.SECP256K1), -1)
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := address.PubKeyToAddr(address.DefaultID, privKey.PubKey().Bytes())
	return addrto, privto
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

func TestEventTxList(t *testing.T) {
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
	require.Nil(t, err)

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
	require.Equal(t, int64(3), reply.GetData().(*types.MempoolSize).Size)
}

func TestAddBlockedTx(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	msg1 := mem.client.NewMessage("mempool", types.EventTx, tx3)
	err := mem.client.Send(msg1, true)
	require.Nil(t, err)
	_, err = mem.client.Wait(msg1)
	require.Nil(t, err)
	blkDetail := &types.BlockDetail{Block: blk}
	msg2 := mem.client.NewMessage("mempool", types.EventAddBlock, blkDetail)
	mem.client.Send(msg2, false)

	msg3 := mem.client.NewMessage("mempool", types.EventTx, tx3)
	err = mem.client.Send(msg3, true)
	require.Nil(t, err)
	resp, err := mem.client.Wait(msg3)
	require.Nil(t, err)
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
	require.Equal(t, mem.Size(), 10)
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
	require.Equal(t, expectFee, fee)
	return fee
}

func TestGetProperFee(t *testing.T) {
	q, mem := initEnv(0)
	cfg := q.GetConfig()
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
	maxTxNum := cfg.GetP(mem.Height()).MaxTxNumber
	maxSize := types.MaxBlockSize
	msg11 := mem.client.NewMessage("mempool", types.EventTx, tx11)
	mem.client.Send(msg11, true)
	mem.client.Wait(msg11)

	baseFee := testProperFee(t, mem.client, nil, mem.cfg.MinTxFeeRate)
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
	ctx1 := types.CloneTx(tx1)
	msg := mem.client.NewMessage("mempool", types.EventTx, ctx1)
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
	require.Equal(t, mem.Size(), 4)
	mem.removeExpired()
	require.Equal(t, mem.Size(), 3)
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

	ad := address.PubKeyToAddr(address.DefaultID, privKey.PubKey().Bytes())
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
	cfg := q.GetConfig()
	defer q.Close()
	defer mem.Close()
	toAddr := "1PjMi9yGTjA9bbqUZa1Sj7dAUKyLA8KqE1"

	//copytx
	crouptx1 := types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: toAddr}
	crouptx2 := types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, To: toAddr}
	crouptx3 := types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr}
	crouptx4 := types.Transaction{Execer: []byte("user.write"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr}

	txGroup, _ := types.CreateTxGroup([]*types.Transaction{&crouptx1, &crouptx2, &crouptx3, &crouptx4}, cfg.GetMinTxFeeRate())

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
	cfg := q.GetConfig()
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
	txGroup, err := types.CreateTxGroup([]*types.Transaction{bigTx4, bigTx5, bigTx6, bigTx7, bigTx8, bigTx9, bigTx10, bigTx11}, cfg.GetMinTxFeeRate())
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
	require.Equal(t, sumFee, mem.cache.TotalFee())
	require.Equal(t, sumFee, int64(200000))

	mem.cache.Remove(string(txb.Hash()))

	var sumFee2 int64
	mem.cache.Walk(mem.cache.Size(), func(it *Item) bool {
		sumFee2 += it.Value.Fee
		return true
	})
	require.Equal(t, sumFee2, mem.cache.TotalFee())
	require.Equal(t, sumFee2, int64(100000))
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
	require.Equal(t, sumByte, mem.GetTotalCacheBytes())
	require.Equal(t, sumByte, int64(19))

	mem.cache.Remove(string(txb.Hash()))

	var sumByte2 int64
	mem.cache.Walk(mem.cache.Size(), func(it *Item) bool {
		sumByte2 += int64(proto.Size(it.Value))
		return true
	})
	require.Equal(t, sumByte2, mem.GetTotalCacheBytes())
	require.Equal(t, sumByte2, int64(9))
}

func BenchmarkMempool(b *testing.B) {
	q, mem := initEnv(10240)
	defer q.Close()
	defer mem.Close()
	maxTxNumPerAccount = 100000
	for i := 0; i < b.N; i++ {
		to, _ := genaddress()
		tx := createTx(q.GetConfig(), mainPriv, to, 10000)
		msg := mem.client.NewMessage("mempool", types.EventTx, tx)
		err := mem.client.Send(msg, true)
		if err != nil {
			println(err)
		}
	}
	to0, _ := genaddress()
	tx0 := createTx(q.GetConfig(), mainPriv, to0, 10000)
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

func rpcProcess(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub("rpc")
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetEvmNonce:
				msg.Reply(client.NewMessage("", types.EventGetEvmNonce, &types.EvmAccountNonce{Nonce: int64(1)}))
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

// BenchmarkGetTxList-8   	    1000	   1631315 ns/op
func BenchmarkGetTxList(b *testing.B) {

	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	txNum := 10000
	subConfig := SubConfig{int64(txNum), mem.cfg.MinTxFeeRate}
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.cache.AccountTxIndex.maxperaccount = txNum
	mem.cache.SHashTxCache.max = txNum
	cfg := q.GetConfig()
	for i := 0; i < txNum; i++ {
		tx := util.CreateCoinsTx(cfg, privKey, toAddr, int64(i+1))
		err := mem.PushTx(tx)
		require.Nil(b, err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mem.getTxList(&types.TxHashList{Hashes: nil, Count: int64(txNum)})
	}
}

func TestGetTxList(t *testing.T) {

	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	cacheSize := 100
	subConfig := SubConfig{int64(cacheSize), mem.cfg.MinTxFeeRate}
	mem.SetQueueCache(NewSimpleQueue(subConfig))
	mem.cache.AccountTxIndex.maxperaccount = cacheSize
	mem.cache.SHashTxCache.max = cacheSize

	cfg := q.GetConfig()
	currentHeight := cfg.GetFork("ForkTxHeight")
	mem.setHeader(&types.Header{Height: currentHeight})
	//push expired invalid tx
	for i := 0; i < 10; i++ {
		tx := util.CreateNoneTxWithTxHeight(cfg, privKey, currentHeight+2+types.LowAllowPackHeight)
		require.True(t, tx.IsExpire(cfg, currentHeight+1, 0))
		err := mem.PushTx(tx)
		require.Nil(t, err)
	}
	// push unexpired valid tx
	for i := 0; i < 10; i++ {
		tx := util.CreateNoneTxWithTxHeight(cfg, privKey, currentHeight+1+types.LowAllowPackHeight)
		require.False(t, tx.IsExpire(cfg, currentHeight+1, 0))
		require.True(t, tx.IsExpire(cfg, currentHeight, 0))
		err := mem.PushTx(tx)
		require.Nil(t, err)
	}

	// 取交易自动过滤过期交易
	txs := mem.getTxList(&types.TxHashList{Hashes: nil, Count: int64(5)})
	require.Equal(t, 5, len(txs))
	require.Equal(t, 20, mem.cache.Size())
	txs = mem.getTxList(&types.TxHashList{Hashes: nil, Count: int64(20)})
	require.Equal(t, 10, len(txs))
}

func TestCheckTxsExist(t *testing.T) {
	q, mem := initEnv(0)
	defer q.Close()
	defer mem.Close()

	txs := util.GenCoinsTxs(q.GetConfig(), privKey, 10)
	for _, tx := range txs {
		err := mem.PushTx(tx)
		require.Nil(t, err)
	}
	_, priv := util.Genaddress()
	txs1 := append(util.GenCoinsTxs(q.GetConfig(), priv, 10))

	checkReq := &types.ReqCheckTxsExist{}
	// 构造请求数据，存在不存在交替
	for i, tx := range txs {
		checkReq.TxHashes = append(checkReq.TxHashes, tx.Hash(), txs1[i].Hash())
	}
	checkReqMsg := mem.client.NewMessage("mempool", types.EventCheckTxsExist, checkReq)
	mem.eventCheckTxsExist(checkReqMsg)
	reply, err := mem.client.Wait(checkReqMsg)
	require.Nil(t, err)
	replyData := reply.GetData().(*types.ReplyCheckTxsExist)

	require.Equal(t, 20, len(replyData.ExistFlags))
	require.Equal(t, 10, int(replyData.ExistCount))
	for i, exist := range replyData.ExistFlags {
		//根据请求数据，结果应该是存在（true）、不存在（false）交替序列，即奇偶交错
		require.Equal(t, i%2 == 0, exist)
	}
	mem.setSync(false)
	mem.eventCheckTxsExist(checkReqMsg)
	reply, err = mem.client.Wait(checkReqMsg)
	require.Nil(t, err)
	replyData = reply.GetData().(*types.ReplyCheckTxsExist)
	require.Equal(t, 0, len(replyData.ExistFlags))
	require.Equal(t, 0, int(replyData.ExistCount))
}

func Test_pushDelayTxRoutine(t *testing.T) {

	_, mem := initEnv(1)
	mockAPI := &mocks.QueueProtocolAPI{}
	mem.setAPI(mockAPI)
	txChan := make(chan *types.Transaction)

	runFn := func(args mock.Arguments) {
		tx := args.Get(0).(*types.Transaction)
		txChan <- tx
	}
	mockAPI.On("SendTx", mock.Anything).Run(runFn).Return(nil, types.ErrMemFull).Once()
	mem.delayTxListChan <- []*types.Transaction{tx1}
	// push to mempool
	tx := <-txChan
	require.Equal(t, tx1.Hash(), tx.Hash())
	mockAPI.On("SendTx", mock.Anything).Run(runFn).Return(nil, nil)
	// retry push
	tx = <-txChan
	require.Equal(t, tx1.Hash(), tx.Hash())
	mem.delayTxListChan <- []*types.Transaction{tx2}
	tx = <-txChan
	require.Equal(t, tx2.Hash(), tx.Hash())
	close(mem.done)
	mem.delayTxListChan <- nil
	txs := <-mem.delayTxListChan
	require.Equal(t, 0, len(txs))
}

func Test_pushDelayTx(t *testing.T) {

	_, mem := initEnv(1)
	mockAPI := &mocks.QueueProtocolAPI{}
	mem.setAPI(mockAPI)

	cache := newDelayTxCache(100)

	txChan := make(chan *types.Transaction)

	runFn := func(args mock.Arguments) {
		tx := args.Get(0).(*types.Transaction)
		txChan <- tx
	}
	mockAPI.On("SendTx", mock.Anything).Run(runFn).Return(nil, nil)
	txList := make([]*types.Transaction, 100)
	for i := 0; i < 100; i++ {
		txList[i] = &types.Transaction{Payload: []byte(fmt.Sprintf("test%d", i))}
		err := cache.addDelayTx(&types.DelayTx{
			Tx:           txList[i],
			EndDelayTime: int64(i)})
		require.Nil(t, err)
		mem.pushExpiredDelayTx(cache, 0, 0, int64(i))
	}

	for i := 0; i < 100; i++ {
		<-txChan
	}
}

func Test_addDelayTxFromBlock(t *testing.T) {

	q, mem := initEnv(1)
	cache := newDelayTxCache(100)
	_, priv := util.Genaddress()

	block := util.CreateNoneBlock(q.GetConfig(), priv, 10)
	block.Height = 10
	mem.addDelayTx(cache, block)
	require.Equal(t, 0, len(cache.hashCache))
	delayTx := util.CreateNoneTx(q.GetConfig(), priv)
	action := &nty.NoneAction{}
	block.Txs[0].Payload = types.Encode(action)
	mem.addDelayTx(cache, block)
	require.Equal(t, 0, len(cache.hashCache))

	action.Ty = nty.TyCommitDelayTxAction
	block.Txs[0].Payload = types.Encode(action)
	mem.addDelayTx(cache, block)
	require.Equal(t, 0, len(cache.hashCache))
	action.Value = &nty.NoneAction_CommitDelayTx{CommitDelayTx: &nty.CommitDelayTx{
		DelayTx:             common.ToHex(types.Encode(delayTx)),
		RelativeDelayHeight: 1,
	}}

	block.Txs[0].Payload = types.Encode(action)
	mem.addDelayTx(cache, block)
	require.Equal(t, 1, len(cache.hashCache))

	//duplicate tx
	mem.addDelayTx(cache, block)
	require.Equal(t, 1, len(cache.hashCache))
	delayTime, ok := cache.contains(delayTx.Hash())
	require.Equal(t, 11, int(delayTime))
	require.True(t, ok)

}

func Test_sortEthSignTyTx(t *testing.T) {
	pub, err := common.FromHex("0x04715e4e07d983c2d98eeac7018bce6e68ef9de25835340f6455f1b1c9686132ac54904f5e04b07966a256140a5f487c4aef3ddc461e02d58f90cc8baa49f9c7ca")
	require.Nil(t, err)
	pub2, err := common.FromHex("0x0490a89752545aa381c2db0979e3af3e365d5d82adc34f9b36b038d7e19f3e6d65dd6230f5f064b58b3dcf195e95fd4cfc1d39a28ede4c1df5b0300a73ccd2d32f")
	require.Nil(t, err)
	sig := &types.Signature{
		Ty:     8452,
		Pubkey: pub,
	}
	sig2 := &types.Signature{
		Ty:     8452,
		Pubkey: pub2,
	}
	tx1 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: toAddr, Nonce: 1, Signature: sig}
	tx2 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0, To: toAddr, Nonce: 2, Signature: sig}
	tx3 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr, Nonce: 3, Signature: sig}
	tx4 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr, Nonce: 4, Signature: sig}
	tx5 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr, Nonce: 6, Signature: sig}
	tx6 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr, Nonce: 1, Signature: sig2}
	tx7 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr, Nonce: 2, Signature: sig2}
	tx8 := &types.Transaction{ChainID: 3999, Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0, To: toAddr, Nonce: 4, Signature: sig2}
	var txs []*types.Transaction
	txs = append(txs, tx4, tx3, tx1, tx2, tx5)
	_, mem := initEnv(1)
	txs = mem.sortEthSignTyTx(txs)
	//对排序后的结果进行校验
	require.Equal(t, 4, len(txs))
	require.Equal(t, txs[0].GetNonce(), tx1.GetNonce())
	require.Equal(t, txs[1].GetNonce(), tx2.GetNonce())
	require.Equal(t, txs[2].GetNonce(), tx3.GetNonce())
	require.Equal(t, txs[3].GetNonce(), tx4.GetNonce())
	txs = append(txs, tx4, tx3, tx1, tx2, tx5, tx6, tx7, tx8)
	_, mem = initEnv(1)
	txs = mem.sortEthSignTyTx(txs)
	txs = mem.sortEthSignTyTx(txs)
	require.Equal(t, 4+2, len(txs))
}

func TestCheckTxsNonce(t *testing.T) {
	var err error
	c, _ = crypto.Load(types.GetSignName("", types.SECP256K1ETH), -1)
	key, _ := common.FromHex(hexPirv)
	privKey, err = c.PrivKeyFromBytes(key)
	assert.Nil(t, err)
	tx0 := &types.Transaction{ChainID: 0, Execer: []byte("evm"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: toAddr, Nonce: 0}
	tx0.Sign(types.EncodeSignID(secp256k1eth.ID, eth.ID), privKey)
	tx1 := &types.Transaction{ChainID: 0, Execer: []byte("evm"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0, To: toAddr, Nonce: 1}
	tx1.Sign(types.EncodeSignID(secp256k1eth.ID, eth.ID), privKey)
	tx2 := &types.Transaction{ChainID: 0, Execer: []byte("evm"), Payload: types.Encode(transfer), Fee: 460000002, Expire: 0, To: toAddr, Nonce: 1}
	tx2.Sign(types.EncodeSignID(secp256k1eth.ID, eth.ID), privKey)
	tx3 := &types.Transaction{ChainID: 0, Execer: []byte("evm"), Payload: types.Encode(transfer), Fee: 460000002 + 45999998, Expire: 0, To: toAddr, Nonce: 2}
	tx3.Sign(types.EncodeSignID(secp256k1eth.ID, eth.ID), privKey)
	_, mem := initEnv(10)
	//测试nonce 较低的情况下进入mempool 检查
	msg := mem.client.NewMessage("mempool", types.EventTx, tx0)
	mem.client.Send(msg, true)
	msg, _ = mem.client.Wait(msg)
	reply := msg.GetData().(*types.Reply)
	assert.False(t, reply.GetIsOk())
	assert.Equal(t, "ErrNonceTooLow", string(reply.GetMsg()))

	msg = mem.client.NewMessage("mempool", types.EventTx, tx1)
	mem.client.Send(msg, true)
	msg, _ = mem.client.Wait(msg)
	reply = msg.GetData().(*types.Reply)
	assert.True(t, reply.GetIsOk())
	//相同的nonce的交易，在gas 不满足条件下，不允许通过过
	msg = mem.client.NewMessage("mempool", types.EventTx, tx2)
	mem.client.Send(msg, true)
	msg, err = mem.client.Wait(msg)
	assert.Nil(t, err)
	reply = msg.GetData().(*types.Reply)
	assert.False(t, reply.GetIsOk())
	assert.Equal(t, "disable transaction acceleration", string(reply.GetMsg()))
	txs := mem.GetLatestTx()
	assert.Equal(t, 1, len(txs))
	assert.Equal(t, txs[0].Hash(), tx1.Hash())
	msg = mem.client.NewMessage("mempool", types.EventTx, tx3)
	mem.client.Send(msg, true)
	msg, err = mem.client.Wait(msg)
	assert.Nil(t, err)
	reply = msg.GetData().(*types.Reply)
	assert.True(t, reply.GetIsOk())

	//此时mempool 中应该之后tx3 ,tx1 已经被自动删除
	txs = mem.GetLatestTx()
	assert.Equal(t, 2, len(txs))
	assert.Equal(t, txs[0].Hash(), tx1.Hash())
}
