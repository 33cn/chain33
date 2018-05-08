package trade

/*
trade执行器支持trade的创建和交易，

主要提供操作有以下几种：
1）挂单出售；
2）购买指定的卖单；
3）撤销卖单；
*/

import (
	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/executor/drivers/token"
	"gitlab.33.cn/chain33/chain33/types"
	"strconv"
	"strings"
	"fmt"
)

var tradelog = log.New("module", "execs.trade")

func init() {
	t := newTrade()
	drivers.Register(t.GetName(), t, types.ForkV2AddToken)
}

type trade struct {
	drivers.DriverBase
}

func newTrade() *trade {
	t := &trade{}
	t.SetChild(t)
	return t
}

func (t *trade) GetName() string {
	return "trade"
}

func (t *trade) Clone() drivers.Driver {
	clone := &trade{}
	clone.DriverBase = *(t.DriverBase.Clone().(*drivers.DriverBase))
	clone.SetChild(clone)
	return clone
}

func (t *trade) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var trade types.Trade
	err := types.Decode(tx.Payload, &trade)
	if err != nil {
		return nil, err
	}
	tradelog.Info("exec trade tx=", "tx hash", common.Bytes2Hex(tx.Hash()), "Ty", trade.GetTy())

	action := newTradeAction(t, tx)
	switch trade.GetTy() {
	case types.TradeSellLimit:
		return action.tradeSell(trade.GetTokensell())

	case types.TradeBuyMarket:
		return action.tradeBuy(trade.GetTokenbuy())

	case types.TradeRevokeSell:
		return action.tradeRevokeSell(trade.GetTokenrevokesell())

	case types.TradeBuyLimit:
		if t.GetHeight() < types.ForTradeBuyLimit {
			return nil, types.ErrActionNotSupport
		}
		return action.tradeBuyLimit(trade.GetTokenbuylimit())

	case types.TradeSellMarket:
		if t.GetHeight() < types.ForTradeBuyLimit {
			return nil, types.ErrActionNotSupport
		}
		return action.tradeSellMarket(trade.GetTokensellmarket())

	case types.TradeRevokeBuy:
		if t.GetHeight() < types.ForTradeBuyLimit {
			return nil, types.ErrActionNotSupport
		}
		return action.tradeRevokeBuyLimit(trade.GetTokenrevokebuy())

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
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogTradeSell || item.Ty == types.TyLogTradeRevoke {
			var receipt types.ReceiptTradeSell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveSell([]byte(receipt.Base.Sellid), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTradeBuy {
			var receipt types.ReceiptBuyBase
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveBuy(&receipt)
			set.KV = append(set.KV, kv...)

			// 添加个人资产列表
			kv = token.AddTokenToAssets(receipt.Owner, t.GetLocalDB(), receipt.TokenSymbol)
			if kv != nil {
				set.KV = append(set.KV, kv...)
			}
		} else if item.Ty == types.TyLogTradeBuyRevoke || item.Ty == types.TyLogTradeBuyLimit {
			var receipt types.ReceiptTradeBuyLimit
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}

			kv := t.saveBuyLimit([]byte(receipt.Base.Buyid), item.Ty)
			set.KV = append(set.KV, kv...)

			// 添加个人资产列表
			kv = token.AddTokenToAssets(receipt.Base.Owner, t.GetLocalDB(), receipt.Base.TokenSymbol)
			if kv != nil {
				set.KV = append(set.KV, kv...)
			}
		} else if item.Ty == types.TyLogTradeSellMarket {
			var receipt types.ReceiptTradeBase
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveSellMarket(&receipt)
			tradelog.Info("saveSellMarket", "kv", kv)
			set.KV = append(set.KV, kv...)
		}
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

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogTradeSell || item.Ty == types.TyLogTradeRevoke {
			var receipt types.ReceiptTradeSell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteSell([]byte(receipt.Base.Sellid), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTradeBuy {
			var receipt types.ReceiptBuyBase
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteBuy(&receipt)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogTradeBuyRevoke || item.Ty == types.TyLogTradeBuyLimit {
			var receipt types.ReceiptTradeBuyLimit
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteBuyLimit([]byte(receipt.Base.Buyid), item.Ty)
			set.KV = append(set.KV, kv...)

		} else if item.Ty == types.TyLogTradeSellMarket {
			var receipt types.ReceiptTradeBase
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteSellMarket(&receipt)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (t *trade) Query(funcName string, params []byte) (types.Message, error) {
	tradelog.Info("trade Query", "name", funcName)
	switch funcName {
	//查询某个特定用户的一个或者多个token的卖单,包括所有状态的卖单
	//TODO:后续可以考虑支持查询不同状态的卖单
	case "GetOnesSellOrder":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetOnesSellOrder(&addrTokens)
	case "GetOnesBuyOrder":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetOnesBuyOrder(&addrTokens)
		//查寻所有的可以进行交易的卖单
	case "GetAllSellOrdersWithStatus":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetAllSellOrdersWithStatus(addrTokens.Status)
	case "GetTokenSellOrderByStatus": // 根据token 分页显示未完成成交卖单
		var req types.ReqTokenSellOrder
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		if req.Status == 0 {
			req.Status = types.TracdOrderStatusOnSale
		}
		return t.GetTokenByStatus(&req, req.Status)
	case "GetTokenBuyLimitOrderByStatus": // 根据token 分页显示未完成成交买单
		var req types.ReqTokenBuyLimitOrder
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		if req.Status == 0 {
			req.Status = types.TracdOrderStatusOnBuy
		}
		return t.GetTokenBuyLimitOrderByStatus(&req, req.Status)
	case "GetAllBuyOrdersWithStatus":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetAllSellOrdersWithStatus(addrTokens.Status)

	default:
	}
	tradelog.Error("trade Query", "Query type not supprt with func name", funcName)
	return nil, types.ErrQueryNotSupport
}

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

func (t *trade) getSellOrderFromDb(sellid []byte) *types.SellOrder {
	value, err := t.GetStateDB().Get(sellid)
	if err != nil {
		panic(err)
	}
	var sellorder types.SellOrder
	types.Decode(value, &sellorder)
	return &sellorder
}

func genSaveSellKv(sellorder *types.SellOrder) []*types.KeyValue {
	status := sellorder.Status
	var kv []*types.KeyValue
	kv = saveSellOrderKeyValue(kv, sellorder, status)
	if types.TracdOrderStatusSoldOut == status || types.TracdOrderStatusRevoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", sellorder.Sellid)
		kv = deleteSellOrderKeyValue(kv, sellorder, types.TracdOrderStatusOnSale)
	}
	return kv
}

func (t *trade) saveSell(sellid []byte, ty int32) []*types.KeyValue {
	sellorder := t.getSellOrderFromDb(sellid)
	return genSaveSellKv(sellorder)
}

func deleteSellOrderKeyValue(kv []*types.KeyValue, sellorder *types.SellOrder, status int32) []*types.KeyValue {
	newkey := calcTokenSellOrderKey(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid, sellorder.Height)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyToken(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcTokensSellOrderKeyStatus(sellorder.Tokensymbol, status,
		calcPriceOfToken(sellorder.Priceperboardlot, sellorder.Amountperboardlot), sellorder.Address, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	return kv
}

func saveSellOrderKeyValue(kv []*types.KeyValue, sellorder *types.SellOrder, status int32) []*types.KeyValue {
	sellid := []byte(sellorder.Sellid)
	newkey := calcTokenSellOrderKey(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid, sellorder.Height)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	newkey = calcOnesSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	newkey = calcOnesSellOrderKeyToken(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	newkey = calcTokensSellOrderKeyStatus(sellorder.Tokensymbol, status,
		calcPriceOfToken(sellorder.Priceperboardlot, sellorder.Amountperboardlot), sellorder.Address, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	return kv
}

func genDeleteSellKv(sellorder *types.SellOrder) []*types.KeyValue {
	status := sellorder.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, sellorder, status)
	if types.TracdOrderStatusSoldOut == status || types.TracdOrderStatusRevoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", sellorder.Sellid)
		kv = saveSellOrderKeyValue(kv, sellorder, types.TracdOrderStatusOnSale)
	}
	return kv
}

func (t *trade) deleteSell(sellid []byte, ty int32) []*types.KeyValue {
	sellorder := t.getSellOrderFromDb(sellid)
	return genDeleteSellKv(sellorder)
}

func (t *trade) saveBuy(receiptTradeBuy *types.ReceiptBuyBase) []*types.KeyValue {
	var kv []*types.KeyValue
	return saveBuyMarketOrderKeyValue(kv, receiptTradeBuy, types.TracdOrderStatusBoughtOut, t.GetHeight())
}

func (t *trade) deleteBuy(receiptTradeBuy *types.ReceiptBuyBase) []*types.KeyValue {
	var kv []*types.KeyValue
	return deleteBuyMarketOrderKeyValue(kv, receiptTradeBuy, types.TracdOrderStatusBoughtOut, t.GetHeight())
}

// BuyLimit Local
func (t *trade) getBuyOrderFromDb(buyid []byte) *types.BuyLimitOrder {
	value, err := t.GetStateDB().Get(buyid)
	if err != nil {
		panic(err)
	}
	var buyOrder types.BuyLimitOrder
	types.Decode(value, &buyOrder)
	return &buyOrder
}

func genSaveBuyLimitKv(buyOrder *types.BuyLimitOrder) []*types.KeyValue {
	status := buyOrder.Status
	var kv []*types.KeyValue
	kv = saveBuyLimitOrderKeyValue(kv, buyOrder, status)
	if types.TracdOrderStatusBoughtOut == status || types.TracdOrderStatusBuyRevoked == status {
		tradelog.Debug("trade saveBuyLimit ", "remove old status with Buyid", buyOrder.Buyid)
		kv = deleteBuyLimitKeyValue(kv, buyOrder, types.TracdOrderStatusOnSale)
	}
	return kv
}

func (t *trade) saveBuyLimit(buyid []byte, ty int32) []*types.KeyValue {
	buyOrder := t.getBuyOrderFromDb(buyid)
	return genSaveBuyLimitKv(buyOrder)
}

func saveBuyLimitOrderKeyValue(kv []*types.KeyValue, buyOrder *types.BuyLimitOrder, status int32) []*types.KeyValue {
	buyid := []byte(buyOrder.Buyid)
	newkey := calcTokenBuyLimitOrderKey(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid, buyOrder.Height)
	kv = append(kv, &types.KeyValue{newkey, buyid})

	newkey = calcOnesBuyLimitOrderKeyStatus(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, buyid})

	newkey = calcOnesBuyLimitOrderKeyToken(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, buyid})

	newkey = calcTokensBuyLimitOrderKeyStatus(buyOrder.TokenSymbol, status,
		calcPriceOfToken(buyOrder.PricePerBoardlot, buyOrder.AmountPerBoardlot), buyOrder.Address, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, buyid})

	return kv
}


func deleteBuyLimitKeyValue(kv []*types.KeyValue, buyOrder *types.BuyLimitOrder, status int32) []*types.KeyValue {
	newkey := calcTokenBuyLimitOrderKey(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid, buyOrder.Height)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesBuyLimitOrderKeyStatus(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesBuyLimitOrderKeyToken(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcTokensBuyLimitOrderKeyStatus(buyOrder.TokenSymbol, status,
		buyOrder.PricePerBoardlot/buyOrder.AmountPerBoardlot, buyOrder.Address, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	return kv
}

func genDeleteBuyLimitKv(buyOrder *types.BuyLimitOrder) []*types.KeyValue {
	status := buyOrder.Status
	var kv []*types.KeyValue
	kv = deleteBuyLimitKeyValue(kv, buyOrder, status)
	if types.TracdOrderStatusBoughtOut == status || types.TracdOrderStatusBuyRevoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", buyOrder.Buyid)
		kv = saveBuyLimitOrderKeyValue(kv, buyOrder, types.TracdOrderStatusOnBuy)
	}
	return kv
}

func (t *trade) deleteBuyLimit(buyid []byte, ty int32) []*types.KeyValue {
	buyOrder := t.getBuyOrderFromDb(buyid)
	return genDeleteBuyLimitKv(buyOrder)
}

func (t *trade) saveSellMarket(receiptTradeBuy *types.ReceiptTradeBase) []*types.KeyValue {
	var kv []*types.KeyValue
	return saveSellMarketOrderKeyValue(kv, receiptTradeBuy, types.TracdOrderStatusSoldOut, t.GetHeight())
}

func (t *trade) deleteSellMarket(receiptTradeBuy *types.ReceiptTradeBase) []*types.KeyValue {
	var kv []*types.KeyValue
	return deleteSellMarketOrderKeyValue(kv, receiptTradeBuy, types.TracdOrderStatusSoldOut, t.GetHeight())
}

func saveSellMarketOrderKeyValue(kv []*types.KeyValue, receipt *types.ReceiptTradeBase, status int32, height int64) []*types.KeyValue {
	txhash := []byte(receipt.Txhash)
	keyID := receipt.Txhash

	newkey := calcTokenSellOrderKey(receipt.Tokensymbol, receipt.Owner, status, keyID, height)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	newkey = calcOnesSellOrderKeyStatus(receipt.Tokensymbol, receipt.Owner, status, keyID)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	newkey = calcOnesSellOrderKeyToken(receipt.Tokensymbol, receipt.Owner, status, keyID)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	priceBoardlot, err := strconv.ParseFloat(receipt.Priceperboardlot, 64)
	if err != nil {
		panic(err)
	}
	priceBoardlotInt64 := int64(priceBoardlot * float64(types.TokenPrecision))
	AmountPerBoardlot, err := strconv.ParseFloat(receipt.Amountperboardlot, 64)
	if err != nil {
		panic(err)
	}
	AmountPerBoardlotInt64 := int64(AmountPerBoardlot * float64(types.Coin))
	price := calcPriceOfToken(priceBoardlotInt64, AmountPerBoardlotInt64)

	newkey = calcTokensSellOrderKeyStatus(receipt.Tokensymbol, status,
		price, receipt.Owner, keyID)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	return kv
}

func deleteSellMarketOrderKeyValue(kv []*types.KeyValue, receipt *types.ReceiptTradeBase, status int32, height int64) []*types.KeyValue {
	keyID := receipt.Txhash

	newkey := calcTokenSellOrderKey(receipt.Tokensymbol, receipt.Owner, status, keyID, height)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyStatus(receipt.Tokensymbol, receipt.Owner, status, keyID)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyToken(receipt.Tokensymbol, receipt.Owner, status, keyID)
	kv = append(kv, &types.KeyValue{newkey, nil})

	priceBoardlot, err := strconv.ParseFloat(receipt.Priceperboardlot, 64)
	if err != nil {
		panic(err)
	}
	priceBoardlotInt64 := int64(priceBoardlot * float64(types.TokenPrecision))
	AmountPerBoardlot, err := strconv.ParseFloat(receipt.Amountperboardlot, 64)
	if err != nil {
		panic(err)
	}
	AmountPerBoardlotInt64 := int64(AmountPerBoardlot * float64(types.Coin))
	price := calcPriceOfToken(priceBoardlotInt64, AmountPerBoardlotInt64)

	newkey = calcTokensSellOrderKeyStatus(receipt.Tokensymbol, status,
		price, receipt.Owner, keyID)
	kv = append(kv, &types.KeyValue{newkey, nil})

	return kv
}

func saveBuyMarketOrderKeyValue(kv []*types.KeyValue, receipt *types.ReceiptBuyBase, status int32, height int64) []*types.KeyValue {
	txhash := []byte(receipt.Txhash)
	keyId := receipt.Txhash

	newkey := calcTokenBuyLimitOrderKey(receipt.TokenSymbol, receipt.Owner, status, keyId, height)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	newkey = calcOnesBuyLimitOrderKeyStatus(receipt.TokenSymbol, receipt.Owner, status, keyId)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	newkey = calcOnesBuyLimitOrderKeyToken(receipt.TokenSymbol, receipt.Owner, status, keyId)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	priceBoardlot, err := strconv.ParseFloat(receipt.PricePerBoardlot, 64)
	if err != nil {
		panic(err)
	}
	priceBoardlotInt64 := int64(priceBoardlot * float64(types.TokenPrecision))
	AmountPerBoardlot, err := strconv.ParseFloat(receipt.AmountPerBoardlot, 64)
	if err != nil {
		panic(err)
	}
	AmountPerBoardlotInt64 := int64(AmountPerBoardlot * float64(types.Coin))
	price := calcPriceOfToken(priceBoardlotInt64, AmountPerBoardlotInt64)

	newkey = calcTokensBuyLimitOrderKeyStatus(receipt.TokenSymbol, status,
		price, receipt.Owner, keyId)
	kv = append(kv, &types.KeyValue{newkey, txhash})

	return kv
}

func deleteBuyMarketOrderKeyValue(kv []*types.KeyValue, receipt *types.ReceiptBuyBase, status int32, height int64) []*types.KeyValue {
	keyId := receipt.Txhash

	newkey := calcTokenBuyLimitOrderKey(receipt.TokenSymbol, receipt.Owner, status, keyId, height)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesBuyLimitOrderKeyStatus(receipt.TokenSymbol, receipt.Owner, status, keyId)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesBuyLimitOrderKeyToken(receipt.TokenSymbol, receipt.Owner, status, keyId)
	kv = append(kv, &types.KeyValue{newkey, nil})


	priceBoardlot, err := strconv.ParseFloat(receipt.PricePerBoardlot, 64)
	if err != nil {
		panic(err)
	}
	priceBoardlotInt64 := int64(priceBoardlot * float64(types.TokenPrecision))
	AmountPerBoardlot, err := strconv.ParseFloat(receipt.AmountPerBoardlot, 64)
	if err != nil {
		panic(err)
	}
	AmountPerBoardlotInt64 := int64(AmountPerBoardlot * float64(types.Coin))
	price := calcPriceOfToken(priceBoardlotInt64, AmountPerBoardlotInt64)

	newkey = calcTokensBuyLimitOrderKeyStatus(receipt.TokenSymbol, status,
		price, receipt.Owner, keyId)
	kv = append(kv, &types.KeyValue{newkey, nil})

	return kv
}

// make a number as token's price whether cheap or dear
// support 1e8 bty pre token or 1/1e8 bty pre token, [1Coins, 1e16Coins]
// the number in key is used to sort buy orders and pages
func calcPriceOfToken(priceBoardlot, AmountPerBoardlot int64) int64 {
	return 1e8 * priceBoardlot / AmountPerBoardlot
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
