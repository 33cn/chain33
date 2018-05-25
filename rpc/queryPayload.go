package rpc

import (
	"encoding/json"
	"fmt"

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
	case "GetAddrReciver":
		req = &types.ReqAddr{}
	case "GetTxsByAddr":
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
	case "MinerAddress":
		req = &types.ReqString{}
	case "MinerSourceList":
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
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func tradePayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetOnesSellOrder":
		req = &types.ReqAddrTokens{}
	case "GetOnesBuyOrder":
		req = &types.ReqAddrTokens{}
	case "GetOnesSellOrderWithStatus":
		req = &types.ReqAddrTokens{}
	case "GetOnesBuyOrderWithStatus":
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
	case "token":
		return tokenPayloadType(funcname)
	case "coins":
		return coinsPayloadType(funcname)
	case "manage":
		return managePayloadType(funcname)
	case "retrieve":
		return retrievePayloadType(funcname)
	case "ticket":
		return ticketPayloadType(funcname)
	case "trade":
		return tradePayloadType(funcname)
	case "user.evm":
		return evmPayloadType(funcname)
	}
	return nil, types.ErrInputPara
}

func protoPayload(execer, funcname string, payload *json.RawMessage) ([]byte, error) {
	req, err := payloadType(execer, funcname)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(*payload, &req)
	if err != nil {
		return nil, types.ErrInputPara
	}
	fmt.Println("req: ", req)
	fmt.Println("req: ", types.Encode(req))
	return types.Encode(req), nil
}
