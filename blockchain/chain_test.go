package blockchain

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var random *rand.Rand

//测试时有两个地方需要使用桩函数，testExecBlock来代替具体的util.ExecBlock
//poolRoutine函数中，以及ProcAddBlockMsg函数中
func init() {
	queue.DisableLog()
	log.SetLogLevel("info")
}

func initEnv() (*BlockChain, queue.Queue) {
	var q = queue.New("channel")
	cfg := config.InitCfg("../chain33.test.toml")

	blockchain := New(cfg.BlockChain)
	blockchain.SetQueueClient(q.Client())
	return blockchain, q
}

func ConstructionBlock(parentHash []byte, height int64, txcount int) (*types.Block, string, string) {
	var block types.Block

	block.BlockTime = height
	block.Height = height
	block.Version = 100
	block.StateHash = common.Hash{}.Bytes()
	block.ParentHash = parentHash
	block.Difficulty = types.GetP(0).PowLimitBits
	txs, fromaddr, toaddr := genTxs(int64(txcount))
	block.Txs = txs
	block.TxHash = merkle.CalcMerkleRoot(block.Txs)
	return &block, fromaddr, toaddr
}

func ConstructionBlockDetail(parentHash []byte, height int64, txcount int) (*types.BlockDetail, string, string) {
	var blockdetail types.BlockDetail
	var block types.Block

	blockdetail.Receipts = make([]*types.ReceiptData, txcount)

	block.BlockTime = height
	block.Height = height
	block.Version = 100
	block.StateHash = common.Hash{}.Bytes()
	block.ParentHash = parentHash
	txs, fromaddr, toaddr := genTxs(int64(txcount))
	block.Txs = txs
	block.TxHash = merkle.CalcMerkleRoot(block.Txs)

	for j := 0; j < txcount; j++ {
		ReceiptData := types.ReceiptData{Ty: 2}
		blockdetail.Receipts[j] = &ReceiptData
	}
	blockdetail.Block = &block
	return &blockdetail, fromaddr, toaddr
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	random = rand.New(rand.NewSource(time.Now().UnixNano()))

	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = random.Int63()
	tx.To = account.ExecAddress("none").String()
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

	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func genTxs(n int64) (txs []*types.Transaction, fromaddr string, to string) {
	fromaddr, priv := genaddress()
	to, _ = genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, createTx(priv, to, types.Coin*(n+1)))
	}
	return txs, fromaddr, to
}

// 打印block的信息
func PrintBlockInfo(block *types.BlockDetail) {
	if block == nil {
		return
	}
	if block.Block != nil {
		fmt.Println("PrintBlockInfo block!")
		fmt.Println("block.ParentHash:", block.Block.ParentHash)
		fmt.Println("block.TxHash:", block.Block.TxHash)
		fmt.Println("block.BlockTime:", block.Block.BlockTime)
		fmt.Println("block.Height:", block.Block.Height)
		fmt.Println("block.Version:", block.Block.Version)
		fmt.Println("block.StateHash:", block.Block.StateHash)

		fmt.Println("txs len:", len(block.Block.Txs))
		for _, tx := range block.Block.Txs {
			transaction := tx
			fmt.Println("tx.Payload:", string(transaction.Payload))
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

func TestBlockChain(t *testing.T) {
	blockchain, client := initEnv()
	defer blockchain.Close()

	execprocess(client)
	consensusprocess(client)

	//同步过来的block
	testProcAddBlockMsg(t, blockchain)

	//共识模块发过来的block
	testProcAddBlockDetail(t, blockchain)

	testGetBlock(t, blockchain)

	testGetTx(t, blockchain)

	testGetTxHashList(t, blockchain)

	testProcQueryTxMsg(t, blockchain)

	testGetBlocksMsg(t, blockchain)

	testProcGetHeadersMsg(t, blockchain)

	testProcGetLastHeaderMsg(t, blockchain)

	testGetBlockByHash(t, blockchain)

	testProcGetTransactionByHashes(t, blockchain)

	testProcGetTransactionByAddr(t, blockchain)
}

//同步过来的block
func testProcAddBlockMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("testProcAddBlockMsg begin --------------------")

	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testProcAddBlockMsg", "curheight", curheight)
	addblockheight := curheight + 10
	var parentHash []byte
	block, err := blockchain.GetBlock(curheight)
	if err != nil {
		parentHash = zeroHash[:]
	} else {
		parentHash = block.Block.Hash()
	}
	chainlog.Info("testProcAddBlockMsg", "addblockheight", addblockheight)
	for i := curheight + 1; i <= addblockheight; i++ {
		block, _, _ := ConstructionBlock(parentHash, i, 5)
		blockchain.ProcAddBlockMsg(false, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
	}

	curheight = blockchain.GetBlockHeight()
	chainlog.Info("testProcAddBlockMsg ", "curheight", curheight)
	block, err = blockchain.GetBlock(curheight)
	require.NoError(t, err)
	PrintBlockInfo(block)

	chainlog.Info("testProcAddBlockMsg end --------------------")
}

//共识模块发过来的block
func testProcAddBlockDetail(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestProcAddBlockDetail begin --------------------")

	curheight := blockchain.GetBlockHeight()
	addblockheight := curheight + 1
	var parentHash []byte
	block, err := blockchain.GetBlock(curheight)
	if err != nil {
		parentHash = zeroHash[:]
	} else {
		parentHash = block.Block.Hash()
	}
	chainlog.Info("TestProcAddBlockDetail", "addblockheight", addblockheight)
	for i := curheight + 1; i <= addblockheight; i++ {
		block, _, _ := ConstructionBlockDetail(parentHash, i, 5)
		blockchain.ProcAddBlockMsg(true, block)
		parentHash = block.Block.Hash()
	}

	//print
	curheight = blockchain.GetBlockHeight()
	chainlog.Info("TestProcAddBlockDetail ", "curheight", curheight)
	block, err = blockchain.GetBlock(curheight)
	require.NoError(t, err)
	PrintBlockInfo(block)

	chainlog.Info("TestProcAddBlockDetail end --------------------")
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

	chainlog.Info("testGetTx curheight:", curheight)
	txResult, err := blockchain.GetTxResultFromDb(block.Block.Txs[0].Hash())
	if err == nil && txResult != nil {
		fmt.Println("testGetTx info:.")
		fmt.Println("txResult.Index:", txResult.Index)
		fmt.Println("txResult.Height:", txResult.Height)
		fmt.Println("tx.Payload:", string(txResult.Tx.Payload))
		fmt.Println("tx.Signature:", txResult.Tx.Signature.String())
		fmt.Println("tx.Receiptdate:", txResult.Receiptdate.String())
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
	duptxhashlist := blockchain.GetDuplicateTxHashList(&txhashlist)
	if duptxhashlist != nil {
		for _, duptxhash := range duptxhashlist.Hashes {
			if duptxhash != nil {
				chainlog.Info("testGetTxHashList", "duptxhash", duptxhash)
			}
		}
	}
	chainlog.Info("TestGetTxHashList end --------------------")
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
		fmt.Println("block.TxHash:", block.Block.TxHash)

		fmt.Println("txs len:", len(block.Block.Txs))
		for index, transaction := range block.Block.Txs {
			fmt.Println("tx.Payload:", string(transaction.Payload))
			fmt.Println("tx.Signature:", transaction.Signature.String())
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
	reqBlock.Isdetail = true

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
	if err == nil && blockheaders != nil {
		for _, head := range blockheaders.Items {
			fmt.Println("TestProcGetHeadersMsg info:.")
			fmt.Println("head.ParentHash:", head.ParentHash)
			fmt.Println("head.TxHash:", head.TxHash)
			fmt.Println("head.BlockTime:", head.BlockTime)
			fmt.Println("head.Height:", head.Height)
			fmt.Println("head.Version:", head.Version)
			fmt.Println("head.StateHash:", head.StateHash)
		}
	}
	chainlog.Info("TestProcGetHeadersMsg end --------------------")
}

func testProcGetLastHeaderMsg(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestProcGetLastHeaderMsg begin --------------------")

	blockheader, err := blockchain.ProcGetLastHeaderMsg()
	if err == nil && blockheader != nil {
		fmt.Println("TestProcGetLastHeaderMsg info:.")
		fmt.Println("head.ParentHash:", blockheader.ParentHash)
		fmt.Println("head.TxHash:", blockheader.TxHash)
		fmt.Println("head.BlockTime:", blockheader.BlockTime)
		fmt.Println("head.Height:", blockheader.Height)
		fmt.Println("head.Version:", blockheader.Version)
		fmt.Println("head.StateHash:", blockheader.StateHash)
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
func testProcGetTransactionByHashes(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestProcGetTransactionByHashes begin --------------------")
	curheight := blockchain.GetBlockHeight()
	block, _ := blockchain.GetBlock(curheight)
	parentHash := block.Block.Hash()

	chainlog.Info("TestProcGetTransactionByHashes ", "curheight", curheight)

	addblockheight := curheight + 10
	txhashs := make([][]byte, 10)
	j := 0
	for i := curheight + 1; i <= addblockheight; i++ {
		block, _, _ := ConstructionBlock(parentHash, i, 5)
		blockchain.ProcAddBlockMsg(false, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
		txhashs[j] = block.Txs[0].Hash()
		j++
	}

	txs, err := blockchain.ProcGetTransactionByHashes(txhashs)
	require.NoError(t, err)

	for _, txdetail := range txs.Txs {
		if txdetail != nil {
			fmt.Println("TestProcGetTransactionByHashes info:.")
			fmt.Println("txresult.Tx:", txdetail.Tx.String())
			fmt.Println("txresult.Receipt:", txdetail.Receipt.String())
		}
	}
	chainlog.Info("TestProcGetTransactionByHashes end --------------------")
}

func testProcGetTransactionByAddr(t *testing.T, blockchain *BlockChain) {
	chainlog.Info("TestProcGetTransactionByAddr begin --------------------")
	curheight := blockchain.GetBlockHeight()
	curblock, _ := blockchain.GetBlock(curheight)
	parentHash := curblock.Block.Hash()

	chainlog.Info("TestProcGetTransactionByAddr from -> to", "curheight", curheight)
	addblockheight := curheight + 5
	var fromaddr string
	var toaddr string
	var block *types.Block

	for i := curheight + 1; i <= addblockheight; i++ {
		block, fromaddr, toaddr = ConstructionBlock(parentHash, i, 5)
		blockchain.ProcAddBlockMsg(false, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
	}

	var reqAddr types.ReqAddr
	reqAddr.Addr = fromaddr
	reqAddr.Flag = 0
	reqAddr.Count = 10
	reqAddr.Direction = 0
	reqAddr.Height = -1
	reqAddr.Index = 0

	//fromaddr
	txs, _ := blockchain.ProcGetTransactionByAddr(&reqAddr)
	fmt.Println("ProcGetTransactionByAddr info:", "fromaddr", fromaddr)
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}
	//toaddr
	reqAddr.Addr = toaddr
	txs, _ = blockchain.ProcGetTransactionByAddr(&reqAddr)
	fmt.Println("ProcGetTransactionByAddr info:", "toaddr", toaddr)
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}

	//toaddr:1
	reqAddr.Addr = toaddr
	reqAddr.Flag = 1
	txs, _ = blockchain.ProcGetTransactionByAddr(&reqAddr)
	fmt.Println("ProcGetTransactionByAddr info:", "addr:1", toaddr)
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}

	//toaddr:2
	reqAddr.Addr = toaddr
	reqAddr.Flag = 2
	txs, _ = blockchain.ProcGetTransactionByAddr(&reqAddr)
	fmt.Println("ProcGetTransactionByAddr info:", "toaddr:2", toaddr)
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}

	chainlog.Info("TestProcGetTransactionByAddr end --------------------")
}

func execprocess(q queue.Queue) {
	//execs
	go func() {
		client := q.Client()
		client.Sub("execs")
		for msg := range client.Recv() {
			chainlog.Info("execprocess exec", "msg.Ty", msg.Ty)
			if msg.Ty == types.EventExecTxList {
				datas := msg.GetData().(*types.ExecTxList)
				var receipts []*types.Receipt
				for i := 0; i < len(datas.Txs); i++ {
					var receipt types.Receipt
					receipt.Ty = 2
					receipt.KV = nil
					receipt.Logs = nil
					//receipt := execute.Exec(datas.Txs[i])
					receipts = append(receipts, &receipt)
				}
				msg.Reply(client.NewMessage("", types.EventReceipts, &types.Receipts{receipts}))
			} else if msg.Ty == types.EventAddBlock {
				datas := msg.GetData().(*types.BlockDetail)
				b := datas.Block
				var kvset types.LocalDBSet

				for i := 0; i < len(b.Txs); i++ {
					tx := b.Txs[i]
					var set types.LocalDBSet
					txhash := tx.Hash()
					//构造txresult 信息保存到db中
					var txresult types.TxResult
					txresult.Height = b.GetHeight()
					txresult.Index = int32(i)
					txresult.Tx = tx
					txresult.Receiptdate = datas.Receipts[i]
					txresult.Blocktime = b.GetBlockTime()
					set.KV = append(set.KV, &types.KeyValue{txhash, types.Encode(&txresult)})
					kvset.KV = append(kvset.KV, set.KV...)
				}
				msg.Reply(client.NewMessage("", types.EventAddBlock, &kvset))

			} else if msg.Ty == types.EventBlockChainQuery {
				data := msg.GetData().(*types.BlockChainQuery)
				driver, err := executor.LoadDriver(data.Driver)
				if err != nil {
					msg.Reply(client.NewMessage("", types.EventBlockChainQuery, err))
				} else {
					driver.SetLocalDB(executor.NewLocalDB(client.Clone()))
					driver.SetStateDB(executor.NewStateDB(client.Clone(), data.StateHash))

					ret, err := driver.Query(data.FuncName, data.Param)
					if err != nil {
						msg.Reply(client.NewMessage("", types.EventBlockChainQuery, err))

					} else {
						msg.Reply(client.NewMessage("", types.EventBlockChainQuery, &ret))
					}
				}
			}
		}
	}()
}

func consensusprocess(q queue.Queue) {
	//execs
	go func() {
		client := q.Client()
		client.Sub("consensus")
		for msg := range client.Recv() {
			chainlog.Info("consensusprocess consensus", "msg.Ty", msg.Ty)
			if msg.Ty == types.EventCheckBlock {
				msg.ReplyErr("EventCheckBlock", nil)
			} else if msg.Ty == types.EventAddBlock {
				//block := msg.GetData().(*types.BlockDetail).Block
				//bc.SetCurrentBlock(block)
			}
		}
	}()
}
