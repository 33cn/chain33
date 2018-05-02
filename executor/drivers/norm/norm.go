package norm

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.norm")

func init() {
	n := newNorm()
	drivers.Register(n.GetName(), n, 0)
}

type Norm struct {
	drivers.DriverBase
}

func newNorm() *Norm {
	n := &Norm{}
	n.SetChild(n)
	n.SetIsFree(true)
	return n
}

func (n *Norm) GetName() string {
	return "norm"
}

func (n *Norm) Clone() drivers.Driver {
	clone := &Norm{}
	clone.DriverBase = *(n.DriverBase.Clone().(*drivers.DriverBase))
	clone.SetChild(clone)
	clone.SetIsFree(true)
	return clone
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
		return &types.KeyValue{[]byte(action.GetNput().Key), action.GetNput().Value}
	}
	return nil
}

func (n *Norm) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Debug("exec norm tx=", "tx=", action)

	receipt := &types.Receipt{types.ExecOk, nil, nil}
	normKV := n.GetKVPair(tx)
	receipt.KV = append(receipt.KV, normKV)
	return receipt, nil
}

func (n *Norm) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//save tx
	hash, result := n.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	return &set, nil
}

func (n *Norm) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//del tx
	hash, _ := n.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, nil})
	return &set, nil
}

func (n *Norm) Query(funcname string, params []byte) (types.Message, error) {
	if funcname == "NormGet" {
		value, err := n.GetStateDB().Get(params)
		if err != nil {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{string(value)}, nil
	} else if funcname == "NormHas" {
		_, err := n.GetStateDB().Get(params)
		if err != nil {
			return &types.ReplyString{"false"}, err
		}
		return &types.ReplyString{"true"}, nil
	}
	return nil, types.ErrActionNotSupport
}
