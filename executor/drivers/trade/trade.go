package trade

/*
trade执行器支持trade的创建和交易，

主要提供操作有以下几种：
1）挂单出售；
2）购买指定的卖单；
3）撤销卖单；
*/

import (
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/executor/drivers"
	"code.aliyun.com/chain33/chain33/types"
	"fmt"
	log "github.com/inconshreveable/log15"
)

var tradelog = log.New("module", "execs.trade")

func init() {
	t := newTrade()
	drivers.Register(t.GetName(), t)
	drivers.RegisterAddress(t.GetName())
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
	default:
	}
	tradelog.Error("trade Query", "Query type not supprt with func name", funcName)
	return nil, types.ErrQueryNotSupport
}

func (t *trade) GetOnesSellOrder(db dbm.KVDB, querydb dbm.DB, addrTokens *types.ReqAddrTokens) (types.Message, error) {
	sellidGotAlready := make(map[string]bool)
	var sellids [][]byte
	if 0 == len(addrTokens.Token) {
		values := querydb.List(calcOnesSellOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
		if len(values) != 0 {
			tradelog.Debug("trade Query", "get number of sellid", len(values))
			sellids = append(sellids, values...)
		}
	} else {
		for _, token := range addrTokens.Token {
			values := querydb.List(calcOnesSellOrderPrefixToken(addrTokens.Addr, token), nil, 0, 0)
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
				reply.Selloders = append(reply.Selloders, sellorder)
			}
			sellidGotAlready[string(sellid)] = true
		}
	}
	return &reply, nil
}

func (t *trade) GetOnesBuyOrder(db dbm.KVDB, querydb dbm.DB, addrTokens *types.ReqAddrTokens) (types.Message, error) {
	sellidGotAlready := make(map[interface{}]bool)
	var reply types.ReplyTradeBuyOrders
	values := querydb.List(calcOnesBuyOrderPrefixAddr(addrTokens.Addr), nil, 0, 0)
	if len(values) != 0 {
		tradelog.Debug("GetOnesBuyOrder", "get number of buy order", len(values))
		for _, value := range values {
			//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
			if !sellidGotAlready[value] {
				var tradeBuyDone types.TradeBuyDone
				if err := types.Decode(value, &tradeBuyDone); err != nil {
					tradelog.Error("GetOnesBuyOrder", "Failed to decode tradebuydoen", value)
					return nil, err
				}

				reply.Tradebuydones = append(reply.Tradebuydones, &tradeBuyDone)
				sellidGotAlready[value] = true
			}
		}
	}
	var resultReply types.ReplyTradeBuyOrders
	if len(addrTokens.Token) != 0 {
		tokenMap := make(map[string]bool)
		for _, token := range addrTokens.Token {
			tokenMap[token] = true
		}

		for _, Tradebuydone := range reply.Tradebuydones {
			if tokenMap[Tradebuydone.Token] {
				resultReply.Tradebuydones = append(resultReply.Tradebuydones, Tradebuydone)
			}
		}
	}

	return &resultReply, nil
}

func (t *trade) GetAllSellOrdersWithStatus(db dbm.KVDB, querydb dbm.DB, status int32) (types.Message, error) {
	sellidGotAlready := make(map[string]bool)
	var sellids [][]byte
	values := querydb.List(calcTokenSellOrderPrefixStatus(status), nil, 0, 0)
	if len(values) != 0 {
		tradelog.Debug("trade Query", "get number of sellid", len(values))
		sellids = append(sellids, values...)
	}

	var reply types.ReplySellOrders
	for _, sellid := range sellids {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !sellidGotAlready[string(sellid)] {
			if sellorder, err := getSellOrderFromID(sellid, db); err == nil {
				//reply.Selloders = append(reply.Selloders, sellorder)
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

func (t *trade) saveSell(sellid []byte, ty int32) []*types.KeyValue {
	db := t.GetDB()
	value, err := db.Get(sellid)
	if err != nil {
		panic(err)
	}
	var sellorder types.SellOrder
	types.Decode(value, &sellorder)
	status := sellorder.Status
	var kv []*types.KeyValue
	newkey := calcTokenSellOrderKey(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid, sellorder.Height)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	newkey = calcOnesSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	newkey = calcOnesSellOrderKeyToken(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, sellid})

	if types.SoldOut == status || types.Revoked == status {
		tradelog.Debug("trade saveSell ", "remove old status onsale to soldout or revoked with sellid", sellorder.Sellid)
		kv = deleteSellOrderKeyValue(kv, &sellorder, types.OnSale)
	}

	return kv
}

func deleteSellOrderKeyValue(kv []*types.KeyValue, sellorder *types.SellOrder, status int32) []*types.KeyValue {
	newkey := calcTokenSellOrderKey(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid, sellorder.Height)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	newkey = calcOnesSellOrderKeyToken(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, nil})

	return kv
}

func (t *trade) deleteSell(sellid []byte, ty int32) []*types.KeyValue {
	db := t.GetDB()
	value, err := db.Get(sellid)
	if err != nil {
		panic(err)
	}
	var sellorder types.SellOrder
	types.Decode(value, &sellorder)
	status := sellorder.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, &sellorder, status)
	return kv
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

//特定状态下的卖单
func calcTokenSellOrderKey(token string, addr string, status int32, sellOrderID string, height int64) []byte {
	key := fmt.Sprintf("token-sellorder-shtas:%d:%d:%s:%s:%s", status, height, token, addr, sellOrderID)
	return []byte(key)
}

//特定账户下特定状态的卖单
func calcOnesSellOrderKeyStatus(token string, addr string, status int32, sellOrderID string) []byte {
	key := fmt.Sprintf("token-sellorder-asts:%s:%d:%s:%s", addr, status, token, sellOrderID)
	return []byte(key)
}

//特定账户下特定token的卖单
func calcOnesSellOrderKeyToken(token string, addr string, status int32, sellOrderID string) []byte {
	key := fmt.Sprintf("token-sellorder-atss:%s:%s:%d:%s", addr, token, status, sellOrderID)
	return []byte(key)
}

//特定账户下指定token的卖单
func calcOnesSellOrderPrefixToken(token string, addr string) []byte {
	return []byte(fmt.Sprintf("token-sellorder-atss:%s:%s", addr, token))
}

//特定账户下的卖单
func calcOnesSellOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf("token-sellorder-asts:%s", addr))
}

//特定状态下的卖单
func calcTokenSellOrderPrefixStatus(status int32) []byte {
	return []byte(fmt.Sprintf("token-sellorder-shtas:%d", status))
}

//考虑到购买者可能在同一个区块时间针对相同的卖单(sellorder)发起购买操作，所以使用sellid：buytxhash组合的方式生成key
func calcOnesBuyOrderKey(addr string, height int64, token string, sellOrderID string, buyTxHash string) []byte {
	return []byte(fmt.Sprintf("token-buyorder:%s:%d:%s:%s:%s", addr, height, token, sellOrderID, buyTxHash))
}

//用于快速查询某个token下的所有成交的买单
func calcBuyOrderKey(addr string, height int64, token string, sellOrderID string, buyTxHash string) []byte {
	return []byte(fmt.Sprintf("token-buyorder:%s:%s:%d:%s:%s", token, addr, height, sellOrderID, buyTxHash))
}

//特定账户下的卖单
func calcOnesBuyOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf("token-buyorder:%s", addr))
}
