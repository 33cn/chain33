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

	//	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/queue"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var TxHeightOffset int64
var sendTxWait = time.Millisecond * 5
var chainlog = log15.New("module", "chain_test")

func addTx(cfg *types.Chain33Config, priv crypto.PrivKey, api client.QueueProtocolAPI) ([]*types.Transaction, string, error) {
	txs := util.GenCoinsTxs(cfg, priv, 1)
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

func addTxTxHeigt(cfg *types.Chain33Config, priv crypto.PrivKey, api client.QueueProtocolAPI, height int64) ([]*types.Transaction, string, error) {
	txs := util.GenTxsTxHeigt(cfg, priv, 1, height+TxHeightOffset)
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
	//log.SetLogLevel("crit")
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	cfg := mock33.GetClient().GetConfig()
	blockchain := mock33.GetBlockChain()
	//等待共识模块增长10个区块
	testProcAddBlockMsg(t, mock33, blockchain)

	testGetTx(t, blockchain)

	testGetTxHashList(t, blockchain)

	testProcQueryTxMsg(cfg, t, blockchain)

	testGetBlocksMsg(t, blockchain)

	testProcGetHeadersMsg(t, blockchain)

	testProcGetLastHeaderMsg(t, blockchain)

	testGetBlockByHash(t, cfg, blockchain)

	testProcGetLastSequence(t, blockchain)

	testGetBlockSequences(t, blockchain)

	testGetBlockByHashes(t, cfg, blockchain)
	testGetSeqByHash(t, blockchain)
	testPrefixCount(t, mock33, blockchain)
	testAddrTxCount(t, mock33, blockchain)

	// QM add
	testGetBlockHerderByHash(t, cfg, blockchain)

	testProcGetTransactionByHashes(t, blockchain)

	textProcGetBlockOverview(t, cfg, blockchain)

	testProcGetAddrOverview(t, cfg, blockchain)

	testProcGetBlockHash(t, cfg, blockchain)

	//	testSendDelBlockEvent(t, blockchain)

	testGetOrphanRoot(t, cfg, blockchain)

	testRemoveOrphanBlock(t, cfg, blockchain)

	testLoadBlockBySequence(t, blockchain)
	testAddBlockSeqCB(t, blockchain)
	testProcDelParaChainBlockMsg(t, mock33, blockchain)

	testProcAddParaChainBlockMsg(t, mock33, blockchain)
	testProcGetBlockBySeqMsg(t, mock33, blockchain)
	testProcBlockChainFork(t, blockchain)
	testDelBlock(t, blockchain)
	testIsRecordFaultErr(t)

	testGetsynBlkHeight(t, blockchain)
	testProcDelChainBlockMsg(t, mock33, blockchain)
	testFaultPeer(t, cfg, blockchain)
	testCheckBlock(t, blockchain)
	testWriteBlockToDbTemp(t, blockchain)
	testReadBlockToExec(t, blockchain)
	testReExecBlock(t, blockchain)
	testUpgradeStore(t, blockchain)

	testProcMainSeqMsg(t, blockchain)
	testAddOrphanBlock(t, blockchain)
	testCheckBestChainProc(t, cfg, blockchain)
}

func testProcAddBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcAddBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}
	cfg := mock33.GetClient().GetConfig()
	for {
		_, _, err = addTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI())
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

func testProcQueryTxMsg(cfg *types.Chain33Config, t *testing.T, blockchain *blockchain.BlockChain) {
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
	txProof, err := blockchain.ProcQueryTxMsg(txhash)
	require.NoError(t, err)
	if len(block.Block.Txs) <= 1 {
		assert.Nil(t, txProof.GetProofs())
		assert.Nil(t, txProof.GetTxProofs()[0].GetProofs())
	}

	blockheight := block.Block.GetHeight()
	if cfg.IsPara() {
		blockheight = block.Block.GetMainHeight()
	}
	if cfg.IsFork(blockheight, "ForkRootHash") {
		txhash = block.Block.Txs[txindex].FullHash()
	}
	//证明txproof的正确性,
	if txProof.GetProofs() != nil { //ForkRootHash 之前的proof证明
		brroothash := merkle.GetMerkleRootFromBranch(txProof.GetProofs(), txhash, uint32(txindex))
		assert.Equal(t, merkleroothash, brroothash)
	} else if txProof.GetTxProofs() != nil { //ForkRootHash 之后的proof证明
		var childhash []byte
		for i, txproof := range txProof.GetTxProofs() {
			if i == 0 {
				childhash = merkle.GetMerkleRootFromBranch(txproof.GetProofs(), txhash, txproof.GetIndex())
				if txproof.GetRootHash() != nil {
					assert.Equal(t, txproof.GetRootHash(), childhash)
				} else {
					assert.Equal(t, txproof.GetIndex(), uint32(txindex))
					assert.Equal(t, merkleroothash, childhash)
				}
			} else {
				brroothash := merkle.GetMerkleRootFromBranch(txproof.GetProofs(), childhash, txproof.GetIndex())
				assert.Equal(t, merkleroothash, brroothash)
			}
		}
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
	reqBlock.Start = 0
	reqBlock.End = 1000
	_, err = blockchain.ProcGetBlockDetailsMsg(&reqBlock)
	assert.Equal(t, err, types.ErrMaxCountPerTime)

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
	reqBlock.Start = 0
	reqBlock.End = 100000
	_, err = blockchain.ProcGetHeadersMsg(&reqBlock)
	assert.Equal(t, err, types.ErrMaxCountPerTime)

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

func testGetBlockByHash(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("TestGetBlockByHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block1, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash1 := block1.Block.Hash(cfg)
	block2, err := blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash1, block2.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash1, block2.Block.ParentHash)
	}
	block3, err := blockchain.ProcGetBlockByHashMsg(block2.Block.Hash(cfg))
	require.NoError(t, err)
	if !bytes.Equal(block2.Block.Hash(cfg), block3.Block.Hash(cfg)) {
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
			if sequence.Type != types.AddBlock {
				t.Error("testGetBlockSequences sequence type check error")
			}
		}
	}
	reqBlock.Start = 0
	reqBlock.End = 1000
	_, err = chain.GetBlockSequences(&reqBlock)
	assert.Equal(t, err, types.ErrMaxCountPerTime)

	chainlog.Info("testGetBlockSequences end --------------------")
}

func testGetBlockByHashes(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
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
			if !bytes.Equal(hashes[index], block.Block.Hash(cfg)) {
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
	cfg := mock33.GetClient().GetConfig()
	var reqkey types.ReqKey
	reqkey.Key = []byte(fmt.Sprintf("AddrTxsCount:%s", "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"))
	count, err := mock33.GetAPI().Query(cfg.ExecName("coins"), "GetAddrTxsCount", &reqkey)
	if err != nil {
		t.Error(err)
		return
	}
	if count.(*types.Int64).GetData() == 0 {
		t.Error("testAddrTxCount count check error ")
	}
	chainlog.Info("testAddrTxCount end --------------------")
}

func testGetBlockHerderByHash(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("testGetBlockHerderByHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash(cfg)
	block, err = blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}
	header, err := blockchain.GetStore().GetBlockHeaderByHash(block.Block.Hash(cfg))
	require.NoError(t, err)
	if !bytes.Equal(header.Hash, block.Block.Hash(cfg)) {
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

	parm.Count = 1001
	_, err = blockchain.ProcGetTransactionByAddr(parm)
	assert.Equal(t, err, types.ErrMaxCountPerTime)

	chainlog.Info("textProcGetTransactionByHashes end --------------------")
}

func textProcGetBlockOverview(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("textProcGetBlockOverview begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	parm := &types.ReqHash{
		Hash: block.Block.Hash(cfg),
	}
	blockOverview, err := blockchain.ProcGetBlockOverview(parm)
	require.NoError(t, err)

	if blockOverview != nil {
		if !bytes.Equal(block.Block.Hash(cfg), blockOverview.Head.Hash) {
			t.Error("textProcGetBlockOverview  block hash check error")
		}

	}
	chainlog.Info("textProcGetBlockOverview end --------------------")
}

func testProcGetAddrOverview(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcGetAddrOverview begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash(cfg)
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

func testProcGetBlockHash(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcGetBlockHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)
	height := &types.ReqInt{Height: curheight - 5}
	hash, err := blockchain.ProcGetBlockHash(height)
	require.NoError(t, err)

	if !bytes.Equal(block.Block.Hash(cfg), hash.Hash) {
		t.Error("testProcGetBlockHash  block hash check error")
	}

	chainlog.Info("testProcGetBlockHash end --------------------")
}

func testGetOrphanRoot(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("testGetOrphanRoot begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	hash := blockchain.GetOrphanPool().GetOrphanRoot(block.Block.Hash(cfg))
	if !bytes.Equal(block.Block.Hash(cfg), hash) {
		t.Error("testGetOrphanRoot  Orphan Root hash check error")
	}
	chainlog.Info("testGetOrphanRoot end --------------------")
}

func testRemoveOrphanBlock(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("testRemoveOrphanBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	ParentHashNotexist, _ := hex.DecodeString("8bd1f23741c90e9db4edd663acc0328a49f7c92d9974f83e2d5b57b25f3f059b")
	block.Block.ParentHash = ParentHashNotexist

	blockchain.GetOrphanPool().RemoveOrphanBlock2(block.Block, time.Time{}, false, "123", 0)
	blockchain.GetOrphanPool().RemoveOrphanBlockByHash(block.Block.Hash(cfg))

	chainlog.Info("testRemoveOrphanBlock end --------------------")
}

func testDelBlock(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testDelBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	newblock := types.BlockDetail{}
	newblock.Block = types.Clone(block.Block).(*types.Block)
	newblock.Block.Difficulty = block.Block.Difficulty - 100

	blockchain.ProcessBlock(true, &newblock, "1", true, 0)
	chainlog.Info("testDelBlock end --------------------")
}

func testLoadBlockBySequence(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testLoadBlockBySequence begin ---------------------")

	curheight := blockchain.GetBlockHeight()
	lastseq, _ := blockchain.GetStore().LoadBlockLastSequence()
	sequence, err := blockchain.GetStore().GetBlockSequence(lastseq)
	require.NoError(t, err)
	block, _, err := blockchain.GetStore().LoadBlockBySequence(lastseq)
	require.NoError(t, err)

	if block.Block.Height != curheight && types.DelBlock != sequence.GetType() {
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
	blockchain.MaxSeqCB = 2
	_, err := chain.ProcAddBlockSeqCB(cb)
	require.NoError(t, err)

	cbs, err := chain.ProcListBlockSeqCB()
	require.NoError(t, err)
	exist := false
	for _, temcb := range cbs.Items {
		if temcb.Name == cb.Name && !temcb.IsHeader {
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

	cb1 := &types.BlockSeqCB{
		Name:     "test-1",
		URL:      "http://192.168.1.107:15760",
		Encode:   "json",
		IsHeader: true,
	}
	_, err = chain.ProcAddBlockSeqCB(cb1)
	require.NoError(t, err)

	cbs, err = chain.ProcListBlockSeqCB()
	require.NoError(t, err)
	exist = false
	for _, temcb := range cbs.Items {
		if temcb.Name == cb1.Name && temcb.IsHeader {
			exist = true
		}
	}
	if !exist {
		t.Error("testAddBlockSeqCB  listSeqCB fail", "cb", cb1, "cbs", cbs)
	}
	num = chain.ProcGetSeqCBLastNum(cb1.Name)
	if num != -1 {
		t.Error("testAddBlockSeqCB  getSeqCBLastNum", "num", num, "name", cb1.Name)
	}

	cb2 := &types.BlockSeqCB{
		Name:   "test1",
		URL:    "http://192.168.1.107:15760",
		Encode: "json",
	}

	_, err = chain.ProcAddBlockSeqCB(cb2)
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

func testFaultPeer(t *testing.T, cfg *types.Chain33Config, chain *blockchain.BlockChain) {
	chainlog.Info("testFaultPeer begin ---------------------")
	curheight := chain.GetBlockHeight()
	block, err := chain.GetBlock(curheight)
	require.NoError(t, err)
	isok := chain.IsFaultPeer("self")
	if isok {
		t.Error("testFaultPeer:IsFaultPeer")
	}
	//记录故障peer信息
	chain.RecordFaultPeer("self", curheight+1, block.Block.Hash(cfg), types.ErrSign)

	var faultnode blockchain.FaultPeerInfo
	var peerinfo blockchain.PeerInfo
	peerinfo.Name = "self"
	faultnode.Peer = &peerinfo
	faultnode.FaultHeight = curheight + 1
	faultnode.FaultHash = block.Block.Hash(cfg)
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
	chain.DownLoadTimeOutProc(curheight - 1)
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
	err = chain.WriteBlockToDbTemp(&rawblock, true)
	if err != nil {
		t.Error("testWriteBlockToDbTemp", "err", err)
	}
	chain.UpdateDownLoadPids()
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

func testProcMainSeqMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Info("testProcMainSeqMsg begin -------------------")

	msg := queue.NewMessage(1, "blockchain", types.EventGetLastBlockMainSequence, nil)
	blockchain.GetLastBlockMainSequence(msg)
	assert.Equal(t, int64(types.EventGetLastBlockMainSequence), msg.Ty)

	msg = queue.NewMessage(1, "blockchain", types.EventGetMainSeqByHash, &types.ReqHash{Hash: []byte("hash")})
	blockchain.GetMainSeqByHash(msg)
	assert.Equal(t, int64(types.EventGetMainSeqByHash), msg.Ty)

	chainlog.Info("testProcMainSeqMsg end --------------------")
}

func testAddOrphanBlock(t *testing.T, blockchain *blockchain.BlockChain) {

	chainlog.Info("testAddOrphanBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)
	block.Block.ParentHash = block.Block.GetTxHash()
	newblock := types.BlockDetail{}
	newblock.Block = block.Block

	blockchain.ProcessBlock(true, &newblock, "1", true, 0)
	chainlog.Info("testAddOrphanBlock end --------------------")
}
func testCheckBestChainProc(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Info("testCheckBestChainProc begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)
	header := block.Block.GetHeader(cfg)

	var headers types.Headers
	headers.Items = append(headers.Items, header)
	blockchain.CheckBestChainProc(&headers, "test")

	blockchain.CheckBestChain(true)
	blockchain.GetNtpClockSyncStatus()
	blockchain.UpdateNtpClockSyncStatus(true)
	blockchain.IsErrExecBlock(curheight, block.Block.Hash(cfg))
	chainlog.Info("testCheckBestChainProc end --------------------")
}

//测试kv对的读写
func TestSetValueByKey(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	chainlog.Info("TestSetValueByKey begin --------------------")
	blockchain := mock33.GetBlockChain()

	//设置kv对到数据库，key的前缀错误
	var kvs types.LocalDBSet
	var kv types.KeyValue
	//ConsensusParaTxsPrefix = []byte("LODB:Consensus:Para:")
	key_1 := []byte("LODBP:Consensus:Parakey-1")
	value_1 := []byte("value-1")
	kv.Key = key_1
	kv.Value = value_1
	kvs.KV = append(kvs.KV, &kv)

	err := blockchain.SetValueByKey(&kvs)
	if err != nil {
		t.Error("TestSetValueByKey:SetValueByKey1", "err", err)
	}
	//读取数据为空
	var keys types.LocalDBGet
	key := []byte("LODBP:Consensus:Parakey-1")
	keys.Keys = append(keys.Keys, key)
	values := blockchain.GetValueByKey(&keys)
	for _, value := range values.Values {
		if bytes.Equal(value, value_1) {
			t.Error("TestSetValueByKey:GetValueByKey1", "value", string(value))
		}
	}
	//设置kv对到数据库
	var kvs2 types.LocalDBSet
	var kv2 types.KeyValue
	key_1 = types.CalcConsensusParaTxsKey([]byte("key-1"))
	kv2.Key = key_1
	kv2.Value = value_1
	kvs2.KV = append(kvs2.KV, &kv2)

	err = blockchain.SetValueByKey(&kvs2)
	if err != nil {
		t.Error("TestSetValueByKey:SetValueByKey2", "err", err)
	}
	//读取数据ok
	var keys2 types.LocalDBGet
	var key_1_exist bool
	keys2.Keys = append(keys2.Keys, key_1)
	values = blockchain.GetValueByKey(&keys2)
	for _, value := range values.Values {
		if bytes.Equal(value, value_1) {
			key_1_exist = true
		} else {
			t.Error("TestSetValueByKey:GetValueByKey", "value", string(value))
		}
	}
	if !key_1_exist {
		t.Error("TestSetValueByKey:GetValueByKey:key_1", "key_1", string(key_1))
	}

	//删除key_1对应的数据,set时设置key_1对应的value为nil
	for _, kv := range kvs2.KV {
		if bytes.Equal(kv.GetKey(), key_1) {
			kv.Value = nil
		}
	}
	err = blockchain.SetValueByKey(&kvs2)
	if err != nil {
		t.Error("TestSetValueByKey:SetValueByKey3", "err", err)
	}
	values = blockchain.GetValueByKey(&keys2)
	for _, value := range values.Values {
		if bytes.Equal(value, value_1) {
			t.Error("TestSetValueByKey:GetValueByKey4", "value", string(value))
		}
	}

	//插入多个kv对到数据库并获取
	var kvs3 types.LocalDBSet
	var kv3 types.KeyValue
	key_1 = types.CalcConsensusParaTxsKey([]byte("key-1"))
	value_1 = []byte("test-1")
	key_2 := types.CalcConsensusParaTxsKey([]byte("key-2"))
	value_2 := []byte("test-2")
	key_3 := types.CalcConsensusParaTxsKey([]byte("key-3"))
	value_3 := []byte("test-3")

	kv.Key = key_1
	kv.Value = value_1
	kv2.Key = key_2
	kv2.Value = value_2
	kv3.Key = key_3
	kv3.Value = value_3

	kvs3.KV = append(kvs2.KV, &kv, &kv2, &kv3)

	err = blockchain.SetValueByKey(&kvs3)
	if err != nil {
		t.Error("TestSetValueByKey:SetValueByKey4", "err", err)
	}
	//读取数据ok
	var keys3 types.LocalDBGet
	count := 0
	for _, kv := range kvs3.GetKV() {
		keys3.Keys = append(keys3.Keys, kv.GetKey())
	}
	values = blockchain.GetValueByKey(&keys3)
	for i, value := range values.Values {
		if bytes.Equal(value, kvs3.GetKV()[i].GetValue()) {
			count++
		} else {
			t.Error("TestSetValueByKey:GetValueByKey", "value", string(value))
		}
	}
	if count < 3 {
		t.Error("TestSetValueByKey:GetValueByKey:fail")
	}
	chainlog.Info("TestSetValueByKey end --------------------")
}

func TestOnChainTimeout(t *testing.T) {
	chainlog.Info("TestOnChainTimeout begin --------------------")

	cfg := testnode.GetDefaultConfig()
	mcfg := cfg.GetModuleConfig()
	mcfg.BlockChain.OnChainTimeout = 1
	mock33 := testnode.NewWithConfig(cfg, nil)

	defer mock33.Close()
	blockchain := mock33.GetBlockChain()

	//等待共识模块增长10个区块
	testProcAddBlockMsg(t, mock33, blockchain)

	curheight := blockchain.GetBlockHeight()

	//没有超时
	isTimeOut := blockchain.OnChainTimeout(curheight)
	assert.Equal(t, isTimeOut, false)

	//2秒后超时
	time.Sleep(2 * time.Second)
	lastheight := blockchain.GetBlockHeight()
	isTimeOut = blockchain.OnChainTimeout(lastheight)
	println("curheight:", curheight)
	println("lastheight:", lastheight)
	if lastheight == curheight {
		isTimeOut = blockchain.OnChainTimeout(lastheight)
		assert.Equal(t, isTimeOut, true)
	} else {
		time.Sleep(2 * time.Second)
		isTimeOut = blockchain.OnChainTimeout(lastheight)
		assert.Equal(t, isTimeOut, true)
	}

	chainlog.Info("TestOnChainTimeout end --------------------")
}

func TestProcessDelBlock(t *testing.T) {
	chainlog.Info("TestProcessDelBlock begin --------------------")
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	blockchain := mock33.GetBlockChain()

	//构造十个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}
	cfg := mock33.GetClient().GetConfig()

	// 确保只出指定数量的区块
	isFirst := true
	prevheight := curheight
	for {
		if isFirst || curheight > prevheight {
			_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.none")
			require.NoError(t, err)
			if curheight > prevheight {
				prevheight = curheight
			}
			isFirst = false
		}
		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}

	curheight = blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	_, ok, _, err := blockchain.ProcessDelParaChainBlock(true, block, "self", curheight)
	require.NoError(t, err)
	assert.Equal(t, true, ok)

	//获取已经删除的区块上的title
	var req types.ReqParaTxByTitle
	req.Start = curheight + 1
	req.End = curheight + 1
	req.Title = "user.p.para."
	req.IsSeq = true

	paratxs, err := blockchain.GetParaTxByTitle(&req)
	require.NoError(t, err)
	assert.NotNil(t, paratxs)
	for _, paratx := range paratxs.Items {
		assert.Equal(t, types.DelBlock, paratx.Type)
		assert.Nil(t, paratx.TxDetails)
		assert.Nil(t, paratx.ChildHash)
		assert.Nil(t, paratx.Proofs)
		assert.Equal(t, uint32(0), paratx.Index)
	}

	req.Title = "user.p.hyb."
	maintxs, err := blockchain.GetParaTxByTitle(&req)
	require.NoError(t, err)
	assert.NotNil(t, maintxs)
	for _, maintx := range maintxs.Items {
		assert.Equal(t, types.DelBlock, maintx.Type)
		roothash := merkle.GetMerkleRootFromBranch(maintx.Proofs, maintx.ChildHash, maintx.Index)
		var txs []*types.Transaction
		for _, tx := range maintx.TxDetails {
			txs = append(txs, tx.Tx)
		}
		hash := merkle.CalcMerkleRoot(cfg, maintx.GetHeader().GetHeight(), txs)
		assert.Equal(t, hash, roothash)
	}
	chainlog.Info("TestProcessDelBlock end --------------------")
}

func TestEnableCmpBestBlock(t *testing.T) {
	chainlog.Info("TestEnableCmpBestBlock begin --------------------")
	defaultCfg := testnode.GetDefaultConfig()
	mcfg := defaultCfg.GetModuleConfig()
	mcfg.Consensus.EnableBestBlockCmp = true
	mock33 := testnode.NewWithConfig(defaultCfg, nil)

	defer mock33.Close()
	blockchain := mock33.GetBlockChain()

	//构造十个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}
	cfg := mock33.GetClient().GetConfig()

	// 确保只出指定数量的区块
	isFirst := true
	prevheight := curheight
	for {
		if isFirst || curheight > prevheight {
			_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.none")
			require.NoError(t, err)
			if curheight > prevheight {
				prevheight = curheight
			}
			isFirst = false
		}
		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}

	curheight = blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	temblock := types.Clone(block.Block)
	newblock := temblock.(*types.Block)
	newblock.GetTxs()[0].Nonce = newblock.GetTxs()[0].Nonce + 1
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.GetHeight(), newblock.GetTxs())
	blockDetail := types.BlockDetail{Block: newblock}
	_, err = blockchain.ProcAddBlockMsg(true, &blockDetail, "peer")

	//直接修改了交易的Nonce，执行时会报ErrSign错误
	if err != nil {
		assert.Equal(t, types.ErrSign, err)
	} else {
		require.NoError(t, err)
	}

	//当前节点做对比
	curheight = blockchain.GetBlockHeight()
	curblock, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	testCmpBestBlock(t, mock33.GetClient(), curblock.Block, cfg)

	//非当前节点做对比
	curheight = blockchain.GetBlockHeight()
	oldblock, err := blockchain.GetBlock(curheight - 1)
	require.NoError(t, err)

	testCmpBestBlock(t, mock33.GetClient(), oldblock.Block, cfg)

	chainlog.Info("TestEnableCmpBestBlock end --------------------")
}

func TestDisableCmpBestBlock(t *testing.T) {
	chainlog.Info("TestDisableCmpBestBlock begin --------------------")
	defaultCfg := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(defaultCfg, nil)

	defer mock33.Close()
	blockchain := mock33.GetBlockChain()

	//构造十个区块
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}
	cfg := mock33.GetClient().GetConfig()

	// 确保只出指定数量的区块
	isFirst := true
	prevheight := curheight
	for {
		if isFirst || curheight > prevheight {
			_, err = addSingleParaTx(cfg, mock33.GetGenesisKey(), mock33.GetAPI(), "user.p.hyb.none")
			require.NoError(t, err)
			if curheight > prevheight {
				prevheight = curheight
			}
			isFirst = false
		}
		curheight = blockchain.GetBlockHeight()
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}

	curheight = blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	temblock := types.Clone(block.Block)
	newblock := temblock.(*types.Block)
	newblock.GetTxs()[0].Nonce = newblock.GetTxs()[0].Nonce + 1
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.GetHeight(), newblock.GetTxs())
	blockDetail := types.BlockDetail{Block: newblock}
	_, err = blockchain.ProcAddBlockMsg(true, &blockDetail, "peer")

	//直接修改了交易的Nonce，执行时会报ErrSign错误
	if err != nil {
		assert.Equal(t, types.ErrSign, err)
	} else {
		require.NoError(t, err)
	}
	chainlog.Info("TestDisableCmpBestBlock end --------------------")
}

func testCmpBestBlock(t *testing.T, client queue.Client, block *types.Block, cfg *types.Chain33Config) {
	temblock := types.Clone(block)
	newblock := temblock.(*types.Block)
	newblock.GetTxs()[0].Nonce = newblock.GetTxs()[0].Nonce + 1
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.GetHeight(), newblock.GetTxs())

	isbestBlock := util.CmpBestBlock(client, newblock, block.Hash(cfg))
	assert.Equal(t, isbestBlock, false)
}
