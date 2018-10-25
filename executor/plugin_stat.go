package executor

import (
	"time"

	"gitlab.33.cn/chain33/chain33/types"
)

	//定制数据统计
	if exec.enableStat {
		kvs, err := exec.stat(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
			return
		}
		kvset.KV = append(kvset.KV, kvs...)
	}

func countInfo(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	var kvset types.LocalDBSet

	//保存挖矿统计数据
	ticketkv, err := countTicket(ex, b)
	if err != nil {
		return &kvset, err
	}
	kvset.KV = append(kvset.KV, ticketkv.KV...)

	return &kvset, nil
}

func delCountInfo(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	var kvset types.LocalDBSet

	//删除挖矿统计数据
	ticketkv, err := delCountTicket(ex, b)
	if err != nil {
		return &kvset, err
	}
	kvset.KV = append(kvset.KV, ticketkv.KV...)

	return &kvset, nil
}

//这两个功能需要重构到 ticket 里面去。
//有些功能需要开启选项，才会启用功能。并且功能必须从0开始
func countTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}

func delCountTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}

func StatisticFlag() []byte {
	return []byte("Statistics:Flag")
}

func StatisticTicketInfoKey(ticketId string) []byte {
	return []byte("Statistics:TicketInfo:TicketId:" + ticketId)
}

func StatisticTicketInfoOrderKey(minerAddr string, createTime int64, ticketId string) []byte {
	return []byte("Statistics:TicketInfoOrder:Addr:" + minerAddr + ":CreateTime:" + time.Unix(createTime, 0).Format("20060102150405") + ":TicketId:" + ticketId)
}

func StatisticTicketKey(minerAddr string) []byte {
	return []byte("Statistics:TicketStat:Addr:" + minerAddr)
}


	//定制数据统计
	if exec.enableStat {
		kvs, err := delCountInfo(execute, datas)
		if err != nil {
			msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
			return
		}
		kvset.KV = append(kvset.KV, kvs.KV...)
	}