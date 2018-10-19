package executor

import (
	"strconv"
	"strings"

	"gitlab.33.cn/chain33/chain33/common"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/types"
)

// 目前设计trade 的query， 有两个部分的大分类
// 1. 按token 分
//    可以用于 token的挂单查询 (按价格排序)： OnBuy/OnSale
//    token 的历史行情 （按价格排序）: SoldOut/BoughtOut--> TODO 是否需要按时间（区块高度排序更合理）
// 2. 按 addr 分。 用于客户个人的钱包
//    自己未完成的交易 （按地址状态来）
//    自己的历史交易 （addr 的所有订单）
//
// 由于现价买/卖是没有orderID的， 用txhash 代替作为key
// key 有两种 orderID， txhash (0xAAAAAAAAAAAAAAA)

// 根据token 分页显示未完成成交卖单
func (t *trade) Query_GetTokenSellOrderByStatus(req *pty.ReqTokenSellOrder) (types.Message, error) {
	return t.GetTokenSellOrderByStatus(req, req.Status)
}

// 根据token 分页显示未完成成交买单
func (t *trade) Query_GetTokenBuyOrderByStatus(req *pty.ReqTokenBuyOrder) (types.Message, error) {
	if req.Status == 0 {
		req.Status = pty.TradeOrderStatusOnBuy
	}
	return t.GetTokenBuyOrderByStatus(req, req.Status)
}

// addr part
// addr(-token) 的所有订单， 不分页
func (t *trade) Query_GetOnesSellOrder(req *pty.ReqAddrAssets) (types.Message, error) {
	return t.GetOnesSellOrder(req)
}

func (t *trade) Query_GetOnesBuyOrder(req *pty.ReqAddrAssets) (types.Message, error) {
	return t.GetOnesBuyOrder(req)
}

// 按 用户状态来 addr-status
func (t *trade) Query_GetOnesSellOrderWithStatus(req *pty.ReqAddrAssets) (types.Message, error) {
	return t.GetOnesSellOrdersWithStatus(req)
}

func (t *trade) Query_GetOnesBuyOrderWithStatus(req *pty.ReqAddrAssets) (types.Message, error) {
	return t.GetOnesBuyOrdersWithStatus(req)
}

func (t *trade) Query_GetOnesOrderWithStatus(req *pty.ReqAddrAssets) (types.Message, error) {
	return t.GetOnesOrderWithStatus(req)
}

func (t *trade) GetOnesSellOrder(addrTokens *pty.ReqAddrAssets) (types.Message, error) {
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
			if err != nil && err != types.ErrNotFound {
				return nil, err
			}
			if len(values) != 0 {
				keys = append(keys, values...)
			}
		}
	}

	var replys pty.ReplyTradeOrders
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

func (t *trade) GetOnesBuyOrder(addrTokens *pty.ReqAddrAssets) (types.Message, error) {
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
			if err != nil && err != types.ErrNotFound {
				return nil, err
			}
			if len(values) != 0 {
				keys = append(keys, values...)
			}
		}
	}

	var replys pty.ReplyTradeOrders
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

func (t *trade) GetOnesSellOrdersWithStatus(req *pty.ReqAddrAssets) (types.Message, error) {
	var sellIDs [][]byte
	values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixStatus(req.Addr, req.Status), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of sellID", len(values))
		sellIDs = append(sellIDs, values...)
	}

	var replys pty.ReplyTradeOrders
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

func (t *trade) GetOnesBuyOrdersWithStatus(req *pty.ReqAddrAssets) (types.Message, error) {
	var sellIDs [][]byte
	values, err := t.GetLocalDB().List(calcOnesBuyOrderPrefixStatus(req.Addr, req.Status), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of buy keys", len(values))
		sellIDs = append(sellIDs, values...)
	}
	var replys pty.ReplyTradeOrders
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

func (t *trade) GetTokenSellOrderByStatus(req *pty.ReqTokenSellOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInvalidParam
	}

	fromKey := []byte("")
	if len(req.FromKey) != 0 {
		sell := t.replyReplySellOrderfromID([]byte(req.FromKey))
		if sell == nil {
			tradelog.Error("GetTokenSellOrderByStatus", "key not exist", req.FromKey)
			return nil, types.ErrInvalidParam
		}
		fromKey = calcTokensSellOrderKeyStatus(sell.TokenSymbol, sell.Status,
			calcPriceOfToken(sell.PricePerBoardlot, sell.AmountPerBoardlot), sell.Owner, sell.Key)
	}
	values, err := t.GetLocalDB().List(calcTokensSellOrderPrefixStatus(req.TokenSymbol, status), fromKey, req.Count, req.Direction)
	if err != nil {
		return nil, err
	}
	var replys pty.ReplyTradeOrders
	for _, key := range values {
		reply := t.loadOrderFromKey(key)
		if reply == nil {
			continue
		}
		replys.Orders = append(replys.Orders, reply)
	}
	return &replys, nil
}

func (t *trade) GetTokenBuyOrderByStatus(req *pty.ReqTokenBuyOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInvalidParam
	}

	fromKey := []byte("")
	if len(req.FromKey) != 0 {
		buy := t.replyReplyBuyOrderfromID([]byte(req.FromKey))
		if buy == nil {
			tradelog.Error("GetTokenBuyOrderByStatus", "key not exist", req.FromKey)
			return nil, types.ErrInvalidParam
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
	var replys pty.ReplyTradeOrders
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
func (t *trade) replyReplySellOrderfromID(key []byte) *pty.ReplySellOrder {
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

func (t *trade) replyReplyBuyOrderfromID(key []byte) *pty.ReplyBuyOrder {
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

func sellOrder2reply(sellOrder *pty.SellOrder) *pty.ReplySellOrder {
	reply := &pty.ReplySellOrder{
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
		sellOrder.AssetExec,
	}
	return reply
}

func txResult2sellOrderReply(txResult *types.TxResult) *pty.ReplySellOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeSellMarket {
			var receipt pty.ReceiptSellMarket
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
			reply := &pty.ReplySellOrder{
				receipt.Base.TokenSymbol,
				receipt.Base.Owner,
				int64(amount * float64(types.TokenPrecision)),
				receipt.Base.MinBoardlot,
				int64(price * float64(types.Coin)),
				receipt.Base.TotalBoardlot,
				receipt.Base.SoldBoardlot,
				receipt.Base.BuyID,
				pty.SellOrderStatus2Int[receipt.Base.Status],
				"",
				txhash,
				receipt.Base.Height,
				txhash,
				receipt.Base.AssetExec,
			}
			tradelog.Debug("txResult2sellOrderReply", "show reply", reply)
			return reply
		}
	}
	return nil
}

func buyOrder2reply(buyOrder *pty.BuyLimitOrder) *pty.ReplyBuyOrder {
	reply := &pty.ReplyBuyOrder{
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
		buyOrder.AssetExec,
	}
	return reply
}

func txResult2buyOrderReply(txResult *types.TxResult) *pty.ReplyBuyOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeBuyMarket {
			var receipt pty.ReceiptTradeBuyMarket
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
			reply := &pty.ReplyBuyOrder{
				receipt.Base.TokenSymbol,
				receipt.Base.Owner,
				int64(amount * float64(types.TokenPrecision)),
				receipt.Base.MinBoardlot,
				int64(price * float64(types.Coin)),
				receipt.Base.TotalBoardlot,
				receipt.Base.BoughtBoardlot,
				"",
				pty.SellOrderStatus2Int[receipt.Base.Status],
				receipt.Base.SellID,
				txhash,
				receipt.Base.Height,
				txhash,
				receipt.Base.AssetExec,
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
	case pty.TradeOrderStatusOnSale:
		return orderStatusOn, orderTypeSell
	case pty.TradeOrderStatusSoldOut:
		return orderStatusDone, orderTypeSell
	case pty.TradeOrderStatusRevoked:
		return orderStatusRevoke, orderTypeSell
	case pty.TradeOrderStatusOnBuy:
		return orderStatusOn, orderTypeBuy
	case pty.TradeOrderStatusBoughtOut:
		return orderStatusDone, orderTypeBuy
	case pty.TradeOrderStatusBuyRevoked:
		return orderStatusRevoke, orderTypeBuy
	}
	return orderStatusInvalid, orderTypeInvalid
}

// SellMarkMarket BuyMarket 没有tradeOrder 需要调用这个函数进行转化
// BuyRevoke, SellRevoke 也需要
// SellLimit/BuyLimit 有order 但order 里面没有 bolcktime， 直接访问 order 还需要再次访问 block， 还不如直接访问交易
func buyBase2Order(base *pty.ReceiptBuyBase, txHash string, blockTime int64) *pty.ReplyTradeOrder {
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
	key := txHash
	if len(base.BuyID) > 0 {
		key = base.BuyID
	}
	//txhash := common.ToHex(txResult.GetTx().Hash())
	reply := &pty.ReplyTradeOrder{
		base.TokenSymbol,
		base.Owner,
		int64(amount * float64(types.TokenPrecision)),
		base.MinBoardlot,
		int64(price * float64(types.Coin)),
		base.TotalBoardlot,
		base.BoughtBoardlot,
		base.BuyID,
		pty.SellOrderStatus2Int[base.Status],
		base.SellID,
		txHash,
		base.Height,
		key,
		blockTime,
		false,
		base.AssetExec,
	}
	tradelog.Debug("txResult2sellOrderReply", "show reply", reply)
	return reply
}

func sellBase2Order(base *pty.ReceiptSellBase, txHash string, blockTime int64) *pty.ReplyTradeOrder {
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
	key := txHash
	if len(base.SellID) > 0 {
		key = base.SellID
	}
	reply := &pty.ReplyTradeOrder{
		base.TokenSymbol,
		base.Owner,
		int64(amount * float64(types.TokenPrecision)),
		base.MinBoardlot,
		int64(price * float64(types.Coin)),
		base.TotalBoardlot,
		base.SoldBoardlot,
		base.BuyID,
		pty.SellOrderStatus2Int[base.Status],
		base.SellID,
		txHash,
		base.Height,
		key,
		blockTime,
		true,
		base.AssetExec,
	}
	tradelog.Debug("txResult2sellOrderReply", "show reply", reply)
	return reply
}

func txResult2OrderReply(txResult *types.TxResult) *pty.ReplyTradeOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeBuyMarket {
			var receipt pty.ReceiptTradeBuyMarket
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return buyBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeBuyRevoke {
			var receipt pty.ReceiptTradeBuyRevoke
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return buyBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeSellRevoke {
			var receipt pty.ReceiptTradeSellRevoke
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show revoke 1 ", receipt)
			return sellBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeSellMarket {
			var receipt pty.ReceiptSellMarket
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

func limitOrderTxResult2Order(txResult *types.TxResult) *pty.ReplyTradeOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Debug("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeSellLimit {
			var receipt pty.ReceiptTradeSellLimit
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Debug("txResult2sellOrderReply", "show logs 1 ", receipt)
			return sellBase2Order(receipt.Base, common.ToHex(txResult.GetTx().Hash()), txResult.Blocktime)
		} else if log.Ty == types.TyLogTradeBuyLimit {
			var receipt pty.ReceiptTradeBuyLimit
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

func (t *trade) loadOrderFromKey(key []byte) *pty.ReplyTradeOrder {
	tradelog.Debug("trade Query", "id", string(key), "check-prefix", sellIDPrefix)
	if strings.HasPrefix(string(key), sellIDPrefix) {
		txHash := strings.Replace(string(key), sellIDPrefix, "0x", 1)
		txResult, err := getTx([]byte(txHash), t.GetLocalDB())
		tradelog.Debug("loadOrderFromKey ", "load txhash", txResult)
		if err != nil {
			return nil
		}
		reply := limitOrderTxResult2Order(txResult)

		sellOrder, err := getSellOrderFromID(key, t.GetStateDB())
		tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
		if err != nil {
			return nil
		}
		reply.TradedBoardlot = sellOrder.SoldBoardlot
		return reply
	} else if strings.HasPrefix(string(key), buyIDPrefix) {
		txHash := strings.Replace(string(key), buyIDPrefix, "0x", 1)
		txResult, err := getTx([]byte(txHash), t.GetLocalDB())
		tradelog.Debug("loadOrderFromKey ", "load txhash", txResult)
		if err != nil {
			return nil
		}
		reply := limitOrderTxResult2Order(txResult)

		buyOrder, err := getBuyOrderFromID(key, t.GetStateDB())
		if err != nil {
			return nil
		}
		reply.TradedBoardlot = buyOrder.BoughtBoardlot
		return reply
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

func (t *trade) GetOnesOrderWithStatus(req *pty.ReqAddrAssets) (types.Message, error) {
	fromKey := []byte("")
	if len(req.FromKey) != 0 {
		order := t.loadOrderFromKey(fromKey)
		if order == nil {
			tradelog.Error("GetOnesOrderWithStatus", "key not exist", req.FromKey)
			return nil, types.ErrInvalidParam
		}
		st, ty := fromStatus(order.Status)
		fromKey = calcOnesOrderKey(order.Owner, st, ty, order.Height, order.Key)
	}

	orderStatus, orderType := fromStatus(req.Status)
	if orderStatus == orderStatusInvalid || orderType == orderTypeInvalid {
		return nil, types.ErrInvalidParam
	}

	keys, err := t.GetLocalDB().List(calcOnesOrderPrefixStatus(req.Addr, orderStatus), fromKey, req.Count, req.Direction)
	if err != nil {
		return nil, err
	}

	var replys pty.ReplyTradeOrders
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
