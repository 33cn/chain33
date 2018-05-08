package trade

import (
	"strings"
	"fmt"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"

)

func (t *trade) GetOnesSellOrder(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	sellidGotAlready := make(map[string]bool)
	var sellids [][]byte
	if 0 == len(addrTokens.Token) {
		//list := dbm.NewListHelper(t.GetLocalDB())
		values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
		if err != nil {
			return nil, err
		}
		if len(values) != 0 {
			tradelog.Debug("trade Query", "get number of sellid", len(values))
			sellids = append(sellids, values...)
		}
	} else {
		for _, token := range addrTokens.Token {
			values, err := t.GetLocalDB().List(calcOnesSellOrderPrefixToken(token, addrTokens.Addr), nil, 0, 0)
			tradelog.Debug("trade Query", "Begin to list addr with token", token, "got values", len(values))
			if err != nil {
				return nil, err
			}
			if len(values) != 0 {
				sellids = append(sellids, values...)
			}
		}
	}

	var reply types.ReplySellOrders
	for _, sellid := range sellids {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !sellidGotAlready[string(sellid)] {
			tradelog.Info("trade Query", "id", string(sellid), "check-prefix", sellOrderPrefix)
			if strings.HasPrefix(string(sellid), sellOrderPrefix) {
				if sellorder, err := getSellOrderFromID(sellid, t.GetStateDB()); err == nil {
					tradelog.Debug("trade Query", "getSellOrderFromID", string(sellid))
					reply.Selloders = insertSellOrderDescending(sellorder, reply.Selloders)
				}
				sellidGotAlready[string(sellid)] = true
			} else { // txhash as sellid for
				sellid2 := fmt.Sprintf("0x%s", sellid)
				selled2, err := common.FromHex(string(sellid))
				if err != nil {
					return nil, err
				}
				txResult, err := getTx(selled2, t.GetLocalDB())
				tradelog.Info("GetOnesSellOrder ", "load txhash", sellid2)
				if err != nil {
					return nil, err
				}

				tradelog.Info("GetOnesSellOrder ", "load txhash", txResult)
				// TODO make detail for sellOrder
				tx := txResult.Tx
				var trade types.Trade
				err = types.Decode(tx.Payload, &trade)
				if err != nil {
					tradelog.Error("GetOnesSellOrder", "bad sellid", sellid)
					continue
				}
				sellMarket := trade.GetTokensellmarket()
				if sellMarket == nil {
					tradelog.Error("GetOnesSellOrder", "bad sellid", sellid)
					continue
				}
				tradelog.Info("GetOnesSellOrder", "show logs", sellMarket)
				logs := txResult.Receiptdate.Logs
				tradelog.Info("GetOnesSellOrder", "show logs", logs)
				for _, log := range logs {
					if log.Ty == types.TyLogTradeSellMarket {
						var receipt types.ReceiptTradeBase
						types.Decode(log.Log, &receipt)
						if err != nil {
							tradelog.Error("GetOnesSellOrder", "bad sellid 1", sellid)
							continue
						}
						tradelog.Info("GetOnesSellOrder", "show logs", receipt)
					} else if log.Ty == types.TyLogTradeBuyLimit {
						var receipt types.ReceiptTradeBuyLimit
						types.Decode(log.Log, &receipt)
						if err != nil {
							tradelog.Error("GetOnesSellOrder", "bad sellid", sellid)
							continue
						}
						tradelog.Info("GetOnesSellOrder", "show logs 2", receipt)
					}
				}
				continue
			}
		}
	}
	return &reply, nil
}

func (t *trade) GetOnesBuyOrder(addrTokens *types.ReqAddrTokens) (types.Message, error) {
	gotAlready := make(map[interface{}]bool)
	var reply types.ReplyTradeBuyOrders
	values, err := t.GetLocalDB().List(calcOnesBuyOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("GetOnesBuyOrder", "get number of buy order", len(values))
		for _, value := range values {
			//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
			var tradeBuyDone types.TradeBuyDone
			if err := types.Decode(value, &tradeBuyDone); err != nil {
				tradelog.Error("GetOnesBuyOrder", "Failed to decode tradebuydoen", value)
				return nil, err
			}
			if !gotAlready[tradeBuyDone.Buytxhash] {
				reply.Tradebuydones = append(reply.Tradebuydones, &tradeBuyDone)
				gotAlready[tradeBuyDone.Buytxhash] = true
			}
		}
	}

	if len(addrTokens.Token) != 0 {
		var resultReply types.ReplyTradeBuyOrders
		tokenMap := make(map[string]bool)
		for _, token := range addrTokens.Token {
			tokenMap[token] = true
		}

		for _, Tradebuydone := range reply.Tradebuydones {
			if tokenMap[Tradebuydone.Token] {
				resultReply.Tradebuydones = append(resultReply.Tradebuydones, Tradebuydone)
			}
		}
		return &resultReply, nil
	}

	return &reply, nil
}

func (t *trade) GetAllSellOrdersWithStatus(status int32) (types.Message, error) {
	sellidGotAlready := make(map[string]bool)
	var sellids [][]byte
	values, err := t.GetLocalDB().List(calcTokenSellOrderPrefixStatus(status), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of sellid", len(values))
		sellids = append(sellids, values...)
	}

	var reply types.ReplySellOrders
	for _, sellid := range sellids {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !sellidGotAlready[string(sellid)] {
			if sellorder, err := getSellOrderFromID(sellid, t.GetStateDB()); err == nil {
				reply.Selloders = insertSellOrderDescending(sellorder, reply.Selloders)
				tradelog.Debug("trade Query", "height of sellid", sellorder.Height,
					"len of reply.Selloders", len(reply.Selloders))
			}
			sellidGotAlready[string(sellid)] = true
		}
	}
	return &reply, nil
}

//根据height进行降序插入,TODO:使用标准的第三方库进行替换
func insertSellOrderDescending(toBeInserted *types.SellOrder, selloders []*types.SellOrder) []*types.SellOrder {
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
			rear := append([]*types.SellOrder{}, selloders[index:]...)
			selloders = append(selloders[0:index], toBeInserted)
			selloders = append(selloders, rear...)
		}
	}
	return selloders
}

func (t *trade) GetTokenByStatus(req *types.ReqTokenSellOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInputPara
	}

	fromKey := []byte("")
	if len(req.FromSellId) != 0 {
		sellorder, err := getSellOrderFromID([]byte(req.FromSellId), t.GetStateDB())
		if err != nil {
			tradelog.Error("GetTokenByStatus get sellorder err", err)
			return nil, err
		}
		fromKey = calcTokensSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Status,
			calcPriceOfToken(sellorder.Priceperboardlot, sellorder.Amountperboardlot), sellorder.Address, sellorder.Sellid)
	}
	values, err := t.GetLocalDB().List(calcTokensSellOrderPrefixStatus(req.TokenSymbol, status), fromKey, req.Count, req.Direction)
	if err != nil {
		return nil, err
	}
	var reply types.ReplySellOrders
	for _, sellid := range values {
		if sellorder, err := getSellOrderFromID(sellid, t.GetStateDB()); err == nil {
			tradelog.Debug("trade Query", "getSellOrderFromID", string(sellid))
			reply.Selloders = append(reply.Selloders, sellorder)
		}
	}
	return &reply, nil
}

func (t *trade) GetTokenBuyLimitOrderByStatus(req *types.ReqTokenBuyLimitOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInputPara
	}

	fromKey := []byte("")
	if len(req.FromBuyId) != 0 {
		buyOrder, err := getBuyOrderFromID([]byte(req.FromBuyId), t.GetStateDB())
		if err != nil {
			tradelog.Error("GetTokenBuyLimitOrderByStatus get sellorder err", err)
			return nil, err
		}
		fromKey = calcTokensBuyLimitOrderKeyStatus(buyOrder.TokenSymbol, buyOrder.Status,
			calcPriceOfToken(buyOrder.PricePerBoardlot, buyOrder.AmountPerBoardlot), buyOrder.Address, buyOrder.Buyid)

	}
	tradelog.Info("GetTokenBuyLimitOrderByStatus","fromKey ", fromKey)

	// List Direction 是升序， 买单是要降序， 把高价买的放前面， 在下一页操作时， 显示买价低的。
	direction := 1 - req.Direction
	values, err := t.GetLocalDB().List(calcTokensBuyLimitOrderPrefixStatus(req.TokenSymbol, status), fromKey, req.Count, direction)
	if err != nil {
		return nil, err
	}
	var reply types.ReplyBuyLimitOrders
	for _, buyid := range values {
		if buyOrder, err := getBuyOrderFromID(buyid, t.GetStateDB()); err == nil {
			tradelog.Debug("trade Query", "getBuyOrderFromID", string(buyid))
			reply.BuyOrders = append(reply.BuyOrders, buyOrder)
		}
	}
	return &reply, nil
}

