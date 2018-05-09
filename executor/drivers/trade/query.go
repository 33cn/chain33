package trade

import (
	"strings"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"

	"strconv"
)

func (t *trade) GetOnesSellOrder(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	sellidGotAlready := make(map[string]bool)
	var keys [][]byte
	if 0 == len(addrTokens.Token) {
		values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
		if err != nil {
			return nil, err
		}
		if len(values) != 0 {
			tradelog.Debug("trade Query", "get number of sellid", len(values))
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

	var replys types.ReplySellOrders
	for _, key := range keys {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !sellidGotAlready[string(key)] {
			reply := t.replyReplySellOrderfromID(key)
			if reply == nil {
				continue
			}
			tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
			replys.Selloders = insertSellOrderDescending(reply, replys.Selloders)
			sellidGotAlready[string(key)] = true
		}
	}
	return &replys, nil
}

func (t *trade) GetOnesBuyOrder(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	gotAlready := make(map[interface{}]bool)
	var replys types.ReplyBuyOrders

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

	if len(keys) != 0 {
		tradelog.Debug("GetOnesBuyOrder", "get number of buy order", len(keys))
		for _, key := range keys {
			//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
			reply := t.replyReplyBuyOrderfromID(key)
			if reply == nil {
				continue
			}

			if !gotAlready[key] {
				// TODO 排序
				replys.BuyOrders = append(replys.BuyOrders, reply)
				gotAlready[key] = true
			}
		}
	}

	return &replys, nil
}

func (t *trade) GetOnesSellOrdersWithStatus(req *types.ReqAddrTokens) (types.Message, error) {
	var sellids [][]byte
	values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixStatus(req.Addr, req.Status), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of sellid", len(values))
		sellids = append(sellids, values...)
	}

	var replys types.ReplySellOrders
	for _, sellid := range sellids {
		reply := t.replyReplySellOrderfromID(sellid)
		if reply != nil {
			replys.Selloders = insertSellOrderDescending(reply, replys.Selloders)
			tradelog.Debug("trade Query", "height of sellid", reply.Height,
					"len of reply.Selloders", len(replys.Selloders))
		}
	}
	return &replys, nil
}

func (t *trade) GetOnesBuyOrdersWithStatus(req *types.ReqAddrTokens) (types.Message, error) {
	var sellids [][]byte
	values, err := t.GetLocalDB().List(calcOnesBuyOrderPrefixStatus(req.Addr, req.Status), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of buy keys", len(values))
		sellids = append(sellids, values...)
	}

	var replys types.ReplyBuyOrders
	for _, sellid := range sellids {
		reply := t.replyReplyBuyOrderfromID(sellid)
		if reply != nil {
			//replys.Selloders = insertBuyOrderDescending(reply, replys.Selloders)
			replys.BuyOrders = append(replys.BuyOrders, reply)
			tradelog.Debug("trade Query", "height of key", reply.Height,
				"len of reply.Selloders", len(replys.BuyOrders))
		}
	}
	return &replys, nil
}

//根据height进行降序插入,TODO:使用标准的第三方库进行替换
func insertSellOrderDescending(toBeInserted *types.ReplySellOrder, selloders []*types.ReplySellOrder) []*types.ReplySellOrder {
	if 0 == len(selloders) {
		selloders = append(selloders, toBeInserted)
	} else {
		index := len(selloders)
		for i, element := range selloders {
			if toBeInserted.Height >= element.Height {
				index = i
				break
			}
		}

		if len(selloders) == index {
			selloders = append(selloders, toBeInserted)
		} else {
			rear := append([]*types.ReplySellOrder{}, selloders[index:]...)
			selloders = append(selloders[0:index], toBeInserted)
			selloders = append(selloders, rear...)
		}
	}
	return selloders
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
	var reply types.ReplySellOrders
	for _, key := range values {
		sell := t.replyReplySellOrderfromID([]byte(key))
		if sell != nil {
			tradelog.Debug("trade Query", "GetTokenSellOrderByStatus", string(key))
			reply.Selloders = append(reply.Selloders, sell)
		}
	}
	return &reply, nil
}

func (t *trade) GetTokenBuyOrderByStatus(req *types.ReqTokenBuyOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInputPara
	}

	fromKey := []byte("")
	if len(req.FromKey) != 0 {
		buy := t.replyReplyBuyOrderfromID([]byte(req.FromKey))
		if buy == nil {
			tradelog.Error("GetTokenBuyLimitOrderByStatus", "key not exist", req.FromKey)
			return nil, types.ErrInputPara
		}
		fromKey = calcTokensBuyOrderKeyStatus(buy.TokenSymbol, buy.Status,
			calcPriceOfToken(buy.PricePerBoardlot, buy.AmountPerBoardlot), buy.Owner, buy.Key)
	}
	tradelog.Info("GetTokenBuyLimitOrderByStatus","fromKey ", fromKey)

	// List Direction 是升序， 买单是要降序， 把高价买的放前面， 在下一页操作时， 显示买价低的。
	direction := 1 - req.Direction
	values, err := t.GetLocalDB().List(calcTokensBuyOrderPrefixStatus(req.TokenSymbol, status), fromKey, req.Count, direction)
	if err != nil {
		return nil, err
	}
	var reply types.ReplyBuyOrders
	for _, key := range values {
		buy := t.replyReplyBuyOrderfromID([]byte(key))
		if buy != nil {
			tradelog.Debug("trade Query", "GetTokenBuyLimitOrderByStatus", string(key))
			reply.BuyOrders = append(reply.BuyOrders, buy)
		}
	}
	return &reply, nil
}

// query reply utils
func (t *trade) replyReplySellOrderfromID(key []byte) *types.ReplySellOrder {
	tradelog.Info("trade Query", "id", string(key), "check-prefix", sellOrderPrefix)
	if strings.HasPrefix(string(key), sellOrderPrefix) {
		if sellorder, err := getSellOrderFromID(key, t.GetStateDB()); err == nil {
			tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
			return sellOrder2reply(sellorder)
		}
	} else { // txhash as key
		txResult, err := getTx(key, t.GetLocalDB())
		tradelog.Info("GetOnesSellOrder ", "load txhash", string(key))
		if err != nil {
			return nil
		}
		return txResult2sellOrderReply(txResult)
	}
	return nil
}

func (t *trade) replyReplyBuyOrderfromID(key []byte) *types.ReplyBuyOrder {
	tradelog.Info("trade Query", "id", string(key), "check-prefix", buyOrderPrefix)
	if strings.HasPrefix(string(key), buyOrderPrefix) {
		if buyOrder, err := getBuyOrderFromID(key, t.GetStateDB()); err == nil {
			tradelog.Debug("trade Query", "getSellOrderFromID", string(key))
			return buyOrder2reply(buyOrder)
		}
	} else { // txhash as key
		txResult, err := getTx(key, t.GetLocalDB())
		tradelog.Info("replyReplyBuyOrderfromID ", "load txhash", string(key))
		if err != nil {
			return nil
		}
		return txResult2buyOrderReply(txResult)
	}
	return nil
}

func sellOrder2reply(sellOrder *types.SellOrder) *types.ReplySellOrder {
	reply := &types.ReplySellOrder{
		sellOrder.Tokensymbol,
		sellOrder.Address,
		sellOrder.Amountperboardlot,
		sellOrder.Minboardlot,
		sellOrder.Priceperboardlot,
		sellOrder.Totalboardlot,
		sellOrder.Soldboardlot,
		"",
		sellOrder.Status,
		sellOrder.Sellid,
		strings.Replace(sellOrder.Sellid, sellIDPrefix, "0x", 1),
		sellOrder.Height,
		sellOrder.Sellid,
	}
	return reply
}

func txResult2sellOrderReply(txResult *types.TxResult) *types.ReplySellOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Info("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeSellMarket {
			var receipt types.ReceiptSellMarket
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Info("txResult2sellOrderReply", "show logs 1 ", receipt)
			amount, err := strconv.ParseFloat(receipt.Base.Amountperboardlot, 64)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			price, err := strconv.ParseFloat(receipt.Base.Priceperboardlot, 64)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}

			txhash := common.ToHex(txResult.GetTx().Hash())
			reply := &types.ReplySellOrder{
				receipt.Base.Tokensymbol,
				receipt.Base.Owner,
				int64(amount * float64(types.TokenPrecision)),
				receipt.Base.Minboardlot,
				int64(price * float64(types.Coin)),
				receipt.Base.Totalboardlot,
				receipt.Base.Soldboardlot,
				receipt.Base.Buyid,
				types.SellOrderStatus2Int[receipt.Base.Status],
				"",
				txhash,
				receipt.Base.Height,
				txhash,
			}
			tradelog.Info("txResult2sellOrderReply", "show reply", reply)
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
		buyOrder.Buyid,
		buyOrder.Status,
		"",
		strings.Replace(buyOrder.Buyid, buyIDPrefix, "0x", 1),
		buyOrder.Height,
		buyOrder.Buyid,
	}
	return reply
}

func txResult2buyOrderReply(txResult *types.TxResult) *types.ReplyBuyOrder {
	logs := txResult.Receiptdate.Logs
	tradelog.Info("txResult2sellOrderReply", "show logs", logs)
	for _, log := range logs {
		if log.Ty == types.TyLogTradeBuyMarket {
			var receipt types.ReceiptTradeBuyMarket
			err := types.Decode(log.Log, &receipt)
			if err != nil {
				tradelog.Error("txResult2sellOrderReply", "decode receipt", err)
				return nil
			}
			tradelog.Info("txResult2sellOrderReply", "show logs 1 ", receipt)
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
				receipt.Base.Sellid,
				txhash,
				receipt.Base.Height,
				txhash,
			}
			tradelog.Info("txResult2sellOrderReply", "show reply", reply)
			return reply
		}
	}
	return nil
}