package executor

import (
	log "github.com/inconshreveable/log15"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.norm")

func Init() {
	drivers.Register(GetName(), newNorm, 0)
}

func GetName() string {
	return newNorm().GetName()
}

type Norm struct {
	drivers.DriverBase
}

func newNorm() drivers.Driver {
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

func Key(str string) (key []byte) {
	key = append(key, []byte("mavl-norm-")...)
	key = append(key, str...)
	return key
}

func (n *Norm) GetKVPair(tx *types.Transaction) *types.KeyValue {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return nil
	}
	if action.Ty == types.NormActionPut && action.GetNput() != nil {
		return &types.KeyValue{Key(action.GetNput().Key), action.GetNput().Value}
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

func (n *Norm) Query(funcname string, params []byte) (types.Message, error) {
	str := string(params)
	if funcname == "NormGet" {
		value, err := n.GetStateDB().Get(Key(str))
		if err != nil {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{string(value)}, nil
	} else if funcname == "NormHas" {
		_, err := n.GetStateDB().Get(Key(str))
		if err != nil {
			return &types.ReplyString{"false"}, err
		}
		return &types.ReplyString{"true"}, nil
	}
	return nil, types.ErrActionNotSupport
}
