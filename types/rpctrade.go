package types

import (
	"encoding/json"
)

func init() {
	registorRpcType("GetTokenSellOrderByStatus", &TradeQueryTokenSellOrder{})
	registorRpcType("GetOnesSellOrderWithStatus", &TradeQueryOnesSellOrder{})
	registorRpcType("GetOnesSellOrder", &TradeQueryOnesSellOrder{})
	registorRpcType("GetTokenBuyOrderByStatus", &TradeQueryTokenBuyOrder{})
	registorRpcType("GetOnesBuyOrderWithStatus", &TradeQueryOnesBuyOrder{})
	registorRpcType("GetOnesBuyOrder", &TradeQueryOnesBuyOrder{})
	registorRpcType("GetOnesOrderWithStatus", &TradeQueryOnesOrder{})

	//tlog.Info("rpc", "typeUtil", rpcTypeUtilMap)
}

// rpc query trade sell order part
type RpcReplySellOrders struct {
	SellOrders []*rpcReplyTradeOrder `json:"sellOrders"`
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
	orders := (*(reply.(*Message))).(*ReplyTradeOrders)
	var rpcReply RpcReplySellOrders
	for _, order := range orders.Orders {
		rpcReply.SellOrders = append(rpcReply.SellOrders, (*rpcReplyTradeOrder)(order))
	}
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
	orders := (*(reply.(*Message))).(*ReplyTradeOrders)
	var rpcReply RpcReplySellOrders
	for _, order := range orders.Orders {
		rpcReply.SellOrders = append(rpcReply.SellOrders, (*rpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

// rpc query trade buy order
type RpcReplyBuyOrders struct {
	BuyOrders []*rpcReplyTradeOrder `json:"buyOrders"`
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
	orders := (*(reply.(*Message))).(*ReplyTradeOrders)
	var rpcReply RpcReplyBuyOrders
	for _, order := range orders.Orders {
		rpcReply.BuyOrders = append(rpcReply.BuyOrders, (*rpcReplyTradeOrder)(order))
	}
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
	orders := (*(reply.(*Message))).(*ReplyTradeOrders)
	var rpcReply RpcReplyBuyOrders
	for _, order := range orders.Orders {
		rpcReply.BuyOrders = append(rpcReply.BuyOrders, (*rpcReplyTradeOrder)(order))
	}
	return &rpcReply, nil
}

// trade order
type rpcReplyTradeOrder struct {
	TokenSymbol       string `protobuf:"bytes,1,opt,name=tokenSymbol" json:"tokenSymbol"`
	Owner             string `protobuf:"bytes,2,opt,name=owner" json:"owner"`
	AmountPerBoardlot int64  `protobuf:"varint,3,opt,name=amountPerBoardlot" json:"amountPerBoardlot"`
	MinBoardlot       int64  `protobuf:"varint,4,opt,name=minBoardlot" json:"minBoardlot"`
	PricePerBoardlot  int64  `protobuf:"varint,5,opt,name=pricePerBoardlot" json:"pricePerBoardlot"`
	TotalBoardlot     int64  `protobuf:"varint,6,opt,name=totalBoardlot" json:"totalBoardlot"`
	TradedBoardlot    int64  `protobuf:"varint,7,opt,name=tradedBoardlot" json:"tradedBoardlot"`
	BuyID             string `protobuf:"bytes,8,opt,name=buyID" json:"buyID"`
	Status            int32  `protobuf:"varint,9,opt,name=status" json:"status"`
	SellID            string `protobuf:"bytes,10,opt,name=sellID" json:"sellID"`
	TxHash            string `protobuf:"bytes,11,opt,name=txHash" json:"txHash"`
	Height            int64  `protobuf:"varint,12,opt,name=height" json:"height"`
	Key               string `protobuf:"bytes,13,opt,name=key" json:"key"`
	BlockTime         int64  `protobuf:"varint,14,opt,name=blockTime" json:"blockTime"`
	IsSellOrder       bool   `protobuf:"varint,15,opt,name=isSellOrder" json:"isSellOrder"`
}

type RpcReplyTradeOrders struct {
	Orders []*rpcReplyTradeOrder `protobuf:"bytes,1,rep,name=orders" json:"orders"`
}

func (reply *ReplyTradeOrder) MarshalJSON() ([]byte, error) {
	r := (*rpcReplyTradeOrder)(reply)
	return json.Marshal(r)
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
	orders := (*(reply.(*Message))).(*ReplyTradeOrders)
	return orders, nil
}
