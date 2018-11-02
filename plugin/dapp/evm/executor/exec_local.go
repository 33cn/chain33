package executor

import (
	"bytes"

	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (evm *EVMExecutor) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := evm.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	if types.IsDappFork(evm.GetHeight(), "evm", "ForkEVMState") {
		// 需要将Exec中生成的合约状态变更信息写入localdb
		for _, logItem := range receipt.Logs {
			if evmtypes.TyLogEVMStateChangeItem == logItem.Ty {
				data := logItem.Log
				var changeItem evmtypes.EVMStateChangeItem
				err = types.Decode(data, &changeItem)
				if err != nil {
					return set, err
				}
				//转换老的log的key-> 新的key
				key := []byte(changeItem.Key)
				if bytes.HasPrefix(key, []byte("mavl-")) {
					key[0] = 'L'
					key[1] = 'O'
					key[2] = 'D'
					key[3] = 'B'
				}
				set.KV = append(set.KV, &types.KeyValue{Key: key, Value: changeItem.CurrentValue})
			}
		}
	}

	return set, err
}
