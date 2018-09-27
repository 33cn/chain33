package rpc

import (
	"encoding/hex"
	"encoding/json"

	bw "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

var jrpc = &Jrpc{}
var grpc = &Grpc{}

func InitRPC(s pluginmgr.RPCServer) {
	cli := channelClient{}
	cli.Init(s.GetQueueClient())
	jrpc.cli = cli
	grpc.channelClient = cli
	s.JRPC().RegisterName(bw.JRPCName, jrpc)
	bw.RegisterBlackwhiteServer(s.GRPC(), grpc)
}

func Init(s pluginmgr.RPCServer) {
	InitRPC(s)
}

type BlackwhiteCreateTxRPC struct{}

func (t *BlackwhiteCreateTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhiteCreateTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteCreateTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type BlackwhitePlayTxRPC struct {
}

func (t *BlackwhitePlayTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhitePlayTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhitePlayTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type BlackwhiteShowTxRPC struct {
}

func (t *BlackwhiteShowTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhiteShowTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteShowTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}

type BlackwhiteTimeoutDoneTxRPC struct {
}

func (t *BlackwhiteTimeoutDoneTxRPC) Input(message json.RawMessage) ([]byte, error) {
	var req bw.BlackwhiteTimeoutDoneTxReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *BlackwhiteTimeoutDoneTxRPC) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if tx, ok := (*replyData).(*types.Transaction); ok {
			data := types.Encode(tx)
			return hex.EncodeToString(data), nil
		}
	}
	return nil, types.ErrTypeAsset
}
