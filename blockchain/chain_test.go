package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"encoding/hex"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
)

var random *rand.Rand

//测试时有两个地方需要使用桩函数，testExecBlock来代替具体的util.ExecBlock
//poolRoutine函数中，以及ProcAddBlockMsg函数中
func init() {
	//queue.DisableLog()
	random = rand.New(rand.NewSource(types.Now().UnixNano()))
	log.SetLogLevel("error")
}

var q queue.Queue
var api client.QueueProtocolAPI
var priv = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
var cacheTxs []*types.Transaction
var cacheTxsTxHeigt []*types.Transaction
var TxHeightOffset int64 = 0

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
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

func initEnv() (*BlockChain, queue.Module, queue.Module, queue.Module, queue.Module, queue.Module) {
	var q = queue.New("channel")

	api, _ = client.New(q.Client(), nil)
	cfg := config.InitCfg("../cmd/chain33/chain33.test.toml")
	blockchain := New(cfg.BlockChain)
	blockchain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())

	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	cons := consensus.New(cfg.Consensus)
	cons.SetQueueClient(q.Client())

	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())
	mem.SetSync(true)
	mem.WaitPollLastHeader()

	network := p2p.New(cfg.P2P)
	network.SetQueueClient(q.Client())

	return blockchain, exec, cons, s, mem, network
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to, Expire: 0}
	tx.Nonce = random.Int63()
	tx.To = to
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func createTxWithTxHeight(priv crypto.PrivKey, to string, amount, expire int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to, Expire: expire + TxHeightOffset + types.TxHeightFlag}
	tx.Nonce = random.Int63()
	tx.To = to
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}

	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}

	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func genTxs(n int64) (txs []*types.Transaction, fromaddr string, to string) {
	to, _ = genaddress()
	for i := 0; i < int(n); i++ {
		tx := createTx(priv, to, types.Coin*(n+1))
		txs = append(txs, tx)
		cacheTxs = append(cacheTxs, tx)
	}
	return txs, fromaddr, to
}

func genTxsTxHeigt(n int64) (txs []*types.Transaction, fromaddr string, to string) {
	to, _ = genaddress()
	for i := 0; i < int(n); i++ {
		tx := createTxWithTxHeight(priv, to, types.Coin*(n+1), n+1)
		txs = append(txs, tx)
		cacheTxsTxHeigt = append(cacheTxsTxHeigt, tx)
	}
	return txs, fromaddr, to
}

// 打印block的信息
func PrintBlockInfo(block *types.BlockDetail) {
	return
	if block == nil {
		return
	}
	if block.Block != nil {
		fmt.Println("PrintBlockInfo block!")
		fmt.Println("block.Hash:", hex.EncodeToString(block.Block.Hash()))
		fmt.Println("block.ParentHash:", hex.EncodeToString(block.Block.ParentHash))
		fmt.Println("block.TxHash:", hex.EncodeToString(block.Block.TxHash))
		fmt.Println("block.BlockTime:", block.Block.BlockTime)
		fmt.Println("block.Height:", block.Block.Height)
		fmt.Println("block.Version:", block.Block.Version)
		fmt.Println("block.Difficulty:", block.Block.Difficulty)
		fmt.Println("block.StateHash:", hex.EncodeToString(block.Block.StateHash))

		fmt.Println("txs len:", len(block.Block.Txs))
		for _, tx := range block.Block.Txs {
			transaction := tx
			fmt.Println("tx.Payload:", hex.EncodeToString(transaction.Payload))
			fmt.Println("tx.Signature:", transaction.Signature.String())
		}
	}
	if block.Receipts != nil {
		fmt.Println("PrintBlockInfo Receipts!", "height", block.Block.Height)
		for index, receipt := range block.Receipts {
			fmt.Println("PrintBlockInfo Receipts!", "txindex", index)
			fmt.Println("PrintBlockInfo Receipts!", "ReceiptData", receipt.String())
		}
	}
}

// 打印header的信息
func PrintHeaderInfo(header *types.Header) {
	return
	if header == nil {
		return
	}

	fmt.Println("header.Version:", header.Version)
	fmt.Println("header.ParentHash:", header.ParentHash)
	fmt.Println("header.TxHash:", header.TxHash)
	fmt.Println("header.StateHash:", header.StateHash)
	fmt.Println("header.Height:", header.Height)
	fmt.Println("header.BlockTime:", header.BlockTime)
	fmt.Println("header.TxCount:", header.TxCount)
	fmt.Println("header.Hash:", header.Hash)
	fmt.Println("header.Difficulty:", header.Difficulty)

	if header.Signature != nil {
		fmt.Println("header Signature!")
		fmt.Println("header.Signature.Ty:", header.Signature.Ty)
		fmt.Println("header.Signature.Pubkey:", header.Signature.Pubkey)
		fmt.Println("header.Signature.Signature:", header.Signature.Signature)
	}
}

// 打印block的信息
func PrintSequenceInfo(Sequence *types.BlockSequence) {
	return
	if Sequence == nil {
		return
	}
	fmt.Println("PrintSequenceInfo!")
	fmt.Println("Sequence.Hash:", Sequence.Hash)
	fmt.Println("Sequence.Type:", Sequence.Type)
}
func addTx() (string, error) {
	txs, _, _ := genTxs(1)
	hash := common.Bytes2Hex(txs[0].Hash())
	reply, err := api.SendTx(txs[0])
	if err != nil {
		return hash, err
	}
	if !reply.GetIsOk() {
		return hash, errors.New("sendtx unknow error")
	}
	return hash, nil
}

func addTxTxHeigt() (string, error) {
	txs, _, _ := genTxsTxHeigt(1)
	hash := common.Bytes2Hex(txs[0].Hash())
	reply, err := api.SendTx(txs[0])
	if err != nil {
		return hash, err
	}
	if !reply.GetIsOk() {
		return hash, errors.New("sendtx unknow error")
	}
	return hash, nil
}

func TestcalcHeightToBlockHeaderKey(t *testing.T) {
	key := calcHeightToBlockHeaderKey(1)
	assert.Equal(t, key, []byte("000000000001"))
	key = calcHeightToBlockHeaderKey(0)
	assert.Equal(t, key, []byte("000000000000"))
	key = calcHeightToBlockHeaderKey(10)
	assert.Equal(t, key, []byte("000000000010"))
}

func TestBlockChain(t *testing.T) {
	blockchain, exec, cons, s, mem, p2p := initEnv()
	defer func() {
		blockchain.Close()
		exec.Close()
		cons.Close()
		s.Close()
		mem.Close()
		p2p.Close()
	}()

	//等待共识模块增长10个区块
	testProcAddBlockMsg(t, blockchain)

	testGetBlock(t, blockchain)

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
	testPrefixCount(t, blockchain)
	testAddrTxCount(t, blockchain)
	//testProcGetTransactionByHashes(t, blockchain)

	//testProcGetTransactionByAddr(t, blockchain)

	// QM add
	testGetBlockHerderByHash(t, blockchain)

	testProcGetTransactionByAddr(t, blockchain)

	textProcGetTransactionByHashes(t, blockchain)

	textProcGetBlockOverview(t, blockchain)

	testProcGetAddrOverview(t, blockchain)

	testProcGetBlockHash(t, blockchain)

	//	testSendDelBlockEvent(t, blockchain)

	testGetOrphanRoot(t, blockchain)

	testRemoveOrphanBlock(t, blockchain)

	testDelBlock(t, blockchain)

	testLoadBlockBySequence(t, blockchain)

	testProcDelParaChainBlockMsg(t, blockchain)

	testProcAddParaChainBlockMsg(t, blockchain)
	testProcBlockChainFork(t, blockchain)
}

func testProcAddBlockMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcAddBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testProcAddBlockMsg", "curheight", curheight)
	addblockheight := curheight + 10

	_, err := blockchain.GetBlock(curheight)
	if err != nil {
		require.NoError(t, err)
	}
	chainlog.Info("testProcAddBlockMsg", "addblockheight", addblockheight)
	for {
		_, err = addTx()
		require.NoError(t, err)
		curheight = blockchain.GetBlockHeight()
		chainlog.Info("testProcAddBlockMsg ", "curheight", curheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		if curheight >= addblockheight {
			break
		}
		time.Sleep(time.Second)
	}
	chainlog.Info("testProcAddBlockMsg end --------------------")
}

func testGetBlock(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testGetBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testGetBlock ", "curheight", curheight)
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)
	PrintBlockInfo(block)
	chainlog.Info("testGetBlock end --------------------")
}

func testGetTx(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestGetTx begin --------------------")
	//构建txhash
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	chainlog.Info("testGetTx :", "curheight", curheight)
	txResult, err := blockchain.GetTxResultFromDb(block.Block.Txs[0].Hash())
	if err != nil || txResult == nil {
		t.Error("testGetTx")
		return
	}
	chainlog.Info("TestGetTx end --------------------")
}

func testGetTxHashList(t *testing.T, blockchain *BlockChain) {
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
		chainlog.Info("testGetTxHashList", "height", i, "count", j, "txhash", txhash)
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

func checkDupTx(blockchain *BlockChain) (*types.TxHashList, error) {
	var txhashlist types.TxHashList
	i := blockchain.GetBlockHeight()
	for j, tx := range cacheTxs {
		txhash := tx.Hash()
		chainlog.Info("checkDupTx", "height", i, "count", j, "txhash", txhash)
		txhashlist.Hashes = append(txhashlist.Hashes, txhash[:])
	}
	duptxhashlist, err := blockchain.GetDuplicateTxHashList(&txhashlist)
	if err != nil {
		return nil, err
	}

	return duptxhashlist, nil
}

func checkDupTxHeight(blockchain *BlockChain) (*types.TxHashList, error) {
	var txhashlist types.TxHashList
	i := blockchain.GetBlockHeight()
	for j, tx := range cacheTxsTxHeigt {
		txhash := tx.Hash()
		chainlog.Info("checkDupTxHeight", "height", i, "count", j, "txhash", txhash)
		txhashlist.Hashes = append(txhashlist.Hashes, txhash[:])
		txhashlist.Expire = append(txhashlist.Expire, tx.Expire)
	}
	duptxhashlist, err := blockchain.GetDuplicateTxHashList(&txhashlist)
	if err != nil {
		return nil, err
	}

	return duptxhashlist, nil
}

func initCache() {
	cacheTxs = make([]*types.Transaction, 0)
	cacheTxsTxHeigt = make([]*types.Transaction, 0)
}

func TestCheckDupTxHashListOne(t *testing.T) {
	blockchain, exec, cons, s, mem, p2p := initEnv()
	defer func() {
		blockchain.Close()
		exec.Close()
		cons.Close()
		s.Close()
		mem.Close()
		p2p.Close()
		types.EnableTxHeight = false
	}()
	chainlog.Info("TestCheckDupTxHashListOne begin --------------------")
	initCache()
	types.EnableTxHeight = true
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 1
	for {
		_, err := addTx()
		require.NoError(t, err)
		chainlog.Info("testCheckDupTxHashListOne", "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		time.Sleep(time.Second)
		curheight := blockchain.GetBlockHeight()
		if curheight >= addblockheight {
			break
		}
	}
}

//构造10个区块，10笔交易不带TxHeight，缓存size128
func TestCheckDupTxHashList01(t *testing.T) {
	blockchain, exec, cons, s, mem, p2p := initEnv()
	defer func() {
		blockchain.Close()
		exec.Close()
		cons.Close()
		s.Close()
		mem.Close()
		p2p.Close()
		types.EnableTxHeight = false
	}()

	chainlog.Info("TestCheckDupTxHashList01 begin --------------------")

	initCache()
	types.EnableTxHeight = true
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	for {
		_, err := addTx()
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList01", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		time.Sleep(time.Second)
		if curheight >= addblockheight {
			break
		}
	}

	//重复交易
	duptxhashlist, err := checkDupTx(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != len(cacheTxs) {
		t.Error("CheckDupCacheFailed-01")
		return
	}

	//非重复交易
	initCache()
	genTxs(50)
	duptxhashlist, err = checkDupTx(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed-02")
		return
	}

	genTxsTxHeigt(50)
	duptxhashlist, err = checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed - 03")
		return
	}
	chainlog.Info("TestCheckDupTxHashList01 end --------------------")
}

//构造10个区块，10笔交易带TxHeight，缓存size128
func TestCheckDupTxHashList02(t *testing.T) {
	blockchain, exec, cons, s, mem, p2p := initEnv()
	defer func() {
		blockchain.Close()
		exec.Close()
		cons.Close()
		s.Close()
		mem.Close()
		p2p.Close()
		types.EnableTxHeight = false
	}()

	chainlog.Info("TestCheckDupTxHashList02 begin --------------------")

	initCache()
	types.EnableTxHeight = true
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	for {
		_, err := addTxTxHeigt()
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList02", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		time.Sleep(time.Second)
		if curheight >= addblockheight {
			break
		}
	}

	//重复交易
	duptxhashlist, err := checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != len(cacheTxsTxHeigt) {
		t.Error("CheckDupCacheFailed")
		return
	}

	//非重复交易
	initCache()
	genTxs(50)
	duptxhashlist, err = checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed")
		return
	}

	genTxsTxHeigt(50)
	duptxhashlist, err = checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed")
		return
	}
	chainlog.Info("TestCheckDupTxHashList02 end --------------------")
}

//构造130个区块，130笔交易不带TxHeight，缓存size128
func TestCheckDupTxHashList03(t *testing.T) {
	blockchain, exec, cons, s, mem, p2p := initEnv()
	defer func() {
		blockchain.Close()
		exec.Close()
		cons.Close()
		s.Close()
		mem.Close()
		p2p.Close()
		types.EnableTxHeight = false
	}()
	chainlog.Info("TestCheckDupTxHashList03 begin --------------------")

	initCache()
	types.EnableTxHeight = true
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 130
	i := 0
	for {
		_, err := addTx()
		i++
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList03", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		time.Sleep(time.Second)
		if curheight >= addblockheight {
			break
		}
	}

	//重复交易
	duptxhashlist, err := checkDupTx(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != i {
		t.Error("CheckDupCacheFailed - 01")
		return
	}

	//非重复交易
	initCache()
	genTxs(50)
	duptxhashlist, err = checkDupTx(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed - 02")
		return
	}

	genTxsTxHeigt(50)
	duptxhashlist, err = checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed - 03")
		return
	}

	chainlog.Info("TestCheckDupTxHashList03 end --------------------")
}

//构造10个区块，10笔交易带TxHeight，缓存已满 size128
func TestCheckDupTxHashList04(t *testing.T) {
	blockchain, exec, cons, s, mem, p2p := initEnv()
	defer func() {
		blockchain.Close()
		exec.Close()
		cons.Close()
		s.Close()
		mem.Close()
		p2p.Close()
		types.EnableTxHeight = false
	}()
	chainlog.Info("TestCheckDupTxHashList04 begin --------------------")

	initCache()
	types.EnableTxHeight = true
	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 10
	for {
		_, err := addTxTxHeigt()
		require.NoError(t, err)
		curheight := blockchain.GetBlockHeight()
		chainlog.Info("testCheckDupTxHashList04", "curheight", curheight, "addblockheight", addblockheight)
		_, err = blockchain.GetBlock(curheight)
		require.NoError(t, err)
		time.Sleep(time.Second)
		if curheight >= addblockheight {
			break
		}
	}

	//重复交易
	duptxhashlist, err := checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != len(cacheTxsTxHeigt) {
		t.Error("CheckDupCacheFailed")
		return
	}

	//非重复交易
	initCache()
	genTxs(50)
	duptxhashlist, err = checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed")
		return
	}

	genTxsTxHeigt(50)
	duptxhashlist, err = checkDupTxHeight(blockchain)
	if err != nil {
		t.Error(err)
		return
	}
	if len(duptxhashlist.Hashes) != 0 {
		t.Error("CheckDupCacheFailed")
		return
	}
	chainlog.Info("TestCheckDupTxHashList04 end --------------------")
}

//异常：构造10个区块，10笔交易带TxHeight，TxHeight不满足条件 size128
func TestCheckDupTxHashList05(t *testing.T) {
	blockchain, exec, cons, s, mem, p2p := initEnv()
	defer func() {
		blockchain.Close()
		exec.Close()
		cons.Close()
		s.Close()
		mem.Close()
		p2p.Close()
		TxHeightOffset = 0
		types.EnableTxHeight = false
	}()
	chainlog.Info("TestCheckDupTxHashList05 begin --------------------")

	initCache()
	TxHeightOffset = 60
	types.EnableTxHeight = true
	for i := 0; i < 10; i++ {
		_, err := addTxTxHeigt()
		require.EqualError(t, err, "ErrTxExpire")
		time.Sleep(time.Second)
	}
	chainlog.Info("TestCheckDupTxHashList05 end --------------------")
}

func testProcQueryTxMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestProcQueryTxMsg begin --------------------")
	curheight := blockchain.GetBlockHeight()
	var merkleroothash []byte
	var txhash []byte
	var txindex int

	//获取当前高度的block信息
	block, err := blockchain.GetBlock(curheight)

	if err == nil {
		merkleroothash = block.Block.TxHash
		//fmt.Println("block.TxHash:", block.Block.TxHash)
		for index, transaction := range block.Block.Txs {
			//fmt.Println("tx.Payload:", string(transaction.Payload))
			//fmt.Println("tx.Signature:", transaction.Signature.String())
			txhash = transaction.Hash()
			txindex = index
		}
	}
	txproof, err := blockchain.ProcQueryTxMsg(txhash)
	require.NoError(t, err)

	//证明txproof的正确性
	brroothash := merkle.GetMerkleRootFromBranch(txproof.GetProofs(), txhash, uint32(txindex))
	if bytes.Equal(merkleroothash, brroothash) {
		chainlog.Info("testProcQueryTxMsg merkleroothash ==  brroothash  ")
	}
	chainlog.Info("testProcQueryTxMsg!", "GetTx", txproof.GetTx().String())
	chainlog.Info("testProcQueryTxMsg!", "GetReceipt", txproof.GetReceipt().String())

	chainlog.Info("TestProcQueryTxMsg end --------------------")
}

func testGetBlocksMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestGetBlocksMsg begin --------------------")
	curheight := blockchain.GetBlockHeight()
	var reqBlock types.ReqBlocks
	if curheight >= 5 {
		reqBlock.Start = curheight - 5
	}
	reqBlock.End = curheight
	reqBlock.IsDetail = true

	blocks, err := blockchain.ProcGetBlockDetailsMsg(&reqBlock)
	if err == nil && blocks != nil {
		for _, block := range blocks.Items {
			PrintBlockInfo(block)
		}
	}
	chainlog.Info("TestGetBlocksMsg end --------------------")
}

func testProcGetHeadersMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestProcGetHeadersMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	var reqBlock types.ReqBlocks
	if curheight >= 5 {
		reqBlock.Start = curheight - 5
	}
	reqBlock.End = curheight

	blockheaders, err := blockchain.ProcGetHeadersMsg(&reqBlock)
	if err != nil || blockheaders == nil {
		t.Error("testProcGetHeadersMsg")
		return
	}
	chainlog.Info("TestProcGetHeadersMsg end --------------------")
}

func testProcGetLastHeaderMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestProcGetLastHeaderMsg begin --------------------")

	blockheader, err := blockchain.ProcGetLastHeaderMsg()
	if err != nil || blockheader == nil {
		t.Error("ProcGetLastHeaderMsg")
	}
	chainlog.Info("TestProcGetLastHeaderMsg end --------------------")
}

func testGetBlockByHash(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestGetBlockByHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("TestGetBlockByHash ", "curheight", curheight)
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash()
	block, err = blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}
	block, err = blockchain.ProcGetBlockByHashMsg(block.Block.Hash())
	require.NoError(t, err)

	PrintBlockInfo(block)
	chainlog.Info("TestGetBlockByHash end --------------------")
}

func testProcGetLastSequence(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcGetLastSequence begin --------------------")

	_, err := blockchain.blockStore.LoadBlockLastSequence()

	if err != nil {
		t.Error(err)
		return
	}
	chainlog.Info("testProcGetLastSequence end --------------------")
}

func testGetBlockSequences(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testGetBlockSequences begin --------------------")
	lastSequence, _ := blockchain.blockStore.LoadBlockLastSequence()
	var reqBlock types.ReqBlocks
	if lastSequence >= 5 {
		reqBlock.Start = lastSequence - 5
	}
	reqBlock.End = lastSequence
	reqBlock.IsDetail = true
	Sequences, err := blockchain.GetBlockSequences(&reqBlock)
	if err == nil && Sequences != nil {
		for _, sequence := range Sequences.Items {
			PrintSequenceInfo(sequence)
		}
	}
	chainlog.Info("testGetBlockSequences end --------------------")
}

func testGetBlockByHashes(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testGetBlockByHashes begin --------------------")
	lastSequence, _ := blockchain.blockStore.LoadBlockLastSequence()
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
		for _, block := range blocks.Items {
			PrintBlockInfo(block)
		}
	}
	chainlog.Info("testGetBlockByHashes end --------------------")
}

func testGetSeqByHash(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testGetSeqByHash begin --------------------")
	lastSequence, _ := blockchain.blockStore.LoadBlockLastSequence()
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

	seq, err := blockchain.ProcGetSeqByHash(hashes[0])
	chainlog.Info("testGetSeqByHash", "seq", seq, "err", err)
	chainlog.Info("testGetSeqByHash end --------------------")
}

func testPrefixCount(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testPrefixCount begin --------------------")

	msgGen := blockchain.client.NewMessage("blockchain", types.EventLocalPrefixCount, &types.ReqKey{Key: []byte("TxAddrHash:14KEKbYtKKQm4wMthSK9J4La4nAiidGozt:")})
	blockchain.client.Send(msgGen, true)
	Res, _ := blockchain.client.Wait(msgGen)
	count := Res.GetData().(*types.Int64).Data
	chainlog.Info("testPrefixCount end --------------------", "count", count)
}

func testAddrTxCount(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testAddrTxCount begin --------------------")
	var reqkey types.ReqKey
	reqkey.Key = []byte(fmt.Sprintf("AddrTxsCount:%s", "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"))
	count, _ := blockchain.query.Query(types.ExecName("coins"), "GetAddrTxsCount", types.Encode(&reqkey))
	fmt.Println("count: ", count.(*types.Int64).GetData())
	chainlog.Info("testAddrTxCount end --------------------")
}

func testGetBlockHerderByHash(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testGetBlockHerderByHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testGetBlockHerderByHash ", "curheight", curheight)
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash()
	block, err = blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}
	header, err := blockchain.blockStore.GetBlockHeaderByHash(block.Block.Hash())
	require.NoError(t, err)
	PrintHeaderInfo(header)
	chainlog.Info("testGetBlockHerderByHash end --------------------")
}

func testProcGetTransactionByAddr(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcGetTransactionByAddr begin --------------------")
	parm := &types.ReqAddr{
		Addr:   "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt",
		Height: -1,
	}
	txinfos, err := blockchain.ProcGetTransactionByAddr(parm)
	require.NoError(t, err)
	require.NotNil(t, txinfos)
	chainlog.Info("testProcGetTransactionByAddr end --------------------")
}

func textProcGetTransactionByHashes(t *testing.T, blockchain *BlockChain) {
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

	require.NotNil(t, TxDetails)

	chainlog.Info("textProcGetTransactionByHashes end --------------------")
}

func textProcGetBlockOverview(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("textProcGetBlockOverview begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	//fmt.Println("Print block.Block.Hash(): ", block.Block.Hash())
	parm := &types.ReqHash{
		Hash: block.Block.Hash(),
	}
	blockOverview, err := blockchain.ProcGetBlockOverview(parm)
	require.NoError(t, err)

	if blockOverview != nil {
		//fmt.Println("PrintaddrOverview Receipts!")
		//fmt.Println("Print blockOverview.Head.Hash: ", hex.EncodeToString(blockOverview.Head.Hash))
		//fmt.Println("Print blockOverview.TxCount: ", blockOverview.TxCount)
		//fmt.Println("Print blockOverview.TxHashes: ", blockOverview.TxHashes)
	}
	chainlog.Info("textProcGetBlockOverview end --------------------")
}

func testProcGetAddrOverview(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcGetAddrOverview begin --------------------")
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testProcGetAddrOverview ", "curheight", curheight)
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash()
	block, err = blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		//fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}

	parm := &types.ReqAddr{
		Addr: "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt",
	}
	addrOverview, err := blockchain.ProcGetAddrOverview(parm)
	require.NoError(t, err)

	if addrOverview != nil {
		//fmt.Println("PrintaddrOverview Receipts!")
		//fmt.Println("Print addrOverview.Reciver: ", addrOverview.Reciver)
		//fmt.Println("Print addrOverview.Balance: ", addrOverview.Balance)
		//fmt.Println("Print addrOverview.TxCount: ", addrOverview.TxCount)
	}
	chainlog.Info("testProcGetAddrOverview end --------------------")
}

func testProcGetBlockHash(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcGetBlockHash begin --------------------")
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testProcGetBlockHash ", "curheight", curheight)
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	blockhash := block.Block.Hash()
	block, err = blockchain.GetBlock(curheight - 4)
	require.NoError(t, err)

	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}

	height := &types.ReqInt{curheight}
	hash, err := blockchain.ProcGetBlockHash(height)
	require.NoError(t, err)

	if hash != nil {
		fmt.Println("Print  hash.Hash: ", hash.Hash)
	}

	chainlog.Info("testProcGetBlockHash end --------------------")
}

/*
func testSendDelBlockEvent(t *testing.T, blockchain *BlockChain)  {

}*/

func testGetOrphanRoot(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testGetOrphanRoot begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	fmt.Println("Print  hash: ", hex.EncodeToString(block.Block.Hash()))
	hash := blockchain.orphanPool.GetOrphanRoot(block.Block.Hash())
	fmt.Println("Print  hash: ", hex.EncodeToString(hash))

	chainlog.Info("testGetOrphanRoot end --------------------")
}

func testRemoveOrphanBlock(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testRemoveOrphanBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 5)
	require.NoError(t, err)

	ParentHashNotexist, _ := hex.DecodeString("8bd1f23741c90e9db4edd663acc0328a49f7c92d9974f83e2d5b57b25f3f059b")
	block.Block.ParentHash = ParentHashNotexist

	oBlock := &orphanBlock{
		block: block.Block,
		//		expiration: time.Second,
		broadcast: false,
		pid:       "123",
		sequence:  0,
	}

	blockchain.orphanPool.RemoveOrphanBlock(oBlock)

	chainlog.Info("testRemoveOrphanBlock end --------------------")
}

func testDelBlock(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testDelBlock begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	block.Block.Difficulty = block.Block.Difficulty - 100
	blockchain.ProcessBlock(true, block, "1", true, 0)

	chainlog.Info("testDelBlock end --------------------")
}

func testLoadBlockBySequence(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testLoadBlockBySequence begin ---------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.blockStore.LoadBlockBySequence(curheight)
	require.NoError(t, err)

	PrintBlockInfo(block)
	chainlog.Info("testLoadBlockBySequence end -------------------------")
}

func testProcDelParaChainBlockMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcDelParaChainBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight - 1)
	require.NoError(t, err)

	var parablockDetail types.ParaChainBlockDetail
	parablockDetail.Blockdetail = block
	parablockDetail.Sequence = 1

	msgGen := blockchain.client.NewMessage("blockchain", types.EventDelParaChainBlockDetail, &parablockDetail)

	blockchain.client.Send(msgGen, true)
	Res, _ := blockchain.client.Wait(msgGen)
	fmt.Println(Res)
	chainlog.Info("testProcDelParaChainBlockMsg end --------------------")
}

func testProcAddParaChainBlockMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcAddParaChainBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)
	require.NoError(t, err)

	var parablockDetail types.ParaChainBlockDetail
	parablockDetail.Blockdetail = block
	parablockDetail.Sequence = 1

	msgGen := blockchain.client.NewMessage("blockchain", types.EventAddParaChainBlockDetail, &parablockDetail)

	blockchain.client.Send(msgGen, true)
	Res, _ := blockchain.client.Wait(msgGen)
	fmt.Println(Res)
	chainlog.Info("testProcAddParaChainBlockMsg end --------------------")
}

func testProcBlockChainFork(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcBlockChainFork begin --------------------")

	curheight := blockchain.GetBlockHeight()
	blockchain.ProcBlockChainFork(curheight-1, curheight+256, "self")
	chainlog.Info("testProcBlockChainFork end --------------------")
}
