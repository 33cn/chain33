package solo

import (
	"log"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
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

	// TODO: solo模式下判断当前节点是否主节点，主节点打包区块，其余节点不用做

	// TODO: Get transaction list size
	listSize := 100

	go func() {
		for {
			// Get transaction list from mempool
			resp, err := client.RequestTx(listSize)
			if err != nil {
				log.Fatal("error happens when get txs from mempool")
			}

			// TODO:Check the duplicated transaction.
			// The efficiency will be relatively low when check the txs one by one??
			//CheckDuplicatedTxs(resp.GetData().(types.ReplyTxList))

			// create the next block
			block := client.ProcessBlock(resp.GetData().(types.ReplyTxList))
			msg := client.qclient.NewMessage("blockchain", types.EventAddBlock, block)
			client.qclient.Send(msg, true)
			resp, err = client.qclient.Wait(msg)

			if err != nil {
				return
			}
			if !(resp.GetData().(types.Reply).IsOk) {
				log.Fatal("Blockchian return fail when write block")
			}
		}
	}()
}

// Get transaction list from mempool
func (client *SoloClient) RequestTx(txNum int) (queue.Message, error) {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTxList, txNum)
	client.qclient.Send(msg, true)
	return client.qclient.Wait(msg)
}

// Create the block
func (client *SoloClient) ProcessBlock(reply types.ReplyTxList) (block *types.Block) {

	msg := client.qclient.NewMessage("blockchian", types.EventGetBlockHeight, nil)
	client.qclient.Send(msg, true)
	replyHeight, err := client.qclient.Wait(msg)

	if err != nil {
		log.Fatal("error happens when get height from blockchain")
		return
	}
	// Get the blockchain height
	height := replyHeight.GetData().(types.ReplyBlockHeight).Height

	newblock := &types.Block{}

	if height == 0 {
		// create the genesis block
		newblock.Height = 0
		newblock.ParentHash = nil
		// TODO: ??
		newblock.Txs = nil
	} else {
		// TODO: It's better supply the function of getBlockbyHeight
		msg = client.qclient.NewMessage("blockchian", types.EventGetBlocks, types.RequestBlocks{height - 1, height})
		client.qclient.Send(msg, true)
		replyblock, err := client.qclient.Wait(msg)

		if err != nil {
			return
		}

		preblock := replyblock.GetData().(types.Blocks).Items[0]
		// TODO: ??
		newblock.ParentHash = preblock.TxHash
		newblock.Height = height
		newblock.Txs = reply.GetTxs()
	}

	return newblock
}
