package blockchain

import (
	"bytes"
	"fmt"
	"testing"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

func init() {
	queue.DisableLog()
}

func initEnv() (*BlockChain, *queue.Queue) {
	var q = queue.New("channel")
	blockchain := New()
	blockchain.SetQueue(q)
	return blockchain, q
}

func TestProcAddBlockMsg(t *testing.T) {
	chainlog.Info("testProcAddBlockMsg begin --------------------")
	blockchain, _ := initEnv()
	curheight := blockchain.GetBlockHeight()

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
			//account := fmt.Sprintf("Account :%d:%d!", i, j)

			transaction.Payload = []byte(payload)
			var signature1 types.Signature
			signature1.Signature = []byte(signature)
			transaction.Signature = &signature1
			//transaction.Account = []byte(account)

			block.Txs[j] = &transaction
			txhashs[j] = transaction.Hash()
		}

		block.TxHash = merkle.GetMerkleRoot(txhashs)

		blockchain.ProcAddBlockMsg(&block)
	}
	chainlog.Info("testProcAddBlockMsg end --------------------")
	blockchain.Close()
}

func TestGetBlock(t *testing.T) {
	chainlog.Info("testGetBlock begin --------------------")
	blockchain, _ := initEnv()
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
			fmt.Println("tx.Signature:", transaction.Signature.String())
			//fmt.Println("tx.Account:", string(transaction.Account))
		}
	}
	chainlog.Info("testGetBlock end --------------------")
	blockchain.Close()
}

func TestGetTx(t *testing.T) {
	chainlog.Info("TestGetTx begin --------------------")
	blockchain, _ := initEnv()
	//构建txhash
	curheight := blockchain.GetBlockHeight()
	j := 0
	var transaction types.Transaction
	payload := fmt.Sprintf("Payload :%d:%d!", curheight, j)
	signature := fmt.Sprintf("Signature :%d:%d!", curheight, j)
	//account := fmt.Sprintf("Account :%d:%d!", curheight, j)

	transaction.Payload = []byte(payload)
	var signature1 types.Signature
	signature1.Signature = []byte(signature)
	transaction.Signature = &signature1

	//transaction.Account = []byte(account)

	txhash := transaction.Hash()
	fmt.Println("testGetTx curheight:", curheight)

	txresult, err := blockchain.GetTxResultFromDb(txhash)
	if err == nil && txresult != nil {
		fmt.Println("testGetTx info:.")
		fmt.Println("txresult.Index:", txresult.Index)
		fmt.Println("txresult.Height:", txresult.Height)

		fmt.Println("tx.Payload:", string(txresult.Tx.Payload))
		fmt.Println("tx.Signature:", txresult.Tx.Signature.String())
		//fmt.Println("tx.Account:", string(txresult.Tx.Account))

	}
	chainlog.Info("TestGetTx end --------------------")
	blockchain.Close()
}

func TestGetTxHashList(t *testing.T) {
	chainlog.Info("TestGetTxHashList begin --------------------")
	blockchain, _ := initEnv()
	var txhashlist types.TxHashList
	total := 10
	Txs := make([]*types.Transaction, total)

	// 构建当前高度的tx信息
	i := blockchain.GetBlockHeight()
	for j := 0; j < total; j++ {
		var transaction types.Transaction
		payload := fmt.Sprintf("Payload :%d:%d!", i, j)
		signature := fmt.Sprintf("Signature :%d:%d!", i, j)
		//account := fmt.Sprintf("Account :%d:%d!", i, j)

		transaction.Payload = []byte(payload)
		var signature1 types.Signature
		signature1.Signature = []byte(signature)
		transaction.Signature = &signature1
		//transaction.Account = []byte(account)

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
	blockchain.Close()
}

func TestProcQueryTxMsg(t *testing.T) {
	chainlog.Info("TestProcQueryTxMsg begin --------------------")
	blockchain, _ := initEnv()
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
			fmt.Println("tx.Signature:", transaction.Signature.String())
			//fmt.Println("tx.Account:", string(transaction.Account))
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
	chainlog.Info("TestProcQueryTxMsg end --------------------")
	blockchain.Close()
}

func TestGetBlocksMsg(t *testing.T) {
	chainlog.Info("TestGetBlocksMsg begin --------------------")
	blockchain, _ := initEnv()
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
				fmt.Println("tx.Signature:", transaction.Signature.String())
				//fmt.Println("tx.Account:", string(transaction.Account))
			}

		}
	}
	chainlog.Info("TestGetBlocksMsg end --------------------")
	blockchain.Close()

}

func TestProcGetHeadersMsg(t *testing.T) {
	chainlog.Info("TestProcGetHeadersMsg begin --------------------")
	blockchain, _ := initEnv()

	curheight := blockchain.GetBlockHeight()
	var reqBlock types.RequestBlocks
	if curheight >= 5 {
		reqBlock.Start = curheight - 5
	}
	reqBlock.End = curheight

	blockheaders, err := blockchain.ProcGetHeadersMsg(&reqBlock)
	if err == nil && blockheaders != nil {
		for _, head := range blockheaders.Items {
			fmt.Println("TestProcGetHeadersMsg info:.")
			fmt.Println("head.ParentHash:", string(head.ParentHash))
			fmt.Println("head.TxHash:", head.TxHash)
			fmt.Println("head.BlockTime:", head.BlockTime)
			fmt.Println("head.Height:", head.Height)
			fmt.Println("head.Version:", head.Version)
		}
	}
	chainlog.Info("TestProcGetHeadersMsg end --------------------")
	blockchain.Close()
}

// 测试addblock，addblocks以及fetchblocks
func TestFetchBlock(t *testing.T) {
	chainlog.Info("TestFetchBlock begin --------------------")
	//blockchain.SetQueue(q)
	blockchain, q := initEnv()

	curheight := blockchain.GetBlockHeight()
	chainlog.Info("TestFetchBlock", "curheight", curheight)

	//添加一个curheight +10的block给blockchain模块，从而出发blockchain向 p2p模块发送FetchBlock消息
	addheight := curheight + 10
	go addBlock(blockchain, addheight)
	var requestblocks *types.RequestBlocks
	//p2p
	go func() {
		client := q.GetClient()
		client.Sub("p2p")
		for msg := range client.Recv() {
			if msg.Ty == types.EventFetchBlocks {
				chainlog.Info("TestFetchBlock", "msg.Ty", msg.Ty)
				requestblocks = (msg.Data).(*types.RequestBlocks)
				go addBlocks(blockchain, requestblocks)
				msg.Reply(client.NewMessage("blockchain", types.EventAddBlocks, &types.Reply{true, nil}))
			}
		}
	}()

	for {
		curheight := blockchain.GetBlockHeight()
		if addheight == curheight {
			break
		}
	}
	chainlog.Info("TestFetchBlock end --------------------")
	blockchain.Close()
}

func addBlock(blockchain *BlockChain, addblockheight int64) {
	chainlog.Info("addBlock", "addblockheight", addblockheight)

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
			//account := fmt.Sprintf("Account :%d:%d!", i, j)

			transaction.Payload = []byte(payload)
			var signature1 types.Signature
			signature1.Signature = []byte(signature)
			transaction.Signature = &signature1
			//transaction.Account = []byte(account)

			block.Txs[j] = &transaction
			txhashs[j] = transaction.Hash()
		}

		block.TxHash = merkle.GetMerkleRoot(txhashs)

		blockchain.ProcAddBlockMsg(&block)
	}
}

func addBlocks(blockchain *BlockChain, requestblock *types.RequestBlocks) {
	if requestblock == nil {
		chainlog.Info("addBlocks requestblock is null")
		return
	}
	end := requestblock.End
	start := requestblock.Start
	chainlog.Info("addBlocks", "start", start, "end", end)
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
		//account := fmt.Sprintf("Account :%d!", i)

		transaction.Payload = []byte(payload)
		var signature1 types.Signature
		signature1.Signature = []byte(signature)
		transaction.Signature = &signature1
		//transaction.Account = []byte(account)

		block.Txs = make([]*types.Transaction, 1)
		block.Txs[0] = &transaction

		txhashs := make([][]byte, 1)
		txhashs[0] = transaction.Hash()
		block.TxHash = merkle.GetMerkleRoot(txhashs)
		blocks.Items[j] = &block
		j++
	}
	blockchain.ProcAddBlocksMsg(&blocks)
	chainlog.Info("addBlocks ok")
}
