package consensus

import (
	"code.aliyun.com/chain33/chain33/consensus/solo"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

func New(cfg *types.Consensus) Consumer {
	consensusType := cfg.Name
	if consensusType == "solo" {
		con := solo.NewSolo(cfg)
		return con
	} else if consensusType == "raft" {
		// TODO:
	} else if consensusType == "pbft" {
		// TODO:
	}
	panic("Unsupported consensus type")
}

type Consumer interface {
	SetQueue(q *queue.Queue)
}

//type ReadLeadger interface {
//	CheckDuplicatedTxs() bool
//}

type Communicator interface {
	ProcessBlock(reply types.ReplyTxList) (block *types.Block)
}
