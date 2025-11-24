// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/require"
)

func addTestBlock(t *testing.T, mock33 *testnode.Chain33Mock, blockNum int, isTxHeight bool) []*types.Transaction {

	var txs []*types.Transaction
	for i := 0; i < blockNum; i++ {
		var txList []*types.Transaction
		var err error
		currHeight := mock33.GetBlockChain().GetBlockHeight()
		if isTxHeight {
			txList, _, err = addTxTxHeight(mock33.GetAPI().GetConfig(), mock33.GetGenesisKey(), mock33.GetAPI(), currHeight)
		} else {
			txList, _, err = addTx(mock33.GetAPI().GetConfig(), mock33.GetGenesisKey(), mock33.GetAPI())
		}
		txHeight := types.GetTxHeight(mock33.GetAPI().GetConfig(), txList[0].Expire, currHeight+1)
		require.Nilf(t, err, "currHeight:%d, txHeight=%d", currHeight, txHeight)
		txs = append(txs, txList...)
		for mock33.GetBlockChain().GetBlockHeight() != currHeight+1 {
			time.Sleep(time.Millisecond)
		}
	}
	return txs
}

// 构造10个区块，10笔交易不带TxHeight，缓存size128
func TestCheckDupTxHashList01(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	chainlog.Debug("TestCheckDupTxHashList01 begin --------------------")

	blockchain := mock33.GetBlockChain()
	txs := addTestBlock(t, mock33, 10, false)

	//重复交易
	duptxhashlist, err := checkDupTx(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), len(txs))
	//非重复交易
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeight(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Debug("TestCheckDupTxHashList01 end --------------------")
}

// 构造10个区块，10笔交易带TxHeight，缓存size128
func TestCheckDupTxHashList02(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	chainlog.Debug("TestCheckDupTxHashList02 begin --------------------")
	blockchain := mock33.GetBlockChain()
	txs := addTestBlock(t, mock33, 10, true)
	//重复交易
	duptxhashlist, err := checkDupTxHeight(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), len(txs))

	//非重复交易
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)

	txList := util.GenTxsTxHeight(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txList...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Debug("TestCheckDupTxHashList02 end --------------------")
}

// 构造130个区块，130笔交易不带TxHeight，缓存满
func TestCheckDupTxHashList03(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	chainlog.Debug("TestCheckDupTxHashList03 begin --------------------")
	blockchain := mock33.GetBlockChain()
	txs := addTestBlock(t, mock33, 130, false)

	//重复交易,不带TxHeight，cache没有会检查db
	duptxhashlist, err := checkDupTx(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), len(txs))

	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeight(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Debug("TestCheckDupTxHashList03 end --------------------")
}

// 构造130个区块，130笔交易带TxHeight，缓存满
func TestCheckDupTxHashList04(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	chainlog.Debug("TestCheckDupTxHashList04 begin --------------------")
	blockchain := mock33.GetBlockChain()

	txs := addTestBlock(t, mock33, 130, true)
	time.Sleep(time.Second)
	duptxhashlist, err := checkDupTx(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), len(txs))

	//非重复交易
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeight(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	require.Nil(t, err)
	require.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Debug("TestCheckDupTxHashList04 end --------------------")
}

// 异常：构造10个区块，10笔交易带TxHeight，TxHeight不满足条件 size128
func TestCheckDupTxHashList05(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	chainlog.Debug("TestCheckDupTxHashList05 begin --------------------")
	//发送带TxHeight交易且TxHeight不满足条件
	for i := 2; i < 10; i++ {
		_, _, err := addTxTxHeight(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), int64(i)+types.LowAllowPackHeight)
		require.EqualErrorf(t, err, types.ErrTxExpire.Error(), "index-%d", i)
		time.Sleep(sendTxWait)
	}
	chainlog.Debug("TestCheckDupTxHashList05 end --------------------")
}
