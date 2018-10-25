package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (this *Ticket) Query_TicketInfos(param *pty.TicketInfos) (types.Message, error) {
	return Infos(this.GetStateDB(), param)
}

func (this *Ticket) Query_TicketList(param *pty.TicketList) (types.Message, error) {
	return List(this.GetLocalDB(), this.GetStateDB(), param)
}

func (this *Ticket) Query_MinerAddress(param *types.ReqString) (types.Message, error) {
	value, err := this.GetLocalDB().Get(calcBindReturnKey(param.Data))
	if value == nil || err != nil {
		return nil, types.ErrNotFound
	}
	return &types.ReplyString{string(value)}, nil
}

func (this *Ticket) Query_MinerSourceList(param *types.ReqString) (types.Message, error) {
	key := calcBindMinerKeyPrefix(param.Data)
	values, err := this.GetLocalDB().List(key, nil, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, types.ErrNotFound
	}
	reply := &types.ReplyStrings{}
	for _, value := range values {
		reply.Datas = append(reply.Datas, string(value))
	}
	return reply, nil
}
