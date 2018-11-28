// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor_test

import (
	"errors"
	"net/http"
	_ "net/http/pprof"
	"testing"

	"github.com/33cn/chain33/common/address"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func init() {
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
	cfg, sub := testnode.GetDefaultConfig()
	cfg.Consensus.Minerstart = false
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	return mock33
}

func TestTxGroup(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	prev := types.GInt("MinFee")
	types.SetMinFee(100000)
	defer types.SetMinFee(prev)
	cfg := mock33.GetCfg()
	genkey := mock33.GetGenesisKey()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	acc := mock33.GetAccount(block.StateHash, cfg.Consensus.Genesis)
	assert.Equal(t, acc.Balance, 100000000*types.Coin)
	var txs []*types.Transaction
	addr2, priv2 := util.Genaddress()
	addr3, priv3 := util.Genaddress()
	addr4, _ := util.Genaddress()
	txs = append(txs, util.CreateCoinsTx(genkey, addr2, types.Coin))
	txs = append(txs, util.CreateCoinsTx(priv2, addr3, types.Coin))
	txs = append(txs, util.CreateCoinsTx(priv3, addr4, types.Coin))
	//执行三笔交易: 全部正确
	txgroup, err := types.CreateTxGroup(txs)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, priv3)
	//返回新的区块
	block, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), types.ExecOk)
	assert.Nil(t, err)
	assert.Equal(t, mock33.GetAccount(block.StateHash, mock33.GetGenesisAddress()).Balance, int64(9999999899700000))
	assert.Equal(t, mock33.GetAccount(block.StateHash, addr2).Balance, int64(0))
	assert.Equal(t, mock33.GetAccount(block.StateHash, addr3).Balance, int64(0))
	assert.Equal(t, mock33.GetAccount(block.StateHash, addr4).Balance, 1*types.Coin)

	//执行三笔交易：第一比错误
	txs = nil
	txs = append(txs, util.CreateCoinsTx(priv2, addr3, 2*types.Coin))
	txs = append(txs, util.CreateCoinsTx(genkey, addr4, types.Coin))
	txs = append(txs, util.CreateCoinsTx(genkey, addr4, types.Coin))

	txgroup, err = types.CreateTxGroup(txs)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, priv2)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, genkey)

	_, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), types.ExecErr)
	assert.Nil(t, err)
	//执行三笔交易：第二比错误
	txs = nil
	txs = append(txs, util.CreateCoinsTx(genkey, addr2, types.Coin))
	txs = append(txs, util.CreateCoinsTx(priv2, addr4, 2*types.Coin))
	txs = append(txs, util.CreateCoinsTx(genkey, addr4, types.Coin))

	txgroup, err = types.CreateTxGroup(txs)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, genkey)

	_, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), types.ExecPack)
	assert.Nil(t, err)
	//执行三笔交易: 第三比错误
	txs = nil
	txs = append(txs, util.CreateCoinsTx(genkey, addr2, types.Coin))
	txs = append(txs, util.CreateCoinsTx(genkey, addr4, types.Coin))
	txs = append(txs, util.CreateCoinsTx(priv2, addr4, 10*types.Coin))

	txgroup, err = types.CreateTxGroup(txs)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, priv2)

	_, err = util.ExecAndCheckBlock(mock33.GetClient(), block, txgroup.GetTxs(), types.ExecPack)
	assert.Nil(t, err)
	//执行三笔交易：其中有一笔是user.xxx的执行器
	txs = nil
	txs = append(txs, util.CreateCoinsTx(genkey, addr2, types.Coin))
	txs = append(txs, util.CreateCoinsTx(genkey, addr4, types.Coin))
	txs = append(txs, util.CreateCoinsTx(priv2, addr4, 10*types.Coin))
	txs[2].Execer = []byte("user.xxx")
	txs[2].To = address.ExecAddress("user.xxx")
	txgroup, err = types.CreateTxGroup(txs)
	assert.Nil(t, err)
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, priv2)

	_, err = util.ExecAndCheckBlock2(mock33.GetClient(), block, txgroup.GetTxs(), []int{2, 2, 1})
	assert.Nil(t, err)
}

func TestExecAllow(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	prev := types.GInt("MinFee")
	types.SetMinFee(100000)
	defer types.SetMinFee(prev)
	genkey := mock33.GetGenesisKey()
	genaddr := mock33.GetGenesisAddress()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	tx1 := util.CreateTxWithExecer(genkey, "user.evm")       //allow
	tx2 := util.CreateTxWithExecer(genkey, "coins")          //allow
	tx3 := util.CreateTxWithExecer(genkey, "evmxx")          //not allow
	tx4 := util.CreateTxWithExecer(genkey, "user.evmxx.xxx") //allow
	assert.Equal(t, mock33.GetAccount(block.StateHash, genaddr).Balance, 100000000*types.Coin)
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
	genkey := mock33.GetGenesisKey()
	genaddr := mock33.GetGenesisAddress()
	mock33.WaitHeight(0)
	block := mock33.GetBlock(0)
	assert.Equal(t, mock33.GetAccount(block.StateHash, genaddr).Balance, 100000000*types.Coin)
	txs := util.GenCoinsTxs(genkey, 2)

	block2 := util.CreateNewBlock(block, txs)
	detail, _, err := util.ExecBlock(mock33.GetClient(), block.StateHash, block2, false, true)
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
			txs := util.GenCoinsTxs(genkey, 2)
			block3 := util.CreateNewBlock(block2, txs)
			detail, _, err := util.ExecBlock(mock33.GetClient(), block2.StateHash, block3, false, true)
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

func TestExecBlock(t *testing.T) {
	mock33 := newMockNode()
	defer mock33.Close()
	block := util.CreateNoneBlock(mock33.GetGenesisKey(), 10)
	util.ExecBlock(mock33.GetClient(), nil, block, false, true)
}

//区块执行性能更好的一个测试
//1. 先生成 10万个账户，每个账户转1000个币
//2. 每个区块随机取1万比交易，然后执行
//3. 执行1000个块，看性能曲线的变化
//4. 排除掉网络掉影响
//5. 先对leveldb 做一个性能的测试

//区块执行新能测试
func BenchmarkExecBlock(b *testing.B) {
	mock33 := newMockNode()
	defer mock33.Close()
	block := util.CreateCoinsBlock(mock33.GetGenesisKey(), 10000)
	mock33.WaitHeight(0)
	block0 := mock33.GetBlock(0)
	account := mock33.GetAccount(block0.StateHash, mock33.GetGenesisAddress())
	assert.Equal(b, int64(10000000000000000), account.Balance)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		util.ExecBlock(mock33.GetClient(), block0.StateHash, block, false, true)
	}
}
