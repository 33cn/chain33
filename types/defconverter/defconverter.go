package defconverter

import (
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/types"
)

type DefaultRpcQueryConverter struct {
	ProtoObj types.Message
}

func (converter *DefaultRpcQueryConverter) JsonToProto(message json.RawMessage) ([]byte, error) {
	if converter.ProtoObj == nil {
		return nil, types.ErrInvalidParams
	}
	err := json.Unmarshal(message, &converter.ProtoObj)
	if err != nil {
		return nil, err
	}
	return types.Encode(converter.ProtoObj), nil
}

func (converter *DefaultRpcQueryConverter) ProtoToJson(reply types.Message) (interface{}, error) {
	return reply, nil
}
