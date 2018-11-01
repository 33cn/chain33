package dapp

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/types"
)

func HeightIndexStr(height, index int64) string {
	v := height*types.MaxTxsPerBlock + index
	return fmt.Sprintf("%018d", v)
}
