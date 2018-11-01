package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *trade) ExecDelLocal_SellLimit(sell *pty.TradeForSell, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.localDelLog(tx, receipt, index)
}

func (t *trade) ExecDelLocal_BuyMarket(buy *pty.TradeForBuy, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.localDelLog(tx, receipt, index)
}

func (t *trade) ExecDelLocal_RevokeSell(revoke *pty.TradeForRevokeSell, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.localDelLog(tx, receipt, index)
}

func (t *trade) ExecDelLocal_BuyLimit(buy *pty.TradeForBuyLimit, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.localDelLog(tx, receipt, index)
}

func (t *trade) ExecDelLocal_SellMarket(sell *pty.TradeForSellMarket, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.localDelLog(tx, receipt, index)
}

func (t *trade) ExecDelLocal_RevokeBuy(revoke *pty.TradeForRevokeBuy, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.localDelLog(tx, receipt, index)
}

func (t *trade) localDelLog(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogTradeSellLimit || item.Ty == types.TyLogTradeSellRevoke {
			var receipt pty.ReceiptTradeSellLimit
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteSell([]byte(receipt.Base.SellID), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTradeBuyMarket {
			var receipt pty.ReceiptTradeBuyMarket
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteBuy(receipt.Base)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTradeBuyRevoke || item.Ty == types.TyLogTradeBuyLimit {
			var receipt pty.ReceiptTradeBuyLimit
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteBuyLimit([]byte(receipt.Base.BuyID), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTradeSellMarket {
			var receipt pty.ReceiptSellMarket
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteSellMarket(receipt.Base)
			set.KV = append(set.KV, kv...)
		}
	}

	return &set, nil
}
