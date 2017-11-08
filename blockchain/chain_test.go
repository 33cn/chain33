package blockchain

import (
	"bytes"
	"fmt"
	"testing"

	//"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

func Test_BlockChainModule(t *testing.T) {
	//b.StopTimer()

	q := queue.New("channel")

	fmt.Println("loading blockchain module")
	chain := New()
	chain.SetQueue(q)
	//b.StartTimer()
	fmt.Println("start")

	Testroutline(chain)
}

func Testroutline(chain *BlockChain) {
	//测试add block
	chainlog.Info("testProcAddBlockMsg begin --------------------")
	testProcAddBlockMsg(chain)

	//测试testGetBlock 获取当前高度的block信息
	chainlog.Info("testGetBlock begin --------------------")
	testGetBlock(chain)

	//测试当前高度的index=0的tx
	chainlog.Info("testGetTx begin --------------------")
	testGetTx(chain)

	//测试当前高度的tx重复
	chainlog.Info("testGetTxHashList begin --------------------")
	testGetTxHashList(chain)

	// 测试txproof已经verify
	chainlog.Info("testProcQueryTxMsg begin --------------------")
	testProcQueryTxMsg(chain)

	//测试GetBlocksMsg
	chainlog.Info("testGetBlocksMsg begin --------------------")
	testGetBlocksMsg(chain)

	// 测试AddBlocks
	chainlog.Info("testAddBlocks begin --------------------")
	testAddBlocks(chain)
}

func testProcAddBlockMsg(chain *BlockChain) {
	curheight := chain.GetBlockHeight()

	addblockheight := curheight + 1
	chainlog.Info("testProcAddBlockMsg", "addblockheight", addblockheight)

	for i := addblockheight; i <= addblockheight; i++ {
		var block types.Block
		parenthash := fmt.Sprintf("ParentHash :%d!", i)
		block.ParentHash = []byte(parenthash)
		block.BlockTime = int64(i)
		block.Height = int64(i)
		block.Version = 100

		total := 5
		block.Txs = make([]*types.Transaction, total)
		txhashs := make([][]byte, total)
		for j := 0; j < total; j++ {
			var transaction types.Transaction
			payload := fmt.Sprintf("Payload :%d:%d!", i, j)
			signature := fmt.Sprintf("Signature :%d:%d!", i, j)
			account := fmt.Sprintf("Account :%d:%d!", i, j)

			transaction.Payload = []byte(payload)
			transaction.Signature = []byte(signature)
			transaction.Account = []byte(account)

			block.Txs[j] = &transaction
			txhashs[j] = transaction.Hash()
		}

		block.TxHash = merkle.GetMerkleRoot(txhashs)

		chain.ProcAddBlockMsg(&block)
	}
}

func testGetBlock(blockchain *BlockChain) {
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testGetBlock ", "curheight", curheight)
	block, err := blockchain.GetBlock(curheight)
	if err == nil {
		fmt.Println("testGetBlock info:.")
		fmt.Println("block.ParentHash:", string(block.ParentHash))
		fmt.Println("block.TxHash:", block.TxHash)
		fmt.Println("block.BlockTime:", block.BlockTime)
		fmt.Println("block.Height:", block.Height)
		fmt.Println("block.Version:", block.Version)
		fmt.Println("txs len:", len(block.Txs))
		for _, tx := range block.Txs {
			transaction := tx
			fmt.Println("tx.Payload:", string(transaction.Payload))
			fmt.Println("tx.Signature:", string(transaction.Signature))
			fmt.Println("tx.Account:", string(transaction.Account))
		}
	}
}

func testGetTx(blockchain *BlockChain) {
	//构建txhash
	curheight := blockchain.GetBlockHeight()
	j := 0
	var transaction types.Transaction
	payload := fmt.Sprintf("Payload :%d:%d!", curheight, j)
	signature := fmt.Sprintf("Signature :%d:%d!", curheight, j)
	account := fmt.Sprintf("Account :%d:%d!", curheight, j)

	transaction.Payload = []byte(payload)
	transaction.Signature = []byte(signature)
	transaction.Account = []byte(account)

	txhash := transaction.Hash()
	fmt.Println("testGetTx curheight:", curheight)

	txresult, err := blockchain.GetTxResultFromDb(txhash)
	if err == nil && txresult != nil {
		fmt.Println("testGetTx info:.")
		fmt.Println("txresult.Index:", txresult.Index)
		fmt.Println("txresult.Height:", txresult.Height)

		fmt.Println("tx.Payload:", string(txresult.Tx.Payload))
		fmt.Println("tx.Signature:", string(txresult.Tx.Signature))
		fmt.Println("tx.Account:", string(txresult.Tx.Account))

	}
}

func testGetTxHashList(blockchain *BlockChain) {
	var txhashlist types.TxHashList
	total := 10
	Txs := make([]*types.Transaction, total)

	// 构建当前高度的tx信息
	i := blockchain.GetBlockHeight()
	for j := 0; j < total; j++ {
		var transaction types.Transaction
		payload := fmt.Sprintf("Payload :%d:%d!", i, j)
		signature := fmt.Sprintf("Signature :%d:%d!", i, j)
		account := fmt.Sprintf("Account :%d:%d!", i, j)

		transaction.Payload = []byte(payload)
		transaction.Signature = []byte(signature)
		transaction.Account = []byte(account)

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
}

func testProcQueryTxMsg(blockchain *BlockChain) {
	curheight := blockchain.GetBlockHeight()
	var merkleroothash []byte
	var txhash []byte
	var txindex int

	//获取当前高度的block信息
	block, err := blockchain.GetBlock(curheight)
	if err == nil {
		merkleroothash = block.TxHash
		fmt.Println("block.TxHash:", block.TxHash)

		fmt.Println("txs len:", len(block.Txs))
		for index, transaction := range block.Txs {
			fmt.Println("tx.Payload:", string(transaction.Payload))
			fmt.Println("tx.Signature:", string(transaction.Signature))
			fmt.Println("tx.Account:", string(transaction.Account))
			txhash = transaction.Hash()
			txindex = index
		}
	}
	txproof, err := blockchain.ProcQueryTxMsg(txhash)
	if err != nil {
		chainlog.Info("testProcQueryTxMsg", "ProcQueryTxMsg err ", err, "txhash", txhash)
		return
	}
	//证明txproof的正确性
	brroothash := merkle.GetMerkleRootFromBranch(txproof.Hashs, txhash, uint32(txindex))
	if bytes.Equal(merkleroothash, brroothash) {
		chainlog.Info("testProcQueryTxMsg merkleroothash ==  brroothash  ")
	}

}

func testGetBlocksMsg(blockchain *BlockChain) {
	curheight := blockchain.GetBlockHeight()
	var reqBlock types.RequestBlocks
	if curheight >= 5 {
		reqBlock.Start = curheight - 5
	}
	reqBlock.End = curheight

	blocks, err := blockchain.ProcGetBlocksMsg(&reqBlock)
	if err == nil && blocks != nil {
		for _, block := range blocks.Items {
			fmt.Println("testGetBlocksMsg info:.")
			fmt.Println("block.ParentHash:", string(block.ParentHash))
			fmt.Println("block.TxHash:", block.TxHash)
			fmt.Println("block.BlockTime:", block.BlockTime)
			fmt.Println("block.Height:", block.Height)
			fmt.Println("txs len:", len(block.Txs))
			for _, tx := range block.Txs {
				transaction := tx
				fmt.Println("tx.Payload:", string(transaction.Payload))
				fmt.Println("tx.Signature:", string(transaction.Signature))
				fmt.Println("tx.Account:", string(transaction.Account))
			}

		}
	}
}

func testAddBlocks(blockchain *BlockChain) {

	curheight := blockchain.GetBlockHeight()
	fmt.Println("testAddBlocks start: curheight ", curheight)

	end := curheight + 5
	start := curheight + 1

	fmt.Println("testAddBlocks start: end ", start, end)
	var blocks types.Blocks
	count := end - start + 1
	blocks.Items = make([]*types.Block, count)

	j := 0
	for i := start; i <= end; i++ {
		var block types.Block
		parenthash := fmt.Sprintf("ParentHash :%d!", i)
		block.ParentHash = []byte(parenthash)
		block.BlockTime = int64(i)
		block.Height = int64(i)

		var transaction types.Transaction
		payload := fmt.Sprintf("Payload :%d!", i)
		signature := fmt.Sprintf("Signature :%d!", i)
		account := fmt.Sprintf("Account :%d!", i)

		transaction.Payload = []byte(payload)
		transaction.Signature = []byte(signature)
		transaction.Account = []byte(account)

		block.Txs = make([]*types.Transaction, 1)
		block.Txs[0] = &transaction

		txhashs := make([][]byte, 1)
		txhashs[0] = transaction.Hash()
		block.TxHash = merkle.GetMerkleRoot(txhashs)
		blocks.Items[j] = &block
		j++
	}
	blockchain.ProcAddBlocksMsg(&blocks)
}
