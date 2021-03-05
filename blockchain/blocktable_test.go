// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"
	"time"

	"github.com/33cn/chain33/blockchain"
	"github.com/33cn/chain33/common/merkle"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockTable(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	cfg.GetModuleConfig().BlockChain.RollbackBlock = 5
	mock33 := testnode.NewWithConfig(cfg, nil)
	defer mock33.Close()
	cfg = mock33.GetClient().GetConfig()
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
		_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.none")
		require.NoError(t, err)

		_, _, err = addGroupParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.", false)
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
	testBlockTable(cfg, t, blockchain)
	//del测试
	blockchain.Rollback()
	testBlockTable(cfg, t, blockchain)
}

func testBlockTable(cfg *types.Chain33Config, t *testing.T, blockchain *blockchain.BlockChain) {
	curheight := blockchain.GetBlockHeight()

	//通过当前高度获取header
	header, err := blockchain.GetStore().GetBlockHeaderByHeight(curheight)
	require.NoError(t, err)

	//通过当前高度获取block
	block, err := blockchain.GetStore().LoadBlock(curheight, nil)
	require.NoError(t, err)

	assert.Equal(t, header.GetHash(), block.Block.Hash(cfg))
	assert.Equal(t, header.GetHash(), block.Block.MainHash)
	assert.Equal(t, curheight, block.Block.MainHeight)

	//通过当前高度+hash获取block
	block1, err := blockchain.GetStore().LoadBlockByHash(header.GetHash())
	require.NoError(t, err)
	assert.Equal(t, header.GetHash(), block1.Block.Hash(cfg))
	assert.Equal(t, header.GetHash(), block1.Block.MainHash)
	assert.Equal(t, curheight, block1.Block.MainHeight)

	//获取当前高度上的所有平行链title
	replyparaTxs, err := blockchain.LoadParaTxByHeight(curheight, "", 0, 0)
	require.NoError(t, err)
	for _, paratx := range replyparaTxs.Items {
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
	paraTxs, err := blockchain.LoadParaTxByTitle(&req)
	require.NoError(t, err)
	assert.Equal(t, paraTxs.Title, "user.p.hyb.")

	//获取平行链title对应的区块高度,向前翻
	startheight := curheight - 2
	req.Height = startheight
	paraTxs, err = blockchain.LoadParaTxByTitle(&req)
	require.NoError(t, err)
	count := 0
	var req1 types.ReqParaTxByHeight
	req1.Title = req.Title
	assert.Equal(t, paraTxs.Title, "user.p.hyb.")

	for _, paratx := range paraTxs.Items {
		count++
		req1.Items = append(req1.Items, paratx.Height)
	}
	assert.Equal(t, int64(count), startheight-1)

	t.Log(paraTxs)
	t.Log(req1)
	//通过title+heightList获取对应平行链的交易信息
	paraChainTxs, err := blockchain.GetParaTxByHeight(&req1)
	require.NoError(t, err)
	for index, pChainTx := range paraChainTxs.Items {
		assert.Equal(t, pChainTx.Header.Height, paraTxs.Items[index].Height)
		assert.Equal(t, pChainTx.Header.Hash, paraTxs.Items[index].Hash)
		blockheight := pChainTx.Header.Height
		//子roothash的proof证明验证
		var hashes [][]byte
		for _, tx := range pChainTx.GetTxDetails() {
			if cfg.IsFork(blockheight, "ForkRootHash") {
				hashes = append(hashes, tx.GetTx().FullHash())
			} else {
				hashes = append(hashes, tx.GetTx().Hash())
			}
		}
		childHash := merkle.GetMerkleRoot(hashes)
		root := merkle.GetMerkleRootFromBranch(pChainTx.GetProofs(), childHash, pChainTx.Index)
		assert.Equal(t, childHash, root)
		assert.Equal(t, childHash, pChainTx.ChildHash)
		count--
	}
	assert.Equal(t, count, 0)

	//获取平行链title对应的区块高度，向后翻
	req.Direction = 1
	paraTxs, err = blockchain.LoadParaTxByTitle(&req)
	require.NoError(t, err)
	assert.Equal(t, paraTxs.Title, "user.p.hyb.")
	count = len(paraTxs.Items)

	if count < 2 {
		t.Error("testBlockTable:Title:fail!")
	}
	//异常测试
	_, err = blockchain.LoadParaTxByTitle(nil)
	assert.Equal(t, types.ErrInvalidParam, err)

	req.Count = 100000
	_, err = blockchain.LoadParaTxByTitle(nil)
	assert.Equal(t, types.ErrInvalidParam, err)

	req.Count = 0
	req.Direction = 3
	_, err = blockchain.LoadParaTxByTitle(nil)
	assert.Equal(t, types.ErrInvalidParam, err)

	req.Count = 0
	req.Direction = 0
	req.Title = ""
	_, err = blockchain.LoadParaTxByTitle(nil)
	assert.Equal(t, types.ErrInvalidParam, err)

	req.Count = 0
	req.Direction = 0
	req.Title = "user.write"
	_, err = blockchain.LoadParaTxByTitle(nil)
	assert.Equal(t, types.ErrInvalidParam, err)

	//GetParaTxByHeight入参检测test
	_, err = blockchain.GetParaTxByHeight(nil)
	assert.Equal(t, types.ErrInvalidParam, err)

	var reqPara types.ReqParaTxByHeight
	reqPara.Title = "user.write"
	reqPara.Items = append(reqPara.Items, 1)
	_, err = blockchain.GetParaTxByHeight(&reqPara)
	assert.Equal(t, types.ErrInvalidParam, err)

	reqPara.Title = "user.p.hyb."
	reqPara.Items = append(reqPara.Items, 2)
	reqPara.Items[0] = -1
	_, err = blockchain.GetParaTxByHeight(&reqPara)
	assert.Equal(t, types.ErrInvalidParam, err)

	for i := 0; i < 10002; i++ {
		reqPara.Items = append(reqPara.Items, int64(i))
	}
	_, err = blockchain.GetParaTxByHeight(&reqPara)
	assert.Equal(t, types.ErrInvalidParam, err)
}
