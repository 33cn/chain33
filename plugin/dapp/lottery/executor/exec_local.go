package executor

import (
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/lottery/types"
)

func (l *Lottery) execLocal(tx *types.Transaction, receipt *types.ReceiptData) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for _, item := range receipt.Logs {
		switch item.Ty {
		case types.TyLogLotteryCreate, types.TyLogLotteryBuy, types.TyLogLotteryDraw, types.TyLogLotteryClose:
			var lotterylog pty.ReceiptLottery
			err := types.Decode(item.Log, &lotterylog)
			if err != nil {
				return nil, err
			}
			kv := l.saveLottery(&lotterylog)
			set.KV = append(set.KV, kv...)

			if item.Ty == types.TyLogLotteryBuy {
				kv := l.saveLotteryBuy(&lotterylog, common.ToHex(tx.Hash()))
				set.KV = append(set.KV, kv...)
			} else if item.Ty == types.TyLogLotteryDraw {
				kv := l.saveLotteryDraw(&lotterylog)
				set.KV = append(set.KV, kv...)
			}
		}
	}
	return set, nil
}

func (l *Lottery) ExecLocal_Create(payload *pty.LotteryCreate, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return l.execLocal(tx, receiptData)
}

func (l *Lottery) ExecLocal_Buy(payload *pty.LotteryBuy, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return l.execLocal(tx, receiptData)
}

func (l *Lottery) ExecLocal_Draw(payload *pty.LotteryDraw, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return l.execLocal(tx, receiptData)
}

func (l *Lottery) ExecLocal_Close(payload *pty.LotteryClose, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return l.execLocal(tx, receiptData)
}
