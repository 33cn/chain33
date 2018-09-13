package convertor

import (
	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/types"
)

type QueryConvertorEncodeTx struct {
	QueryConvertor
}

func (convertor *QueryConvertorEncodeTx) ProtoToJson(reply *types.Message) (interface{}, error) {
	if tx, ok := (*reply).(*types.Transaction); ok {
		data := types.Encode(tx)
		return hex.EncodeToString(data), nil
	}
	return nil, types.ErrTypeAsset
}
