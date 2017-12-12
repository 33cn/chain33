package solo

import (
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/util"
	log "github.com/inconshreveable/log15"
)

var genesisAddr = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"

var slog = log.New("module", "solo")

var (
	listSize     int = 10000
	zeroHash     [32]byte
	currentBlock *types.Block
	mulock       sync.Mutex
)

type SoloClient struct {
	qclient queue.IClient
	q       *queue.Queue
}

func NewSolo(cfg *types.Consensus) *SoloClient {
	if cfg.Genesis != "" {
		genesisAddr = cfg.Genesis
	}
	log.Info("Enter consensus solo")
	return &SoloClient{}
}

func (client *SoloClient) SetQueue(q *queue.Queue) {
	log.Info("Enter SetQueue method of consensus")
	client.qclient = q.GetClient()
	client.q = q
	// TODO: solo模式下通过配置判断是否主节点，主节点打包区块，其余节点不用做

	// 程序初始化时，先从blockchain取区块链高度

	height := client.getInitHeight()

	if height == -1 {
		// 创世区块
		newblock := &types.Block{}
		newblock.Height = 0
		// TODO: 下面这些值在创世区块中赋值nil，是否合理？
		newblock.ParentHash = zeroHash[:]
		tx := createGenesisTx()
		newblock.Txs = append(newblock.Txs, tx)
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		client.writeBlock(zeroHash[:], newblock)
	} else {
		block := client.RequestBlock(height)
		setCurrentBlock(block)
	}
	go client.eventLoop()
	go client.createBlock()
}

func (client *SoloClient) Close() {

}

func (client *SoloClient) checkTxDup(txs []*types.Transaction) (transactions []*types.Transaction) {
	var checkHashList types.TxHashList
	txMap := make(map[string]*types.Transaction)
	for _, tx := range txs {
		hash := tx.Hash()
		txMap[string(hash)] = tx
		checkHashList.Hashes = append(checkHashList.Hashes, hash)
	}

	// 发送Hash过后的交易列表给blockchain模块
	hashList := client.qclient.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
	client.qclient.Send(hashList, true)
	dupTxList, _ := client.qclient.Wait(hashList)

	// 取出blockchain返回的重复交易列表
	dupTxs := dupTxList.GetData().(*types.TxHashList).Hashes

	for _, hash := range dupTxs {
		delete(txMap, string(hash))
	}

	for _, tx := range txMap {
		transactions = append(transactions, tx)
	}
	return transactions
}

func (client *SoloClient) createBlock() {
	issleep := true
	for {
		if issleep {
			time.Sleep(time.Second)
		}
		txs := client.RequestTx()
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		//check dup
		txs = client.checkTxDup(txs)
		lastBlock := getCurrentBlock()
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		newblock.Txs = txs
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		client.writeBlock(lastBlock.StateHash, &newblock)
	}
}

// 准备新区块
func (client *SoloClient) eventLoop() {
	// 监听blockchain模块，获取当前最高区块
	client.qclient.Sub("consensus")
	go func() {
		for msg := range client.qclient.Recv() {
			slog.Info("consensus recv", "msg", msg)
			if msg.Ty == types.EventAddBlock {
				block := msg.GetData().(*types.BlockDetail).Block
				setCurrentBlock(block)
			}
		}
	}()
}

// Mempool中取交易列表
func (client *SoloClient) RequestTx() []*types.Transaction {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTxList, listSize)
	client.qclient.Send(msg, true)
	resp, _ := client.qclient.Wait(msg)
	return resp.GetData().(*types.ReplyTxList).GetTxs()
}

func (client *SoloClient) RequestBlock(start int64) *types.Block {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, start, false})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		panic(err)
	}
	blocks := resp.GetData().(*types.BlockDetails)
	return blocks.Items[0].Block
}

// solo初始化时，取一次区块高度放在内存中，后面自增长，不用再重复去blockchain取
func (client *SoloClient) getInitHeight() int64 {

	msg := client.qclient.NewMessage("blockchain", types.EventGetBlockHeight, nil)

	client.qclient.Send(msg, true)
	replyHeight, err := client.qclient.Wait(msg)
	h := replyHeight.GetData().(*types.ReplyBlockHeight).Height
	slog.Info("init = ", "height", h)
	if err != nil {
		panic("error happens when get height from blockchain")
	}
	return h
}

// 向blockchain写区块
func (client *SoloClient) writeBlock(prevHash []byte, block *types.Block) {
	blockdetail, err := util.ExecBlock(client.q, prevHash, block, false)
	if err != nil { //never happen
		panic(err)
	}
	for {
		msg := client.qclient.NewMessage("blockchain", types.EventAddBlockDetail, blockdetail)
		client.qclient.Send(msg, true)
		resp, _ := client.qclient.Wait(msg)

		if resp.GetData().(*types.Reply).IsOk {
			setCurrentBlock(block)
			break
		} else {
			log.Info("Send block to blockchian return fail, retry!")
		}
	}
}

func setCurrentBlock(b *types.Block) {
	mulock.Lock()
	if currentBlock == nil || currentBlock.Height <= b.Height {
		currentBlock = b
	}
	mulock.Unlock()
}

func getCurrentBlock() (b *types.Block) {
	mulock.Lock()
	defer mulock.Unlock()
	return currentBlock
}

func getCurrentHeight() int64 {
	mulock.Lock()
	start := currentBlock.Height
	mulock.Unlock()
	return start
}

func createGenesisTx() *types.Transaction {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = genesisAddr
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{1e8 * types.Coin}
	tx.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	return &tx
}
