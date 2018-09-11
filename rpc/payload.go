package rpc

import (
	"bytes"
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

func privacyPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "ShowAmountsOfUTXO":
		req = &types.ReqPrivacyToken{}
	case "ShowUTXOs4SpecifiedAmount":
		req = &types.ReqPrivacyToken{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func relayPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetRelayOrderByStatus":
		req = &types.ReqRelayAddrCoins{}
	case "GetSellRelayOrder":
		req = &types.ReqRelayAddrCoins{}
	case "GetBuyRelayOrder":
		req = &types.ReqRelayAddrCoins{}
	case "GetBTCHeaderList":
		req = &types.ReqRelayBtcHeaderHeightList{}
	case "GetBTCHeaderMissList":
		req = &types.ReqRelayBtcHeaderHeightList{}
	case "GetBTCHeaderCurHeight":
		req = &types.ReqRelayQryBTCHeadHeight{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func lotteryPayloadType(funcname string) (proto.Message, error) {
	var req proto.Message
	switch funcname {
	case "GetLotteryNormalInfo":
		req = &types.ReqLotteryInfo{}
	case "GetLotteryCurrentInfo":
		req = &types.ReqLotteryInfo{}
	case "GetLotteryHistoryLuckyNumber":
		req = &types.ReqLotteryInfo{}
	case "GetLotteryRoundLuckyNumber":
		req = &types.ReqLotteryLuckyInfo{}
	case "GetLotteryHistoryBuyInfo":
		req = &types.ReqLotteryBuyHistory{}
	case "GetLotteryBuyRoundInfo":
		req = &types.ReqLotteryBuyInfo{}
	default:
		return nil, types.ErrInputPara
	}
	return req, nil
}

func payloadType(execer, funcname string) (proto.Message, error) {
	switch execer {
	case types.ExecName(types.TokenX): // D
		return tokenPayloadType(funcname)
	case types.ExecName(types.CoinsX): // D
		return coinsPayloadType(funcname)
	case types.ExecName(types.ManageX): // D
		return managePayloadType(funcname)
	case types.ExecName(types.RetrieveX): // D
		return retrievePayloadType(funcname)
	case types.ExecName(types.TicketX): // D
		return ticketPayloadType(funcname)
	case types.ExecName(types.TradeX): // D
		return tradePayloadType(funcname)
	case types.ExecName(types.EvmX):
		return evmPayloadType(funcname)
	case types.ExecName(types.PrivacyX):
		return privacyPayloadType(funcname)
	case types.ExecName(types.RelayX):
		return relayPayloadType(funcname)
	case types.ExecName(types.LotteryX):
		return lotteryPayloadType(funcname)
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

func decodeUserWrite(payload []byte) *userWrite {
	var article userWrite
	if len(payload) != 0 {
		if payload[0] == '#' {
			data := bytes.SplitN(payload[1:], []byte("#"), 2)
			if len(data) == 2 {
				article.Topic = string(data[0])
				article.Content = string(data[1])
				return &article
			}
		}
	}
	article.Topic = ""
	article.Content = string(payload)
	return &article
}
