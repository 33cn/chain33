package executor

import (
	"encoding/binary"
	"fmt"

	log "github.com/inconshreveable/log15"
	"github.com/pkg/errors"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.valnode")
var driverName = "valnode"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&ValNode{}))
}

func CalcValNodeUpdateHeightIndexKey(height int64, index int) []byte {
	return []byte(fmt.Sprintf("ValNodeUpdate:%18d:%18d", height, int64(index)))
}

func CalcValNodeUpdateHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("ValNodeUpdate:%18d:", height))
}

func CalcValNodeBlockInfoHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("ValNodeBlockInfo:%18d:", height))
}

func Init(name string) {
	drivers.Register(GetName(), newValNode, 0)
}

func GetName() string {
	return newValNode().GetName()
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

func (val *ValNode) GetDriverName() string {
	return driverName
}

func (val *ValNode) GetActionValue(tx *types.Transaction) (*types.ValNodeAction, error) {
	action := &types.ValNodeAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}

	return action, nil
}

func (val *ValNode) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := val.GetActionValue(tx)
	if err != nil {
		return nil, err
	}

	receipt := &types.Receipt{types.ExecOk, nil, nil}
	return receipt, nil
}

func (val *ValNode) GetActionName(tx *types.Transaction) string {
	action, err := val.GetActionValue(tx)
	if err != nil {
		return "unknow"
	}
	if action.Ty == types.ValNodeActionUpdate && action.GetNode() != nil {
		return "upadate"
	}
	if action.Ty == types.ValNodeActionBlockInfo && action.GetBlockInfo() != nil {
		return "blockInfo"
	}
	return "unknow"
}

func (val *ValNode) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := val.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		clog.Error("ExecLocal")
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	action, err := val.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Debug("ExecLocal valnode tx", "tx=", action)

	if action.Ty == types.ValNodeActionUpdate && action.GetNode() != nil {
		if len(action.GetNode().PubKey) == 0 {
			return nil, errors.New("validator pubkey is empty")
		}
		if action.GetNode().Power < 0 {
			return nil, errors.New("validator power must not be negative")
		}
		key := CalcValNodeUpdateHeightIndexKey(val.GetHeight(), index)
		set.KV = append(set.KV, &types.KeyValue{Key: key, Value: types.Encode(action.GetNode())})
	} else if action.Ty == types.ValNodeActionBlockInfo && action.GetBlockInfo() != nil {
		key := CalcValNodeBlockInfoHeightKey(val.GetHeight())
		set.KV = append(set.KV, &types.KeyValue{Key: key, Value: types.Encode(action.GetBlockInfo())})
	}
	return set, nil
}

func (val *ValNode) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := val.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	action, err := val.GetActionValue(tx)
	if err != nil {
		return nil, err
	}
	clog.Debug("ExecDelLocal valnode tx", "tx=", action)

	if action.Ty == types.ValNodeActionUpdate && action.GetNode() != nil {
		if len(action.GetNode().PubKey) == 0 {
			return nil, errors.New("validator pubkey is empty")
		}
		if action.GetNode().Power < 0 {
			return nil, errors.New("validator power must not be negative")
		}
		key := CalcValNodeUpdateHeightIndexKey(val.GetHeight(), index)
		set.KV = append(set.KV, &types.KeyValue{Key: key, Value: types.Encode(action.GetNode())})
	} else if action.Ty == types.ValNodeActionBlockInfo && action.GetBlockInfo() != nil {
		key := CalcValNodeBlockInfoHeightKey(val.GetHeight())
		set.KV = append(set.KV, &types.KeyValue{Key: key, Value: types.Encode(action.GetBlockInfo())})
	}
	return set, nil
}

func (val *ValNode) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "GetValNodeByHeight" {
		height, size := binary.Varint(params)
		if size <= 0 || height <= 0 {
			return nil, types.ErrInvalidParam
		}
		key := CalcValNodeUpdateHeightKey(height)
		values, err := val.GetLocalDB().List(key, nil, 0, 1)
		if err != nil {
			return nil, err
		}
		if len(values) == 0 {
			return nil, types.ErrNotFound
		}
		reply := &types.ValNodes{}
		for _, valnodeByte := range values {
			var valnode types.ValNode
			err := types.Decode(valnodeByte, &valnode)
			if err != nil {
				clog.Error("GetValNodeByHeight proto.Unmarshal!", "err:", err)
				return nil, err
			}
			reply.Nodes = append(reply.Nodes, &valnode)
		}
		return reply, nil
	}
	if funcName == "GetBlockInfoByHeight" {
		height, size := binary.Varint(params)
		if size <= 0 || height <= 0 {
			return nil, types.ErrInvalidParam
		}
		key := CalcValNodeBlockInfoHeightKey(height)
		value, err := val.GetLocalDB().Get(key)
		if err != nil {
			return nil, err
		}
		if len(value) == 0 {
			return nil, types.ErrNotFound
		}
		reply := &types.TendermintBlockInfo{}
		err = types.Decode(value, reply)
		if err != nil {
			clog.Error("GetBlockInfoByHeight proto.Unmarshal!", "err:", err)
			return nil, err
		}

		return reply, nil
	}
	return nil, types.ErrActionNotSupport
}
