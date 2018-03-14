package blacklist

import (
	log "github.com/inconshreveable/log15"
	"code.aliyun.com/chain33/chain33/executor/drivers"
	"code.aliyun.com/chain33/chain33/types"
	//"fmt"
)

var clog = log.New("module", "execs.blacklist")

func init() {
	drivers.Register("user.blacklist",newBlackList())
}
type BlackList struct {
	drivers.DriverBase
}

func newBlackList() *BlackList{
	n := &BlackList{}
	n.SetChild(n)
	n.SetIsFree(true)
	return n
}
//func(b *BlackList)putBlackRecord(){
//
//}
//func (b *BlackList)getBlackRecord(){
//
//}
func (n *BlackList) GetName() string {
	return "user.blacklist"
}

func (n *BlackList) GetActionValue(tx *types.Transaction) (*BlackAction, error) {
	action := &BlackAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (n *BlackList) GetActionName(tx *types.Transaction) string {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return "unknow"
	}
	if action.Ty == BlackRecordPut && action.GetRc()!=nil{
		return "BlackRecordPut"
	}else if action.Ty==OrgGet && action.GetOr()!=nil {
		return "OrgGet"
	}else if action.Ty==BlackRecordGet {
		return "BlackRecordGet"
	}else if action.Ty==OrgPut&& action.GetOr()!=nil {
		return "OrgPut"
	}
	return "unknow"
}

func (n *BlackList) GetKVPair(tx *types.Transaction) *types.KeyValue {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return nil
	}
	if action.Ty == BlackRecordPut && action.GetRc()!=nil {
		return &types.KeyValue{[]byte(n.GetName()+action.GetRc().GetRecordId()), []byte(action.GetRc().String())}
	}else if action.Ty== OrgPut && action.GetOr()!=nil {
		return &types.KeyValue{[]byte(n.GetName()+action.GetOr().GetOrgId()), []byte(action.GetOr().String())}
	}

	return nil
}

func (n *BlackList) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	action, err := n.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Debug("exec blacklist tx=", "tx=", action)
	receipt := &types.Receipt{types.ExecOk, nil, nil}
    if action.Ty==BlackRecordPut&&action.GetRc()!=nil||action.Ty==OrgPut&&action.GetOr()!=nil {
		receipt.KV = append(receipt.KV, n.GetKVPair(tx))
	}
	return receipt, nil
}

func (n *BlackList) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	//save tx
	//hash, result := n.GetTx(tx, receipt, index)
	//set.KV = append(set.KV, &types.KeyValue{hash, types.Encode(result)})
	if n.GetKVPair(tx) != nil {
		set.KV = append(set.KV, n.GetKVPair(tx))
	}

	return &set, nil
}

func (n *BlackList) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
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

func (n *BlackList) Query(funcname string, params []byte) (types.Message, error) {
	if funcname == "BlackRecordGet" {
		value := n.GetQueryDB().Get([]byte(n.GetName()+string(params)))
		if value == nil {
			return nil, types.ErrNotFound
		}
		return &types.ReplyString{string(value)}, nil
	} else if funcname == "OrgGet" {
		//fmt.Println(n.GetName()+string(params))
		value:= n.GetQueryDB().Get([]byte(n.GetName()+string(params)))
		if value == nil {
			return  nil, types.ErrNotFound
		}
		return &types.ReplyString{string(value)}, nil
	}
	return nil, types.ErrActionNotSupport
}