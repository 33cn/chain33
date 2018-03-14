package norm

import (
	"code.aliyun.com/chain33/chain33/executor/drivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.norm")

func init() {
	drivers.Register("norm", newNorm())
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
	normKV := &types.KeyValue{[]byte("Key4Norm"), []byte("Value4Norm")}
	receipt.KV = append(receipt.KV, normKV)
	return receipt, nil
	//actiondb := NewNormAction(n.GetDB(), tx, n.GetAddr(), n.GetBlockTime(), n.GetHeight())
	//if action.Ty == types.NormActionPut && action.GetNput() != nil {
	//	return actiondb.Normput(action.GetNput())
	//}
	//return nil, types.ErrActionNotSupport
}

func (n *Norm) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//save tx
	//hash, result := n.GetTx(tx, receipt, index)
	//set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	if n.GetKVPair(tx) != nil {
		set.KV = append(set.KV, n.GetKVPair(tx))
	}

	return &set, nil
}

func (n *Norm) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	pair := n.GetKVPair(tx)
	if pair != nil {
		set.KV = append(set.KV, &types.KeyValue{pair.Key, nil})
	}
	//del tx
	//hash, _ := n.GetTx(tx, receipt, index)
	//set.KV = append(set.KV, &types.KeyValue{hash, nil})

	return &set, nil
}

func (n *Norm) Query(funcname string, params []byte) (types.Message, error) {
	if funcname == "NormGet" {
		value:= n.GetQueryDB().Get(params)
		if value == nil {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{string(value)}, nil
	} else if funcname == "NormHas" {
		value:= n.GetQueryDB().Get(params)
		if value == nil {
			return &types.ReplyString{"false"}, nil
		}
		return &types.ReplyString{"true"}, nil
	}
	return nil, types.ErrActionNotSupport
}
