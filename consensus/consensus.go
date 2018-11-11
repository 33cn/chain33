package consensus

import (
	"reflect"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/types"
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
