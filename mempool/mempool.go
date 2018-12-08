package mempool

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/mempool"
	"github.com/33cn/chain33/types"
)

// New new mempool queue module
func New(cfg *types.Mempool, sub map[string][]byte) queue.Module {
	con, err := mempool.Load(cfg.Name)
	if err != nil {
		panic("Unsupported mempool type:" + cfg.Name + " " + err.Error())
	}
	subcfg, ok := sub[cfg.Name]
	if !ok {
		subcfg = nil
	}
	obj := con(cfg, subcfg)
	return obj
}
