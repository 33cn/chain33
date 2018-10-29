package store

//store package store the world - state data
import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/system/store"
	"gitlab.33.cn/chain33/chain33/types"
)

func New(cfg *types.Store, sub map[string][]byte) queue.Module {
	s, err := store.Load(cfg.Name)
	if err != nil {
		panic("Unsupported store type:" + cfg.Name + " " + err.Error())
	}
	subcfg, ok := sub[cfg.Name]
	if !ok {
		subcfg = nil
	}
	return s(cfg, subcfg)
}
