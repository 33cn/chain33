package types

import (
	"encoding/json"
	"math/rand"
	"time"

	"reflect"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	nameX string
	tlog  = log.New("module", types.TradeX)

	actionName = map[string]int32{
		"Sell":       TradeSellLimit,
		"Buy":        TradeBuyMarket,
		"RevokeSell": TradeRevokeSell,
		"BuyLimit":   TradeBuyLimit,
		"SellMarket": TradeSellMarket,
		"RevokeBuy":  TradeRevokeBuy,
	}

	logInfo = map[int64]*types.LogInfo{
		types.TyLogTradeSellLimit:  {reflect.TypeOf(ReceiptTradeSellLimit{}), "LogTradeSell"},
		types.TyLogTradeBuyMarket:  {reflect.TypeOf(ReceiptTradeBuyMarket{}), "LogTradeBuyMarket"},
		types.TyLogTradeSellRevoke: {reflect.TypeOf(ReceiptTradeSellRevoke{}), "LogTradeSellRevoke"},
		types.TyLogTradeSellMarket: {reflect.TypeOf(ReceiptSellMarket{}), "LogTradeSellMarket"},
		types.TyLogTradeBuyLimit:   {reflect.TypeOf(ReceiptTradeBuyLimit{}), "LogTradeBuyLimit"},
		types.TyLogTradeBuyRevoke:  {reflect.TypeOf(ReceiptTradeBuyRevoke{}), "LogTradeBuyRevoke"},
	}
)

func (t *tradeType) GetTypeMap() map[string]int32 {
	return actionName
}

func (at *tradeType) GetLogMap() map[int64]*types.LogInfo {
	return logInfo
}

func init() {
	nameX = types.ExecName(types.TradeX)
	// init executor type
	types.RegistorExecutor(types.TradeX, NewType())

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

func NewType() *tradeType {
	c := &tradeType{}
	c.SetChild(c)
	return c
}

func (at *tradeType) GetPayload() types.Message {
	return &Trade{}
}

func (trade tradeType) ActionName(tx *types.Transaction) string {
	var action Trade
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknown-err"
	}
	if action.Ty == TradeSellLimit && action.GetSell() != nil {
		return "selltoken"
	} else if action.Ty == TradeBuyMarket && action.GetBuy() != nil {
		return "buytoken"
	} else if action.Ty == TradeRevokeSell && action.GetRevokeSell() != nil {
		return "revokeselltoken"
	} else if action.Ty == TradeBuyLimit && action.GetBuyLimit() != nil {
		return "buylimittoken"
	} else if action.Ty == TradeSellMarket && action.GetSellMarket() != nil {
		return "sellmarkettoken"
	} else if action.Ty == TradeRevokeBuy && action.GetRevokeBuy() != nil {
		return "revokebuytoken"
	}
	return "unknown"
}

func (t tradeType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action Trade
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (t tradeType) Amount(tx *types.Transaction) (int64, error) {
	//TODO: 补充和完善token和trade分支的amount的计算, added by hzj
	var trade Trade
	err := types.Decode(tx.GetPayload(), &trade)
	if err != nil {
		return 0, types.ErrDecode
	}

	if TradeSellLimit == trade.Ty && trade.GetSell() != nil {
		return 0, nil
	} else if TradeBuyMarket == trade.Ty && trade.GetBuy() != nil {
		return 0, nil
	} else if TradeRevokeSell == trade.Ty && trade.GetRevokeSell() != nil {
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
	v := &TradeForSell{
		TokenSymbol:       parm.TokenSymbol,
		AmountPerBoardlot: parm.AmountPerBoardlot,
		MinBoardlot:       parm.MinBoardlot,
		PricePerBoardlot:  parm.PricePerBoardlot,
		TotalBoardlot:     parm.TotalBoardlot,
		Starttime:         0,
		Stoptime:          0,
		Crowdfund:         false,
	}
	sell := &Trade{
		Ty:    TradeSellLimit,
		Value: &Trade_Sell{v},
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
	v := &TradeForBuy{SellID: parm.SellID, BoardlotCnt: parm.BoardlotCnt}
	buy := &Trade{
		Ty:    TradeBuyMarket,
		Value: &Trade_Buy{v},
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

	v := &TradeForRevokeSell{SellID: parm.SellID}
	buy := &Trade{
		Ty:    TradeRevokeSell,
		Value: &Trade_RevokeSell{v},
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
	v := &TradeForBuyLimit{
		TokenSymbol:       parm.TokenSymbol,
		AmountPerBoardlot: parm.AmountPerBoardlot,
		MinBoardlot:       parm.MinBoardlot,
		PricePerBoardlot:  parm.PricePerBoardlot,
		TotalBoardlot:     parm.TotalBoardlot,
	}
	buyLimit := &Trade{
		Ty:    TradeBuyLimit,
		Value: &Trade_BuyLimit{v},
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
	v := &TradeForSellMarket{BuyID: parm.BuyID, BoardlotCnt: parm.BoardlotCnt}
	sellMarket := &Trade{
		Ty:    TradeSellMarket,
		Value: &Trade_SellMarket{v},
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

	v := &TradeForRevokeBuy{BuyID: parm.BuyID}
	buy := &Trade{
		Ty:    TradeRevokeBuy,
		Value: &Trade_RevokeBuy{v},
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
	var logTmp ReceiptTradeSellLimit
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
	var logTmp ReceiptSellMarket
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
	var logTmp ReceiptTradeBuyMarket
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
	var logTmp ReceiptTradeBuyLimit
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
	var logTmp ReceiptTradeBuyRevoke
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
	var logTmp ReceiptTradeSellRevoke
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

// query
type RpcReplySellOrders struct {
	SellOrders []*RpcReplyTradeOrder `json:"sellOrders"`
}

type TradeQueryTokenSellOrder struct {
}

func (t *TradeQueryTokenSellOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqTokenSellOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryTokenSellOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*ReplyTradeOrders)
	var rpcReply RpcReplySellOrders
	for _, order := range orders.Orders {
		rpcReply.SellOrders = append(rpcReply.SellOrders, (*RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

type TradeQueryOnesSellOrder struct {
}

func (t *TradeQueryOnesSellOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryOnesSellOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*ReplyTradeOrders)
	var rpcReply RpcReplySellOrders
	for _, order := range orders.Orders {
		rpcReply.SellOrders = append(rpcReply.SellOrders, (*RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

// rpc query trade buy order
type RpcReplyBuyOrders struct {
	BuyOrders []*RpcReplyTradeOrder `json:"buyOrders"`
}

type TradeQueryTokenBuyOrder struct {
}

func (t *TradeQueryTokenBuyOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqTokenBuyOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryTokenBuyOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*ReplyTradeOrders)
	var rpcReply RpcReplyBuyOrders
	for _, order := range orders.Orders {
		rpcReply.BuyOrders = append(rpcReply.BuyOrders, (*RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

type TradeQueryOnesBuyOrder struct {
}

func (t *TradeQueryOnesBuyOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryOnesBuyOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*ReplyTradeOrders)
	var rpcReply RpcReplyBuyOrders
	for _, order := range orders.Orders {
		rpcReply.BuyOrders = append(rpcReply.BuyOrders, (*RpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

// trade order
type RpcReplyTradeOrders struct {
	Orders []*RpcReplyTradeOrder `protobuf:"bytes,1,rep,name=orders" json:"orders"`
}

type TradeQueryOnesOrder struct {
}

func (t *TradeQueryOnesOrder) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TradeQueryOnesOrder) ProtoToJson(reply *types.Message) (interface{}, error) {
	orders := (*reply).(*ReplyTradeOrders)
	return orders, nil
}
