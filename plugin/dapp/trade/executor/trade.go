package executor

/*
trade执行器支持trade的创建和交易，

主要提供操作有以下几种：
1）挂单出售；
2）购买指定的卖单；
3）撤销卖单；
4）挂单购买；
5）出售指定的买单；
6）撤销买单；
*/

import (
	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common"
	token "gitlab.33.cn/chain33/chain33/plugin/dapp/token/executor"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
	"reflect"
)

var tradelog = log.New("module", "execs.trade")

var executorFunList = make(map[string]reflect.Method)
var executorType = pty.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&trade{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func (t *trade) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func Init(name string) {
	drivers.Register(GetName(), newTrade, types.ForkV2AddToken)
}

func GetName() string {
	return newTrade().GetName()
}

type trade struct {
	drivers.DriverBase
}

func newTrade() drivers.Driver {
	t := &trade{}
	t.SetChild(t)
	t.SetExecutorType(executorType)
	return t
}

func (t *trade) GetDriverName() string {
	return "trade"
}

func (t *trade) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var trade pty.Trade
	err := types.Decode(tx.Payload, &trade)
	if err != nil {
		return nil, err
	}
	tradelog.Info("exec trade tx=", "tx hash", common.Bytes2Hex(tx.Hash()), "Ty", trade.GetTy())

	action := newTradeAction(t, tx)
	switch trade.GetTy() {
	case pty.TradeSellLimit:
		if trade.GetSell() == nil {
			return nil, types.ErrInputPara
		}
		return action.tradeSell(trade.GetSell())

	case pty.TradeBuyMarket:
		if trade.GetBuy() == nil {
			return nil, types.ErrInputPara
		}
		return action.tradeBuy(trade.GetBuy())

	case pty.TradeRevokeSell:
		if trade.GetRevokeSell() == nil {
			return nil, types.ErrInputPara
		}
		return action.tradeRevokeSell(trade.GetRevokeSell())

	case pty.TradeBuyLimit:
		if t.GetHeight() < types.ForkV10TradeBuyLimit {
			return nil, types.ErrActionNotSupport
		}
		if trade.GetBuyLimit() == nil {
			return nil, types.ErrInputPara
		}
		return action.tradeBuyLimit(trade.GetBuyLimit())

	case pty.TradeSellMarket:
		if t.GetHeight() < types.ForkV10TradeBuyLimit {
			return nil, types.ErrActionNotSupport
		}
		if trade.GetSellMarket() == nil {
			return nil, types.ErrInputPara
		}
		return action.tradeSellMarket(trade.GetSellMarket())

	case pty.TradeRevokeBuy:
		if t.GetHeight() < types.ForkV10TradeBuyLimit {
			return nil, types.ErrActionNotSupport
		}
		if trade.GetRevokeBuy() == nil {
			return nil, types.ErrInputPara
		}
		return action.tradeRevokeBuyLimit(trade.GetRevokeBuy())

	default:
		return nil, types.ErrActionNotSupport
	}
}

func (t *trade) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	var symbol string
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogTradeSellLimit || item.Ty == types.TyLogTradeSellRevoke {
			var receipt pty.ReceiptTradeSell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveSell([]byte(receipt.Base.SellID), item.Ty)
			set.KV = append(set.KV, kv...)
			symbol = receipt.Base.TokenSymbol
		} else if item.Ty == types.TyLogTradeBuyMarket {
			var receipt pty.ReceiptTradeBuyMarket
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveBuy(receipt.Base)
			set.KV = append(set.KV, kv...)

			// 添加个人资产列表
			//trade 合约原则上不要影响其他的合约,调用withdraw的时候，在显示也可以
			/*
				kv = token.AddTokenToAssets(receipt.Base.Owner, t.GetLocalDB(), receipt.Base.TokenSymbol)
				if kv != nil {
					set.KV = append(set.KV, kv...)
				}
			*/
			symbol = receipt.Base.TokenSymbol
		} else if item.Ty == types.TyLogTradeBuyRevoke || item.Ty == types.TyLogTradeBuyLimit {
			var receipt pty.ReceiptTradeBuyLimit
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}

			kv := t.saveBuyLimit([]byte(receipt.Base.BuyID), item.Ty)
			set.KV = append(set.KV, kv...)

			// 添加个人资产列表
			//trade 合约原则上不要影响其他的合约,调用withdraw的时候，在显示也可以
			/*
				kv = token.AddTokenToAssets(receipt.Base.Owner, t.GetLocalDB(), receipt.Base.TokenSymbol)
				if kv != nil {
					set.KV = append(set.KV, kv...)
				}
			*/
			symbol = receipt.Base.TokenSymbol
		} else if item.Ty == types.TyLogTradeSellMarket {
			var receipt pty.ReceiptSellMarket
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveSellMarket(receipt.Base)
			//tradelog.Info("saveSellMarket", "kv", kv)
			set.KV = append(set.KV, kv...)
			symbol = receipt.Base.TokenSymbol
		}
	}
	if types.GetSaveTokenTxList() {
		kvs, err := token.TokenTxKvs(tx, symbol, t.GetHeight(), int64(index), false)
		// t.makeT1okenTxKvs(tx, &action, receipt, index, false)
		if err != nil {
			return nil, err
		}
		set.KV = append(set.KV, kvs...)
	}

	return set, nil
}

func (t *trade) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	var symbol string
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogTradeSellLimit || item.Ty == types.TyLogTradeSellRevoke {
			var receipt pty.ReceiptTradeSell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteSell([]byte(receipt.Base.SellID), item.Ty)
			set.KV = append(set.KV, kv...)
			symbol = receipt.Base.TokenSymbol
		} else if item.Ty == types.TyLogTradeBuyMarket {
			var receipt pty.ReceiptTradeBuyMarket
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteBuy(receipt.Base)
			set.KV = append(set.KV, kv...)
			symbol = receipt.Base.TokenSymbol
		} else if item.Ty == types.TyLogTradeBuyRevoke || item.Ty == types.TyLogTradeBuyLimit {
			var receipt pty.ReceiptTradeBuyLimit
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteBuyLimit([]byte(receipt.Base.BuyID), item.Ty)
			set.KV = append(set.KV, kv...)
			symbol = receipt.Base.TokenSymbol
		} else if item.Ty == types.TyLogTradeSellMarket {
			var receipt pty.ReceiptSellMarket
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteSellMarket(receipt.Base)
			set.KV = append(set.KV, kv...)
			symbol = receipt.Base.TokenSymbol
		}
	}
	if types.GetSaveTokenTxList() {
		kvs, err := token.TokenTxKvs(tx, symbol, t.GetHeight(), int64(index), true)
		// t.makeT1okenTxKvs(tx, &action, receipt, index, false)
		if err != nil {
			return nil, err
		}
		set.KV = append(set.KV, kvs...)
	}
	return set, nil
}



func (t *trade) getSellOrderFromDb(sellID []byte) *pty.SellOrder {
	value, err := t.GetStateDB().Get(sellID)
	if err != nil {
		panic(err)
	}
	var sellorder pty.SellOrder
	types.Decode(value, &sellorder)
	return &sellorder
}

func genSaveSellKv(sellorder *pty.SellOrder) []*types.KeyValue {
	status := sellorder.Status
	var kv []*types.KeyValue
	kv = saveSellOrderKeyValue(kv, sellorder, status)
	if types.TradeOrderStatusSoldOut == status || types.TradeOrderStatusRevoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", sellorder.SellID)
		kv = deleteSellOrderKeyValue(kv, sellorder, types.TradeOrderStatusOnSale)
	}
	return kv
}

func (t *trade) saveSell(sellID []byte, ty int32) []*types.KeyValue {
	sellorder := t.getSellOrderFromDb(sellID)
	return genSaveSellKv(sellorder)
}

func deleteSellOrderKeyValue(kv []*types.KeyValue, sellorder *pty.SellOrder, status int32) []*types.KeyValue {
	return genSellOrderKeyValue(kv, sellorder, status, nil)
}

func saveSellOrderKeyValue(kv []*types.KeyValue, sellorder *pty.SellOrder, status int32) []*types.KeyValue {
	sellID := []byte(sellorder.SellID)
	return genSellOrderKeyValue(kv, sellorder, status, sellID)
}

func genDeleteSellKv(sellorder *pty.SellOrder) []*types.KeyValue {
	status := sellorder.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, sellorder, status)
	if types.TradeOrderStatusSoldOut == status || types.TradeOrderStatusRevoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellID", sellorder.SellID)
		kv = saveSellOrderKeyValue(kv, sellorder, types.TradeOrderStatusOnSale)
	}
	return kv
}

func (t *trade) deleteSell(sellID []byte, ty int32) []*types.KeyValue {
	sellorder := t.getSellOrderFromDb(sellID)
	return genDeleteSellKv(sellorder)
}

func (t *trade) saveBuy(receiptTradeBuy *pty.ReceiptBuyBase) []*types.KeyValue {
	//tradelog.Info("save", "buy", receiptTradeBuy)

	var kv []*types.KeyValue
	return saveBuyMarketOrderKeyValue(kv, receiptTradeBuy, types.TradeOrderStatusBoughtOut, t.GetHeight())
}

func (t *trade) deleteBuy(receiptTradeBuy *pty.ReceiptBuyBase) []*types.KeyValue {
	var kv []*types.KeyValue
	return deleteBuyMarketOrderKeyValue(kv, receiptTradeBuy, types.TradeOrderStatusBoughtOut, t.GetHeight())
}

// BuyLimit Local
func (t *trade) getBuyOrderFromDb(buyID []byte) *pty.BuyLimitOrder {
	value, err := t.GetStateDB().Get(buyID)
	if err != nil {
		panic(err)
	}
	var buyOrder pty.BuyLimitOrder
	types.Decode(value, &buyOrder)
	return &buyOrder
}

func genSaveBuyLimitKv(buyOrder *pty.BuyLimitOrder) []*types.KeyValue {
	status := buyOrder.Status
	var kv []*types.KeyValue
	kv = saveBuyLimitOrderKeyValue(kv, buyOrder, status)
	if types.TradeOrderStatusBoughtOut == status || types.TradeOrderStatusBuyRevoked == status {
		tradelog.Debug("trade saveBuyLimit ", "remove old status with Buyid", buyOrder.BuyID)
		kv = deleteBuyLimitKeyValue(kv, buyOrder, types.TradeOrderStatusOnSale)
	}
	return kv
}

func (t *trade) saveBuyLimit(buyID []byte, ty int32) []*types.KeyValue {
	buyOrder := t.getBuyOrderFromDb(buyID)
	return genSaveBuyLimitKv(buyOrder)
}

func saveBuyLimitOrderKeyValue(kv []*types.KeyValue, buyOrder *pty.BuyLimitOrder, status int32) []*types.KeyValue {
	buyID := []byte(buyOrder.BuyID)
	return genBuyLimitOrderKeyValue(kv, buyOrder, status, buyID)
}

func deleteBuyLimitKeyValue(kv []*types.KeyValue, buyOrder *pty.BuyLimitOrder, status int32) []*types.KeyValue {
	return genBuyLimitOrderKeyValue(kv, buyOrder, status, nil)
}

func genDeleteBuyLimitKv(buyOrder *pty.BuyLimitOrder) []*types.KeyValue {
	status := buyOrder.Status
	var kv []*types.KeyValue
	kv = deleteBuyLimitKeyValue(kv, buyOrder, status)
	if types.TradeOrderStatusBoughtOut == status || types.TradeOrderStatusBuyRevoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", buyOrder.BuyID)
		kv = saveBuyLimitOrderKeyValue(kv, buyOrder, types.TradeOrderStatusOnBuy)
	}
	return kv
}

func (t *trade) deleteBuyLimit(buyID []byte, ty int32) []*types.KeyValue {
	buyOrder := t.getBuyOrderFromDb(buyID)
	return genDeleteBuyLimitKv(buyOrder)
}

func (t *trade) saveSellMarket(receiptTradeBuy *pty.ReceiptSellBase) []*types.KeyValue {
	var kv []*types.KeyValue
	return saveSellMarketOrderKeyValue(kv, receiptTradeBuy, types.TradeOrderStatusSoldOut, t.GetHeight())
}

func (t *trade) deleteSellMarket(receiptTradeBuy *pty.ReceiptSellBase) []*types.KeyValue {
	var kv []*types.KeyValue
	return deleteSellMarketOrderKeyValue(kv, receiptTradeBuy, types.TradeOrderStatusSoldOut, t.GetHeight())
}

func saveSellMarketOrderKeyValue(kv []*types.KeyValue, receipt *pty.ReceiptSellBase, status int32, height int64) []*types.KeyValue {
	txhash := []byte(receipt.TxHash)
	return genSellMarketOrderKeyValue(kv, receipt, status, height, txhash)
}

func deleteSellMarketOrderKeyValue(kv []*types.KeyValue, receipt *pty.ReceiptSellBase, status int32, height int64) []*types.KeyValue {
	return genSellMarketOrderKeyValue(kv, receipt, status, height, nil)
}

func saveBuyMarketOrderKeyValue(kv []*types.KeyValue, receipt *pty.ReceiptBuyBase, status int32, height int64) []*types.KeyValue {
	txhash := []byte(receipt.TxHash)
	return genBuyMarketOrderKeyValue(kv, receipt, status, height, txhash)
}

func deleteBuyMarketOrderKeyValue(kv []*types.KeyValue, receipt *pty.ReceiptBuyBase, status int32, height int64) []*types.KeyValue {
	return genBuyMarketOrderKeyValue(kv, receipt, status, height, nil)
}
