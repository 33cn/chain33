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

func addTxTxHeight(cfg *types.Chain33Config, priv crypto.PrivKey, api client.QueueProtocolAPI, height int64) ([]*types.Transaction, string, error) {
	txs := util.GenTxsTxHeight(cfg, priv, 1, height)
	hash := common.ToHex(txs[0].Hash())
	reply, err := api.SendTx(txs[0])
	if err != nil {
		return txs, hash, err
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

	// chunk
	testGetChunkRecordMsg(t, mock33, blockchain)
	testAddChunkRecordMsg(t, mock33, blockchain)
	testGetChunkBlockBodyMsg(t, mock33, blockchain)
	testAddChunkBlockMsg(t, mock33, blockchain)
}

func testGetChunkRecordMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testGetChunkRecordMsg begin --------------------")

	records := &types.ReqChunkRecords{
		Start:    1,
		End:      1,
		IsDetail: false,
	}

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventGetChunkRecord, records)
	err := mock33.GetClient().Send(msgGen, false)
	assert.NoError(t, err)
	chainlog.Debug("testGetChunkRecordMsg end --------------------")
}

func testAddChunkRecordMsg(t *testing.T, mock33 *testnode.Chain33Mock, chain *blockchain.BlockChain) {
	chainlog.Debug("testAddChunkRecordMsg begin --------------------")

	records := &types.ChunkRecords{
		Infos: []*types.ChunkInfo{{ChunkNum: 1, ChunkHash: []byte("11111111111")}},
	}

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventAddChunkRecord, records)
	err := mock33.GetClient().Send(msgGen, false)
	assert.Nil(t, err)
	chainlog.Debug("testAddChunkRecordMsg end --------------------")
}

func testGetChunkBlockBodyMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testGetChunkBlockBodyMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 1)
	require.NoError(t, err)

	end := block.Block.Height
	start := block.Block.Height - 2
	if start < 0 {
		start = 0
	}

	blocks := &types.ChunkInfoMsg{
		ChunkHash: []byte{},
		Start:     start,
		End:       end,
	}

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventGetChunkBlockBody, blocks)
	mock33.GetClient().Send(msgGen, true)
	resp, _ := mock33.GetClient().Wait(msgGen)
	if resp.GetData() != nil {
		bds := resp.Data.(*types.BlockBodys)
		assert.Equal(t, len(bds.Items), int(end-start+1))
	}
	chainlog.Debug("testGetChunkBlockBodyMsg end --------------------")
}

func testAddChunkBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testAddChunkBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block1, err := blockchain.GetBlock(curheight - 1)
	require.NoError(t, err)
	block2, err := blockchain.GetBlock(curheight - 2)
	require.NoError(t, err)

	blocks := &types.Blocks{
		Items: []*types.Block{block1.Block, block2.Block},
	}

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventAddChunkBlock, blocks)
	mock33.GetClient().Send(msgGen, false)
	chainlog.Debug("testAddChunkBlockMsg end --------------------")
}

func testProcAddBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcAddBlockMsg begin --------------------")

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
		chainlog.Debug("testProcAddBlockMsg ", "curheight", curheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(sendTxWait)
	}
	chainlog.Debug("testProcAddBlockMsg end --------------------")
}

func testGetTx(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("TestGetTx begin --------------------")
	//构建txhash
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	chainlog.Debug("testGetTx :", "curheight", curheight)
	txResult, err := blockchain.GetTxResultFromDb(block.Block.Txs[0].Hash())
	require.NoError(t, err)

	if err == nil && txResult != nil {
		Execer := string(txResult.GetTx().Execer)
		if "coins" != Execer {
			t.Error("ExecerName error")
		}
	}
	chainlog.Debug("TestGetTx end --------------------")
}

func testGetTxHashList(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("TestGetTxHashList begin --------------------")
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
		//chainlog.Debug("testGetTxHashList", "height", i, "count", j, "txhash", txhash)
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
				chainlog.Debug("testGetTxHashList", "duptxhash", duptxhash)
			}
		}
	}
	chainlog.Debug("TestGetTxHashList end --------------------")
}

func checkDupTx(cacheTxs []*types.Transaction, blockchain *blockchain.BlockChain) (*types.TxHashList, error) {
	var txhashlist types.TxHashList
	i := blockchain.GetBlockHeight()
	for j, tx := range cacheTxs {
		txhash := tx.Hash()
		chainlog.Debug("checkDupTx", "height", i, "count", j, "txhash", txhash)
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
		chainlog.Debug("checkDupTxHeight", "height", i, "count", j, "txhash", common.ToHex(txhash))
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
	chainlog.Debug("TestProcQueryTxMsg begin --------------------")
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
	chainlog.Debug("TestProcQueryTxMsg end --------------------")
}

func testGetBlocksMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("TestGetBlocksMsg begin --------------------")
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

	chainlog.Debug("TestGetBlocksMsg end --------------------")
}

func testProcGetHeadersMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("TestProcGetHeadersMsg begin --------------------")

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

	chainlog.Debug("TestProcGetHeadersMsg end --------------------")
}

// 新增区块时代码中是先更新UpdateHeight2，然后再更新UpdateLastBlock2
// 可能存在调用GetBlockHeight时已经更新，但UpdateLastBlock2还没有来得及更新最新区块
// GetBlockHeight()获取的最新高度 >= ProcGetLastHeaderMsg()获取的区块高度
func testProcGetLastHeaderMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("TestProcGetLastHeaderMsg begin --------------------")
	curheight := blockchain.GetBlockHeight()
	blockheader, err := blockchain.ProcGetLastHeaderMsg()
	if err == nil && blockheader != nil {
		if curheight < blockheader.Height {
			chainlog.Debug("TestProcGetLastHeaderMsg", "curheight", curheight, "blockheader.Height", blockheader.Height)
			t.Error("testProcGetLastHeaderMsg Last Header  check error")
		}
	}
	chainlog.Debug("TestProcGetLastHeaderMsg end --------------------")
}

func testGetBlockByHash(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("TestGetBlockByHash begin --------------------")
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
	chainlog.Debug("TestGetBlockByHash end --------------------")
}

func testProcGetLastSequence(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcGetLastSequence begin --------------------")

	curheight := blockchain.GetBlockHeight()

	lastSequence, err := blockchain.GetStore().LoadBlockLastSequence()
	require.NoError(t, err)
	if curheight != lastSequence {
		t.Error("testProcGetLastSequence Last Sequence check error")
	}
	chainlog.Debug("testProcGetLastSequence end --------------------")
}

func testGetBlockSequences(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Debug("testGetBlockSequences begin --------------------")
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

	chainlog.Debug("testGetBlockSequences end --------------------")
}

func testGetBlockByHashes(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testGetBlockByHashes begin --------------------")
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
	chainlog.Debug("testGetBlockByHashes end --------------------")
}

func testGetSeqByHash(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testGetSeqByHash begin --------------------")
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

	chainlog.Debug("testGetSeqByHash end --------------------")
}

func testPrefixCount(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testPrefixCount begin --------------------")

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventLocalPrefixCount, &types.ReqKey{Key: []byte("TxAddrHash:14KEKbYtKKQm4wMthSK9J4La4nAiidGozt:")})
	mock33.GetClient().Send(msgGen, true)
	Res, _ := mock33.GetClient().Wait(msgGen)
	count := Res.GetData().(*types.Int64).Data
	if count == 0 {
		t.Error("testPrefixCount count check error ")
	}
	chainlog.Debug("testPrefixCount end --------------------")
}

func testAddrTxCount(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testAddrTxCount begin --------------------")
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
	chainlog.Debug("testAddrTxCount end --------------------")
}

func testGetBlockHerderByHash(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testGetBlockHerderByHash begin --------------------")
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
	chainlog.Debug("testGetBlockHerderByHash end --------------------")
}

func testProcGetTransactionByHashes(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("textProcGetTransactionByHashes begin --------------------")
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

	chainlog.Debug("textProcGetTransactionByHashes end --------------------")
}

func textProcGetBlockOverview(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("textProcGetBlockOverview begin --------------------")
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
	chainlog.Debug("textProcGetBlockOverview end --------------------")
}

func testProcGetAddrOverview(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcGetAddrOverview begin --------------------")
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
	chainlog.Debug("testProcGetAddrOverview end --------------------")
}

func testProcGetBlockHash(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcGetBlockHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)
	height := &types.ReqInt{Height: curheight - 5}
	hash, err := blockchain.ProcGetBlockHash(height)
	require.NoError(t, err)

	if !bytes.Equal(block.Block.Hash(cfg), hash.Hash) {
		t.Error("testProcGetBlockHash  block hash check error")
	}

	chainlog.Debug("testProcGetBlockHash end --------------------")
}

func testGetOrphanRoot(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testGetOrphanRoot begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	hash := blockchain.GetOrphanPool().GetOrphanRoot(block.Block.Hash(cfg))
	if !bytes.Equal(block.Block.Hash(cfg), hash) {
		t.Error("testGetOrphanRoot  Orphan Root hash check error")
	}
	chainlog.Debug("testGetOrphanRoot end --------------------")
}

func testRemoveOrphanBlock(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testRemoveOrphanBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	ParentHashNotexist, _ := hex.DecodeString("8bd1f23741c90e9db4edd663acc0328a49f7c92d9974f83e2d5b57b25f3f059b")
	block.Block.ParentHash = ParentHashNotexist

	blockchain.GetOrphanPool().RemoveOrphanBlock2(block.Block, time.Time{}, false, "123", 0)
	blockchain.GetOrphanPool().RemoveOrphanBlockByHash(block.Block.Hash(cfg))

	chainlog.Debug("testRemoveOrphanBlock end --------------------")
}

func testDelBlock(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testDelBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	newblock := types.BlockDetail{}
	newblock.Block = types.Clone(block.Block).(*types.Block)
	newblock.Block.Difficulty = block.Block.Difficulty - 100

	blockchain.ProcessBlock(true, &newblock, "1", true, 0)
	chainlog.Debug("testDelBlock end --------------------")
}

func testLoadBlockBySequence(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testLoadBlockBySequence begin ---------------------")

	curheight := blockchain.GetBlockHeight()
	lastseq, _ := blockchain.GetStore().LoadBlockLastSequence()
	sequence, err := blockchain.GetStore().GetBlockSequence(lastseq)
	require.NoError(t, err)
	block, _, err := blockchain.GetStore().LoadBlockBySequence(lastseq)
	require.NoError(t, err)

	if block.Block.Height != curheight && types.DelBlock != sequence.GetType() {
		t.Error("testLoadBlockBySequence", "curheight", curheight, "lastseq", lastseq, "Block.Height", block.Block.Height)
	}
	chainlog.Debug("testLoadBlockBySequence end -------------------------")
}

func testProcDelParaChainBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcDelParaChainBlockMsg begin --------------------")

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
	chainlog.Debug("testProcDelParaChainBlockMsg end --------------------")
}

func testProcAddParaChainBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcAddParaChainBlockMsg begin --------------------")

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
	chainlog.Debug("testProcAddParaChainBlockMsg end --------------------")
}

func testProcGetBlockBySeqMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcGetBlockBySeqMsg begin --------------------")

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
	chainlog.Debug("testProcGetBlockBySeqMsg end --------------------")
}

func testProcBlockChainFork(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcBlockChainFork begin --------------------")

	curheight := blockchain.GetBlockHeight()
	blockchain.ProcDownLoadBlocks(curheight-1, curheight+256, []string{"self"})
	chainlog.Debug("testProcBlockChainFork end --------------------")
}

func testIsRecordFaultErr(t *testing.T) {
	chainlog.Debug("testIsRecordFaultErr begin ---------------------")
	isok := blockchain.IsRecordFaultErr(types.ErrFutureBlock)
	if isok {
		t.Error("testIsRecordFaultErr  IsRecordFaultErr", "isok", isok)
	}
	chainlog.Debug("testIsRecordFaultErr end ---------------------")
}

func testProcDelChainBlockMsg(t *testing.T, mock33 *testnode.Chain33Mock, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcDelChainBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	var parablockDetail types.ParaChainBlockDetail
	parablockDetail.Blockdetail = block
	parablockDetail.Sequence = curheight

	msgGen := mock33.GetClient().NewMessage("blockchain", types.EventDelParaChainBlockDetail, &parablockDetail)
	mock33.GetClient().Send(msgGen, true)
	mock33.GetClient().Wait(msgGen)

	chainlog.Debug("testProcDelChainBlockMsg end --------------------")
}
func testGetsynBlkHeight(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Debug("testGetsynBlkHeight begin --------------------")
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

	chainlog.Debug("testGetsynBlkHeight end --------------------")
}

func testFaultPeer(t *testing.T, cfg *types.Chain33Config, chain *blockchain.BlockChain) {
	chainlog.Debug("testFaultPeer begin ---------------------")
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
	chainlog.Debug("testFaultPeer end ---------------------")
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
	chainlog.Debug("testReadBlockToExec begin ---------------------")
	curheight := chain.GetBlockHeight()
	chain.ReadBlockToExec(curheight+1, false)
	chain.DownLoadTimeOutProc(curheight - 1)
	chainlog.Debug("testReadBlockToExec end ---------------------")
}
func testWriteBlockToDbTemp(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Debug("WriteBlockToDbTemp begin ---------------------")
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
	chainlog.Debug("WriteBlockToDbTemp end ---------------------")
}

func testReExecBlock(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Debug("ReExecBlock begin ---------------------")
	curheight := chain.GetBlockHeight()
	chain.ReExecBlock(0, curheight)
	chainlog.Debug("ReExecBlock end ---------------------")
}

func testUpgradeStore(t *testing.T, chain *blockchain.BlockChain) {
	chainlog.Debug("UpgradeStore begin ---------------------")
	chain.UpgradeStore()
	chainlog.Debug("UpgradeStore end ---------------------")
}

func testProcMainSeqMsg(t *testing.T, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testProcMainSeqMsg begin -------------------")

	msg := queue.NewMessage(1, "blockchain", types.EventGetLastBlockMainSequence, nil)
	blockchain.GetLastBlockMainSequence(msg)
	assert.Equal(t, int64(types.EventGetLastBlockMainSequence), msg.Ty)

	msg = queue.NewMessage(1, "blockchain", types.EventGetMainSeqByHash, &types.ReqHash{Hash: []byte("hash")})
	blockchain.GetMainSeqByHash(msg)
	assert.Equal(t, int64(types.EventGetMainSeqByHash), msg.Ty)

	chainlog.Debug("testProcMainSeqMsg end --------------------")
}

func testAddOrphanBlock(t *testing.T, blockchain *blockchain.BlockChain) {

	chainlog.Debug("testAddOrphanBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)
	block.Block.ParentHash = block.Block.GetTxHash()
	newblock := types.BlockDetail{}
	newblock.Block = block.Block

	blockchain.ProcessBlock(true, &newblock, "1", true, 0)
	chainlog.Debug("testAddOrphanBlock end --------------------")
}
func testCheckBestChainProc(t *testing.T, cfg *types.Chain33Config, blockchain *blockchain.BlockChain) {
	chainlog.Debug("testCheckBestChainProc begin --------------------")
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
	chainlog.Debug("testCheckBestChainProc end --------------------")
}

// 测试kv对的读写
func TestSetValueByKey(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer func() {
		defer mock33.Close()
	}()
	chainlog.Debug("TestSetValueByKey begin --------------------")
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
	chainlog.Debug("TestSetValueByKey end --------------------")
}

func TestOnChainTimeout(t *testing.T) {
	chainlog.Debug("TestOnChainTimeout begin --------------------")

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
	blockchain.OnChainTimeout(lastheight)
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

	chainlog.Debug("TestOnChainTimeout end --------------------")
}

func TestProcessDelBlock(t *testing.T) {
	chainlog.Debug("TestProcessDelBlock begin --------------------")
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

	//删除最新的区块:可能存在GetBlockHeight()获取的是最新区块，
	//但是在删除时b.bestChain.Tip()中还没有更新成最新的区块,此时会返回ErrBlockHashNoMatch错误信息

	isok := false
	count := 0
	for {
		_, ok, _, err := blockchain.ProcessDelParaChainBlock(true, block, "self", curheight)
		if err != nil {
			time.Sleep(sendTxWait)
			count++
		} else if true == ok && err == nil {
			isok = true
			break
		} else if count == 10 {
			isok = false
			chainlog.Error("TestProcessDelBlock 50ms timeout --------------------")
			break
		}
	}
	if !isok {
		chainlog.Error("TestProcessDelBlock:ProcessDelParaChainBlock:fail!")
		return
	}

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
	chainlog.Debug("TestProcessDelBlock end --------------------")
}

func TestEnableCmpBestBlock(t *testing.T) {
	chainlog.Debug("TestEnableCmpBestBlock begin --------------------")
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

	chainlog.Debug("TestEnableCmpBestBlock end --------------------")
}

func TestDisableCmpBestBlock(t *testing.T) {
	chainlog.Debug("TestDisableCmpBestBlock begin --------------------")
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
	chainlog.Debug("TestDisableCmpBestBlock end --------------------")
}

func testCmpBestBlock(t *testing.T, client queue.Client, block *types.Block, cfg *types.Chain33Config) {
	temblock := types.Clone(block)
	newblock := temblock.(*types.Block)
	newblock.GetTxs()[0].Nonce = newblock.GetTxs()[0].Nonce + 1
	newblock.TxHash = merkle.CalcMerkleRoot(cfg, newblock.GetHeight(), newblock.GetTxs())

	isbestBlock := util.CmpBestBlock(client, newblock, block.Hash(cfg))
	assert.Equal(t, isbestBlock, false)
}
