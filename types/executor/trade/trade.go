package trade


import (
	"gitlab.33.cn/chain33/chain33/types"
	"encoding/json"
	"time"
	log "github.com/inconshreveable/log15"
	"math/rand"
	"gitlab.33.cn/chain33/chain33/common/address"
	rpctype "gitlab.33.cn/chain33/chain33/rpc"
)

const name  = "trade"

var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &tradeType{})

	// init log
	types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})
	types.RegistorLog(types.TyLogTransfer, &CoinsTransferLog{})
	types.RegistorLog(types.TyLogGenesis, &CoinsGenesisLog{})

	// init query rpc
	types.RegistorRpcType("GetAddrReciver", &CoinsGetAddrReciver{})
	types.RegistorRpcType("GetTxsByAddr", &CoinsGetTxsByAddr{})
}

type tradeType struct {
}

func (trade tradeType) ActionName(tx *types.Transaction) string {
	var action types.Trade
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == types.TradeSellLimit && action.GetTokensell() != nil {
		return "selltoken"
	} else if action.Ty == types.TradeBuyMarket && action.GetTokenbuy() != nil {
		return "buytoken"
	} else if action.Ty == types.TradeRevokeSell && action.GetTokenrevokesell() != nil {
		return "revokeselltoken"
	} else if action.Ty == types.CoinsActionTransferToExec && action.GetTokenbuylimit() != nil {
		return "buylimittoken"
	} else if action.Ty == types.CoinsActionTransferToExec && action.GetTokensellmarket() != nil {
		return "sellmarkettoken"
	} else if action.Ty == types.CoinsActionTransferToExec && action.GetTokenrevokebuy() != nil {
		return "revokebuytoken"
	}
	return "unknow"
}

func (trade tradeType) NewTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	if action == "TradeSellLimit" {
		var param rpctype.TradeSellTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("NewTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeSellTx(&param)
	} else if action == "TradeBuyMarket" {
		var param rpctype.TradeBuyTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("NewTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeBuyTx(&param)
	} else if action == "TradeSellRevoke" {
		var param rpctype.TradeRevokeTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("NewTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeRevokeTx(&param)
	} else if action == "TradeBuyLimit" {
		var param rpctype.TradeBuyLimitTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("NewTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeBuyLimitTx(&param)
	} else if action == "TradeSellMarket" {
		var param rpctype.TradeSellMarketTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("NewTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeSellMarketTx(&param)
	} else if action == "TradeRevokeBuy" {
		var param rpctype.TradeRevokeBuyTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("NewTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeRevokeBuyTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

func CreateRawTradeSellTx(parm *rpctype.TradeSellTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForSell{
		TokenSymbol:       parm.TokenSymbol,
		AmountPerBoardlot: parm.AmountPerBoardlot,
		MinBoardlot:       parm.MinBoardlot,
		PricePerBoardlot:  parm.PricePerBoardlot,
		TotalBoardlot:     parm.TotalBoardlot,
		Starttime:         0,
		Stoptime:          0,
		Crowdfund:         false,
	}
	sell := &types.Trade{
		Ty:    types.TradeSellLimit,
		Value: &types.Trade_Tokensell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawTradeBuyTx(parm *rpctype.TradeBuyTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForBuy{SellID: parm.SellID, BoardlotCnt: parm.BoardlotCnt}
	buy := &types.Trade{
		Ty:    types.TradeBuyMarket,
		Value: &types.Trade_Tokenbuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawTradeRevokeTx(parm *rpctype.TradeRevokeTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeSell{SellID: parm.SellID}
	buy := &types.Trade{
		Ty:    types.TradeRevokeSell,
		Value: &types.Trade_Tokenrevokesell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawTradeBuyLimitTx(parm *rpctype.TradeBuyLimitTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForBuyLimit{
		TokenSymbol:       parm.TokenSymbol,
		AmountPerBoardlot: parm.AmountPerBoardlot,
		MinBoardlot:       parm.MinBoardlot,
		PricePerBoardlot:  parm.PricePerBoardlot,
		TotalBoardlot:     parm.TotalBoardlot,
	}
	buyLimit := &types.Trade{
		Ty:    types.TradeBuyLimit,
		Value: &types.Trade_Tokenbuylimit{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(buyLimit),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawTradeSellMarketTx(parm *rpctype.TradeSellMarketTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForSellMarket{BuyID: parm.BuyID, BoardlotCnt: parm.BoardlotCnt}
	sellMarket := &types.Trade{
		Ty:    types.TradeSellMarket,
		Value: &types.Trade_Tokensellmarket{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(sellMarket),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawTradeRevokeBuyTx(parm *rpctype.TradeRevokeBuyTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeBuy{BuyID: parm.BuyID}
	buy := &types.Trade{
		Ty:    types.TradeRevokeBuy,
		Value: &types.Trade_Tokenrevokebuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

// log

// query