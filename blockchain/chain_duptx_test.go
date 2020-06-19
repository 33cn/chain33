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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//构造10个区块，10笔交易不带TxHeight，缓存size128
func TestCheckDupTxHashList01(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList01 begin --------------------")

	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	var txs []*types.Transaction
	for {
		txlist, _, err := addTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)
		txs = append(txs, txlist...)
		curheight := blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList01", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	//重复交易
	duptxhashlist, err := checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))
	//非重复交易
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Debug("TestCheckDupTxHashList01 end --------------------")
}

//构造10个区块，10笔交易带TxHeight，缓存size128
func TestCheckDupTxHashList02(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList02 begin --------------------")
	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	var txs []*types.Transaction
	for {
		txlist, _, err := addTxTxHeigt(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), curheight)
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList02", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	//重复交易
	duptxhashlist, err := checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))

	//非重复交易
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Debug("TestCheckDupTxHashList02 end --------------------")
}

//构造130个区块，130笔交易不带TxHeight，缓存满
func TestCheckDupTxHashList03(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList03 begin --------------------")
	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 130
	var txs []*types.Transaction
	for {
		txlist, _, err := addTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI())
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList03", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	//重复交易,不带TxHeight，cache没有会检查db
	duptxhashlist, err := checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))

	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Debug("TestCheckDupTxHashList03 end --------------------")
}

//构造130个区块，130笔交易带TxHeight，缓存满
func TestCheckDupTxHashList04(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList04 begin --------------------")
	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 130
	curheightForExpire := curheight
	var txs []*types.Transaction
	for {
		txlist, _, err := addTxTxHeigt(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), curheightForExpire)
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheightForExpire = blockchain.GetBlockHeight()
		chainlog.Debug("testCheckDupTxHashList04", "curheight", curheightForExpire, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheightForExpire)
		require.NoError(t, err)
		if curheightForExpire >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(time.Second)
	duptxhashlist, err := checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), len(txs))

	//非重复交易
	txs = util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(cfg, mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Debug("TestCheckDupTxHashList04 end --------------------")
}

//异常：构造10个区块，10笔交易带TxHeight，TxHeight不满足条件 size128
func TestCheckDupTxHashList05(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	cfg := mock33.GetClient().GetConfig()
	cfg.S("TxHeight", true)
	chainlog.Debug("TestCheckDupTxHashList05 begin --------------------")
	TxHeightOffset = 60
	//发送带TxHeight交易且TxHeight不满足条件
	for i := 1; i < 10; i++ {
		_, _, err := addTxTxHeigt(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), int64(i))
		require.EqualError(t, err, "ErrTxExpire")
		time.Sleep(sendTxWait)
	}
	chainlog.Debug("TestCheckDupTxHashList05 end --------------------")
}
