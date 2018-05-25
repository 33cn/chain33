package trade

import (
	"strings"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"

	"strconv"
)

func (t *trade) GetOnesSellOrder(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	var keys [][]byte
	if 0 == len(addrTokens.Token) {
		values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
		if err != nil {
			return nil, err
		}
		if len(values) != 0 {
			tradelog.Debug("trade Query", "get number of sellID", len(values))
			keys = append(keys, values...)
		}
	} else {
		for _, token := range addrTokens.Token {
			values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixToken(token, addrTokens.Addr), nil, 0, 0)
			tradelog.Debug("trade Query", "Begin to list addr with token", token, "got values", len(values))
			if err != nil {
				return nil, err
			}
			if len(values) != 0 {
				keys = append(keys, values...)
			}
		}
	}

	var replys types.ReplyTradeOrders
	for _, key := range keys {
		reply := t.loadOrderFromKey(key)
		if reply == nil {
			continue
		}
		tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
		replys.Orders = append(replys.Orders, reply)
	}
	return &replys, nil
}

func (t *trade) GetOnesBuyOrder(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	var keys [][]byte
	if 0 == len(addrTokens.Token) {
		values, err := t.GetLocalDB().List(calcOnesBuyOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
		if err != nil {
			return nil, err
		}
		if len(values) != 0 {
			tradelog.Debug("trade Query", "get number of buy keys", len(values))
			keys = append(keys, values...)
		}
	} else {
		for _, token := range addrTokens.Token {
			values, err := t.GetLocalDB().List(calcOnesBuyOrderPrefixToken(token, addrTokens.Addr), nil, 0, 0)
			tradelog.Debug("trade Query", "Begin to list addr with token", token, "got values", len(values))
			if err != nil {
				return nil, err
			}
			if len(values) != 0 {
				keys = append(keys, values...)
			}
		}
	}

	var replys types.ReplyTradeOrders
	for _, key := range keys {
		reply := t.loadOrderFromKey(key)
		if reply == nil {
			continue
		}
		tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
		replys.Orders = append(replys.Orders, reply)
	}

	return &replys, nil
}

func (t *trade) GetOnesSellOrdersWithStatus(req *types.ReqAddrTokens) (types.Message, error) {
	var sellIDs [][]byte
	values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixStatus(req.Addr, req.Status), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of sellID", len(values))
		sellIDs = append(sellIDs, values...)
	}

	var replys types.ReplyTradeOrders
	for _, key := range sellIDs {
		reply := t.loadOrderFromKey(key)
		if reply == nil {
			continue
		}
		tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
		replys.Orders = append(replys.Orders, reply)
	}

	return &replys, nil
}

func (t *trade) GetOnesBuyOrdersWithStatus(req *types.ReqAddrTokens) (types.Message, error) {
	var sellIDs [][]byte
	values, err := t.GetLocalDB().List(calcOnesBuyOrderPrefixStatus(req.Addr, req.Status), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of buy keys", len(values))
		sellIDs = append(sellIDs, values...)
	}
	var replys types.ReplyTradeOrders
	for _, key := range sellIDs {
		reply := t.loadOrderFromKey(key)
		if reply == nil {
			continue
		}
		tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
		replys.Orders = append(replys.Orders, reply)
	}

	return &replys, nil
}

func (t *trade) GetTokenSellOrderByStatus(req *types.ReqTokenSellOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInputPara
	}

	fromKey := []byte("")
	if len(req.FromKey) != 0 {
		sell := t.replyReplySellOrderfromID([]byte(req.FromKey))
		if sell == nil {
			tradelog.Error("GetTokenSellOrderByStatus", "key not exist", req.FromKey)
			return nil, types.ErrInputPara
		}
		fromKey = calcTokensSellOrderKeyStatus(sell.TokenSymbol, sell.Status,
			calcPriceOfToken(sell.PricePerBoardlot, sell.AmountPerBoardlot), sell.Owner, sell.Key)
	}
	values, err := t.GetLocalDB().List(calcTokensSellOrderPrefixStatus(req.TokenSymbol, status), fromKey, req.Count, req.Direction)
	if err != nil {
		return nil, err
	}
	var replys types.ReplyTradeOrders
	for _, key := range values {
		reply := t.loadOrderFromKey(key)
		if reply == nil {
			continue
		}
		replys.Orders = append(replys.Orders, reply)
	}
	return &replys, nil
}

func (t *trade) GetTokenBuyOrderByStatus(req *types.ReqTokenBuyOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInputPara
	}

	fromKey := []byte("")
	if len(req.FromKey) != 0 {
		buy := t.replyReplyBuyOrderfromID([]byte(req.FromKey))
		if buy == nil {
			tradelog.Error("GetTokenBuyOrderByStatus", "key not exist", req.FromKey)
			return nil, types.ErrInputPara
		}
		fromKey = calcTokensBuyOrderKeyStatus(buy.TokenSymbol, buy.Status,
			calcPriceOfToken(buy.PricePerBoardlot, buy.AmountPerBoardlot), buy.Owner, buy.Key)
	}
	tradelog.Debug("GetTokenBuyOrderByStatus", "fromKey ", fromKey)

	// List Direction 是升序， 买单是要降序， 把高价买的放前面， 在下一页操作时， 显示买价低的。
	direction := 1 - req.Direction
	values, err := t.GetLocalDB().List(calcTokensBuyOrderPrefixStatus(req.TokenSymbol, status), fromKey, req.Count, direction)
	if err != nil {
		return nil, err
	}
	var replys types.ReplyTradeOrders
	for _, key := range values {
		reply := t.loadOrderFromKey(key)
		if reply == nil {
			continue
		}
		tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
		replys.Orders = append(replys.Orders, reply)
	}
	return &replys, nil
}

// query reply utils
func (t *trade) replyReplySellOrderfromID(key []byte) *types.ReplySellOrder {
	tradelog.Debug("trade Query", "id", string(key), "check-prefix", sellIDPrefix)
	if strings.HasPrefix(string(key), sellIDPrefix) {
		if sellorder, err := getSellOrderFromID(key, t.GetStateDB()); err == nil {
			tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
			return sellOrder2reply(sellorder)
		}
	} else { // txhash as key
		txResult, err := getTx(key, t.GetLocalDB())
		tradelog.Debug("GetOnesSellOrder ", "load txhash", string(key))
		if err != nil {
			return nil
		}
		return txResult2sellOrderReply(txResult)
	}
	return nil
}

func (t *trade) replyReplyBuyOrderfromID(key []byte) *types.ReplyBuyOrder {
	tradelog.Debug("trade Query", "id", string(key), "check-prefix", buyIDPrefix)
	if strings.HasPrefix(string(key), buyIDPrefix) {
		if buyOrder, err := getBuyOrderFromID(key, t.GetStateDB()); err == nil {
			tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
			return buyOrder2reply(buyOrder)
		}
	} else { // txhash as key
		txResult, err := getTx(key, t.GetLocalDB())
		tradelog.Debug("replyReplyBuyOrderfromID ", "load txhash", string(key))
		if err != nil {
			return nil
		}
		return txResult2buyOrderReply(txResult)
	}
	return nil
}

func sellOrder2reply(sellOrder *types.SellOrder) *types.ReplySellOrder {
	reply := &types.ReplySellOrder{
		sellOrder.TokenSymbol,
		sellOrder.Address,
		sellOrder.AmountPerBoardlot,
		sellOrder.MinBoardlot,
		sellOrder.PricePerBoardlot,
		sellOrder.TotalBoardlot,
		sellOrder.SoldBoardlot,
		"",
		sellOrder.Status,
		sellOrder.SellID,
		strings.Replace(sellOrder.SellID, sellIDPrefix, "0x", 1),
		sellOrder.Height,
		sellOrder.SellID,
	}
	return reply
}

func txResult2sellOrderReply(txResult *types.TxResult) *types.ReplySellOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeSellMarket {
			var receipt types.ReceiptSellMarket
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			amount, err := strconv.ParseFloat(receipt.Base.AmountPerBoardlot, 64)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			price, err := strconv.ParseFloat(receipt.Base.PricePerBoardlot, 64)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}

			txhash := common.ToHex(txResult.GetTx().Hash())
			reply := &types.ReplySellOrder{
				receipt.Base.TokenSymbol,
				receipt.Base.Owner,
				int64(amount * float64(types.TokenPrecision)),
				receipt.Base.MinBoardlot,
				int64(price * float64(types.Coin)),
				receipt.Base.TotalBoardlot,
				receipt.Base.SoldBoardlot,
				receipt.Base.BuyID,
				types.SellOrderStatus2Int[receipt.Base.Status],
				"",
				txhash,
				receipt.Base.Height,
				txhash,
			}
			tradelog.Debug("txResult2sellOrderReply", "show reply", reply)
			return reply
		}
	}
	return nil
}

func buyOrder2reply(buyOrder *types.BuyLimitOrder) *types.ReplyBuyOrder {
	reply := &types.ReplyBuyOrder{
		buyOrder.TokenSymbol,
		buyOrder.Address,
		buyOrder.AmountPerBoardlot,
		buyOrder.MinBoardlot,
		buyOrder.PricePerBoardlot,
		buyOrder.TotalBoardlot,
		buyOrder.BoughtBoardlot,
		buyOrder.BuyID,
		buyOrder.Status,
		"",
		strings.Replace(buyOrder.BuyID, buyIDPrefix, "0x", 1),
		buyOrder.Height,
		buyOrder.BuyID,
	}
	return reply
}

func txResult2buyOrderReply(txResult *types.TxResult) *types.ReplyBuyOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeBuyMarket {
			var receipt types.ReceiptTradeBuyMarket
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			amount, err := strconv.ParseFloat(receipt.Base.AmountPerBoardlot, 64)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			price, err := strconv.ParseFloat(receipt.Base.PricePerBoardlot, 64)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			txhash := common.ToHex(txResult.GetTx().Hash())
			reply := &types.ReplyBuyOrder{
				receipt.Base.TokenSymbol,
				receipt.Base.Owner,
				int64(amount * float64(types.TokenPrecision)),
				receipt.Base.MinBoardlot,
				int64(price * float64(types.Coin)),
				receipt.Base.TotalBoardlot,
				receipt.Base.BoughtBoardlot,
				"",
				types.SellOrderStatus2Int[receipt.Base.Status],
				receipt.Base.SellID,
				txhash,
				receipt.Base.Height,
				txhash,
			}
			tradelog.Debug("txResult2sellOrderReply", "show reply", reply)
			return reply
		}
	}
	return nil
}

const (
	orderStatusInvalid = iota
	orderStatusOn
	orderStatusDone
	orderStatusRevoke
)

const (
	orderTypeInvalid = iota
	orderTypeSell
	orderTypeBuy
)

func fromStatus(status int32) (st, ty int32) {
	switch status {
	case types.TradeOrderStatusOnSale:
		return orderStatusOn, orderTypeSell
	case types.TradeOrderStatusSoldOut:
		return orderStatusDone, orderTypeSell
	case types.TradeOrderStatusRevoked:
		return orderStatusRevoke, orderTypeSell
	case types.TradeOrderStatusOnBuy:
		return orderStatusOn, orderTypeBuy
	case types.TradeOrderStatusBoughtOut:
		return orderStatusDone, orderTypeBuy
	case types.TradeOrderStatusBuyRevoked:
		return orderStatusRevoke, orderTypeBuy
	}
	return orderStatusInvalid, orderTypeInvalid
}

// SellMarkMarket BuyMarket 没有tradeOrder 需要调用这个函数进行转化
// BuyRevoke, SellRevoke 也需要
// SellLimit/BuyLimit 有order 但order 里面没有 bolcktime， 直接访问 order 还需要再次访问 block， 还不如直接访问交易
func buyBase2Order(base *types.ReceiptBuyBase, txHash string, blockTime int64) *types.ReplyTradeOrder {
	amount, err := strconv.ParseFloat(base.AmountPerBoardlot, 64)
	if err != nil {
		tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
		return nil
	}
	price, err := strconv.ParseFloat(base.PricePerBoardlot, 64)
	if err != nil {
		tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
		return nil
	}
	//txhash := common.ToHex(txResult.GetTx().Hash())
	reply := &types.ReplyTradeOrder{
		base.TokenSymbol,
		base.Owner,
		int64(amount * float64(types.TokenPrecision)),
		base.MinBoardlot,
		int64(price * float64(types.Coin)),
		base.TotalBoardlot,
		base.BoughtBoardlot,
		base.BuyID,
		types.SellOrderStatus2Int[base.Status],
		base.SellID,
		txHash,
		base.Height,
		txHash,
		blockTime,
		false,
	}
	tradelog.Debug("txResult2sellOrderReply", "show reply", reply)
	return reply
}

func sellBase2Order(base *types.ReceiptSellBase, txHash string, blockTime int64) *types.ReplyTradeOrder {
	amount, err := strconv.ParseFloat(base.AmountPerBoardlot, 64)
	if err != nil {
		tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
		return nil
	}
	price, err := strconv.ParseFloat(base.PricePerBoardlot, 64)
	if err != nil {
		tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
		return nil
	}
	//txhash := common.ToHex(txResult.GetTx().Hash())
	reply := &types.ReplyTradeOrder{
		base.TokenSymbol,
		base.Owner,
		int64(amount * float64(types.TokenPrecision)),
		base.MinBoardlot,
		int64(price * float64(types.Coin)),
		base.TotalBoardlot,
		base.SoldBoardlot,
		base.BuyID,
		types.SellOrderStatus2Int[base.Status],
		base.SellID,
		txHash,
		base.Height,
		txHash,
		blockTime,
		true,
	}
	tradelog.Debug("txResult2sellOrderReply", "show reply", reply)
	return reply
}

func txResult2OrderReply(txResult *types.TxResult) *types.ReplyTradeOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeBuyMarket {
			var receipt types.ReceiptTradeBuyMarket
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return buyBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeBuyRevoke {
			var receipt types.ReceiptTradeBuyRevoke
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return buyBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeSellRevoke {
			var receipt types.ReceiptTradeRevoke
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show revoke 1 ", receipt)
			return sellBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeSellMarket {
			var receipt types.ReceiptSellMarket
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return sellBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		}
	}
	return nil
}

func limitOrderTxResult2Order(txResult *types.TxResult) *types.ReplyTradeOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeSellLimit {
			var receipt types.ReceiptTradeSell
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return sellBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeBuyLimit {
			var receipt types.ReceiptTradeBuyLimit
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return buyBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		}
	}
	return nil
}

func (t *trade) loadOrderFromKey(key []byte) *types.ReplyTradeOrder {
	tradelog.Debug("trade Query", "id", string(key), "check-prefix", sellIDPrefix)
	if strings.HasPrefix(string(key), sellIDPrefix) {
		txHash := strings.Replace(string(key), sellIDPrefix, "0x", 1)
		txResult, err := getTx([]byte(txHash), t.GetLocalDB())
		tradelog.Debug("loadOrderFromKey ", "load txhash", txResult)
		if err != nil {
			return nil
		}
		return limitOrderTxResult2Order(txResult)
		// sellOrder 中没有 blocktime， 需要查询两次， 不如直接查询
		//if sellorder, err := getSellOrderFromID(key, t.GetStateDB()); err == nil {
		//	tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
		//	return sellOrder2Order(sellorder)
		//}
	} else if strings.HasPrefix(string(key), buyIDPrefix) {
		txHash := strings.Replace(string(key), buyIDPrefix, "0x", 1)
		txResult, err := getTx([]byte(txHash), t.GetLocalDB())
		tradelog.Debug("loadOrderFromKey ", "load txhash", txResult)
		if err != nil {
			return nil
		}
		return limitOrderTxResult2Order(txResult)
	} else { // txhash as key
		txResult, err := getTx(key, t.GetLocalDB())
		tradelog.Debug("loadOrderFromKey ", "load txhash", string(key))
		if err != nil {
			return nil
		}
		return txResult2OrderReply(txResult)
	}
	return nil
}

func (t *trade) GetOnesOrderWithStatus(req *types.ReqAddrTokens) (types.Message, error) {
	fromKey := []byte("")
	if len(req.FromKey) != 0 {
		order := t.loadOrderFromKey(fromKey)
		if order == nil {
			tradelog.Error("GetOnesOrderWithStatus", "key not exist", req.FromKey)
			return nil, types.ErrInputPara
		}
		st, ty := fromStatus(order.Status)
		fromKey = calcOnesOrderKey(order.Owner, st, ty, order.Height, order.Key)
	}

	orderStatus, orderType := fromStatus(req.Status)
	if orderStatus == orderStatusInvalid || orderType == orderTypeInvalid {
		return nil, types.ErrInputPara
	}

	keys, err := t.GetLocalDB().List(calcOnesOrderPrefixStatus(req.Addr, orderStatus), fromKey, req.Count, req.Direction)
	if err != nil {
		return nil, err
	}

	var replys types.ReplyTradeOrders
	for _, key := range keys {
		reply := t.loadOrderFromKey(key)
		if reply != nil {
			replys.Orders = append(replys.Orders, reply)
			tradelog.Debug("trade Query", "height of key", reply.Height,
				"len of reply.Selloders", len(replys.Orders))
		}
	}
	return &replys, nil
}
