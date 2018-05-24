package types

import (
	"encoding/json"
)

func init() {
	registorRpcTypeUtil("GetTokenSellOrderByStatus", &TradeQueryTokenSellOrder{})
	registorRpcTypeUtil("GetOnesSellOrderWithStatus",
		&TradeQueryOnesSellOrder{})
	registorRpcTypeUtil("GetOnesSellOrder",
		&TradeQueryOnesSellOrder{})
	registorRpcTypeUtil("GetTokenBuyOrderByStatus",
		&TradeQueryTokenBuyOrder{})
	registorRpcTypeUtil("GetOnesBuyOrderWithStatus",
		&TradeQueryOnesBuyOrder{})
	registorRpcTypeUtil("GetOnesBuyOrder",
		&TradeQueryOnesBuyOrder{})
	registorRpcTypeUtil("GetOnesOrderWithStatus",
		&TradeQueryOnesOrder{})

	tlog.Info("rpc", "typeUtil", RpcTypeUtilMap)
}

// rpc query trade sell order part
type RpcReplySellOrders struct {
	SellOrders []*RpcReplyTradeOrder `json:"sellOrders"`
}

type TradeQueryTokenSellOrder struct {
}

func (t *TradeQueryTokenSellOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqTokenSellOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryTokenSellOrder) Output(reply interface{}) (interface{}, error) {
	str, err := json.Marshal(*reply.(*Message))
	if err != nil {
		return nil, err
	}

	var rpcReplyTmp RpcReplyTradeOrders
	json.Unmarshal(str, &rpcReplyTmp)

	var rpcReply RpcReplySellOrders
	rpcReply.SellOrders = rpcReplyTmp.Orders
	return &rpcReply, nil
}

type TradeQueryOnesSellOrder struct {
}

func (t *TradeQueryOnesSellOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryOnesSellOrder) Output(reply interface{}) (interface{}, error) {
	str, err := json.Marshal(*reply.(*Message))
	if err != nil {
		return nil, err
	}

	var rpcReplyTmp RpcReplyTradeOrders
	json.Unmarshal(str, &rpcReplyTmp)

	var rpcReply RpcReplySellOrders
	rpcReply.SellOrders = rpcReplyTmp.Orders
	return &rpcReply, nil
}

// rpc query trade buy order
type RpcReplyBuyOrders struct {
	BuyOrders []*RpcReplyTradeOrder `json:"buyOrders"`
}

type TradeQueryTokenBuyOrder struct {
}

func (t *TradeQueryTokenBuyOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqTokenBuyOrder
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryTokenBuyOrder) Output(reply interface{}) (interface{}, error) {
	str, err := json.Marshal(*reply.(*Message))
	if err != nil {
		return nil, err
	}
	var rpcReplyTmp RpcReplyTradeOrders
	json.Unmarshal(str, &rpcReplyTmp)

	var rpcReply RpcReplyBuyOrders
	rpcReply.BuyOrders = rpcReplyTmp.Orders
	return &rpcReply, nil
}

type TradeQueryOnesBuyOrder struct {
}

func (t *TradeQueryOnesBuyOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryOnesBuyOrder) Output(reply interface{}) (interface{}, error) {
	str, err := json.Marshal(*reply.(*Message))
	if err != nil {
		return nil, err
	}

	var rpcReplyTmp RpcReplyTradeOrders
	json.Unmarshal(str, &rpcReplyTmp)

	var rpcReply RpcReplyBuyOrders
	rpcReply.BuyOrders = rpcReplyTmp.Orders
	return &rpcReply, nil
}

// trade order
type RpcReplyTradeOrder struct {
	TokenSymbol       string `json:"tokenSymbol"`
	Owner             string `json:"owner"`
	AmountPerBoardlot int64  `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  int64  `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	TradedBoardlot    int64  `json:"tradedBoardlot"`
	BuyID             string `json:"buyID"`
	Status            int32  `json:"status"`
	SellID            string `json:"sellID"`
	TxHash            string `json:"txHash"`
	Height            int64  `json:"height"`
	Key               string `json:"key"`
	BlockTime         int64  `json:"blockTime"`
	IsSellOrder       bool   `json:"isSellOrder"`
}

type RpcReplyTradeOrders struct {
	Orders []*RpcReplyTradeOrder `json:"orders"`
}

type TradeQueryOnesOrder struct {
}

func (t *TradeQueryOnesOrder) Input(message json.RawMessage) ([]byte, error) {
	var req ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return Encode(&req), nil
}

func (t *TradeQueryOnesOrder) Output(reply interface{}) (interface{}, error) {
	str, err := json.Marshal(*reply.(*Message))
	if err != nil {
		return nil, err
	}
	var rpcReply RpcReplyTradeOrders
	json.Unmarshal(str, &rpcReply)
	return &rpcReply, nil
}
