// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/33cn/chain33/blockchain"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/merkle"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	types.Init("local", nil)
}

var TxHeightOffset int64 = 0
var sendTxWait = time.Millisecond * 5
var chainlog = log15.New("module", "chain_test")

func addTx(priv crypto.PrivKey, api client.QueueProtocolAPI) ([]*types.Transaction, string, error) {
	txs := util.GenCoinsTxs(priv, 1)
	hash := common.ToHex(txs[0].Hash())
	reply, err := api.SendTx(txs[0])
	if err != nil {
		return nil, hash, err
	}
	if !reply.GetIsOk() {
		return nil, hash, errors.New("sendtx unknow error")
	}
	return txs, hash, nil
}

func addTxTxHeigt(priv crypto.PrivKey, api client.QueueProtocolAPI, height int64) ([]*types.Transaction, string, error) {
	txs := util.GenTxsTxHeigt(priv, 1, height+TxHeightOffset)
	hash := common.ToHex(txs[0].Hash())
	reply, err := api.SendTx(txs[0])
	if err != nil {
		return nil, hash, err
	}
	if !reply.GetIsOk() {
		return nil, hash, errors.New("sendtx unknow error")
	}
	return txs, hash, nil
}

func TestBlockChain(t *testing.T) {
	log.SetLogLevel("crit")
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	blockchain := mock33.GetBlockChain()
	//等待共识模块增长10个区块
	testProcAddBlockMsg(t, mock33, blockchain)

	curBlock := testGetBlock(t, blockchain)

	testGetTx(t, blockchain)

	testGetTxHashList(t, blockchain)

	testProcQueryTxMsg(t, blockchain)

	testGetBlocksMsg(t, blockchain)

	testProcGetHeadersMsg(t, blockchain)

	testProcGetLastHeaderMsg(t, blockchain)

	testGetBlockByHash(t, blockchain)

	testProcGetLastSequence(t, blockchain)

	testGetBlockSequences(t, blockchain)

	testGetBlockByHashes(t, blockchain)
	testGetSeqByHash(t, blockchain)
	testPrefixCount(t, mock33, blockchain)
	testAddrTxCount(t, mock33, blockchain)

	// QM add
	testGetBlockHerderByHash(t, blockchain)

	testProcGetTransactionByHashes(t, blockchain)

	textProcGetBlockOverview(t, blockchain)

	testProcGetAddrOverview(t, blockchain)

	testProcGetBlockHash(t, blockchain)

	//	testSendDelBlockEvent(t, blockchain)

	testGetOrphanRoot(t, blockchain)

	testRemoveOrphanBlock(t, blockchain)

	testLoadBlockBySequence(t, blockchain)
	testAddBlockSeqCB(t, blockchain)
	testProcDelParaChainBlockMsg(t, mock33, blockchain)

	testProcAddParaChainBlockMsg(t, mock33, blockchain)
	testProcGetBlockBySeqMsg(t, mock33, blockchain)
	testProcBlockChainFork(t, blockchain)
	testDelBlock(t, blockchain, curBlock)
	testIsRecordFaultErr(t)

	testGetsynBlkHeight(t, blockchain)
	testProcDelChainBlockMsg(t, mock33, blockchain)
	testFaultPeer(t, blockchain)
	testCheckBlock(t, blockchain)
	testWriteBlockToDbTemp(t, blockchain)
	testReadBlockToExec(t, blockchain)
	testReExecBlock(t, blockchain)
	testUpgradeStore(t, blockchain)
}

func testProcAddBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcAddBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}

	for {
		_, _, err = addTx(mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)
		curheight = blockchain.GetBlockHeight()
		chainlog.Info("testProcAddBlockMsg ", "curheight", curheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	chainlog.Info("testProcAddBlockMsg end --------------------")
}

func testGetBlock(t *testing.T, blockchain *blockchain.BlockChain) *types.Block {
	chainlog.Info("testGetBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)
	if curheight != block.Block.Height {
		t.Error("get block height error")
	}
	chainlog.Info("testGetBlock end --------------------")
	return block.Block

}

func testGetTx(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestGetTx begin --------------------")
	//构建txhash
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	chainlog.Info("testGetTx :", "curheight", curheight)
	txResult, err := blockchain.GetTxResultFromDb(block.Block.Txs[0].Hash())
	require.NoError(t, err)

	if err == nil && txResult != nil {
		Execer := string(txResult.GetTx().Execer)
		if "coins" != Execer {
			t.Error("ExecerName error")
		}
	}
	chainlog.Info("TestGetTx end --------------------")
}

func testGetTxHashList(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestGetTxHashList begin --------------------")
	var txhashlist types.TxHashList
	total := 10
	Txs := make([]*types.Transaction, total)

	// 构建当前高度的tx信息
	i := blockchain.GetBlockHeight()
	for j := 0; j < total; j++ {
		var transaction types.Transaction
		payload := fmt.Sprintf("Payload :%d:%d!", i, j)
		signature := fmt.Sprintf("Signature :%d:%d!", i, j)

		transaction.Payload = []byte(payload)
		var signature1 types.Signature
		signature1.Signature = []byte(signature)
		transaction.Signature = &signature1

		Txs[j] = &transaction
		txhash := Txs[j].Hash()
		//chainlog.Info("testGetTxHashList", "height", i, "count", j, "txhash", txhash)
		txhashlist.Hashes = append(txhashlist.Hashes, txhash[:])
	}
	duptxhashlist, err := blockchain.GetDuplicateTxHashList(&txhashlist)
	if err != nil {
		t.Error(err)
		return
	}
	if duptxhashlist != nil {
		for _, duptxhash := range duptxhashlist.Hashes {
			if duptxhash != nil {
				chainlog.Info("testGetTxHashList", "duptxhash", duptxhash)
			}
		}
	}
	chainlog.Info("TestGetTxHashList end --------------------")
}

func checkDupTx(cacheTxs []*types.Transaction, blockchain *blockchain.BlockChain) (*types.TxHashList, error) {
	var txhashlist types.TxHashList
	i := blockchain.GetBlockHeight()
	for j, tx := range cacheTxs {
		txhash := tx.Hash()
		chainlog.Info("checkDupTx", "height", i, "count", j, "txhash", txhash)
		txhashlist.Hashes = append(txhashlist.Hashes, txhash[:])
	}
	//count 现在是高度，当前的高度
	txhashlist.Count = i
	duptxhashlist, err := blockchain.GetDuplicateTxHashList(&txhashlist)
	if err != nil {
		return nil, err
	}

	return duptxhashlist, nil
}

func checkDupTxHeight(cacheTxsTxHeigt []*types.Transaction, blockchain *blockchain.BlockChain) (*types.TxHashList, error) {
	var txhashlist types.TxHashList
	i := blockchain.GetBlockHeight()
	for j, tx := range cacheTxsTxHeigt {
		txhash := tx.Hash()
		chainlog.Info("checkDupTxHeight", "height", i, "count", j, "txhash", txhash)
		txhashlist.Hashes = append(txhashlist.Hashes, txhash[:])
		txhashlist.Expire = append(txhashlist.Expire, tx.Expire)
	}
	txhashlist.Count = i
	duptxhashlist, err := blockchain.GetDuplicateTxHashList(&txhashlist)
	if err != nil {
		return nil, err
	}
	return duptxhashlist, nil
}

//构造10个区块，10笔交易不带TxHeight，缓存size128
func TestCheckDupTxHashList01(t *testing.T) {
	types.S("TxHeight", true)
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
		types.S("TxHeight", false)
	}()
	chainlog.Info("TestCheckDupTxHashList01 begin --------------------")

	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	var txs []*types.Transaction
	for {
		txlist, _, err := addTx(mock33.GetGenesisKey(), mock33.GetAPI())
		require.NoError(t, err)
		txs = append(txs, txlist...)
		curheight := blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList01", "curheight", curheight, "addblockheight", addblockheight)
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
	txs = util.GenCoinsTxs(mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Info("TestCheckDupTxHashList01 end --------------------")
}

//构造10个区块，10笔交易带TxHeight，缓存size128
func TestCheckDupTxHashList02(t *testing.T) {
	types.S("TxHeight", true)
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
		types.S("TxHeight", false)
	}()
	chainlog.Info("TestCheckDupTxHashList02 begin --------------------")
	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	var txs []*types.Transaction
	for {
		txlist, _, err := addTxTxHeigt(mock33.GetGenesisKey(), mock33.GetAPI(), int64(curheight))
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList02", "curheight", curheight, "addblockheight", addblockheight)
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
	txs = util.GenCoinsTxs(mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Info("TestCheckDupTxHashList02 end --------------------")
}

//构造130个区块，130笔交易不带TxHeight，缓存满
func TestCheckDupTxHashList03(t *testing.T) {
	types.S("TxHeight", true)
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
		types.S("TxHeight", false)
	}()
	chainlog.Info("TestCheckDupTxHashList03 begin --------------------")
	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 130
	var txs []*types.Transaction
	for {
		txlist, _, err := addTx(mock33.GetGenesisKey(), mock33.GetAPI())
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList03", "curheight", curheight, "addblockheight", addblockheight)
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

	txs = util.GenCoinsTxs(mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)
	chainlog.Info("TestCheckDupTxHashList03 end --------------------")
}

//构造130个区块，130笔交易带TxHeight，缓存满
func TestCheckDupTxHashList04(t *testing.T) {
	types.S("TxHeight", true)
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
		types.S("TxHeight", false)
	}()
	chainlog.Info("TestCheckDupTxHashList04 begin --------------------")
	blockchain := mock33.GetBlockChain()
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 130
	curheightForExpire := curheight
	var txs []*types.Transaction
	for {
		txlist, _, err := addTxTxHeigt(mock33.GetGenesisKey(), mock33.GetAPI(), int64(curheightForExpire))
		txs = append(txs, txlist...)
		require.NoError(t, err)
		curheightForExpire = blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList04", "curheight", curheightForExpire, "addblockheight", addblockheight)
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
	txs = util.GenCoinsTxs(mock33.GetGenesisKey(), 50)
	duptxhashlist, err = checkDupTx(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	txlist := util.GenTxsTxHeigt(mock33.GetGenesisKey(), 50, 10)
	txs = append(txs, txlist...)
	duptxhashlist, err = checkDupTxHeight(txs, blockchain)
	assert.Nil(t, err)
	assert.Equal(t, len(duptxhashlist.Hashes), 0)

	chainlog.Info("TestCheckDupTxHashList04 end --------------------")
}

//异常：构造10个区块，10笔交易带TxHeight，TxHeight不满足条件 size128
func TestCheckDupTxHashList05(t *testing.T) {
	types.S("TxHeight", true)
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
		types.S("TxHeight", false)
	}()
	chainlog.Info("TestCheckDupTxHashList05 begin --------------------")
	TxHeightOffset = 60
	//发送带TxHeight交易且TxHeight不满足条件
	for i := 1; i < 10; i++ {
		_, _, err := addTxTxHeigt(mock33.GetGenesisKey(), mock33.GetAPI(), int64(i))
		require.EqualError(t, err, "ErrTxExpire")
		time.Sleep(sendTxWait)
	}
	chainlog.Info("TestCheckDupTxHashList05 end --------------------")
}

func testProcQueryTxMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestProcQueryTxMsg begin --------------------")
	curheight := blockchain.GetBlockHeight()
	var merkleroothash []byte
	var txhash []byte
	var txindex int

	//获取当前高度的block信息
	block, err := blockchain.GetBlock(curheight)

	if err == nil {
		merkleroothash = block.Block.TxHash
		for index, transaction := range block.Block.Txs {
			txhash = transaction.Hash()
			txindex = index
		}
	}
	txproof, err := blockchain.ProcQueryTxMsg(txhash)
	require.NoError(t, err)

	//证明txproof的正确性
	brroothash := merkle.GetMerkleRootFromBranch(txproof.GetProofs(), txhash, uint32(txindex))
	if !bytes.Equal(merkleroothash, brroothash) {
		t.Error("txproof roothash error")
	}

	chainlog.Info("TestProcQueryTxMsg end --------------------")
}

func testGetBlocksMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestGetBlocksMsg begin --------------------")
	curheight := blockchain.GetBlockHeight()
	var reqBlock types.ReqBlocks
	if curheight >= 5 {
		reqBlock.Start = curheight - 5
	}
	reqBlock.End = curheight
	reqBlock.IsDetail = true
	checkheight := reqBlock.Start
	blocks, err := blockchain.ProcGetBlockDetailsMsg(&reqBlock)
	if err == nil && blocks != nil {
		for _, block := range blocks.Items {
			if checkheight != block.Block.Height {
				t.Error("TestGetBlocksMsg", "checkheight", checkheight, "block", block)
			}
			checkheight++
		}
	}
	chainlog.Info("TestGetBlocksMsg end --------------------")
}

func testProcGetHeadersMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestProcGetHeadersMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	var reqBlock types.ReqBlocks
	if curheight >= 5 {
		reqBlock.Start = curheight - 5
	}
	reqBlock.End = curheight
	checkheight := reqBlock.Start
	blockheaders, err := blockchain.ProcGetHeadersMsg(&reqBlock)
	if err == nil && blockheaders != nil {
		for _, head := range blockheaders.Items {
			if checkheight != head.Height {
				t.Error("testProcGetHeadersMsg Block header  check error")
			}
			checkheight++
		}
	}
	chainlog.Info("TestProcGetHeadersMsg end --------------------")
}

func testProcGetLastHeaderMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestProcGetLastHeaderMsg begin --------------------")
	curheight := blockchain.GetBlockHeight()
	blockheader, err := blockchain.ProcGetLastHeaderMsg()
	if err == nil && blockheader != nil {
		if curheight != blockheader.Height {
			t.Error("testProcGetLastHeaderMsg Last Header  check error")
		}
	}
	chainlog.Info("TestProcGetLastHeaderMsg end --------------------")
}

func testGetBlockByHash(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestGetBlockByHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block1, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash1 := block1.Block.Hash()
	block2, err := blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash1, block2.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash1, block2.Block.ParentHash)
	}
	block3, err := blockchain.ProcGetBlockByHashMsg(block2.Block.Hash())
	require.NoError(t, err)
	if !bytes.Equal(block2.Block.Hash(), block3.Block.Hash()) {
		t.Error("testGetBlockByHash Block Hash check error")
	}
	chainlog.Info("TestGetBlockByHash end --------------------")
}

func testProcGetLastSequence(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcGetLastSequence begin --------------------")

	curheight := blockchain.GetBlockHeight()

	lastSequence, err := blockchain.GetStore().LoadBlockLastSequence()
	require.NoError(t, err)
	if curheight != lastSequence {
		t.Error("testProcGetLastSequence Last Sequence check error")
	}
	chainlog.Info("testProcGetLastSequence end --------------------")
}

func testGetBlockSequences(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("testGetBlockSequences begin --------------------")
	lastSequence, _ := chain.GetStore().LoadBlockLastSequence()
	var reqBlock types.ReqBlocks
	if lastSequence >= 5 {
		reqBlock.Start = lastSequence - 5
	}
	reqBlock.End = lastSequence
	reqBlock.IsDetail = true
	Sequences, err := chain.GetBlockSequences(&reqBlock)
	if err == nil && Sequences != nil {
		for _, sequence := range Sequences.Items {
			if sequence.Type != blockchain.AddBlock {
				t.Error("testGetBlockSequences sequence type check error")
			}
		}
	}
	chainlog.Info("testGetBlockSequences end --------------------")
}

func testGetBlockByHashes(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testGetBlockByHashes begin --------------------")
	lastSequence, _ := blockchain.GetStore().LoadBlockLastSequence()
	var reqBlock types.ReqBlocks
	if lastSequence >= 5 {
		reqBlock.Start = lastSequence - 5
	}
	reqBlock.End = lastSequence
	reqBlock.IsDetail = true
	hashes := make([][]byte, 6)
	Sequences, err := blockchain.GetBlockSequences(&reqBlock)
	if err == nil && Sequences != nil {
		for index, sequence := range Sequences.Items {
			hashes[index] = sequence.Hash
		}
	}

	blocks, err := blockchain.GetBlockByHashes(hashes)
	if err == nil && blocks != nil {
		for index, block := range blocks.Items {
			if !bytes.Equal(hashes[index], block.Block.Hash()) {
				t.Error("testGetBlockByHashes block hash check error")
			}
		}
	}
	chainlog.Info("testGetBlockByHashes end --------------------")
}

func testGetSeqByHash(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testGetSeqByHash begin --------------------")
	lastSequence, _ := blockchain.GetStore().LoadBlockLastSequence()
	var reqBlock types.ReqBlocks

	reqBlock.Start = lastSequence
	reqBlock.End = lastSequence
	reqBlock.IsDetail = true
	hashes := make([][]byte, 1)
	Sequences, err := blockchain.GetBlockSequences(&reqBlock)

	if err == nil && Sequences != nil {
		for index, sequence := range Sequences.Items {
			hashes[index] = sequence.Hash
		}
	}

	seq, _ := blockchain.ProcGetSeqByHash(hashes[0])
	if seq == -1 {
		t.Error(" GetSeqByHash err")
	}

	chainlog.Info("testGetSeqByHash end --------------------")
}

func testPrefixCount(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testPrefixCount begin --------------------")

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventLocalPrefixCount, &types.ReqKey{Key: []byte("TxAddrHash:14KEKbYtKKQm4wMthSK9J4La4nAiidGozt:")})
	mock33.GetClient().Send(msgGen, true)
	Res, _ := mock33.GetClient().Wait(msgGen)
	count := Res.GetData().(*types.Int64).Data
	if count == 0 {
		t.Error("testPrefixCount count check error ")
	}
	chainlog.Info("testPrefixCount end --------------------")
}

func testAddrTxCount(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testAddrTxCount begin --------------------")
	var reqkey types.ReqKey
	reqkey.Key = []byte(fmt.Sprintf("AddrTxsCount:%s", "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"))
	count, err := mock33.GetAPI().Query(types.ExecName("coins"), "GetAddrTxsCount", &reqkey)
	if err != nil {
		t.Error(err)
		return
	}
	if count.(*types.Int64).GetData() == 0 {
		t.Error("testAddrTxCount count check error ")
	}
	chainlog.Info("testAddrTxCount end --------------------")
}

func testGetBlockHerderByHash(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testGetBlockHerderByHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash()
	block, err = blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}
	header, err := blockchain.GetStore().GetBlockHeaderByHash(block.Block.Hash())
	require.NoError(t, err)
	if !bytes.Equal(header.Hash, block.Block.Hash()) {
		t.Error("testGetBlockHerderByHash block header hash check error")
	}
	chainlog.Info("testGetBlockHerderByHash end --------------------")
}

func testProcGetTransactionByHashes(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("textProcGetTransactionByHashes begin --------------------")
	parm := &types.ReqAddr{
		Addr:   "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt",
		Height: -1,
	}
	txinfos, err := blockchain.ProcGetTransactionByAddr(parm)
	require.NoError(t, err)

	var Hashes [][]byte
	if txinfos != nil {
		for _, receipt := range txinfos.TxInfos {
			Hashes = append(Hashes, receipt.Hash)
		}
	}

	TxDetails, err := blockchain.ProcGetTransactionByHashes(Hashes)
	require.NoError(t, err)

	if TxDetails != nil {
		for index, tx := range TxDetails.Txs {
			if tx.Tx != nil {
				if !bytes.Equal(Hashes[index], tx.Tx.Hash()) {
					t.Error("testProcGetTransactionByHashes  hash check error")
				}
			}
		}
	}

	chainlog.Info("textProcGetTransactionByHashes end --------------------")
}

func textProcGetBlockOverview(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("textProcGetBlockOverview begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	parm := &types.ReqHash{
		Hash: block.Block.Hash(),
	}
	blockOverview, err := blockchain.ProcGetBlockOverview(parm)
	require.NoError(t, err)

	if blockOverview != nil {
		if !bytes.Equal(block.Block.Hash(), blockOverview.Head.Hash) {
			t.Error("textProcGetBlockOverview  block hash check error")
		}

	}
	chainlog.Info("textProcGetBlockOverview end --------------------")
}

func testProcGetAddrOverview(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcGetAddrOverview begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash()
	block, err = blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}

	parm := &types.ReqAddr{
		Addr: "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt",
	}
	addrOverview, err := blockchain.ProcGetAddrOverview(parm)
	require.NoError(t, err)

	if addrOverview != nil {
		if addrOverview.TxCount == 0 {
			t.Error("testProcGetAddrOverview  TxCount check error")
		}
	}
	chainlog.Info("testProcGetAddrOverview end --------------------")
}

func testProcGetBlockHash(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcGetBlockHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)
	height := &types.ReqInt{Height: curheight - 5}
	hash, err := blockchain.ProcGetBlockHash(height)
	require.NoError(t, err)

	if !bytes.Equal(block.Block.Hash(), hash.Hash) {
		t.Error("testProcGetBlockHash  block hash check error")
	}

	chainlog.Info("testProcGetBlockHash end --------------------")
}

func testGetOrphanRoot(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testGetOrphanRoot begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	hash := blockchain.GetOrphanPool().GetOrphanRoot(block.Block.Hash())
	if !bytes.Equal(block.Block.Hash(), hash) {
		t.Error("testGetOrphanRoot  Orphan Root hash check error")
	}
	chainlog.Info("testGetOrphanRoot end --------------------")
}

func testRemoveOrphanBlock(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testRemoveOrphanBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	ParentHashNotexist, _ := hex.DecodeString("8bd1f23741c90e9db4edd663acc0328a49f7c92d9974f83e2d5b57b25f3f059b")
	block.Block.ParentHash = ParentHashNotexist

	blockchain.GetOrphanPool().RemoveOrphanBlock2(block.Block, time.Time{}, false, "123", 0)

	chainlog.Info("testRemoveOrphanBlock end --------------------")
}

func testDelBlock(t *testing.T, blockchain *blockchain.BlockChain, curBlock *types.Block) {
	chainlog.Info("testDelBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)
	if curBlock == nil {
		t.Error("testDelBlock curBlock is nil")
	}

	curBlock.Difficulty = block.Block.Difficulty - 100
	newblock := types.BlockDetail{}
	newblock.Block = curBlock

	blockchain.ProcessBlock(true, &newblock, "1", true, 0)
	chainlog.Info("testDelBlock end --------------------")
}

func testLoadBlockBySequence(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testLoadBlockBySequence begin ---------------------")

	curheight := blockchain.GetBlockHeight()
	lastseq, _ := blockchain.GetStore().LoadBlockLastSequence()
	block, err := blockchain.GetStore().LoadBlockBySequence(lastseq)
	require.NoError(t, err)

	if block.Block.Height != curheight {
		t.Error("testLoadBlockBySequence", "curheight", curheight, "lastseq", lastseq, "Block.Height", block.Block.Height)
	}
	chainlog.Info("testLoadBlockBySequence end -------------------------")
}

func testProcDelParaChainBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcDelParaChainBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 1)
	require.NoError(t, err)

	var parablockDetail types.ParaChainBlockDetail
	parablockDetail.Blockdetail = block
	parablockDetail.Sequence = 1

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventDelParaChainBlockDetail, &parablockDetail)
	mock33.GetClient().Send(msgGen, true)
	resp, _ := mock33.GetClient().Wait(msgGen)
	if resp.GetData().(*types.Reply).IsOk {
		t.Error("testProcDelParaChainBlockMsg  only in parachain ")
	}
	chainlog.Info("testProcDelParaChainBlockMsg end --------------------")
}

func testProcAddParaChainBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcAddParaChainBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	var parablockDetail types.ParaChainBlockDetail
	parablockDetail.Blockdetail = block
	parablockDetail.Sequence = 1

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventAddParaChainBlockDetail, &parablockDetail)

	mock33.GetClient().Send(msgGen, true)
	_, err = mock33.GetClient().Wait(msgGen)
	if err != nil {
		t.Log(err)
		//t.Error("testProcAddParaChainBlockMsg  only in parachain ")
	}
	chainlog.Info("testProcAddParaChainBlockMsg end --------------------")
}

func testProcGetBlockBySeqMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcGetBlockBySeqMsg begin --------------------")

	seq, err := blockchain.GetStore().LoadBlockLastSequence()
	assert.Nil(t, err)

	//block, err := blockchain.GetBlock(curheight)
	//require.NoError(t, err)

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventGetBlockBySeq, &types.Int64{Data: seq})

	mock33.GetClient().Send(msgGen, true)
	msg, err := mock33.GetClient().Wait(msgGen)
	if err != nil {
		t.Log(err)
		//t.Error("testProcAddParaChainBlockMsg  only in parachain ")
	}
	blockseq := msg.Data.(*types.BlockSeq)
	assert.Equal(t, seq, blockseq.Num)
	chainlog.Info("testProcGetBlockBySeqMsg end --------------------")
}

func testProcBlockChainFork(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcBlockChainFork begin --------------------")

	curheight := blockchain.GetBlockHeight()
	blockchain.ProcDownLoadBlocks(curheight-1, curheight+256, []string{"self"})
	chainlog.Info("testProcBlockChainFork end --------------------")
}

func testAddBlockSeqCB(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("testAddBlockSeqCB begin ---------------------")

	cb := &types.BlockSeqCB{
		Name:   "test",
		URL:    "http://192.168.1.107:15760",
		Encode: "json",
	}
	blockchain.MaxSeqCB = 1
	err := chain.ProcAddBlockSeqCB(cb)
	require.NoError(t, err)

	cbs, err := chain.ProcListBlockSeqCB()
	require.NoError(t, err)
	exist := false
	for _, temcb := range cbs.Items {
		if temcb.Name == cb.Name {
			exist = true
		}
	}
	if !exist {
		t.Error("testAddBlockSeqCB  listSeqCB fail", "cb", cb, "cbs", cbs)
	}
	num := chain.ProcGetSeqCBLastNum(cb.Name)
	if num != -1 {
		t.Error("testAddBlockSeqCB  getSeqCBLastNum", "num", num, "name", cb.Name)
	}

	cb2 := &types.BlockSeqCB{
		Name:   "test1",
		URL:    "http://192.168.1.107:15760",
		Encode: "json",
	}

	err = chain.ProcAddBlockSeqCB(cb2)
	if err != types.ErrTooManySeqCB {
		t.Error("testAddBlockSeqCB", "cb", cb2, "err", err)
	}

	chainlog.Info("testAddBlockSeqCB end -------------------------")
}
func testIsRecordFaultErr(t *testing.T) {
	chainlog.Info("testIsRecordFaultErr begin ---------------------")
	isok := blockchain.IsRecordFaultErr(types.ErrFutureBlock)
	if isok {
		t.Error("testIsRecordFaultErr  IsRecordFaultErr", "isok", isok)
	}
	chainlog.Info("testIsRecordFaultErr end ---------------------")
}

func testProcDelChainBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcDelChainBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	var parablockDetail types.ParaChainBlockDetail
	parablockDetail.Blockdetail = block
	parablockDetail.Sequence = curheight

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventDelParaChainBlockDetail, &parablockDetail)
	mock33.GetClient().Send(msgGen, true)
	mock33.GetClient().Wait(msgGen)

	chainlog.Info("testProcDelChainBlockMsg end --------------------")
}
func testGetsynBlkHeight(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("testGetsynBlkHeight begin --------------------")
	curheight := chain.GetBlockHeight()
	chain.UpdatesynBlkHeight(curheight)

	height := chain.GetsynBlkHeight()
	if height != curheight {
		chainlog.Error("testGetsynBlkHeight", "curheight", curheight, "height", height)
	}
	//get peerinfo
	peerinfo := chain.GetPeerInfo("self")
	if peerinfo != nil {
		chainlog.Error("testGetsynBlkHeight:GetPeerInfo", "peerinfo", peerinfo)
	}
	maxpeer := chain.GetMaxPeerInfo()
	if maxpeer != nil {
		chainlog.Error("testGetsynBlkHeight:GetMaxPeerInfo", "maxpeer", maxpeer)
	}

	chainlog.Info("testGetsynBlkHeight end --------------------")
}

func testFaultPeer(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("testFaultPeer begin ---------------------")
	curheight := chain.GetBlockHeight()
	block, err := chain.GetBlock(curheight)
	require.NoError(t, err)
	isok := chain.IsFaultPeer("self")
	if isok {
		t.Error("testFaultPeer:IsFaultPeer")
	}
	//记录故障peer信息
	chain.RecordFaultPeer("self", curheight+1, block.Block.Hash(), types.ErrSign)

	var faultnode blockchain.FaultPeerInfo
	var peerinfo blockchain.PeerInfo
	peerinfo.Name = "self"
	faultnode.Peer = &peerinfo
	faultnode.FaultHeight = curheight + 1
	faultnode.FaultHash = block.Block.Hash()
	faultnode.ErrInfo = types.ErrSign
	faultnode.ReqFlag = false
	chain.AddFaultPeer(&faultnode)

	peer := chain.GetFaultPeer("self")
	if peer == nil {
		t.Error("testFaultPeer:GetFaultPeer is nil")
	}

	chain.UpdateFaultPeer("self", true)

	chain.RemoveFaultPeer("self")
	chainlog.Info("testFaultPeer end ---------------------")
}
func testCheckBlock(t *testing.T, chain *blockchain.BlockChain) {
	curheight := chain.GetBlockHeight()
	//chain.CheckHeightNoIncrease()
	chain.FetchBlockHeaders(0, curheight, "self")
	header, err := chain.ProcGetLastHeaderMsg()

	var blockheader types.Header
	if header != nil && err == nil {
		blockheader.Version = header.Version
		blockheader.ParentHash = header.ParentHash
		blockheader.TxHash = header.TxHash
		blockheader.StateHash = header.StateHash
		blockheader.Height = header.Height
		blockheader.BlockTime = header.BlockTime

		blockheader.TxCount = header.TxCount
		blockheader.Hash = header.Hash
		blockheader.Difficulty = header.Difficulty
		blockheader.Signature = header.Signature
	}
	var blockheaders types.Headers
	blockheaders.Items = append(blockheaders.Items, &blockheader)
	chain.ProcAddBlockHeadersMsg(&blockheaders, "self")
	chain.ProcBlockHeaders(&blockheaders, "self")
}
func testReadBlockToExec(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("testReadBlockToExec begin ---------------------")
	curheight := chain.GetBlockHeight()
	chain.ReadBlockToExec(curheight+1, false)
	chainlog.Info("testReadBlockToExec end ---------------------")
}
func testWriteBlockToDbTemp(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("WriteBlockToDbTemp begin ---------------------")
	curheight := chain.GetBlockHeight()
	block, err := chain.GetBlock(curheight)
	if err != nil {
		t.Error("testWriteBlockToDbTemp", "err", err)
	}
	var rawblock types.Block
	rawblock.Version = block.Block.Version
	rawblock.ParentHash = block.Block.ParentHash
	rawblock.TxHash = block.Block.TxHash
	rawblock.StateHash = block.Block.StateHash
	rawblock.BlockTime = block.Block.BlockTime
	rawblock.Difficulty = block.Block.Difficulty
	rawblock.MainHash = block.Block.MainHash
	rawblock.MainHeight = block.Block.MainHeight

	rawblock.Height = block.Block.Height + 1
	err = chain.WriteBlockToDbTemp(&rawblock)
	if err != nil {
		t.Error("testWriteBlockToDbTemp", "err", err)
	}
	chainlog.Info("WriteBlockToDbTemp end ---------------------")
}

func testReExecBlock(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("ReExecBlock begin ---------------------")
	curheight := chain.GetBlockHeight()
	chain.ReExecBlock(0, curheight)
	chainlog.Info("ReExecBlock end ---------------------")
}

func testUpgradeStore(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Info("UpgradeStore begin ---------------------")
	chain.UpgradeStore()
	chainlog.Info("UpgradeStore end ---------------------")
}
