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
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/executor/drivers/token"
	"gitlab.33.cn/chain33/chain33/types"
)

var tradelog = log.New("module", "execs.trade")

func init() {
	t := newTrade()
	drivers.Register(t.GetName(), t, types.ForkV2_add_token)
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

func (t *trade) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var trade types.Trade
	err := types.Decode(tx.Payload, &trade)
	if err != nil {
		return nil, err
	}
	tradelog.Info("exec trade tx=", "tx hash", common.Bytes2Hex(tx.Hash()), "Ty", trade.GetTy())

	action := newTradeAction(t, tx)
	switch trade.GetTy() {
	case types.TradeSell:
		return action.tradeSell(trade.GetTokensell())

	case types.TradeBuy:
		return action.tradeBuy(trade.GetTokenbuy())

	case types.TradeRevokeSell:
		return action.tradeRevokeSell(trade.GetTokenrevokesell())

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
			var receipt types.ReceiptTradeBuy
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveBuy(&receipt)
			set.KV = append(set.KV, kv...)

			// 添加个人资产列表
			kv = token.AddTokenToAssets(receipt.Buyeraddr, t.GetLocalDB(), receipt.Token)
			if kv != nil {
				set.KV = append(set.KV, kv...)
			}
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
			var receipt types.ReceiptTradeBuy
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.deleteBuy(&receipt)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (t *trade) Query(funcName string, params []byte) (types.Message, error) {
	switch funcName {
	//查询某个特定用户的一个或者多个token的卖单,包括所有状态的卖单
	//TODO:后续可以考虑支持查询不同状态的卖单
	case "GetOnesSellOrder":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetOnesSellOrder(t.GetDB(), t.GetQueryDB(), &addrTokens)
	case "GetOnesBuyOrder":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetOnesBuyOrder(t.GetDB(), t.GetQueryDB(), &addrTokens)
		//查寻所有的可以进行交易的卖单
	case "GetAllSellOrdersWithStatus":
		var addrTokens types.ReqAddrTokens
		err := types.Decode(params, &addrTokens)
		if err != nil {
			return nil, err
		}
		return t.GetAllSellOrdersWithStatus(t.GetDB(), t.GetQueryDB(), addrTokens.Status)
	case "GetTokenSellOrderByStatus":
		var req types.ReqTokenSellOrder
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return t.GetTokenByStatus(t.GetDB(), t.GetQueryDB(), &req, types.OnSale)
	default:
	}
	tradelog.Error("trade Query", "Query type not supprt with func name", funcName)
	return nil, types.ErrQueryNotSupport
}

func (t *trade) GetOnesSellOrder(db dbm.KVDB, querydb dbm.DB, addrTokens *types.ReqAddrTokens) (types.Message, error) {
	sellidGotAlready := make(map[string]bool)
	var sellids [][]byte
	if 0 == len(addrTokens.Token) {
		list := dbm.NewListHelper(querydb)
		values := list.List(calcOnesSellOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
		if len(values) != 0 {
			tradelog.Debug("trade Query", "get number of sellid", len(values))
			sellids = append(sellids, values...)
		}
	} else {
		for _, token := range addrTokens.Token {
			list := dbm.NewListHelper(querydb)
			values := list.List(calcOnesSellOrderPrefixToken(token, addrTokens.Addr), nil, 0, 0)
			tradelog.Debug("trade Query", "Begin to list addr with token", token, "got values", len(values))
			if len(values) != 0 {
				sellids = append(sellids, values...)
			}
		}
	}

	var reply types.ReplySellOrders
	for _, sellid := range sellids {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !sellidGotAlready[string(sellid)] {
			if sellorder, err := getSellOrderFromID(sellid, db); err == nil {
				tradelog.Debug("trade Query", "getSellOrderFromID", string(sellid))
				reply.Selloders = insertSellOrderDescending(sellorder, reply.Selloders)
			}
			sellidGotAlready[string(sellid)] = true
		}
	}
	return &reply, nil
}

func (t *trade) GetOnesBuyOrder(db dbm.KVDB, querydb dbm.DB, addrTokens *types.ReqAddrTokens) (types.Message, error) {
	gotAlready := make(map[interface{}]bool)
	var reply types.ReplyTradeBuyOrders
	list := dbm.NewListHelper(querydb)
	values := list.List(calcOnesBuyOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
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

func (t *trade) GetAllSellOrdersWithStatus(db dbm.KVDB, querydb dbm.DB, status int32) (types.Message, error) {
	sellidGotAlready := make(map[string]bool)
	var sellids [][]byte
	list := dbm.NewListHelper(querydb)
	values := list.List(calcTokenSellOrderPrefixStatus(status), nil, 0, 0)
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of sellid", len(values))
		sellids = append(sellids, values...)
	}

	var reply types.ReplySellOrders
	for _, sellid := range sellids {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !sellidGotAlready[string(sellid)] {
			if sellorder, err := getSellOrderFromID(sellid, db); err == nil {
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

func (t *trade) GetTokenByStatus(db dbm.KVDB, querydb dbm.DB, req *types.ReqTokenSellOrder, status int32) (types.Message, error) {
	if req.Count <= 0 || (req.Direction != 1 && req.Direction != 0) {
		return nil, types.ErrInputPara
	}

	fromKey := []byte("")
	if len(req.FromSellId) != 0 {
		sellorder, err := getSellOrderFromID([]byte(req.FromSellId), db)
		if err != nil {
			tradelog.Error("GetTokenByStatus get sellorder err", err)
			return nil, err
		}
		fromKey = calcTokensSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Status, 1e8*sellorder.Priceperboardlot/sellorder.Amountperboardlot, sellorder.Address, sellorder.Sellid)
	}

	list := dbm.NewListHelper(querydb)
	prefix := calcTokensSellOrderPrefixStatus(req.TokenSymbol, status)
	values := list.List(prefix, fromKey, req.Count, req.Direction)

	var reply types.ReplySellOrders
	for _, sellid := range values {
		if sellorder, err := getSellOrderFromID(sellid, db); err == nil {
			tradelog.Debug("trade Query", "getSellOrderFromID", string(sellid))
			reply.Selloders = append(reply.Selloders, sellorder)
		}
	}
	return &reply, nil
}

func (t *trade) getSellOrderFromDb(db dbm.KVDB, sellid []byte) *types.SellOrder {
	value, err := db.Get(sellid)
	if err != nil {
		panic(err)
	}
	var sellorder types.SellOrder
	types.Decode(value, &sellorder)
	return &sellorder
}

func genSaveSellKv(sellorder *types.SellOrder ) []*types.KeyValue {
	status := sellorder.Status
	var kv []*types.KeyValue
	kv = saveSellOrderKeyValue(kv, sellorder, status)
	if types.SoldOut == status || types.Revoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", sellorder.Sellid)
		kv = deleteSellOrderKeyValue(kv, sellorder, types.OnSale)
	}
	return kv
}

func (t *trade) saveSell(sellid []byte, ty int32) []*types.KeyValue {
	sellorder := t.getSellOrderFromDb(t.GetDB(), sellid)
	return genSaveSellKv(sellorder)
}

func deleteSellOrderKeyValue(kv []*types.KeyValue, sellorder *types.SellOrder, status int32) []*types.KeyValue {
	newkey := calcTokenSellOrderKey(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid, sellorder.Height)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyToken(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcTokensSellOrderKeyStatus(sellorder.Tokensymbol, status, sellorder.Priceperboardlot/sellorder.Amountperboardlot, sellorder.Address, sellorder.Sellid)
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

	// make a number as token's price whether cheap or dear
	// support 1e8 bty pre token or 1/1e8 bty pre token,
	// the number in key is used to sort sell orders and pages
	newkey = calcTokensSellOrderKeyStatus(sellorder.Tokensymbol, status, sellorder.Priceperboardlot/sellorder.Amountperboardlot, sellorder.Address, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	return kv
}

func genDeleteSellKv(sellorder *types.SellOrder) []*types.KeyValue {
	status := sellorder.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, sellorder, status)
	if types.SoldOut == status || types.Revoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", sellorder.Sellid)
		kv = saveSellOrderKeyValue(kv, sellorder, types.OnSale)
	}
	return kv
}

func (t *trade) deleteSell(sellid []byte, ty int32) []*types.KeyValue {
	sellorder := t.getSellOrderFromDb(t.GetDB(), sellid)
	return genDeleteSellKv(sellorder)
}

func (t *trade) saveBuy(receiptTradeBuy *types.ReceiptTradeBuy) []*types.KeyValue {

	tradeBuyDone := types.TradeBuyDone{}
	tradeBuyDone.Token = receiptTradeBuy.Token
	tradeBuyDone.Boardlotcnt = receiptTradeBuy.Boardlotcnt
	tradeBuyDone.Amountperboardlot = receiptTradeBuy.Amountperboardlot
	tradeBuyDone.Priceperboardlot = receiptTradeBuy.Priceperboardlot
	tradeBuyDone.Buytxhash = receiptTradeBuy.Buytxhash
	tradeBuyDone.Height = t.GetHeight()

	var kv []*types.KeyValue
	value := types.Encode(&tradeBuyDone)

	key := calcOnesBuyOrderKey(receiptTradeBuy.Buyeraddr, t.GetHeight(), receiptTradeBuy.Token, receiptTradeBuy.Sellid, receiptTradeBuy.Buytxhash)
	kv = append(kv, &types.KeyValue{key, value})

	key = calcBuyOrderKey(receiptTradeBuy.Buyeraddr, t.GetHeight(), receiptTradeBuy.Token, receiptTradeBuy.Sellid, receiptTradeBuy.Buytxhash)
	kv = append(kv, &types.KeyValue{key, value})

	return kv
}

func (t *trade) deleteBuy(receiptTradeBuy *types.ReceiptTradeBuy) []*types.KeyValue {
	var kv []*types.KeyValue

	key := calcOnesBuyOrderKey(receiptTradeBuy.Buyeraddr, t.GetHeight(), receiptTradeBuy.Token, receiptTradeBuy.Sellid, receiptTradeBuy.Buytxhash)
	kv = append(kv, &types.KeyValue{key, nil})

	key = calcBuyOrderKey(receiptTradeBuy.Buyeraddr, t.GetHeight(), receiptTradeBuy.Token, receiptTradeBuy.Sellid, receiptTradeBuy.Buytxhash)
	kv = append(kv, &types.KeyValue{key, nil})

	return kv
}
