// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	"github.com/33cn/chain33/blockchain"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	types.Init("local", nil)
}
func TestBlockTable(t *testing.T) {
	cfg, sub := testnode.GetDefaultConfig()
	cfg.BlockChain.RollbackBlock = 5
	mock33 := testnode.NewWithConfig(cfg, sub, nil)
	defer mock33.Close()
	blockchain := mock33.GetBlockChain()
	chainlog.Info("TestBlockTable begin --------------------")

	//构造十个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}

	for {
		_, err = addSingleParaTx(mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)

		_, _, err = addGroupParaTx(mock33.GetGenesisKey(), mock33.GetAPI(), false)
		require.NoError(t, err)

		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	time.Sleep(sendTxWait * 2)
	testBlockTable(t, blockchain)
	//del测试
	blockchain.Rollback()
	testBlockTable(t, blockchain)
}

func testBlockTable(t *testing.T, blockchain *blockchain.BlockChain) {
	curheight := blockchain.GetBlockHeight()

	//通过当前高度获取header
	header, err := blockchain.GetStore().GetBlockHeaderByHeight(curheight)
	require.NoError(t, err)

	//通过当前高度获取block
	block, err := blockchain.GetStore().LoadBlockByHeight(curheight)
	require.NoError(t, err)

	assert.Equal(t, header.GetHash(), block.Block.Hash())
	assert.Equal(t, header.GetHash(), block.Block.MainHash)
	assert.Equal(t, curheight, block.Block.MainHeight)

	//通过当前高度+hash获取block
	block1, err := blockchain.GetStore().LoadBlockByHash(header.GetHash())
	require.NoError(t, err)
	assert.Equal(t, header.GetHash(), block1.Block.Hash())
	assert.Equal(t, header.GetHash(), block1.Block.MainHash)
	assert.Equal(t, curheight, block1.Block.MainHeight)

	//获取当前高度上的所有平行链title
	paraTxs, err := blockchain.LoadParaTxByHeight(curheight, "", 0, 0)
	require.NoError(t, err)
	for _, paratx := range paraTxs.Items {
		assert.Equal(t, paratx.Height, curheight)
		assert.Equal(t, paratx.Hash, header.GetHash())
		if paratx.Title != "user.p.hyb." && paratx.Title != "user.p.fzm." {
			t.Error("testBlockTable:Title:fail!")
		}
	}

	var req types.ReqHeightByTitle
	req.Height = -1
	req.Title = "user.p.hyb."
	req.Count = 0
	req.Direction = 0

	//获取平行链title="user.p.hyb."对应的区块高度
	paraTxs, err = blockchain.LoadParaTxByTitle(&req)
	require.NoError(t, err)
	for _, paratx := range paraTxs.Items {
		assert.Equal(t, paratx.Title, "user.p.hyb.")
		chainlog.Error("TestBlockTable:LoadParaTxByTitle", "paratx", paratx)
	}

	//获取平行链title对应的区块高度,向前翻
	startheight := curheight - 2
	req.Height = startheight
	paraTxs, err = blockchain.LoadParaTxByTitle(&req)
	require.NoError(t, err)
	count := 0
	var req1 types.ReqParaTxByHeight
	req1.Title = req.Title
	for _, paratx := range paraTxs.Items {
		assert.Equal(t, paratx.Title, "user.p.hyb.")
		count++
		req1.Items = append(req1.Items, paratx.Height)
	}
	assert.Equal(t, int64(count), startheight-1)

	//通过title+heightList获取对应平行链的交易信息
	paraChainTxs, err := blockchain.GetParaTxByHeight(&req1)
	require.NoError(t, err)
	for index, pChainTx := range paraChainTxs.Items {
		assert.Equal(t, pChainTx.Header.Height, paraTxs.Items[index].Height)
		assert.Equal(t, pChainTx.Header.Hash, paraTxs.Items[index].Hash)
		count--
	}
	assert.Equal(t, count, 0)

	//获取平行链title对应的区块高度，向后翻
	count = 0
	req.Direction = 1
	paraTxs, err = blockchain.LoadParaTxByTitle(&req)
	require.NoError(t, err)
	for _, paratx := range paraTxs.Items {
		assert.Equal(t, paratx.Title, "user.p.hyb.")
		count++
	}
	if count < 2 {
		t.Error("testBlockTable:Title:fail!")
	}

}
