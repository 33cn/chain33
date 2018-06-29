package valnode

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/pkg/errors"
	"fmt"
	"gitlab.33.cn/chain33/chain33/consensus/drivers/tendermint"
)

var clog = log.New("module", "execs.valnode")

func Init() {
	drivers.Register(newValNode().GetName(), newValNode, 0)
}

type ValNode struct {
	drivers.DriverBase
}

func newValNode() drivers.Driver {
	n := &ValNode{}
	n.SetChild(n)
	n.SetIsFree(true)
	return n
}

func (n *ValNode) GetName() string {
	return "valnode"
}

func (n *ValNode) GetActionValue(tx *types.Transaction) (*types.ValNodeAction, error) {
	action := &types.ValNodeAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (n *ValNode) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Debug("exec valnode tx", "tx=", action)

	receipt := &types.Receipt{types.ExecErr, nil, nil}
	if action.Ty == types.ValNodeActionUpdate && action.GetNode() != nil{
		if len(action.GetNode().Pubkey) == 0 {
			return nil, errors.New("validator pubkey is empty")
		}
		updateVal := tendermint.ValidatorCache{
			PubKey:action.GetNode().Pubkey,
			Power: action.GetNode().Power,
		}
		tendermint.UpdateValidator2Cache(updateVal)
		receipt.Ty = types.ExecOk
	} else {
		return nil, errors.New(fmt.Sprintf("validator node action type %v not support", action.Ty))
	}

	return receipt, nil
}