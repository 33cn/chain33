package para

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.para")
var keyPart string

func Init() {
	keyPart = "mavl-user.para-"
	drivers.Register("user."+newPara().GetName(), newPara, 0)
}

type Para struct {
	drivers.DriverBase
}

func newPara() drivers.Driver {
	p := &Para{}
	p.SetChild(p)
	p.SetIsFree(true)
	return p
}

func (p *Para) GetName() string {
	return "para"
}

func (p *Para) GetActionValue(tx *types.Transaction) (*types.ParaAction, error) {
	action := &types.ParaAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (p *Para) GetActionName(tx *types.Transaction) string {
	action, err := p.GetActionValue(tx)
	if err != nil {
		return "unknow"
	}
	if action.Ty == types.ParaActionPut && action.GetPut() != nil {
		return "put"
	}
	return "unknow"
}

func Key(str string) (key []byte) {
	key = append(key, []byte(keyPart)...)
	key = append(key, str...)
	return key
}

func (p *Para) GetKVPair(tx *types.Transaction) *types.KeyValue {
	action, err := p.GetActionValue(tx)
	if err != nil {
		return nil
	}
	if action.Ty == types.ParaActionPut && action.GetPut() != nil {
		return &types.KeyValue{Key(action.GetPut().Key), action.GetPut().Value}
	}
	return nil
}

func (p *Para) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	action, err := p.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Debug("exec para tx=", "tx=", action)

	receipt := &types.Receipt{types.ExecOk, nil, nil}
	paraKV := p.GetKVPair(tx)
	receipt.KV = append(receipt.KV, paraKV)
	return receipt, nil
}

func (p *Para) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//save tx
	hash, result := p.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	return &set, nil
}

func (p *Para) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//del tx
	hash, _ := p.GetTx(tx, receipt, index)
	set.KV = append(set.KV, &types.KeyValue{hash, nil})
	return &set, nil
}

func (p *Para) Query(funcname string, params []byte) (types.Message, error) {
	str := string(params)
	if funcname == "ParaGet" {
		value, err := p.GetStateDB().Get(Key(str))
		if err != nil {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{string(value)}, nil
	} else if funcname == "ParaHas" {
		_, err := p.GetStateDB().Get(Key(str))
		if err != nil {
			return &types.ReplyString{"false"}, err
		}
		return &types.ReplyString{"true"}, nil
	}
	return nil, types.ErrActionNotSupport
}
