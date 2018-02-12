package norm

import (
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.norm")

func init() {
	execdrivers.Register("norm", newNorm())
	execdrivers.RegisterAddress("norm")
}

type Norm struct {
	execdrivers.ExecBase
}

func newNorm() *Norm {
	n := &Norm{}
	n.SetChild(n)
	return n
}

func (n *Norm) GetName() string {
	return "norm"
}

func (n *Norm) GetActionValue(tx *types.Transaction) (*types.NormAction, error) {
	action := &types.NormAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (n *Norm) GetActionName(tx *types.Transaction) string {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return "unknow"
	}
	if action.Ty == types.NormActionPut && action.GetNput() != nil {
		return "put"
	}
	return "unknow"
}

func (n *Norm) GetKVPair(tx *types.Transaction) *types.KeyValue {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return nil
	}
	if action.Ty == types.NormActionPut && action.GetNput() != nil {
		return &types.KeyValue{[]byte(action.GetNput().Key), []byte(action.GetNput().Value)}
	}
	return nil
}

func (n *Norm) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Info("exec norm tx=", "tx=", action)
	actiondb := NewNormAction(n.GetDB(), tx, n.GetAddr(), n.GetBlockTime(), n.GetHeight())
	if action.Ty == types.NormActionPut && action.GetNput() != nil {
		return actiondb.Normput(action.GetNput())
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (n *Norm) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//save tx
	hash, result := n.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	if n.GetKVPair(tx) != nil {
		set.KV = append(set.KV, n.GetKVPair(tx))
	}

	return &set, nil
}

func (n *Norm) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//del tx
	hash, _ := n.GetTx(tx, receipt, index)
	pair := n.GetKVPair(tx)
	if pair != nil {
		set.KV = append(set.KV, &types.KeyValue{pair.Key, nil})
	}
	set.KV = append(set.KV, &types.KeyValue{hash, nil})

	return &set, nil
}
