package consensus

import "code.aliyun.com/chain33/chain33/queue"

// 从mempool获取消息接口
type Consumer interface {
	SetQueue(q *queue.Queue)
	RequestTx(txNum int64) queue.Message
}

// 查询区块相关信息接口
//type ReadLeadger interface {
//	GetPreBlock()
//	QueryTransaction()
//}

// 广播消息接口
type Communicator interface {
	ProcessBlock(reply types.ReplyTxList) (block *types.Block)
}
