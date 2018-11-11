package dapp

import (
	"fmt"

	"github.com/33cn/chain33/types"
)

func HeightIndexStr(height, index int64) string {
	v := height*types.MaxTxsPerBlock + index
	return fmt.Sprintf("%018d", v)
}
