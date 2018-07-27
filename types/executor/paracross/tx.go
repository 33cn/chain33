package paracross

import (
	"gitlab.33.cn/chain33/chain33/types"
)

type ParacrossCommitTx struct {
	Fee       int64  `json:"fee"`
	Status    types.ParacrossNodeStatus `json:"status"`
}
