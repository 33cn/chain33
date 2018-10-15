package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (val *ValNode) ExecDelLocal_Node(node *pty.ValNode, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	key := CalcValNodeUpdateHeightIndexKey(val.GetHeight(), index)
	set.KV = append(set.KV, &types.KeyValue{Key: key, Value: nil})
	return set, nil
}

func (val *ValNode) ExecDelLocal_BlockInfo(blockInfo *pty.TendermintBlockInfo, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	key := CalcValNodeBlockInfoHeightKey(val.GetHeight())
	set.KV = append(set.KV, &types.KeyValue{Key: key, Value: nil})
	return set, nil
}
