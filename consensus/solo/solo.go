package solo

import (
	"log"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	//"github.com/golang/protobuf/proto"
)

var height int64

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

	// TODO: Get transaction list size
	listSize := 100

	go func() {
		for {
			// mempool中取交易列表
			resp, err := client.RequestTx(listSize)
			if err != nil {
				log.Fatal("error happens when get txs from mempool")
			}

			// TODO:检查重复交易(一条条通过消息队列查会不会低效？)
			//CheckDuplicatedTxs(resp.GetData().(types.ReplyTxList))

			// 创建新区块
			block := client.ProcessBlock(resp.GetData().(types.ReplyTxList))

			for {
				msg := client.qclient.NewMessage("blockchain", types.EventAddBlock, block)
				client.qclient.Send(msg, true)
				resp, _ = client.qclient.Wait(msg)

				if resp.GetData().(types.Reply).IsOk {
					// 写区块返回成功，高度增长
					height++
					break
				} else {
					log.Fatal("Blockchian return fail when write block,retry!")
				}
			}
		}
	}()
}

// Mempool中取交易列表
func (client *SoloClient) RequestTx(txNum int) (queue.Message, error) {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTxList, txNum)
	client.qclient.Send(msg, true)
	return client.qclient.Wait(msg)
}

// 获取新区块
func (client *SoloClient) ProcessBlock(reply types.ReplyTxList) (block *types.Block) {

	newblock := &types.Block{}

	if height == 0 {
		// 创世区块
		newblock.Height = 0
		// TODO: 下面这些值在创世区块中赋值nil，是否合理？
		newblock.ParentHash = nil
		newblock.Txs = nil
		newblock.TxHash = nil

	} else {
		// TODO: blockchain模块最好能提供getblockbyHeight的函数，看上去比较清晰
		msg := client.qclient.NewMessage("blockchian", types.EventGetBlocks, types.RequestBlocks{height - 1, height})
		client.qclient.Send(msg, true)
		replyblock, err := client.qclient.Wait(msg)

		if err != nil {
			return
		}

		preblock := replyblock.GetData().(types.Blocks).Items[0]

		// TODO:
		newblock.ParentHash = preblock.TxHash
		newblock.Height = height
		newblock.Txs = reply.GetTxs()
	}

	return newblock
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
