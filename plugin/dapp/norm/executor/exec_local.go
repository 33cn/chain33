package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/norm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (n *Norm) ExecLocal_Nput(nput *pty.NormPut, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}
