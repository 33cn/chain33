package blockchain

import (
	"bytes"
	"fmt"
	"testing"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

//测试时有两个地方需要使用桩函数，testExecBlock来代替具体的util.ExecBlock
//poolRoutine函数中，以及ProcAddBlockMsg函数中
func init() {
	queue.DisableLog()
}

func initEnv() (*BlockChain, queue.Client) {
	var q = queue.New("channel")
	var cfg types.BlockChain
	cfg.DefCacheSize = 500
	cfg.MaxFetchBlockNum = 100
	cfg.TimeoutSeconds = 5
	cfg.BatchBlockNum = 10
	cfg.Driver = "leveldb"
	cfg.DbPath = "datadir"

	blockchain := New(&cfg)
	blockchain.SetQueueClient(q.Client())
	return blockchain, q.Client()
}

/*
type BlockDetail struct {
	Block    *Block
	Receipts []*ReceiptData
}
type Block struct {
	Version    int64
	ParentHash []byte
	TxHash     []byte
	StateHash  []byte
	Height     int64
	BlockTime  int64
	Txs        []*Transaction
}
type ReceiptData struct {
	Ty   int32
	Logs []*ReceiptLog
}
*/
func ConstructionBlock(Pubkey string, toaddr string, parentHash []byte, height int64, txcount int) *types.Block {
	var block types.Block

	block.BlockTime = height
	block.Height = height
	block.Version = 100
	block.StateHash = common.Hash{}.Bytes()
	block.ParentHash = parentHash

	block.Txs = make([]*types.Transaction, txcount)
	txhashs := make([][]byte, txcount)
	for j := 0; j < txcount; j++ {
		var transaction types.Transaction

		payload := fmt.Sprintf("Payload :%d:%d!", height, j)
		transaction.Payload = []byte(payload)

		var signature1 types.Signature

		signature := fmt.Sprintf("Signature :%d:%d!", height, j)
		signature1.Signature = []byte(signature)
		var pubkey string
		if len(Pubkey) == 0 {
			pubkey = fmt.Sprintf("Pubkey :%d:%d!", height, j)
		} else {
			pubkey = Pubkey
		}
		signature1.Pubkey = []byte(pubkey)

		if len(toaddr) == 0 {
			transaction.To = fmt.Sprintf("toaddr :%d:%d!", height, j)
		} else {
			transaction.To = toaddr
		}
		transaction.Signature = &signature1
		transaction.Execer = []byte("coins")
		block.Txs[j] = &transaction
		txhashs[j] = transaction.Hash()
	}
	block.TxHash = merkle.GetMerkleRoot(txhashs)
	return &block
}

func ConstructionBlockDetail(Pubkey string, toaddr string, parentHash []byte, height int64, txcount int) *types.BlockDetail {
	var blockdetail types.BlockDetail
	var block types.Block

	blockdetail.Receipts = make([]*types.ReceiptData, txcount)

	block.BlockTime = height
	block.Height = height
	block.Version = 100
	block.StateHash = common.Hash{}.Bytes()
	block.ParentHash = parentHash

	block.Txs = make([]*types.Transaction, txcount)
	txhashs := make([][]byte, txcount)
	for j := 0; j < txcount; j++ {
		var transaction types.Transaction

		payload := fmt.Sprintf("Payload :%d:%d!", height, j)
		transaction.Payload = []byte(payload)

		var signature1 types.Signature

		signature := fmt.Sprintf("Signature :%d:%d!", height, j)
		signature1.Signature = []byte(signature)
		var pubkey string
		if len(Pubkey) == 0 {
			pubkey = fmt.Sprintf("Pubkey :%d:%d!", height, j)
		} else {
			pubkey = Pubkey
		}
		signature1.Pubkey = []byte(pubkey)

		if len(toaddr) == 0 {
			transaction.To = fmt.Sprintf("toaddr :%d:%d!", height, j)
		} else {
			transaction.To = toaddr
		}
		transaction.Signature = &signature1
		transaction.Execer = []byte("coins")
		block.Txs[j] = &transaction
		txhashs[j] = transaction.Hash()

		ReceiptData := types.ReceiptData{Ty: 2}
		blockdetail.Receipts[j] = &ReceiptData
	}
	block.TxHash = merkle.GetMerkleRoot(txhashs)
	blockdetail.Block = &block
	return &blockdetail
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

//同步过来的block
func TestProcAddBlockMsg(t *testing.T) {
	chainlog.Info("testProcAddBlockMsg begin --------------------")
	blockchain, client := initEnv()

	execprocess(client)
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
		block := ConstructionBlock("", "", parentHash, i, 5)
		blockchain.ProcAddBlockMsg(false, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
	}

	curheight = blockchain.GetBlockHeight()
	chainlog.Info("testProcAddBlockMsg ", "curheight", curheight)
	block, err = blockchain.GetBlock(curheight)
	if err != nil {
		chainlog.Error("testProcAddBlockMsg GetBlock err", "err", err)
	}
	PrintBlockInfo(block)

	chainlog.Info("testProcAddBlockMsg end --------------------")
	blockchain.Close()
}

//共识模块发过来的block
func TestProcAddBlockDetail(t *testing.T) {
	chainlog.Info("TestProcAddBlockDetail begin --------------------")
	blockchain, client := initEnv()

	execprocess(client)
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
		block := ConstructionBlockDetail("", "", parentHash, i, 5)
		blockchain.ProcAddBlockMsg(true, block)
		parentHash = block.Block.Hash()
	}

	//print
	curheight = blockchain.GetBlockHeight()
	chainlog.Info("TestProcAddBlockDetail ", "curheight", curheight)
	block, err = blockchain.GetBlock(curheight)
	if err != nil {
		chainlog.Error("TestProcAddBlockDetail GetBlock err", "err", err)
	}
	PrintBlockInfo(block)

	chainlog.Info("TestProcAddBlockDetail end --------------------")
	blockchain.Close()
}

func TestGetBlock(t *testing.T) {
	chainlog.Info("testGetBlock begin --------------------")
	blockchain, _ := initEnv()
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("testGetBlock ", "curheight", curheight)
	block, err := blockchain.GetBlock(curheight)
	if err != nil {
		chainlog.Error("testGetBlock GetBlock err", "err", err)
	}
	PrintBlockInfo(block)
	chainlog.Info("testGetBlock end --------------------")
	blockchain.Close()
}

func TestGetTx(t *testing.T) {
	chainlog.Info("TestGetTx begin --------------------")
	blockchain, _ := initEnv()
	//构建txhash
	curheight := blockchain.GetBlockHeight()
	block, err := blockchain.GetBlock(curheight)

	if err != nil {
		chainlog.Error("TestGetTx GetBlock err", "err", err)
	} else {
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
	if err != nil {
		chainlog.Info("testProcQueryTxMsg", "ProcQueryTxMsg err ", err, "txhash", txhash)
		return
	}
	//证明txproof的正确性
	brroothash := merkle.GetMerkleRootFromBranch(txproof.GetProofs(), txhash, uint32(txindex))
	if bytes.Equal(merkleroothash, brroothash) {
		chainlog.Info("testProcQueryTxMsg merkleroothash ==  brroothash  ")
	}
	chainlog.Info("testProcQueryTxMsg!", "GetTx", txproof.GetTx().String())
	chainlog.Info("testProcQueryTxMsg!", "GetReceipt", txproof.GetReceipt().String())

	chainlog.Info("TestProcQueryTxMsg end --------------------")
	blockchain.Close()
}

func TestGetBlocksMsg(t *testing.T) {
	chainlog.Info("TestGetBlocksMsg begin --------------------")
	blockchain, _ := initEnv()
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
	blockchain.Close()
}

func TestProcGetHeadersMsg(t *testing.T) {
	chainlog.Info("TestProcGetHeadersMsg begin --------------------")
	blockchain, _ := initEnv()

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
	blockchain.Close()
}

func TestProcGetLastHeaderMsg(t *testing.T) {
	chainlog.Info("TestProcGetLastHeaderMsg begin --------------------")
	blockchain, _ := initEnv()

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
	blockchain.Close()
}

func TestGetBlockByHash(t *testing.T) {
	chainlog.Info("TestGetBlockByHash begin --------------------")
	blockchain, _ := initEnv()
	curheight := blockchain.GetBlockHeight()
	chainlog.Info("TestGetBlockByHash ", "curheight", curheight)

	block, err := blockchain.GetBlock(curheight - 5)
	blockhash := block.Block.Hash()
	block, err = blockchain.GetBlock(curheight - 4)
	if !bytes.Equal(blockhash, block.Block.ParentHash) {
		fmt.Println("block.ParentHash != prehash: nextParentHash", blockhash, block.Block.ParentHash)
	}
	block, err = blockchain.ProcGetBlockByHashMsg(block.Block.Hash())

	if err == nil {
		PrintBlockInfo(block)
	}
	chainlog.Info("TestGetBlockByHash end --------------------")
	blockchain.Close()
}
func TestProcGetTransactionByHashes(t *testing.T) {
	chainlog.Info("TestProcGetTransactionByHashes begin --------------------")

	blockchain, client := initEnv()
	execprocess(client)

	curheight := blockchain.GetBlockHeight()
	block, _ := blockchain.GetBlock(curheight)
	parentHash := block.Block.Hash()

	chainlog.Info("TestProcGetTransactionByHashes ", "curheight", curheight)
	pubkey := "TestProcGetTransactionByHashes"
	toaddr := "TestProcGetTransactionByHashes-"
	addblockheight := curheight + 10
	txhashs := make([][]byte, 10)
	j := 0
	for i := curheight + 1; i <= addblockheight; i++ {
		block := ConstructionBlock(pubkey, toaddr, parentHash, i, 5)
		blockchain.ProcAddBlockMsg(false, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
		txhashs[j] = block.Txs[0].Hash()
		j++
	}

	txs, err := blockchain.ProcGetTransactionByHashes(txhashs)
	if err != nil {
		chainlog.Info("TestProcGetTransactionByHashes", "ProcGetTransactionByHashes err:", err)
	}
	for _, txdetail := range txs.Txs {
		if txdetail != nil {
			fmt.Println("TestProcGetTransactionByHashes info:.")
			fmt.Println("txresult.Tx:", txdetail.Tx.String())
			fmt.Println("txresult.Receipt:", txdetail.Receipt.String())
		}
	}
	chainlog.Info("TestProcGetTransactionByHashes end --------------------")
	blockchain.Close()
}

func TestProcGetTransactionByAddr(t *testing.T) {
	chainlog.Info("TestProcGetTransactionByAddr begin --------------------")

	blockchain, client := initEnv()
	execprocess(client)

	curheight := blockchain.GetBlockHeight()
	block, _ := blockchain.GetBlock(curheight)
	parentHash := block.Block.Hash()

	chainlog.Info("TestProcGetTransactionByAddr from -> to", "curheight", curheight)
	pubkey := "TestProcGetTransactionByAddr-3333"
	toaddr := "TestProcGetTransactionByAddr-4444"
	addblockheight := curheight + 5
	for i := curheight + 1; i <= addblockheight; i++ {
		block := ConstructionBlock(pubkey, toaddr, parentHash, i, 5)
		blockchain.ProcAddBlockMsg(false, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
	}
	addr := account.PubKeyToAddress([]byte(pubkey))
	address := addr.String()
	txs, _ := blockchain.ProcGetTransactionByAddr(&types.ReqAddr{Addr: address})
	fmt.Println("ProcGetTransactionByAddr info:", "address", address)
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}

	//form <->to
	toaddr = "TestProcGetTransactionByAddr-3333"
	pubkey = "TestProcGetTransactionByAddr-4444"

	curheight = blockchain.GetBlockHeight()
	block, _ = blockchain.GetBlock(curheight)
	parentHash = block.Block.Hash()

	chainlog.Info(" to ->from", "curheight", curheight)
	addblockheight = curheight + 5
	for i := curheight + 1; i <= addblockheight; i++ {
		block := ConstructionBlock(pubkey, toaddr, parentHash, i, 5)
		blockchain.ProcAddBlockMsg(false, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
	}
	addr = account.PubKeyToAddress([]byte(pubkey))

	chainlog.Info(" get txs by addr:TestProcGetTransactionByAddr-4444")
	addrr := "TestProcGetTransactionByAddr-4444"
	txs, _ = blockchain.ProcGetTransactionByAddr(&types.ReqAddr{Addr: addrr})
	fmt.Println("ProcGetTransactionByAddr info:", "addr", addrr)
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}

	chainlog.Info(" get txs by addr:TestProcGetTransactionByAddr-3333")
	addrr = "TestProcGetTransactionByAddr-3333"
	txs, _ = blockchain.ProcGetTransactionByAddr(&types.ReqAddr{Addr: addrr})
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}
	addr = account.PubKeyToAddress([]byte("TestProcGetTransactionByAddr-3333"))
	address = addr.String()
	fromaddr := fmt.Sprintf("%s:0", address)
	chainlog.Info(" get txs by addr:TestProcGetTransactionByAddr-3333:0")
	txs, _ = blockchain.ProcGetTransactionByAddr(&types.ReqAddr{Addr: fromaddr})
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("ProcGetTransactionByAddr info:.")
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}
	chainlog.Info(" get txs by addr:TestProcGetTransactionByAddr-3333:1")
	addrr = "TestProcGetTransactionByAddr-3333:1"
	txs, _ = blockchain.ProcGetTransactionByAddr(&types.ReqAddr{Addr: addrr})
	if txs != nil {
		for _, txresult := range txs.TxInfos {
			if txresult != nil {
				fmt.Println("ProcGetTransactionByAddr info:.")
				fmt.Println("TxInfo.Index:", txresult.Index)
				fmt.Println("TxInfo.Height:", txresult.Height)
				fmt.Println("TxInfo.Hash:", txresult.GetHash())
			}
		}
	}
	chainlog.Info("TestProcGetTransactionByAddr end --------------------")
	blockchain.Close()
}

var CurHeight int64 = 0

//广播一个block
func addBlock(blockchain *BlockChain, parentHash []byte, addblockheight int64) {
	chainlog.Info("addBlock", "addblockheight", addblockheight)

	for i := addblockheight; i <= addblockheight; i++ {
		block := ConstructionBlock("", "", parentHash, i, 5)
		blockchain.ProcAddBlockMsg(true, &types.BlockDetail{block, nil})
		parentHash = block.Hash()
	}
}

//
//type Peer struct {
//	Addr        string  `protobuf:"bytes,1,opt,name=addr" json:"addr,omitempty"`
//	Port        int32   `protobuf:"varint,2,opt,name=port" json:"port,omitempty"`
//	Name        string  `protobuf:"bytes,3,opt,name=name" json:"name,omitempty"`
//	MempoolSize int32   `protobuf:"varint,4,opt,name=mempoolSize" json:"mempoolSize,omitempty"`
//	Header      *Header `protobuf:"bytes,5,opt,name=header" json:"header,omitempty"`
//}
//
func constructpeerlist() *types.PeerList {

	chainlog.Info("constructpeerlist", "CurHeight", CurHeight)
	var peerlist types.PeerList
	count := 10
	peerlist.Peers = make([]*types.Peer, count)

	for i := 1; i <= count; i++ {
		var peer types.Peer
		var header types.Header
		peer.Addr = "Addr"
		header.Height = CurHeight + int64(i)
		peer.Header = &header
		peerlist.Peers[i-1] = &peer
	}
	return &peerlist
}

func execprocess(client queue.Client) {
	//execs
	go func() {
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
			}
		}
	}()
}

// test
func testExecBlock(q queue.Queue, prevStateRoot []byte, block *types.Block, errReturn bool) (*types.BlockDetail, error) {
	var blockdetal types.BlockDetail
	blockdetal.Block = block
	var rdata []*types.ReceiptData
	chainlog.Info("testExecBlock", "height", block.Height)
	for index, _ := range block.Txs {
		var receiptdata types.ReceiptData
		var receiptLog types.ReceiptLog
		var receiptLogdata []*types.ReceiptLog

		Log := fmt.Sprintf("Log :%d:%d!", block.Height, index)
		receiptLog.Log = []byte(Log)
		receiptLog.Ty = 2

		receiptLogdata = append(receiptLogdata, &receiptLog)

		receiptdata.Logs = receiptLogdata
		receiptdata.Ty = 2
		rdata = append(rdata, &receiptdata)
	}
	blockdetal.Receipts = rdata
	return &blockdetal, nil
}
