package mempool

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/mempool"
	"github.com/33cn/chain33/types"
)

// New new mempool queue module
func New(cfg *types.Chain33Config) queue.Module {
	mcfg := cfg.GetMConfig().Mempool
	sub := cfg.GetSConfig().Mempool
	con, err := mempool.Load(cfg.GetMConfig().Mempool.Name)
	if err != nil {
		panic("Unsupported mempool type:" + cfg.GetMConfig().Mempool.Name + " " + err.Error())
	}
	subcfg, ok := sub[mcfg.Name]
	if !ok {
		subcfg = nil
	}
	obj := con(mcfg, subcfg)
	return obj
}
