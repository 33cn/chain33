package executor

import (
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Blackwhite) Query_GetBlackwhiteRoundInfo(in *gt.ReqBlackwhiteRoundInfo) (types.Message, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	return c.GetBlackwhiteRoundInfo(in)
}

func (c *Blackwhite) Query_GetBlackwhiteByStatusAndAddr(in *gt.ReqBlackwhiteRoundList) (types.Message, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	return c.GetBwRoundListInfo(in)
}

func (c *Blackwhite) Query_GetBlackwhiteloopResult(in *gt.ReqLoopResult) (types.Message, error) {
	if in == nil {
		return nil, types.ErrInvalidParam
	}
	return c.GetBwRoundLoopResult(in)
}
