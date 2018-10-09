package executor

import (
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Blackwhite) execLocal(receiptData *types.ReceiptData) ([]*types.KeyValue, error) {
	for _, log := range receiptData.Logs {
		switch log.Ty {
		case gt.TyLogBlackwhiteCreate,
			gt.TyLogBlackwhitePlay,
			gt.TyLogBlackwhiteShow,
			gt.TyLogBlackwhiteTimeout,
			gt.TyLogBlackwhiteDone:
			{
				var receipt gt.ReceiptBlackwhiteStatus
				err := types.Decode(log.Log, &receipt)
				if err != nil {
					return nil, err
				}
				return c.saveHeightIndex(&receipt), nil
			}
		case gt.TyLogBlackwhiteLoopInfo:
			{
				var res gt.ReplyLoopResults
				err := types.Decode(log.Log, &res)
				if err != nil {
					return nil, err
				}
				return c.saveLoopResult(&res), nil
			}
		default:
			break
		}
	}
	return nil, types.ErrNotSupport
}

func (c *Blackwhite) ExecLocal_Create(payload *gt.BlackwhiteCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := c.execLocal(receiptData)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}

func (c *Blackwhite) ExecLocal_Play(payload *gt.BlackwhitePlay, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := c.execLocal(receiptData)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}

func (c *Blackwhite) ExecLocal_Show(payload *gt.BlackwhiteShow, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := c.execLocal(receiptData)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}

func (c *Blackwhite) ExecLocal_TimeoutDone(payload *gt.BlackwhiteTimeoutDone, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := c.execLocal(receiptData)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: kv}, nil
}
