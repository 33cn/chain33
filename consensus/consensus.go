package consensus

import (
	"code.aliyun.com/chain33/chain33/consensus/solo"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

func New(consensusType string, q *queue.Queue) {

	if consensusType == "solo" {
		con := solo.NewSolo()
		con.SetQueue(q)
	} else if consensusType == "raft" {
		// TODO:
	} else if consensusType == "pbft" {
		// TODO:
	} else {
		panic("Unsupported consensus type")
	}
}

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
