package rpc

import (
	"encoding/json"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/types"
)

func tokenPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetTokens":
		req = &types.ReqTokens{}
	case "GetTokenInfo":
		req = &types.ReqString{}
	case "GetAddrReceiverforTokens":
		req = &types.ReqAddrTokens{}
	case "GetAccountTokenAssets":
		req = &types.ReqAccountTokenAssets{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func coinsPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetAddrReciver", "GetTxsByAddr":
		req = &types.ReqAddr{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func managePayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetConfigItem":
		req = &types.ReqString{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func retrievePayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetRetrieveInfo":
		req = &types.ReqRetrieveInfo{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func ticketPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "TicketInfos":
		req = &types.TicketInfos{}
	case "TicketList":
		req = &types.TicketList{}
	case "MinerAddress", "MinerSourceList":
		req = &types.ReqString{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func evmPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "CheckAddrExists":
		req = &types.CheckEVMAddrReq{}
	case "EstimateGas":
		req = &types.EstimateEVMGasReq{}
	case "EvmDebug":
		req = &types.EvmDebugReq{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func tradePayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetOnesSellOrder", "GetOnesBuyOrder", "GetOnesSellOrderWithStatus", "GetOnesBuyOrderWithStatus":
		req = &types.ReqAddrTokens{}
	case "GetTokenSellOrderByStatus":
		req = &types.ReqTokenSellOrder{}
	case "GetTokenBuyOrderByStatus":
		req = &types.ReqTokenBuyOrder{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func payloadType(execer, funcname string) (proto.Message, error) {
	switch execer {
	case "token": // D
		return tokenPayloadType(funcname)
	case "coins": // D
		return coinsPayloadType(funcname)
	case "manage": // D
		return managePayloadType(funcname)
	case "retrieve": // D
		return retrievePayloadType(funcname)
	case "ticket": // D
		return ticketPayloadType(funcname)
	case "trade": // D
		return tradePayloadType(funcname)
	case "evm":
		return evmPayloadType(funcname)
	}
	return nil, types.ErrInputPara
}

func protoPayload(execer, funcname string, payload *json.RawMessage) ([]byte, error) {
	if payload == nil {
		return nil, types.ErrInputPara
	}

	req, err := payloadType(execer, funcname)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*payload, &req)
	if err != nil {
		return nil, types.ErrInputPara
	}
	return types.Encode(req), nil
}
