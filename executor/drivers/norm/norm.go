package norm

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"encoding/asn1"
	"fmt"
)

var clog = log.New("module", "execs.norm")

func Init() {
	drivers.Register(newNorm().GetName(), newNorm, 0)
}

type Norm struct {
	drivers.DriverBase
	certValidateCache map[string]error
}

func newNorm() drivers.Driver {
	n := &Norm{}
	n.SetChild(n)
	n.SetIsFree(true)

	n.certValidateCache = make(map[string]error)
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

func (n *Norm) CheckTx(tx *types.Transaction, index int) error {
	// 基类检查
	err := n.DriverBase.CheckTx(tx, index)
	if err != nil {
		return err
	}

	// auth模块关闭则返回
	if !types.IsAuthEnable {
		return nil
	}

	var certSignature crypto.CertSignature
	_, err = asn1.Unmarshal(tx.Signature.Signature, &certSignature)
	if err != nil {
		clog.Error(fmt.Sprintf("unmashal certificate from signature failed. %s", err.Error()))
		return err
	}
	if len(certSignature.Cert) == 0 {
		clog.Error("cert can not be null")
		return types.ErrInvalidParam
	}

	// 本地缓存是否存在
	str := string(certSignature.Cert)
	result, ok := n.certValidateCache[str]
	if ok {
		return result
	}

	// 调用auth模块api校验
	param := types.ReqAuthCheckCert{tx.GetSignature()}
	reply, err := n.GetApi().ValidateCert(&param)
	if err != nil {
		return err
	}

	if reply.Result {
		n.certValidateCache[str] = nil
		return nil
	}

	n.certValidateCache[str] = types.ErrValidateCertFailed
	return types.ErrValidateCertFailed
}
