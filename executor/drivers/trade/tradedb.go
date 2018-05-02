package trade

import (
	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/token"
	"gitlab.33.cn/chain33/chain33/types"
)

type sellDB struct {
	types.SellOrder
}

func newSellDB(sellOrder types.SellOrder) (selldb *sellDB) {
	selldb = &sellDB{sellOrder}
	if types.InvalidStartTime != selldb.Starttime {
		selldb.Status = types.TracdOrderStatusNotStart
	}
	return
}

func (selldb *sellDB) save(db dbm.KV) []*types.KeyValue {
	set := selldb.getKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}

	return set
}

func (selldb *sellDB) getSellLogs(tradeType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = tradeType
	base := &types.ReceiptTradeBase{
		selldb.Tokensymbol,
		selldb.Address,
		strconv.FormatFloat(float64(selldb.Amountperboardlot)/float64(types.TokenPrecision), 'f', 4, 64),
		selldb.Minboardlot,
		strconv.FormatFloat(float64(selldb.Priceperboardlot)/float64(types.Coin), 'f', 8, 64),
		selldb.Totalboardlot,
		selldb.Soldboardlot,
		selldb.Starttime,
		selldb.Stoptime,
		selldb.Crowdfund,
		selldb.Sellid,
		types.SellOrderStatus[selldb.Status],
	}
	if types.TyLogTradeSell == tradeType {
		receiptTrade := &types.ReceiptTradeSell{base}
		log.Log = types.Encode(receiptTrade)

	} else if types.TyLogTradeRevoke == tradeType {
		receiptTrade := &types.ReceiptTradeRevoke{base}
		log.Log = types.Encode(receiptTrade)
	}

	return log
}

func (selldb *sellDB) getBuyLogs(buyerAddr string, sellid string, boardlotcnt int64, sellOrder *types.SellOrder, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogTradeBuy
	receiptBuy := &types.ReceiptTradeBuy{
		buyerAddr,
		sellid,
		selldb.Tokensymbol,
		boardlotcnt,
		strconv.FormatFloat(float64(sellOrder.Amountperboardlot)/float64(types.TokenPrecision), 'f', 4, 64),
		strconv.FormatFloat(float64(sellOrder.Priceperboardlot)/float64(types.Coin), 'f', 8, 64),
		txhash,
	}
	log.Log = types.Encode(receiptBuy)

	return log
}

func getSellOrderFromID(sellid []byte, db dbm.KV) (*types.SellOrder, error) {
	value, err := db.Get(sellid)
	if err != nil {
		tradelog.Error("getSellOrderFromID", "Failed to get value frim db wiht sellid", string(sellid))
		return nil, err
	}

	var sellOrder types.SellOrder
	if err = types.Decode(value, &sellOrder); err != nil {
		tradelog.Error("getSellOrderFromID", "Failed to decode sell order", string(sellid))
		return nil, err
	}
	return &sellOrder, nil
}

func (selldb *sellDB) getKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&selldb.SellOrder)
	key := []byte(selldb.Sellid)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
}

type buyDB struct {
	types.BuyLimitOrder
}

func newBuyDB(sellOrder types.BuyLimitOrder) (buydb *buyDB) {
	buydb = &buyDB{sellOrder}
	return
}

func (selldb *buyDB) save(db dbm.KV) []*types.KeyValue {
	set := selldb.getKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}

	return set
}

func (buydb *buyDB) getKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&buydb.BuyLimitOrder)
	key := []byte(buydb.Buyid)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
}

func (buydb *buyDB) getBuyLogs(tradeType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = tradeType
	base := &types.ReceiptBuyLimitBase{
		buydb.TokenSymbol,
		buydb.Address,
		strconv.FormatFloat(float64(buydb.AmountPerBoardlot)/float64(types.TokenPrecision), 'f', 4, 64),
		buydb.MinBoardlot,
		strconv.FormatFloat(float64(buydb.PricePerBoardlot)/float64(types.Coin), 'f', 8, 64),
		buydb.TotalBoardlot,
		buydb.BoughtBoardlot,
		buydb.Buyid,
		types.SellOrderStatus[buydb.Status],
	}
	if types.TyLogTradeBuyLimit == tradeType {
		receiptTrade := &types.ReceiptTradeBuyLimit{base}
		log.Log = types.Encode(receiptTrade)

	} else if types.TyLogTradeBuyRevoke == tradeType {
		receiptTrade := &types.ReceiptTradeBuyRevoke{base}
		log.Log = types.Encode(receiptTrade)
	}

	return log
}

func getBuyOrderFromID(buyid []byte, db dbm.KV) (*types.BuyLimitOrder, error) {
	value, err := db.Get(buyid)
	if err != nil {
		tradelog.Error("getBuyOrderFromID", "Failed to get value frim db wiht buyid", string(buyid))
		return nil, err
	}

	var sellOrder types.BuyLimitOrder
	if err = types.Decode(value, &sellOrder); err != nil {
		tradelog.Error("getBuyOrderFromID", "Failed to decode buy order", string(buyid))
		return nil, err
	}
	return &sellOrder, nil
}

func (buydb *buyDB) getSellLogs(sellerAddr string, buyid string, boardlotCnt int64, buyOrder *types.BuyLimitOrder, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogTradeSellMarket
	receiptSellMartet := &types.ReceiptTradeSellMarket{
		sellerAddr,
		buyid,
		buydb.TokenSymbol,
		boardlotCnt,
		strconv.FormatFloat(float64(buyOrder.AmountPerBoardlot)/float64(types.TokenPrecision), 'f', 4, 64),
		strconv.FormatFloat(float64(buyOrder.PricePerBoardlot)/float64(types.Coin), 'f', 8, 64),
		txhash,
	}
	log.Log = types.Encode(receiptSellMartet)

	return log
}

type tradeAction struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       string
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

func newTradeAction(t *trade, tx *types.Transaction) *tradeAction {
	hash := common.Bytes2Hex(tx.Hash())
	fromaddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &tradeAction{t.GetCoinsAccount(), t.GetStateDB(), hash, fromaddr,
		t.GetBlockTime(), t.GetHeight(), t.GetAddr()}
}

func (action *tradeAction) tradeSell(sell *types.TradeForSell) (*types.Receipt, error) {
	if sell.Totalboardlot < 0 || sell.Priceperboardlot < 0 || sell.Minboardlot < 0 || sell.Amountperboardlot < 0 {
		return nil, types.ErrInputPara
	}

	tokenAccDB := account.NewTokenAccount(sell.Tokensymbol, action.db)

	//确认发起此次出售或者众筹的余额是否足够
	totalAmount := sell.GetTotalboardlot() * sell.GetAmountperboardlot()
	receipt, err := tokenAccDB.ExecFrozen(action.fromaddr, action.execaddr, totalAmount)
	if err != nil {
		tradelog.Error("trade sell ", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", totalAmount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	sellOrder := types.SellOrder{
		sell.GetTokensymbol(),
		action.fromaddr,
		sell.GetAmountperboardlot(),
		sell.GetMinboardlot(),
		sell.GetPriceperboardlot(),
		sell.GetTotalboardlot(),
		0,
		sell.GetStarttime(),
		sell.GetStoptime(),
		sell.GetCrowdfund(),
		calcTokenSellID(action.txhash),
		types.TracdOrderStatusOnSale,
		action.height,
	}

	tokendb := newSellDB(sellOrder)
	sellOrderKV := tokendb.save(action.db)
	logs = append(logs, receipt.Logs...)
	logs = append(logs, tokendb.getSellLogs(types.TyLogTradeSell))
	kv = append(kv, receipt.KV...)
	kv = append(kv, sellOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *tradeAction) tradeBuy(buyOrder *types.TradeForBuy) (*types.Receipt, error) {
	if buyOrder.Boardlotcnt < 0 {
		return nil, types.ErrInputPara
	}

	sellidByte := []byte(buyOrder.Sellid)
	sellOrder, err := getSellOrderFromID(sellidByte, action.db)
	if err != nil {
		return nil, types.ErrTSellOrderNotExist
	}

	if sellOrder.Status == types.TracdOrderStatusNotStart && sellOrder.Starttime > action.blocktime {
		return nil, types.ErrTSellOrderNotStart
	} else if sellOrder.Status == types.TracdOrderStatusSoldOut {
		return nil, types.ErrTSellOrderSoldout
	} else if sellOrder.Status == types.TracdOrderStatusOnSale && sellOrder.Totalboardlot-sellOrder.Soldboardlot < buyOrder.Boardlotcnt {
		return nil, types.ErrTSellOrderNotEnough
	} else if sellOrder.Status == types.TracdOrderStatusRevoked {
		return nil, types.ErrTSellOrderRevoked
	} else if sellOrder.Status == types.TracdOrderStatusExpired {
		return nil, types.ErrTSellOrderExpired
	}

	//首先购买费用的划转
	receiptFromAcc, err := action.coinsAccount.ExecTransfer(action.fromaddr, sellOrder.Address, action.execaddr, buyOrder.Boardlotcnt*sellOrder.Priceperboardlot)
	if err != nil {
		tradelog.Error("account.Transfer ", "addrFrom", action.fromaddr, "addrTo", sellOrder.Address,
			"amount", buyOrder.Boardlotcnt*sellOrder.Priceperboardlot)
		return nil, err
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	//TODO: 创建一个LRU用来保存token对应的子合约账户的地址
	tokenAccDB := account.NewTokenAccount(sellOrder.Tokensymbol, action.db)
	receiptFromExecAcc, err := tokenAccDB.ExecTransferFrozen(sellOrder.Address, action.fromaddr, action.execaddr, buyOrder.Boardlotcnt*sellOrder.Amountperboardlot)
	if err != nil {
		tradelog.Error("account.ExecTransfer token ", "error info", err, "addrFrom", sellOrder.Address,
			"addrTo", action.fromaddr, "execaddr", action.execaddr,
			"amount", buyOrder.Boardlotcnt*sellOrder.Amountperboardlot)
		//因为未能成功将对应的token进行转账，所以需要将购买方的账户资金进行回退
		action.coinsAccount.ExecTransfer(sellOrder.Address, action.fromaddr, action.execaddr, buyOrder.Boardlotcnt*sellOrder.Priceperboardlot)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	tradelog.Debug("tradeBuy", "Soldboardlot before this buy", sellOrder.Soldboardlot)
	sellOrder.Soldboardlot += buyOrder.Boardlotcnt
	tradelog.Debug("tradeBuy", "Soldboardlot after this buy", sellOrder.Soldboardlot)
	if sellOrder.Soldboardlot == sellOrder.Totalboardlot {
		sellOrder.Status = types.TracdOrderStatusSoldOut
	}
	sellTokendb := newSellDB(*sellOrder)
	sellOrderKV := sellTokendb.save(action.db)

	logs = append(logs, receiptFromAcc.Logs...)
	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, sellTokendb.getSellLogs(types.TyLogTradeSell))
	logs = append(logs, sellTokendb.getBuyLogs(action.fromaddr, buyOrder.Sellid, buyOrder.Boardlotcnt, sellOrder, action.txhash))
	kv = append(kv, receiptFromAcc.KV...)
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *tradeAction) tradeRevokeSell(revoke *types.TradeForRevokeSell) (*types.Receipt, error) {
	sellidByte := []byte(revoke.Sellid)
	sellOrder, err := getSellOrderFromID(sellidByte, action.db)
	if err != nil {
		return nil, types.ErrTSellOrderNotExist
	}

	if sellOrder.Status == types.TracdOrderStatusSoldOut {
		return nil, types.ErrTSellOrderSoldout
	} else if sellOrder.Status == types.TracdOrderStatusRevoked {
		return nil, types.ErrTSellOrderRevoked
	} else if sellOrder.Status == types.TracdOrderStatusExpired {
		return nil, types.ErrTSellOrderExpired
	}

	if action.fromaddr != sellOrder.Address {
		return nil, types.ErrTSellOrderRevoke
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	tokenAccDB := account.NewTokenAccount(sellOrder.Tokensymbol, action.db)
	tradeRest := (sellOrder.Totalboardlot - sellOrder.Soldboardlot) * sellOrder.Amountperboardlot
	receiptFromExecAcc, err := tokenAccDB.ExecActive(sellOrder.Address, action.execaddr, tradeRest)
	if err != nil {
		tradelog.Error("account.ExecActive token ", "addrFrom", sellOrder.Address, "execaddr", action.execaddr, "amount", tradeRest)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	sellOrder.Status = types.TracdOrderStatusRevoked
	tokendb := newSellDB(*sellOrder)
	sellOrderKV := tokendb.save(action.db)

	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, tokendb.getSellLogs(types.TyLogTradeRevoke))
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *tradeAction) tradeBuyLimit(buy *types.TradeForBuyLimit) (*types.Receipt, error) {
	if buy.TotalBoardlot < 0 || buy.PricePerBoardlot < 0 || buy.MinBoardlot < 0 || buy.AmountPerBoardlot < 0 {
		return nil, types.ErrInputPara
	}
	// check token exist
	if token.CheckTokenExist(buy.TokenSymbol, action.db) {
		return nil, types.ErrTokenExist
	}

	// check enough bty
	amount := buy.PricePerBoardlot * buy.TotalBoardlot
	receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, amount)
	if err != nil {
		tradelog.Error("trade tradeBuyLimit ", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", amount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	buyOrder := types.BuyLimitOrder{
		buy.GetTokenSymbol(),
		action.fromaddr,
		buy.GetAmountPerBoardlot(),
		buy.GetMinBoardlot(),
		buy.GetPricePerBoardlot(),
		buy.GetTotalBoardlot(),
		0,
		calcTokenBuyID(action.txhash),
		types.TracdOrderStatusOnBuy,
		action.height,
	}

	tokendb := newBuyDB(buyOrder)
	sellOrderKV := tokendb.save(action.db)
	logs = append(logs, receipt.Logs...)
	logs = append(logs, tokendb.getBuyLogs(types.TyLogTradeBuyLimit))
	kv = append(kv, receipt.KV...)
	kv = append(kv, sellOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *tradeAction) tradeSellMarket(sellOrder *types.TradeForSellMarket) (*types.Receipt, error) {
	if sellOrder.BoardlotCnt < 0 {
		return nil, types.ErrInputPara
	}

	idByte := []byte(sellOrder.Buyid)
	buyOrder, err := getBuyOrderFromID(idByte, action.db)
	if err != nil {
		return nil, types.ErrTBuyOrderNotExist
	}

	if buyOrder.Status == types.TracdOrderStatusBoughtOut {
		return nil, types.ErrTBuyOrderSoldout
	} else if buyOrder.Status == types.TracdOrderStatusRevoked {
		return nil, types.ErrTBuyOrderRevoked
	} else if buyOrder.Status == types.TracdOrderStatusOnBuy && buyOrder.TotalBoardlot-buyOrder.BoughtBoardlot < sellOrder.BoardlotCnt {
		return nil, types.ErrTBuyOrderNotEnough
	}

	//首先购买费用的划转
	amount := sellOrder.BoardlotCnt * buyOrder.PricePerBoardlot
	receiptFromAcc, err := action.coinsAccount.ExecTransfer(buyOrder.Address, action.fromaddr, action.execaddr, amount)
	if err != nil {
		tradelog.Error("account.Transfer ", "addrFrom", buyOrder.Address, "addrTo", action.fromaddr,
			"amount", amount)
		return nil, err
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	//TODO: 创建一个LRU用来保存token对应的子合约账户的地址
	tokenAccDB := account.NewTokenAccount(buyOrder.TokenSymbol, action.db)
	amountToken := sellOrder.BoardlotCnt * buyOrder.AmountPerBoardlot
	receiptFromExecAcc, err := tokenAccDB.ExecTransferFrozen(action.fromaddr, buyOrder.Address, action.execaddr, amountToken)
	if err != nil {
		tradelog.Error("account.ExecTransfer token ", "error info", err, "addrFrom", buyOrder.Address,
			"addrTo", action.fromaddr, "execaddr", action.execaddr,
			"amount", amountToken)
		//因为未能成功将对应的token进行转账，所以需要将购买方的账户资金进行回退
		action.coinsAccount.ExecTransfer(action.fromaddr, buyOrder.Address, action.execaddr, amount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	tradelog.Debug("tradeBuy", "BoughtBoardlot before this buy", buyOrder.BoughtBoardlot)
	buyOrder.BoughtBoardlot += sellOrder.BoardlotCnt
	tradelog.Debug("tradeBuy", "BoughtBoardlot after this buy", buyOrder.BoughtBoardlot)
	if buyOrder.BoughtBoardlot == buyOrder.TotalBoardlot {
		buyOrder.Status = types.TracdOrderStatusBoughtOut
	}
	buyTokendb := newBuyDB(*buyOrder)
	sellOrderKV := buyTokendb.save(action.db)

	logs = append(logs, receiptFromAcc.Logs...)
	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, buyTokendb.getBuyLogs(types.TyLogTradeBuyLimit))
	logs = append(logs, buyTokendb.getSellLogs(action.fromaddr, buyOrder.Buyid, sellOrder.BoardlotCnt, buyOrder, action.txhash))
	kv = append(kv, receiptFromAcc.KV...)
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *tradeAction) tradeRevokeBuyLimit(revoke *types.TradeForRevokeBuy) (*types.Receipt, error) {
	buyidByte := []byte(revoke.Buyid)
	buyOrder, err := getBuyOrderFromID(buyidByte, action.db)
	if err != nil {
		return nil, types.ErrTBuyOrderNotExist
	}

	if buyOrder.Status == types.TracdOrderStatusBoughtOut {
		return nil, types.ErrTBuyOrderSoldout
	} else if buyOrder.Status == types.TracdOrderStatusRevoked {
		return nil, types.ErrTBuyOrderRevoked
	}

	if action.fromaddr != buyOrder.Address {
		return nil, types.ErrTBuyOrderRevoke
	}

	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	tradeRest := (buyOrder.TotalBoardlot - buyOrder.BoughtBoardlot) * buyOrder.AmountPerBoardlot
	receiptFromExecAcc, err := action.coinsAccount.ExecActive(buyOrder.Address, action.execaddr, tradeRest)
	if err != nil {
		tradelog.Error("account.ExecActive bty ", "addrFrom", buyOrder.Address, "execaddr", action.execaddr, "amount", tradeRest)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	buyOrder.Status = types.TracdOrderStatusRevoked
	tokendb := newBuyDB(*buyOrder)
	sellOrderKV := tokendb.save(action.db)

	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, tokendb.getBuyLogs(types.TyLogTradeBuyRevoke))
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}
