package consensus

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/system/consensus"
	"gitlab.33.cn/chain33/chain33/types"
)

func New(cfg *types.Consensus) queue.Module {
	con, err := consensus.Load(cfg.Name)
	if err != nil {
		panic("Unsupported consensus type:" + cfg.Name + " " + err.Error())
	}
	return con(cfg)
}
