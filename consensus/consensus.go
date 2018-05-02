package consensus

import (
	"gitlab.33.cn/chain33/chain33/consensus/drivers/pbft"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/raft"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/solo"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/ticket"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint"
)

func New(cfg *types.Consensus) queue.Module {
	consensusType := cfg.Name
	if consensusType == "solo" {
		con := solo.New(cfg)
		return con
	} else if consensusType == "raft" {
		con := raft.NewRaftCluster(cfg)
		return con
	} else if consensusType == "pbft" {
		con := pbft.NewPbft(cfg)
		return con
	} else if consensusType == "ticket" {
		t := ticket.New(cfg)
		return t
	} else if consensusType == "tendermint" {
		con := tendermint.New(cfg)
		return con
	}
	panic("Unsupported consensus type")
}

type Consensus interface {
	SetQueue(q *queue.Queue)
	Close()
}
