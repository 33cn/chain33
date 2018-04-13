package trade

import (
	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

type sellDB struct {
	types.SellOrder
}

func newSellDB(sellorder types.SellOrder) (selldb *sellDB) {
	selldb = &sellDB{sellorder}
	if types.InvalidStartTime != selldb.Starttime {
		selldb.Status = types.NotStart
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
	var base *types.ReceiptTradeBase
	base = &types.ReceiptTradeBase{
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

func (selldb *sellDB) getBuyLogs(buyerAddr string, sellid string, boardlotcnt int64, sellorder *types.SellOrder, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogTradeBuy
	receiptBuy := &types.ReceiptTradeBuy{
		buyerAddr,
		sellid,
		selldb.Tokensymbol,
		boardlotcnt,
		strconv.FormatFloat(float64(sellorder.Amountperboardlot)/float64(types.TokenPrecision), 'f', 4, 64),
		strconv.FormatFloat(float64(sellorder.Priceperboardlot)/float64(types.Coin), 'f', 8, 64),
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

	var sellorder types.SellOrder
	if err = types.Decode(value, &sellorder); err != nil {
		tradelog.Error("getSellOrderFromID", "Failed to decode sell order", string(sellid))
		return nil, err
	}
	return &sellorder, nil
}

func (selldb *sellDB) getKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&selldb.SellOrder)
	key := []byte(selldb.Sellid)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
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
	sellorder := types.SellOrder{
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
		types.OnSale,
		action.height,
	}

	tokendb := newSellDB(sellorder)
	sellOrderKV := tokendb.save(action.db)
	logs = append(logs, receipt.Logs...)
	logs = append(logs, tokendb.getSellLogs(types.TyLogTradeSell))
	kv = append(kv, receipt.KV...)
	kv = append(kv, sellOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *tradeAction) tradeBuy(buyorder *types.TradeForBuy) (*types.Receipt, error) {
	sellidByte := []byte(buyorder.Sellid)
	sellorder, err := getSellOrderFromID(sellidByte, action.db)
	if err != nil {
		return nil, types.ErrTSellOrderNotExist
	}

	if sellorder.Status == types.NotStart && sellorder.Starttime > action.blocktime {
		return nil, types.ErrTSellOrderNotStart
	} else if sellorder.Status == types.SoldOut {
		return nil, types.ErrTSellOrderSoldout
	} else if sellorder.Status == types.OnSale && sellorder.Totalboardlot-sellorder.Soldboardlot < buyorder.Boardlotcnt {
		return nil, types.ErrTSellOrderNotEnough
	} else if sellorder.Status == types.Revoked {
		return nil, types.ErrTSellOrderRevoked
	} else if sellorder.Status == types.Expired {
		return nil, types.ErrTSellOrderExpired
	}

	//首先购买费用的划转
	receiptFromAcc, err := action.coinsAccount.ExecTransfer(action.fromaddr, sellorder.Address, action.execaddr, buyorder.Boardlotcnt*sellorder.Priceperboardlot)
	if err != nil {
		tradelog.Error("account.Transfer ", "addrFrom", action.fromaddr, "addrTo", sellorder.Address,
			"amount", buyorder.Boardlotcnt*sellorder.Priceperboardlot)
		return nil, err
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	//TODO: 创建一个LRU用来保存token对应的子合约账户的地址
	tokenAccDB := account.NewTokenAccount(sellorder.Tokensymbol, action.db)
	receiptFromExecAcc, err := tokenAccDB.ExecTransferFrozen(sellorder.Address, action.fromaddr, action.execaddr, buyorder.Boardlotcnt*sellorder.Amountperboardlot)
	if err != nil {
		tradelog.Error("account.ExecTransfer token ", "error info", err, "addrFrom", sellorder.Address,
			"addrTo", action.fromaddr, "execaddr", action.execaddr,
			"amount", buyorder.Boardlotcnt*sellorder.Amountperboardlot)
		//因为未能成功将对应的token进行转账，所以需要将购买方的账户资金进行回退
		action.coinsAccount.ExecTransfer(sellorder.Address, action.fromaddr, action.execaddr, buyorder.Boardlotcnt*sellorder.Priceperboardlot)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	tradelog.Debug("tradeBuy", "Soldboardlot before this buy", sellorder.Soldboardlot)
	sellorder.Soldboardlot += buyorder.Boardlotcnt
	tradelog.Debug("tradeBuy", "Soldboardlot after this buy", sellorder.Soldboardlot)
	if sellorder.Soldboardlot == sellorder.Totalboardlot {
		sellorder.Status = types.SoldOut
	}
	sellTokendb := newSellDB(*sellorder)
	sellOrderKV := sellTokendb.save(action.db)

	logs = append(logs, receiptFromAcc.Logs...)
	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, sellTokendb.getSellLogs(types.TyLogTradeSell))
	logs = append(logs, sellTokendb.getBuyLogs(action.fromaddr, buyorder.Sellid, buyorder.Boardlotcnt, sellorder, action.txhash))
	kv = append(kv, receiptFromAcc.KV...)
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *tradeAction) tradeRevokeSell(revoke *types.TradeForRevokeSell) (*types.Receipt, error) {
	sellidByte := []byte(revoke.Sellid)
	sellorder, err := getSellOrderFromID(sellidByte, action.db)
	if err != nil {
		return nil, types.ErrTSellOrderNotExist
	}

	if sellorder.Status == types.SoldOut {
		return nil, types.ErrTSellOrderSoldout
	} else if sellorder.Status == types.Revoked {
		return nil, types.ErrTSellOrderRevoked
	} else if sellorder.Status == types.Expired {
		return nil, types.ErrTSellOrderExpired
	}

	if action.fromaddr != sellorder.Address {
		return nil, types.ErrTSellOrderRevoke
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	tokenAccDB := account.NewTokenAccount(sellorder.Tokensymbol, action.db)
	tradeRest := (sellorder.Totalboardlot - sellorder.Soldboardlot) * sellorder.Amountperboardlot
	receiptFromExecAcc, err := tokenAccDB.ExecActive(sellorder.Address, action.execaddr, tradeRest)
	if err != nil {
		tradelog.Error("account.ExecActive token ", "addrFrom", sellorder.Address, "execaddr", action.execaddr, "amount", tradeRest)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	sellorder.Status = types.Revoked
	tokendb := newSellDB(*sellorder)
	sellOrderKV := tokendb.save(action.db)

	logs = append(logs, receiptFromExecAcc.Logs...)
	logs = append(logs, tokendb.getSellLogs(types.TyLogTradeRevoke))
	kv = append(kv, receiptFromExecAcc.KV...)
	kv = append(kv, sellOrderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}
