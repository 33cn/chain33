package trade

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

var tlog = log.New("module", types.TradeX)

func Init() {
	nameX = types.ExecName(types.TradeX)
	// init executor type
	types.RegistorExecutor(types.TradeX, &tradeType{})

	// init log
	types.RegistorLog(types.TyLogTradeSellLimit, &TradeSellLimitLog{})
	types.RegistorLog(types.TyLogTradeBuyMarket, &TradeBuyMarketLog{})
	types.RegistorLog(types.TyLogTradeSellRevoke, &TradeSellRevokeLog{})
	types.RegistorLog(types.TyLogTradeBuyLimit, &TradeBuyLimitLog{})
	types.RegistorLog(types.TyLogTradeSellMarket, &TradeSellMarketLog{})
	types.RegistorLog(types.TyLogTradeBuyRevoke, &TradeBuyRevokeLog{})

	// init query rpc
	types.RegisterRPCQueryHandle("GetTokenSellOrderByStatus", &TradeQueryTokenSellOrder{})
	types.RegisterRPCQueryHandle("GetOnesSellOrderWithStatus", &TradeQueryOnesSellOrder{})
	types.RegisterRPCQueryHandle("GetOnesSellOrder", &TradeQueryOnesSellOrder{})
	types.RegisterRPCQueryHandle("GetTokenBuyOrderByStatus", &TradeQueryTokenBuyOrder{})
	types.RegisterRPCQueryHandle("GetOnesBuyOrderWithStatus", &TradeQueryOnesBuyOrder{})
	types.RegisterRPCQueryHandle("GetOnesBuyOrder", &TradeQueryOnesBuyOrder{})
	types.RegisterRPCQueryHandle("GetOnesOrderWithStatus", &TradeQueryOnesOrder{})
}

type tradeType struct {
	types.ExecTypeBase
}

func (trade tradeType) ActionName(tx *types.Transaction) string {
	var action types.Trade
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknown-err"
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
	return "unknown"
}

func (t tradeType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action types.Trade
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (t tradeType) Amount(tx *types.Transaction) (int64, error) {
	//TODO: 补充和完善token和trade分支的amount的计算, added by hzj
	var trade types.Trade
	err := types.Decode(tx.GetPayload(), &trade)
	if err != nil {
		return 0, types.ErrDecode
	}

	if types.TradeSellLimit == trade.Ty && trade.GetTokensell() != nil {
		return 0, nil
	} else if types.TradeBuyMarket == trade.Ty && trade.GetTokenbuy() != nil {
		return 0, nil
	} else if types.TradeRevokeSell == trade.Ty && trade.GetTokenrevokesell() != nil {
		return 0, nil
	}
	return 0, nil
}

func (trade tradeType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	if action == "TradeSellLimit" {
		var param TradeSellTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeSellTx(&param)
	} else if action == "TradeBuyMarket" {
		var param TradeBuyTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeBuyTx(&param)
	} else if action == "TradeSellRevoke" {
		var param TradeRevokeTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeRevokeTx(&param)
	} else if action == "TradeBuyLimit" {
		var param TradeBuyLimitTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeBuyLimitTx(&param)
	} else if action == "TradeSellMarket" {
		var param TradeSellMarketTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeSellMarketTx(&param)
	} else if action == "TradeRevokeBuy" {
		var param TradeRevokeBuyTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTradeRevokeBuyTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

func CreateRawTradeSellTx(parm *TradeSellTx) (*types.Transaction, error) {
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
		Execer:  []byte(nameX),
		Payload: types.Encode(sell),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(nameX),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawTradeBuyTx(parm *TradeBuyTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForBuy{SellID: parm.SellID, BoardlotCnt: parm.BoardlotCnt}
	buy := &types.Trade{
		Ty:    types.TradeBuyMarket,
		Value: &types.Trade_Tokenbuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(nameX),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(nameX),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawTradeRevokeTx(parm *TradeRevokeTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeSell{SellID: parm.SellID}
	buy := &types.Trade{
		Ty:    types.TradeRevokeSell,
		Value: &types.Trade_Tokenrevokesell{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(nameX),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(nameX),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawTradeBuyLimitTx(parm *TradeBuyLimitTx) (*types.Transaction, error) {
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
		Execer:  []byte(nameX),
		Payload: types.Encode(buyLimit),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(nameX),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawTradeSellMarketTx(parm *TradeSellMarketTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TradeForSellMarket{BuyID: parm.BuyID, BoardlotCnt: parm.BoardlotCnt}
	sellMarket := &types.Trade{
		Ty:    types.TradeSellMarket,
		Value: &types.Trade_Tokensellmarket{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(nameX),
		Payload: types.Encode(sellMarket),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(nameX),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawTradeRevokeBuyTx(parm *TradeRevokeBuyTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TradeForRevokeBuy{BuyID: parm.BuyID}
	buy := &types.Trade{
		Ty:    types.TradeRevokeBuy,
		Value: &types.Trade_Tokenrevokebuy{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(nameX),
		Payload: types.Encode(buy),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(nameX),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

// log
type TradeSellLimitLog struct {
}

func (l TradeSellLimitLog) Name() string {
	return "LogTradeSell"
}

func (l TradeSellLimitLog) Decode(msg []byte) (interface{}, error) {
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

func (l TradeSellMarketLog) Decode(msg []byte) (interface{}, error) {
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

func (l TradeBuyMarketLog) Decode(msg []byte) (interface{}, error) {
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

func (l TradeBuyLimitLog) Decode(msg []byte) (interface{}, error) {
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

func (l TradeBuyRevokeLog) Decode(msg []byte) (interface{}, error) {
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

func (l TradeSellRevokeLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptTradeRevoke
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

// query
type RpcReplySellOrders struct {
	SellOrders []*types.RpcReplyTradeOrder `json:"sellOrders"`
}

type TradeQueryTokenSellOrder struct {
}

func (t *TradeQueryTokenSellOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqTokenSellOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryTokenSellOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*types.ReplyTradeOrders)
	var rpcReply RpcReplySellOrders
	for _, order := range orders.Orders {
		rpcReply.SellOrders = append(rpcReply.SellOrders, (*types.RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

type TradeQueryOnesSellOrder struct {
}

func (t *TradeQueryOnesSellOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryOnesSellOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*types.ReplyTradeOrders)
	var rpcReply RpcReplySellOrders
	for _, order := range orders.Orders {
		rpcReply.SellOrders = append(rpcReply.SellOrders, (*types.RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

// rpc query trade buy order
type RpcReplyBuyOrders struct {
	BuyOrders []*types.RpcReplyTradeOrder `json:"buyOrders"`
}

type TradeQueryTokenBuyOrder struct {
}

func (t *TradeQueryTokenBuyOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqTokenBuyOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryTokenBuyOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*types.ReplyTradeOrders)
	var rpcReply RpcReplyBuyOrders
	for _, order := range orders.Orders {
		rpcReply.BuyOrders = append(rpcReply.BuyOrders, (*types.RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

type TradeQueryOnesBuyOrder struct {
}

func (t *TradeQueryOnesBuyOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryOnesBuyOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*types.ReplyTradeOrders)
	var rpcReply RpcReplyBuyOrders
	for _, order := range orders.Orders {
		rpcReply.BuyOrders = append(rpcReply.BuyOrders, (*types.RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

// trade order
type RpcReplyTradeOrders struct {
	Orders []*types.RpcReplyTradeOrder `protobuf:"bytes,1,rep,name=orders" json:"orders"`
}

type TradeQueryOnesOrder struct {
}

func (t *TradeQueryOnesOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryOnesOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*types.ReplyTradeOrders)
	return orders, nil
}
