package consensus

import (
	"reflect"

	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/system/consensus"
	"gitlab.33.cn/chain33/chain33/types"
)

func New(cfg *types.Consensus, sub map[string][]byte) queue.Module {
	con, err := consensus.Load(cfg.Name)
	if err != nil {
		panic("Unsupported consensus type:" + cfg.Name + " " + err.Error())
	}
	subcfg, ok := sub[cfg.Name]
	if !ok {
		subcfg = nil
	}
	obj := con(cfg, subcfg)
	consensus.QueryData.SetThis(cfg.Name, reflect.ValueOf(obj))
	return obj
}
