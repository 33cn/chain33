package convertor

import (
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/types"
)

type QueryConvertor struct {
	ProtoObj types.Message
}

func (converter *QueryConvertor) JsonToProto(message json.RawMessage) ([]byte, error) {
	if converter.ProtoObj == nil {
		return nil, types.ErrInvalidParams
	}
	err := json.Unmarshal(message, &converter.ProtoObj)
	if err != nil {
		return nil, err
	}
	return types.Encode(converter.ProtoObj), nil
}

func (converter *QueryConvertor) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
