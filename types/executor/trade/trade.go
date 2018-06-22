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
	types.RegistorLog(types.TyLogTradeSellLimit, &TradeSellLimitLog{})
	types.RegistorLog(types.TyLogTradeBuyMarket, &TradeBuyMarketLog{})
	types.RegistorLog(types.TyLogTradeSellRevoke, &TradeSellRevokeLog{})
	types.RegistorLog(types.TyLogTradeBuyLimit, &TradeBuyLimitLog{})
	types.RegistorLog(types.TyLogTradeSellMarket, &TradeSellMarketLog{})
	types.RegistorLog(types.TyLogTradeBuyRevoke, &TradeBuyRevokeLog{})

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
type TradeSellLimitLog struct {
}

func (l TradeSellLimitLog) Name() string {
	return "LogTradeSell"
}

func (l TradeSellLimitLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptTradeSell
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TradeSellMarketLog struct {
}

func (l TradeSellMarketLog) Name() string {
	return "LogTradeSellMarket"
}

func (l TradeSellMarketLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptSellMarket
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TradeBuyMarketLog struct {
}

func (l TradeBuyMarketLog) Name() string {
	return "LogTradeBuyMarket"
}

func (l TradeBuyMarketLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptTradeBuyMarket
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TradeBuyLimitLog struct {
}

func (l TradeBuyLimitLog) Name() string {
	return "LogTradeBuyLimit"
}

func (l TradeBuyLimitLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptTradeBuyLimit
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TradeBuyRevokeLog struct {
}

func (l TradeBuyRevokeLog) Name() string {
	return "LogTradeBuyRevoke"
}

func (l TradeBuyRevokeLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptTradeBuyRevoke
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TradeSellRevokeLog struct {
}

func (l TradeSellRevokeLog) Name() string {
	return "LogTradeSellRevoke"
}

func (l TradeSellRevokeLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptTradeRevoke
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

// query

type CoinsGetAddrReciver struct {
}

func (t *CoinsGetAddrReciver) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetAddrReciver) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type CoinsGetTxsByAddr struct {
}

func (t *CoinsGetTxsByAddr) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetTxsByAddr) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

