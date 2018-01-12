package coins

import (
	hashlockdb "code.aliyun.com/chain33/chain33/execs/db/hashlock"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.hashlock")

func init() {
	execdrivers.Register("hashlock", newHashlock())
	execdrivers.RegisterAddress("hashlock")
}

type Hashlock struct {
	execdrivers.ExecBase
}

func newHashlock() *Hashlock {
	h := &Hashlock{}
	h.SetChild(h)
	return h
}

//not called currently
func (h *Hashlock) GetName() string {
	return "hashlock"
}

func (h *Hashlock) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.HashlockAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	clog.Info("exec hashlock tx=", "tx=", action)

	actiondb := hashlockdb.NewHashlockAction(h.GetDB(), tx, h.GetAddr(), h.GetBlockTime(), h.GetHeight())

	if action.Ty == types.HashlockActionGenesis {
		//genesis := action.GetGenesis()
		//if genesis.Count <= 0 {
		//return nil, types.ErrTicketCount
		//}
		//new hashlock
		//return actiondb.GenesisInit(genesis)
		return nil, types.ErrActionNotSupport
	} else if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		hlock := action.GetHlock()
		if hlock.Amount <= 0 {
			return nil, types.ErrHashlockAmount
		}
		return actiondb.Hashlocklock(hlock)
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		hunlock := action.GetHunlock()
		return actiondb.Hashlockunlock(hunlock)
	}

	//return error
	return nil, types.ErrActionNotSupport
}
