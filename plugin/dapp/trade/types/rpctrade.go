package types

import (
	"encoding/json"
)

// trade order
type RpcReplyTradeOrder struct {
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
	AssetExec         string `protobuf:"bytes,16,opt,name=assetExec" json:"assetExec"`
}

func (reply *ReplyTradeOrder) MarshalJSON() ([]byte, error) {
	r := (*RpcReplyTradeOrder)(reply)
	return json.Marshal(r)
}
