package executor

import (
	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

type sellDB struct {
	pty.SellOrder
}

func newSellDB(sellOrder pty.SellOrder) (selldb *sellDB) {
	selldb = &sellDB{sellOrder}
	if pty.InvalidStartTime != selldb.Starttime {
		selldb.Status = pty.TradeOrderStatusNotStart
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

func (selldb *sellDB) getSellLogs(tradeType int32, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = tradeType
	base := &pty.ReceiptSellBase{
		selldb.TokenSymbol,
		selldb.Address,
		strconv.FormatFloat(float64(selldb.AmountPerBoardlot)/float64(types.TokenPrecision), 'f', 8, 64),
		selldb.MinBoardlot,
		strconv.FormatFloat(float64(selldb.PricePerBoardlot)/float64(types.Coin), 'f', 8, 64),
		selldb.TotalBoardlot,
		selldb.SoldBoardlot,
		selldb.Starttime,
		selldb.Stoptime,
		selldb.Crowdfund,
		selldb.SellID,
		pty.SellOrderStatus[selldb.Status],
		"",
		txhash,
		selldb.Height,
		selldb.AssetExec,
	}
	if pty.TyLogTradeSellLimit == tradeType {
		receiptTrade := &pty.ReceiptTradeSellLimit{base}
		log.Log = types.Encode(receiptTrade)

	} else if pty.TyLogTradeSellRevoke == tradeType {
		receiptTrade := &pty.ReceiptTradeSellRevoke{base}
		log.Log = types.Encode(receiptTrade)
	}

	return log
}

func (selldb *sellDB) getBuyLogs(buyerAddr string, boardlotcnt int64, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = pty.TyLogTradeBuyMarket
	base := &pty.ReceiptBuyBase{
		selldb.TokenSymbol,
		buyerAddr,
		strconv.FormatFloat(float64(selldb.AmountPerBoardlot)/float64(types.TokenPrecision), 'f', 8, 64),
		selldb.MinBoardlot,
		strconv.FormatFloat(float64(selldb.PricePerBoardlot)/float64(types.Coin), 'f', 8, 64),
		boardlotcnt,
		boardlotcnt,
		"",
		pty.SellOrderStatus[pty.TradeOrderStatusBoughtOut],
		selldb.SellID,
		txhash,
		selldb.Height,
		selldb.AssetExec,
	}

	receipt := &pty.ReceiptTradeBuyMarket{base}
	log.Log = types.Encode(receipt)
	return log
}

func getSellOrderFromID(sellID []byte, db dbm.KV) (*pty.SellOrder, error) {
	value, err := db.Get(sellID)
	if err != nil {
		tradelog.Error("getSellOrderFromID", "Failed to get value from db with sellid", string(sellID))
		return nil, err
	}

	var sellOrder pty.SellOrder
	if err = types.Decode(value, &sellOrder); err != nil {
		tradelog.Error("getSellOrderFromID", "Failed to decode sell order", string(sellID))
		return nil, err
	}
	return &sellOrder, nil
}

func getTx(txHash []byte, db dbm.KV) (*types.TxResult, error) {
	hash, err := common.FromHex(string(txHash))
	if err != nil {
		return nil, err
	}
	value, err := db.Get(hash)
	if err != nil {
		tradelog.Error("getTx", "Failed to get value from db with getTx", string(txHash))
		return nil, err
	}
	var txResult types.TxResult
	err = types.Decode(value, &txResult)

	if err != nil {
		tradelog.Error("getTx", "Failed to decode TxResult", string(txHash))
		return nil, err
	}
	return &txResult, nil
}

func (selldb *sellDB) getKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&selldb.SellOrder)
	key := []byte(selldb.SellID)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
}

type buyDB struct {
	pty.BuyLimitOrder
}

func newBuyDB(sellOrder pty.BuyLimitOrder) (buydb *buyDB) {
	buydb = &buyDB{sellOrder}
	return
}

func (buydb *buyDB) save(db dbm.KV) []*types.KeyValue {
	set := buydb.getKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}

	return set
}

func (buydb *buyDB) getKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&buydb.BuyLimitOrder)
	key := []byte(buydb.BuyID)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
}

func (buydb *buyDB) getBuyLogs(tradeType int32, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = tradeType
	base := &pty.ReceiptBuyBase{
		buydb.TokenSymbol,
		buydb.Address,
		strconv.FormatFloat(float64(buydb.AmountPerBoardlot)/float64(types.TokenPrecision), 'f', 8, 64),
		buydb.MinBoardlot,
		strconv.FormatFloat(float64(buydb.PricePerBoardlot)/float64(types.Coin), 'f', 8, 64),
		buydb.TotalBoardlot,
		buydb.BoughtBoardlot,
		buydb.BuyID,
		pty.SellOrderStatus[buydb.Status],
		"",
		txhash,
		buydb.Height,
		buydb.AssetExec,
	}
	if pty.TyLogTradeBuyLimit == tradeType {
		receiptTrade := &pty.ReceiptTradeBuyLimit{base}
		log.Log = types.Encode(receiptTrade)

	} else if pty.TyLogTradeBuyRevoke == tradeType {
		receiptTrade := &pty.ReceiptTradeBuyRevoke{base}
		log.Log = types.Encode(receiptTrade)
	}

	return log
}

func getBuyOrderFromID(buyID []byte, db dbm.KV) (*pty.BuyLimitOrder, error) {
	value, err := db.Get(buyID)
	if err != nil {
		tradelog.Error("getBuyOrderFromID", "Failed to get value from db with buyID", string(buyID))
		return nil, err
	}

	var buy pty.BuyLimitOrder
	if err = types.Decode(value, &buy); err != nil {
		tradelog.Error("getBuyOrderFromID", "Failed to decode buy order", string(buyID))
		return nil, err
	}
	return &buy, nil
}

func (buydb *buyDB) getSellLogs(sellerAddr string, sellID string, boardlotCnt int64, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = pty.TyLogTradeSellMarket
	base := &pty.ReceiptSellBase{
		buydb.TokenSymbol,
		sellerAddr,
		strconv.FormatFloat(float64(buydb.AmountPerBoardlot)/float64(types.TokenPrecision), 'f', 8, 64),
		buydb.MinBoardlot,
		strconv.FormatFloat(float64(buydb.PricePerBoardlot)/float64(types.Coin), 'f', 8, 64),
		boardlotCnt,
		boardlotCnt,
		0,
		0,
		false,
		"",
		pty.SellOrderStatus[pty.TradeOrderStatusSoldOut],
		buydb.BuyID,
		txhash,
		buydb.Height,
		buydb.AssetExec,
	}
	receiptSellMarket := &pty.ReceiptSellMarket{Base: base}
	log.Log = types.Encode(receiptSellMarket)

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
	fromaddr := tx.From()
	return &tradeAction{t.GetCoinsAccount(), t.GetStateDB(), hash, fromaddr,
		t.GetBlockTime(), t.GetHeight(), dapp.ExecAddress(string(tx.Execer))}
}

func (action *tradeAction) tradeSell(sell *pty.TradeForSell) (*types.Receipt, error) {
	if sell.TotalBoardlot < 0 || sell.PricePerBoardlot < 0 || sell.MinBoardlot < 0 || sell.AmountPerBoardlot < 0 {
		return nil, types.ErrInvalidParam
	}
	if !checkAsset(action.height, sell.AssetExec, sell.TokenSymbol) {
		return nil, types.ErrInvalidParam
	}

	accDB, err := createAccountDB(action.height, action.db, sell.AssetExec, sell.TokenSymbol)
	if err != nil {
		return nil, err
	}
	//确认发起此次出售或者众筹的余额是否足够
	totalAmount := sell.GetTotalBoardlot() * sell.GetAmountPerBoardlot()
	receipt, err := accDB.ExecFrozen(action.fromaddr, action.execaddr, totalAmount)
	if err != nil {
		tradelog.Error("trade sell ", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", totalAmount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	sellOrder := pty.SellOrder{
		sell.GetTokenSymbol(),
		action.fromaddr,
		sell.GetAmountPerBoardlot(),
		sell.GetMinBoardlot(),
		sell.GetPricePerBoardlot(),
		sell.GetTotalBoardlot(),
		0,
		sell.GetStarttime(),
		sell.GetStoptime(),
		sell.GetCrowdfund(),
		calcTokenSellID(action.txhash),
		pty.TradeOrderStatusOnSale,
		action.height,
		sell.AssetExec,
	}

	tokendb := newSellDB(sellOrder)
	sellOrderKV := tokendb.save(action.db)
	logs = append(logs, receipt.Logs...)
	logs = append(logs, tokendb.getSellLogs(pty.TyLogTradeSellLimit, action.txhash))
	kv = append(kv, receipt.KV...)
	kv = append(kv, sellOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *tradeAction) tradeBuy(buyOrder *pty.TradeForBuy) (*types.Receipt, error) {
	if buyOrder.BoardlotCnt < 0 {
		return nil, types.ErrInvalidParam
	}

	sellidByte := []byte(buyOrder.SellID)
	sellOrder, err := getSellOrderFromID(sellidByte, action.db)
	if err != nil {
		return nil, pty.ErrTSellOrderNotExist
	}

	if sellOrder.Status == pty.TradeOrderStatusNotStart && sellOrder.Starttime > action.blocktime {
		return nil, pty.ErrTSellOrderNotStart
	} else if sellOrder.Status == pty.TradeOrderStatusSoldOut {
		return nil, pty.ErrTSellOrderSoldout
	} else if sellOrder.Status == pty.TradeOrderStatusOnSale && sellOrder.TotalBoardlot-sellOrder.SoldBoardlot < buyOrder.BoardlotCnt {
		return nil, pty.ErrTSellOrderNotEnough
	} else if sellOrder.Status == pty.TradeOrderStatusRevoked {
		return nil, pty.ErrTSellOrderRevoked
	} else if sellOrder.Status == pty.TradeOrderStatusExpired {
		return nil, pty.ErrTSellOrderExpired
	} else if sellOrder.Status == pty.TradeOrderStatusOnSale && buyOrder.BoardlotCnt < sellOrder.MinBoardlot {
		return nil, pty.ErrTCntLessThanMinBoardlot
	}

	//首先购买费用的划转
	receiptFromAcc, err := action.coinsAccount.ExecTransfer(action.fromaddr, sellOrder.Address, action.execaddr, buyOrder.BoardlotCnt*sellOrder.PricePerBoardlot)
	if err != nil {
		tradelog.Error("account.Transfer ", "addrFrom", action.fromaddr, "addrTo", sellOrder.Address,
			"amount", buyOrder.BoardlotCnt*sellOrder.PricePerBoardlot)
		return nil, err
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	//TODO: 创建一个LRU用来保存token对应的子合约账户的地址
	accDB, err := createAccountDB(action.height, action.db, sellOrder.AssetExec, sellOrder.TokenSymbol)
	if err != nil {
		return nil, err
	}
	receiptFromExecAcc, err := accDB.ExecTransferFrozen(sellOrder.Address, action.fromaddr, action.execaddr, buyOrder.BoardlotCnt*sellOrder.AmountPerBoardlot)
	if err != nil {
		tradelog.Error("account.ExecTransfer token ", "error info", err, "addrFrom", sellOrder.Address,
			"addrTo", action.fromaddr, "execaddr", action.execaddr,
			"amount", buyOrder.BoardlotCnt*sellOrder.AmountPerBoardlot)
		//因为未能成功将对应的token进行转账，所以需要将购买方的账户资金进行回退
		action.coinsAccount.ExecTransfer(sellOrder.Address, action.fromaddr, action.execaddr, buyOrder.BoardlotCnt*sellOrder.PricePerBoardlot)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	tradelog.Debug("tradeBuy", "Soldboardlot before this buy", sellOrder.SoldBoardlot)
	sellOrder.SoldBoardlot += buyOrder.BoardlotCnt
	tradelog.Debug("tradeBuy", "Soldboardlot after this buy", sellOrder.SoldBoardlot)
	if sellOrder.SoldBoardlot == sellOrder.TotalBoardlot {
		sellOrder.Status = pty.TradeOrderStatusSoldOut
	}
	sellTokendb := newSellDB(*sellOrder)
	sellOrderKV := sellTokendb.save(action.db)

	logs = append(logs, receiptFromAcc.Logs...)
	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, sellTokendb.getSellLogs(pty.TyLogTradeSellLimit, action.txhash))
	logs = append(logs, sellTokendb.getBuyLogs(action.fromaddr, buyOrder.BoardlotCnt, action.txhash))
	kv = append(kv, receiptFromAcc.KV...)
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *tradeAction) tradeRevokeSell(revoke *pty.TradeForRevokeSell) (*types.Receipt, error) {
	sellidByte := []byte(revoke.SellID)
	sellOrder, err := getSellOrderFromID(sellidByte, action.db)
	if err != nil {
		return nil, pty.ErrTSellOrderNotExist
	}

	if sellOrder.Status == pty.TradeOrderStatusSoldOut {
		return nil, pty.ErrTSellOrderSoldout
	} else if sellOrder.Status == pty.TradeOrderStatusRevoked {
		return nil, pty.ErrTSellOrderRevoked
	} else if sellOrder.Status == pty.TradeOrderStatusExpired {
		return nil, pty.ErrTSellOrderExpired
	}

	if action.fromaddr != sellOrder.Address {
		return nil, pty.ErrTSellOrderRevoke
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	accDB, err := createAccountDB(action.height, action.db, sellOrder.AssetExec, sellOrder.TokenSymbol)
	if err != nil {
		return nil, err
	}
	tradeRest := (sellOrder.TotalBoardlot - sellOrder.SoldBoardlot) * sellOrder.AmountPerBoardlot
	receiptFromExecAcc, err := accDB.ExecActive(sellOrder.Address, action.execaddr, tradeRest)
	if err != nil {
		tradelog.Error("account.ExecActive token ", "addrFrom", sellOrder.Address, "execaddr", action.execaddr, "amount", tradeRest)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	sellOrder.Status = pty.TradeOrderStatusRevoked
	tokendb := newSellDB(*sellOrder)
	sellOrderKV := tokendb.save(action.db)

	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, tokendb.getSellLogs(pty.TyLogTradeSellRevoke, action.txhash))
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *tradeAction) tradeBuyLimit(buy *pty.TradeForBuyLimit) (*types.Receipt, error) {
	if buy.TotalBoardlot < 0 || buy.PricePerBoardlot < 0 || buy.MinBoardlot < 0 || buy.AmountPerBoardlot < 0 {
		return nil, types.ErrInvalidParam
	}

	if !checkAsset(action.height, buy.AssetExec, buy.TokenSymbol) {
		return nil, types.ErrInvalidParam
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
	buyOrder := pty.BuyLimitOrder{
		buy.GetTokenSymbol(),
		action.fromaddr,
		buy.GetAmountPerBoardlot(),
		buy.GetMinBoardlot(),
		buy.GetPricePerBoardlot(),
		buy.GetTotalBoardlot(),
		0,
		calcTokenBuyID(action.txhash),
		pty.TradeOrderStatusOnBuy,
		action.height,
		buy.AssetExec,
	}

	tokendb := newBuyDB(buyOrder)
	buyOrderKV := tokendb.save(action.db)
	logs = append(logs, receipt.Logs...)
	logs = append(logs, tokendb.getBuyLogs(pty.TyLogTradeBuyLimit, action.txhash))
	kv = append(kv, receipt.KV...)
	kv = append(kv, buyOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *tradeAction) tradeSellMarket(sellOrder *pty.TradeForSellMarket) (*types.Receipt, error) {
	if sellOrder.BoardlotCnt < 0 {
		return nil, types.ErrInvalidParam
	}

	idByte := []byte(sellOrder.BuyID)
	buyOrder, err := getBuyOrderFromID(idByte, action.db)
	if err != nil {
		return nil, pty.ErrTBuyOrderNotExist
	}

	if buyOrder.Status == pty.TradeOrderStatusBoughtOut {
		return nil, pty.ErrTBuyOrderSoldout
	} else if buyOrder.Status == pty.TradeOrderStatusRevoked {
		return nil, pty.ErrTBuyOrderRevoked
	} else if buyOrder.Status == pty.TradeOrderStatusOnBuy && buyOrder.TotalBoardlot-buyOrder.BoughtBoardlot < sellOrder.BoardlotCnt {
		return nil, pty.ErrTBuyOrderNotEnough
	} else if buyOrder.Status == pty.TradeOrderStatusOnBuy && sellOrder.BoardlotCnt < buyOrder.MinBoardlot {
		return nil, pty.ErrTCntLessThanMinBoardlot
	}

	// 打token
	accDB, err := createAccountDB(action.height, action.db, buyOrder.AssetExec, buyOrder.TokenSymbol)
	if err != nil {
		return nil, err
	}
	amountToken := sellOrder.BoardlotCnt * buyOrder.AmountPerBoardlot
	tradelog.Debug("tradeSellMarket", "step1 cnt", sellOrder.BoardlotCnt, "amountToken", amountToken)
	receiptFromExecAcc, err := accDB.ExecTransfer(action.fromaddr, buyOrder.Address, action.execaddr, amountToken)
	if err != nil {
		tradelog.Error("account.ExecTransfer token ", "error info", err, "addrFrom", buyOrder.Address,
			"addrTo", action.fromaddr, "execaddr", action.execaddr,
			"amountToken", amountToken)
		return nil, err
	}

	//首先购买费用的划转
	amount := sellOrder.BoardlotCnt * buyOrder.PricePerBoardlot
	tradelog.Debug("tradeSellMarket", "step2 cnt", sellOrder.BoardlotCnt, "price", buyOrder.PricePerBoardlot, "amount", amount)
	receiptFromAcc, err := action.coinsAccount.ExecTransferFrozen(buyOrder.Address, action.fromaddr, action.execaddr, amount)
	if err != nil {
		tradelog.Error("account.Transfer ", "addrFrom", buyOrder.Address, "addrTo", action.fromaddr,
			"amount", amount)
		// 因为未能成功将对应的币进行转账，所以需要回退
		accDB.ExecTransfer(buyOrder.Address, action.fromaddr, action.execaddr, amountToken)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	tradelog.Debug("tradeBuy", "BoughtBoardlot before this buy", buyOrder.BoughtBoardlot)
	buyOrder.BoughtBoardlot += sellOrder.BoardlotCnt
	tradelog.Debug("tradeBuy", "BoughtBoardlot after this buy", buyOrder.BoughtBoardlot)
	if buyOrder.BoughtBoardlot == buyOrder.TotalBoardlot {
		buyOrder.Status = pty.TradeOrderStatusBoughtOut
	}
	buyTokendb := newBuyDB(*buyOrder)
	sellOrderKV := buyTokendb.save(action.db)

	logs = append(logs, receiptFromAcc.Logs...)
	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, buyTokendb.getBuyLogs(pty.TyLogTradeBuyLimit, action.txhash))
	logs = append(logs, buyTokendb.getSellLogs(action.fromaddr, action.txhash, sellOrder.BoardlotCnt, action.txhash))
	kv = append(kv, receiptFromAcc.KV...)
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *tradeAction) tradeRevokeBuyLimit(revoke *pty.TradeForRevokeBuy) (*types.Receipt, error) {
	buyIDByte := []byte(revoke.BuyID)
	buyOrder, err := getBuyOrderFromID(buyIDByte, action.db)
	if err != nil {
		return nil, pty.ErrTBuyOrderNotExist
	}

	if buyOrder.Status == pty.TradeOrderStatusBoughtOut {
		return nil, pty.ErrTBuyOrderSoldout
	} else if buyOrder.Status == pty.TradeOrderStatusBuyRevoked {
		return nil, pty.ErrTBuyOrderRevoked
	}

	if action.fromaddr != buyOrder.Address {
		return nil, pty.ErrTBuyOrderRevoke
	}

	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	tradeRest := (buyOrder.TotalBoardlot - buyOrder.BoughtBoardlot) * buyOrder.PricePerBoardlot
	//tradelog.Info("tradeRevokeBuyLimit", "total-b", buyOrder.TotalBoardlot, "price", buyOrder.PricePerBoardlot, "amount", tradeRest)
	receiptFromExecAcc, err := action.coinsAccount.ExecActive(buyOrder.Address, action.execaddr, tradeRest)
	if err != nil {
		tradelog.Error("account.ExecActive bty ", "addrFrom", buyOrder.Address, "execaddr", action.execaddr, "amount", tradeRest)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	buyOrder.Status = pty.TradeOrderStatusBuyRevoked
	tokendb := newBuyDB(*buyOrder)
	sellOrderKV := tokendb.save(action.db)

	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, tokendb.getBuyLogs(pty.TyLogTradeBuyRevoke, action.txhash))
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}
