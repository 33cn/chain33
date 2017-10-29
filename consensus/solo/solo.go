package solo

import (
	"log"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

var (
	height int64 = 0
	// 交易列表大小
	listSize int = 10000
)

type SoloClient struct {
	qclient queue.IClient
}

func NewSolo() *SoloClient {
	log.Println("consensus/solo")
	return &SoloClient{}
}

func (client *SoloClient) SetQueue(q *queue.Queue) {
	client.qclient = q.GetClient()

	// TODO: solo模式下通过配置判断是否主节点，主节点打包区块，其余节点不用做

	// 程序初始化时，先从blockchain取区块链高度
	height = client.getInitHeight()

	if height == 0 {
		// 创世区块
		newblock := &types.Block{}

		newblock.Height = 0
		// TODO: 下面这些值在创世区块中赋值nil，是否合理？
		newblock.ParentHash = nil
		newblock.Txs = nil
		newblock.TxHash = nil

		client.writeBlock(newblock)

	} else {
		// TODO: 配置文件中读取
		listSize = 100

		txsChannel := make(chan types.ReplyTxList)
		// 从mempool中取交易列表
		go client.RequestTx(txsChannel)

		// 对交易列表验重，并打包发给blockchain
		client.ProcessBlock(txsChannel)
	}
}

// Mempool中取交易列表
func (client *SoloClient) RequestTx(txChannel chan<- types.ReplyTxList) {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	go func() {
		for {
			msg := client.qclient.NewMessage("mempool", types.EventTxList, listSize)
			client.qclient.Send(msg, true)
			resp, _ := client.qclient.Wait(msg)
			txChannel <- resp.GetData().(types.ReplyTxList)
		}
	}()
}

// 准备新区块
func (client *SoloClient) ProcessBlock(txChannel <-chan types.ReplyTxList) {

	// 监听blockchain模块，获取当前最高区块
	client.qclient.Sub("blockchain")

	go func() {
		for msg := range client.qclient.Recv() {
			if msg.Ty == types.EventAddBlock {
				var block *types.Block
				block = msg.GetData().(*types.Block)
				preHeight := block.Height

				// solo模式下，只有solo节点打包区块。每向blockchain成功发送完一个区块（不管blockchain当时有没写成功）
				// 高度计数器height自动增长1，当height == preHeight+1的情况下，说明之前所有区块都已经写完成
				if height == preHeight+1 {

					txlist := <-txChannel

					if len(txlist.Txs) > 0 {
						// TODO: 交易验重,将交易列表hash后发给blockchain，blockchain返回重复交易hash
					}

					// 打包新区块
					newblock := &types.Block{}
					newblock.ParentHash = block.TxHash
					newblock.Height = height
					newblock.Txs = txlist.Txs
					// TODO: 交易hash设置，等待API
					//newblock.TxHash =

					client.writeBlock(newblock)

				} else if height > preHeight+1 {
					log.Println("Not the latest block, continue.")

				} else {
					log.Println("This will not happen")
				}
			}
		}
	}()
}

// solo初始化时，取一次区块高度放在内存中，后面自增长，不用再重复去blockchain取
func (client *SoloClient) getInitHeight() int64 {

	msg := client.qclient.NewMessage("blockchian", types.EventGetBlockHeight, nil)
	client.qclient.Send(msg, true)
	replyHeight, err := client.qclient.Wait(msg)

	if err != nil {
		panic("error happens when get height from blockchain")
	}

	return replyHeight.GetData().(types.ReplyBlockHeight).Height
}

// 向blockchain写区块
func (client *SoloClient) writeBlock(block *types.Block) {
	for {
		msg := client.qclient.NewMessage("blockchain", types.EventAddBlock, block)
		client.qclient.Send(msg, true)
		resp, _ := client.qclient.Wait(msg)

		if resp.GetData().(types.Reply).IsOk {
			// 写区块返回成功，高度增长
			height++
			break
		} else {
			log.Fatal("Send block to blockchian return fail,retry!")
		}
	}
}
