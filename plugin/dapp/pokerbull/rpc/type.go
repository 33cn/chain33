package rpc

import (
	"encoding/hex"
	"encoding/json"

	pb "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string
var jrpc = &Jrpc{}
var grpc = &Grpc{}

type PokerBullStartTxRPC struct{}

func (t *PokerBullStartTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBStartTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullStartTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type PokerBullContinueTxRPC struct {
}

func (t *PokerBullContinueTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBContinueTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullContinueTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type PokerBullQuitTxRPC struct {
}

func (t *PokerBullQuitTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBQuitTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullQuitTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type PokerBullQueryTxRPC struct {
}

func (t *PokerBullQueryTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req pb.PBQuitTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PokerBullQueryTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}
