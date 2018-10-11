package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/norm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (n *Norm) Exec_Nput(nput *pty.NormPut, tx *types.Transaction, index int) (*types.Receipt, error) {
	receipt := &types.Receipt{types.ExecOk, nil, nil}
	normKV := &types.KeyValue{Key(nput.Key), nput.Value}
	receipt.KV = append(receipt.KV, normKV)
	return receipt, nil
}
