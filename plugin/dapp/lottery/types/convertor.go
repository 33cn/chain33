package types

import (
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/types"
)

type LotteryGetInfo struct {
}

func (t *LotteryGetInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqLotteryInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryGetInfo) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type LotteryLuckyRoundInfo struct {
}

func (t *LotteryLuckyRoundInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqLotteryLuckyInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryLuckyRoundInfo) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type LotteryBuyInfo struct {
}

func (t *LotteryBuyInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqLotteryBuyHistory
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryBuyInfo) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type LotteryBuyRoundInfo struct {
}

func (t *LotteryBuyRoundInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqLotteryBuyInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *LotteryBuyRoundInfo) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
