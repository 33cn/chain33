// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor_test

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/merkle"
	_ "github.com/33cn/chain33/system"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

var runonce sync.Once

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte("demo2"))
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
}

func TestExecGenesisBlock(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	assert.Equal(t, block.Height, int64(0))
}

func newMockNode() *testnode.Chain33Mock {
	cfg := testnode.GetDefaultConfig()
	cfg.GetModuleConfig().Consensus.Minerstart = false
	runonce.Do(func() {
		drivers.Register(cfg, "demo2", newdemoApp, 1)
	})
	mock33 := testnode.NewWithConfig(cfg, nil)
	return mock33
}

func TestTxGroup(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	prev := cfg.GetMinTxFeeRate()
	cfg.SetMinFee(100000)
	defer cfg.SetMinFee(prev)
	mcfg := mock33.GetCfg()
	genkey := mock33.GetGenesisKey()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	acc := mock33.GetAccount(block.StateHash, mcfg.Consensus.Genesis)
	assert.Equal(t, acc.Balance, 100000000*types.DefaultCoinPrecision)
	var txs []*types.Transaction
	addr2, priv2 := util.Genaddress()
	addr3, priv3 := util.Genaddress()
	addr4, _ := util.Genaddress()
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr2, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, priv2, addr3, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, priv3, addr4, types.DefaultCoinPrecision))
	//执行三笔交易: 全部正确
	feeRate := cfg.GetMinTxFeeRate()
	txgroup, err := types.CreateTxGroup(txs, feeRate)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, priv3)
	//返回新的区块
	block, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), []int{types.ExecOk, types.ExecOk, types.ExecOk})
	assert.Nil(t, err)
	assert.Equal(t, mock33.GetAccount(block.StateHash, mock33.GetGenesisAddress()).Balance, int64(9999999899700000))
	assert.Equal(t, mock33.GetAccount(block.StateHash, addr2).Balance, int64(0))
	assert.Equal(t, mock33.GetAccount(block.StateHash, addr3).Balance, int64(0))
	assert.Equal(t, mock33.GetAccount(block.StateHash, addr4).Balance, 1*types.DefaultCoinPrecision)

	//执行三笔交易：第一比错误
	txs = nil
	txs = append(txs, util.CreateCoinsTx(cfg, priv2, addr3, 2*types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr4, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr4, types.DefaultCoinPrecision))

	txgroup, err = types.CreateTxGroup(txs, feeRate)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, priv2)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, genkey)

	_, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), []int{types.ExecErr, types.ExecErr, types.ExecErr})
	assert.Nil(t, err)
	//执行三笔交易：第二比错误
	txs = nil
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr2, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, priv2, addr4, 2*types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr4, types.DefaultCoinPrecision))

	txgroup, err = types.CreateTxGroup(txs, feeRate)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, genkey)

	_, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), []int{types.ExecPack, types.ExecPack, types.ExecPack})
	assert.Nil(t, err)
	//执行三笔交易: 第三比错误
	txs = nil
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr2, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr4, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, priv2, addr4, 10*types.DefaultCoinPrecision))

	txgroup, err = types.CreateTxGroup(txs, feeRate)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, priv2)

	_, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), []int{types.ExecPack, types.ExecPack, types.ExecPack})
	assert.Nil(t, err)
	//执行三笔交易：其中有一笔是user.xxx的执行器
	txs = nil
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr2, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr4, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateCoinsTx(cfg, priv2, addr4, 10*types.DefaultCoinPrecision))
	txs[2].Execer = []byte("user.xxx")
	txs[2].To = address.ExecAddress("user.xxx")
	txgroup, err = types.CreateTxGroup(txs, feeRate)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, priv2)

	_, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), []int{2, 2, 1})
	assert.Nil(t, err)
}

func TestExecAllow(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	prev := cfg.GetMinTxFeeRate()
	cfg.SetMinFee(100000)
	defer cfg.SetMinFee(prev)
	genkey := mock33.GetGenesisKey()
	genaddr := mock33.GetGenesisAddress()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	tx1 := util.CreateTxWithExecer(cfg, genkey, "user.evm")       //allow
	tx2 := util.CreateTxWithExecer(cfg, genkey, "coins")          //allow
	tx3 := util.CreateTxWithExecer(cfg, genkey, "evmxx")          //not allow
	tx4 := util.CreateTxWithExecer(cfg, genkey, "user.evmxx.xxx") //allow
	assert.Equal(t, mock33.GetAccount(block.StateHash, genaddr).Balance, 100000000*types.DefaultCoinPrecision)
	txs := []*types.Transaction{tx1, tx2, tx3, tx4}
	var err error
	block, err = util.ExecAndCheckBlockCB(mock33.GetClient(), block, txs, func(index int, receipt *types.ReceiptData) error {
		if index == 0 && receipt.GetTy() != 1 {
			return errors.New("user.evm is allow")
		}
		if index == 1 && receipt.GetTy() != 2 {
			return errors.New("coins exec ok")
		}
		if index == 2 && receipt != nil {
			return errors.New("evmxx is not allow")
		}
		if index == 3 && receipt.GetTy() != 1 {
			return errors.New("user.evmxx.xxx is allow")
		}
		return nil
	})
	assert.Nil(t, err)
}

func TestExecBlock2(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	genkey := mock33.GetGenesisKey()
	genaddr := mock33.GetGenesisAddress()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	assert.Equal(t, mock33.GetAccount(block.StateHash, genaddr).Balance, 100000000*types.DefaultCoinPrecision)
	txs := util.GenCoinsTxs(cfg, genkey, 2)

	block2 := util.CreateNewBlock(cfg, block, txs)
	detail, _, err := util.ExecBlock(mock33.GetClient(), block.StateHash, block2, false, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	for _, Receipt := range detail.Receipts {
		if Receipt.GetTy() != 2 {
			t.Errorf("exec expect true, but now false")
		}
	}

	N := 1000
	done := make(chan struct{}, N)
	for i := 0; i < N; i++ {
		go func() {
			txs := util.GenCoinsTxs(cfg, genkey, 2)
			block3 := util.CreateNewBlock(cfg, block2, txs)
			detail, _, err := util.ExecBlock(mock33.GetClient(), block2.StateHash, block3, false, true, false)
			assert.Nil(t, err)
			for _, Receipt := range detail.Receipts {
				if Receipt.GetTy() != 2 {
					t.Errorf("exec expect true, but now false")
				}
			}
			done <- struct{}{}
		}()
	}
	for n := 0; n < N; n++ {
		<-done
	}
}

var zeroHash [32]byte

func TestSameTx(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	newblock := &types.Block{}
	newblock.Height = 1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 3)
	hash1 := merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
	newblock.Txs = append(newblock.Txs, newblock.Txs[2])
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
	assert.Equal(t, hash1, newblock.TxHash)
	_, _, err := util.ExecBlock(mock33.GetClient(), nil, newblock, true, true, false)
	assert.Equal(t, types.ErrTxDup, err)

	//情况2
	//[tx1,xt2,tx3,tx4,tx5,tx6] and [tx1,xt2,tx3,tx4,tx5,tx6,tx5,tx6]
	newblock.Txs = util.GenNoneTxs(cfg, mock33.GetGenesisKey(), 6)
	hash1 = merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
	newblock.Txs = append(newblock.Txs, newblock.Txs[4:]...)
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.Height, newblock.Txs)
	assert.Equal(t, hash1, newblock.TxHash)
	_, _, err = util.ExecBlock(mock33.GetClient(), nil, newblock, true, true, false)
	assert.Equal(t, types.ErrTxDup, err)
}

func TestExecBlock(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	mock33.WaitHeight(0)
	block0 := mock33.GetBlock(0)
	block := util.CreateCoinsBlock(cfg, mock33.GetGenesisKey(), 10)
	util.ExecBlock(mock33.GetClient(), block0.StateHash, block, false, true, false)
}

//区块执行性能更好的一个测试
//1. 先生成 10万个账户，每个账户转1000个币
//2. 每个区块随机取1万比交易，然后执行
//3. 执行1000个块，看性能曲线的变化
//4. 排除掉网络掉影响
//5. 先对leveldb 做一个性能的测试

// 区块执行新能测试
func BenchmarkExecBlock(b *testing.B) {
	b.ReportAllocs()
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	block := util.CreateCoinsBlock(cfg, mock33.GetGenesisKey(), 10000)
	mock33.WaitHeight(0)
	block0 := mock33.GetBlock(0)
	account := mock33.GetAccount(block0.StateHash, mock33.GetGenesisAddress())
	assert.Equal(b, int64(10000000000000000), account.Balance)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		util.ExecBlock(mock33.GetClient(), block0.StateHash, block, false, true, false)
	}
}

/*
ExecLocalSameTime test
*/
type demoApp struct {
	*drivers.DriverBase
}

func newdemoApp() drivers.Driver {
	demo := &demoApp{DriverBase: &drivers.DriverBase{}}
	demo.SetChild(demo)
	return demo
}

func (demo *demoApp) GetDriverName() string {
	return "demo2"
}

var orderflag = drivers.ExecLocalSameTime

func (demo *demoApp) ExecutorOrder() int64 {
	return orderflag
}

func (demo *demoApp) Exec(tx *types.Transaction, index int) (receipt *types.Receipt, err error) {
	addr := tx.From()
	id := common.ToHex(tx.Hash())
	values, err := demo.GetLocalDB().List(demoCalcLocalKey(addr, ""), nil, 0, 0)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	if seterrkey {
		println("set err key value")
		err = demo.GetLocalDB().Set([]byte("key1"), []byte("value1"))
		if err != nil {
			return nil, err
		}
	}
	receipt = &types.Receipt{Ty: types.ExecOk}
	receipt.KV = append(receipt.KV, &types.KeyValue{
		Key:   demoCalcStateKey(addr, id),
		Value: []byte(fmt.Sprint(len(values))),
	})
	k := []byte("LODB-demo2-localkey")
	data, err := demo.GetLocalDB().Get(k)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	count := &types.Int64{Data: 0}
	if err == nil {
		err = types.Decode(data, count)
		if err != nil {
			return nil, err
		}
	}
	count.Data++
	receipt.Logs = append(receipt.Logs, &types.ReceiptLog{Ty: int32(len(values)),
		Log: types.Encode(count)})
	return receipt, nil
}

func (demo *demoApp) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (localkv *types.LocalDBSet, err error) {
	localkv = &types.LocalDBSet{}
	var count types.Int64
	err = types.Decode(receipt.Logs[0].Log, &count)
	if err != nil {
		return nil, err
	}
	localkv.KV = append(localkv.KV, &types.KeyValue{
		Key:   []byte("LODB-demo2-localkey"),
		Value: receipt.Logs[0].Log,
	})
	localkv.KV = append(localkv.KV, &types.KeyValue{
		Key:   []byte("LODB-demo2-localkey" + fmt.Sprint(count.GetData())),
		Value: receipt.Logs[0].Log,
	})
	if count.GetData() > 0 {
		localkv.KV = append(localkv.KV, &types.KeyValue{
			Key:   []byte("LODB-demo2-localkey" + fmt.Sprint(count.GetData()-1)),
			Value: nil,
		})
	}
	return localkv, nil
}

func demoCalcStateKey(addr string, id string) []byte {
	key := append([]byte("mavl-demo2-"), []byte(addr)...)
	key = append(key, []byte(":")...)
	key = append(key, []byte(id)...)
	return key
}

func demoCalcLocalKey(addr string, id string) []byte {
	key := append([]byte("LODB-demo2-"), []byte(addr)...)
	key = append(key, []byte(":")...)
	if len(id) > 0 {
		key = append(key, []byte(id)...)
	}
	return key
}

func TestExecLocalSameTime1(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	types.AllowUserExec = append(types.AllowUserExec, []byte("demo2"))
	cfg := mock33.GetClient().GetConfig()
	orderflag = 1
	genkey := mock33.GetGenesisKey()
	genaddr := mock33.GetGenesisAddress()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	assert.Equal(t, mock33.GetAccount(block.StateHash, genaddr).Balance, 100000000*types.DefaultCoinPrecision)
	var txs []*types.Transaction
	addr1, priv1 := util.Genaddress()
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr1, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateTxWithExecer(cfg, priv1, "demo2"))
	txs = append(txs, util.CreateTxWithExecer(cfg, priv1, "demo2"))
	block2 := util.CreateNewBlock(cfg, block, txs)
	detail, _, err := util.ExecBlock(mock33.GetClient(), block.StateHash, block2, false, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	for i, receipt := range detail.Receipts {
		assert.Equal(t, receipt.GetTy(), int32(2), fmt.Sprint(i))
	}
	receipt1 := detail.Receipts[1]
	assert.Equal(t, receipt1.Logs[1].Log, types.Encode(&types.Int64{Data: 1}))
	receipt2 := detail.Receipts[2]
	assert.Equal(t, receipt2.Logs[1].Log, types.Encode(&types.Int64{Data: 2}))
}

func TestExecLocalSameTime0(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	orderflag = 0
	genkey := mock33.GetGenesisKey()
	genaddr := mock33.GetGenesisAddress()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	assert.Equal(t, mock33.GetAccount(block.StateHash, genaddr).Balance, 100000000*types.DefaultCoinPrecision)
	var txs []*types.Transaction
	addr1, priv1 := util.Genaddress()
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr1, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateTxWithExecer(cfg, priv1, "demo2"))
	txs = append(txs, util.CreateTxWithExecer(cfg, priv1, "demo2"))
	block2 := util.CreateNewBlock(cfg, block, txs)
	detail, _, err := util.ExecBlock(mock33.GetClient(), block.StateHash, block2, false, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	assert.Equal(t, detail.Receipts[0].GetTy(), int32(2))
	receipt1 := detail.Receipts[1]
	assert.Equal(t, receipt1.Logs[1].Log, []byte(types.ErrDisableRead.Error()))
	receipt2 := detail.Receipts[2]
	assert.Equal(t, receipt2.Logs[1].Log, []byte(types.ErrDisableRead.Error()))
}

var seterrkey = false

func TestExecLocalSameTimeSetErrKey(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	orderflag = 1
	seterrkey = true
	genkey := mock33.GetGenesisKey()
	genaddr := mock33.GetGenesisAddress()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	assert.Equal(t, mock33.GetAccount(block.StateHash, genaddr).Balance, 100000000*types.DefaultCoinPrecision)
	var txs []*types.Transaction
	addr1, priv1 := util.Genaddress()
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr1, types.DefaultCoinPrecision))
	txs = append(txs, util.CreateTxWithExecer(cfg, priv1, "demo2"))
	txs = append(txs, util.CreateTxWithExecer(cfg, priv1, "demo2"))
	block2 := util.CreateNewBlock(cfg, block, txs)
	detail, _, err := util.ExecBlock(mock33.GetClient(), block.StateHash, block2, false, true, false)
	if err != nil {
		t.Error(err)
		return
	}
	for i, receipt := range detail.Receipts {
		if i == 0 {
			assert.Equal(t, receipt.GetTy(), int32(2))
		}
		if i >= 1 {
			assert.Equal(t, receipt.GetTy(), int32(1))
			assert.Equal(t, len(receipt.Logs), 2)
			assert.Equal(t, receipt.Logs[1].Ty, int32(1))
		}
	}
}
