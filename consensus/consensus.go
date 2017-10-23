package consensus

import "code.aliyun.com/chain33/chain33/queue"

type Consumer interface {
	SetQueue(q *queue.Queue)
	RequestTx(txNum int64) queue.Message
}

//type ReadLeadger interface {
//	CheckDuplicatedTxs() bool
//}

type Communicator interface {
	ProcessBlock(reply types.ReplyTxList) (block *types.Block)
}
