package consensus

import (
	"code.aliyun.com/chain33/chain33/consensus/drivers/solo"
	"code.aliyun.com/chain33/chain33/consensus/drivers/ticket"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/consensus/drivers/tendermint"
)

func New(cfg *types.Consensus) Consensus {
	consensusType := cfg.Name
	if consensusType == "solo" {
		con := solo.New(cfg)
		return con
	} else if consensusType == "raft" {
		// TODO:
	} else if consensusType == "pbft" {
		// TODO:
		con := tendermint.New(cfg)
		return con
	} else if consensusType == "ticket" {
		t := ticket.New(cfg)
		return t
	}
	panic("Unsupported consensus type")
}

type Consensus interface {
	SetQueue(q *queue.Queue)
	Close()
}
