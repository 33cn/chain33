package executor

import (
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (evm *EVMExecutor) ExecDelLocal_EvmCreate(evmAction *evmtypes.EVMContractAction, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return evm._execDelLocal(evmAction, tx, receipt, index)
}

func (evm *EVMExecutor) ExecDelLocal_EvmCall(evmAction *evmtypes.EVMContractAction, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return evm._execDelLocal(evmAction, tx, receipt, index)
}

func (evm *EVMExecutor) _execDelLocal(evmAction *evmtypes.EVMContractAction, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := evm.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	if types.IsMatchFork(evm.GetHeight(), types.ForkV20EVMState) {
		// 需要将Exec中生成的合约状态变更信息从localdb中恢复
		for _, logItem := range receipt.Logs {
			if evmtypes.TyLogEVMStateChangeItem == logItem.Ty {
				data := logItem.Log
				var changeItem evmtypes.EVMStateChangeItem
				err = types.Decode(data, &changeItem)
				if err != nil {
					return set, err
				}
				set.KV = append(set.KV, &types.KeyValue{Key: []byte(changeItem.Key), Value: changeItem.PreValue})
			}
		}
	}

	return set, err
}
