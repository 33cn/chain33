package executor

import "gitlab.33.cn/chain33/chain33/types"

func (h *Hashlock) Query_GetHashlocKById(in []byte) (types.Message, error) {
	differTime := types.Now().UnixNano()/1e9 - h.GetBlockTime()
	clog.Error("Query action")
	return h.GetTxsByHashlockID(in, differTime)
}
