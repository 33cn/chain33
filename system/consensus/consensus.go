package consensus

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type ConsensusCreate func(cfg *types.Consensus) queue.Module

var regConsensus = make(map[string]ConsensusCreate)

func Reg(name string, create ConsensusCreate) {
	if create == nil {
		panic("Consensus: Register driver is nil")
	}
	if _, dup := regConsensus[name]; dup {
		panic("Consensus: Register called twice for driver " + name)
	}
	regConsensus[name] = create
	return
}

func Load(name string) (create ConsensusCreate, err error) {
	if driver, ok := regConsensus[name]; ok {
		return driver, nil
	}
	return nil, types.ErrNotFound
}
