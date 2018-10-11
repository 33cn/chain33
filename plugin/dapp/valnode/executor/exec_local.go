package executor

import (
	"errors"

	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (val *ValNode) ExecLocal_Node(node *pty.ValNode, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	if len(node.GetPubKey()) == 0 {
		return nil, errors.New("validator pubkey is empty")
	}
	if node.GetPower() < 0 {
		return nil, errors.New("validator power must not be negative")
	}
	key := CalcValNodeUpdateHeightIndexKey(val.GetHeight(), index)
	set.KV = append(set.KV, &types.KeyValue{Key: key, Value: types.Encode(node)})
	return set, nil
}

func (val *ValNode) ExecLocal_BlockInfo(blockInfo *types.TendermintBlockInfo, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	key := CalcValNodeBlockInfoHeightKey(val.GetHeight())
	set.KV = append(set.KV, &types.KeyValue{Key: key, Value: types.Encode(blockInfo)})
	return set, nil
}
