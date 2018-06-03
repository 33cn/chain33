package executor

import (
	"gitlab.33.cn/chain33/chain33/types"
	"time"
)

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

func countTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	var kvset types.LocalDBSet

	for i := 0; i < len(b.Block.Txs); i++ {
		if string(b.Block.Txs[i].Execer) == "ticket" {
			if b.Receipts[i].GetTy() != types.ExecOk {
				continue
			}
			for j := 0; j < len(b.Receipts[i].Logs); j++ {
				var ticketStatistic types.TicketStatistic
				item := b.Receipts[i].Logs[j]

				if item.Ty == types.TyLogNewTicket ||
					item.Ty == types.TyLogNewTicket ||
					item.Ty == types.TyLogMinerTicket ||
					item.Ty == types.TyLogCloseTicket {
					//item.Ty == types.TyLogTicketBind {
				} else {
					continue
				}

				var ticketlog types.ReceiptTicket
				err := types.Decode(item.Log, &ticketlog)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}

				ticketBytes, err := ex.localDB.Get(StatisticTicketKey(ticketlog.Addr))
				if err == nil {
					err = types.Decode(ticketBytes, &ticketStatistic)
					if err != nil {
						return &kvset, err
					}
				} else if err != types.ErrNotFound {
					return &kvset, err
				}

				switch item.Ty {
				case types.TyLogNewTicket:
					var ticketInfo types.TicketMinerInfo
					ticketStatistic.CurrentOpenCount += 1

					ticketInfo.TicketId = ticketlog.TicketId
					ticketInfo.Status = ticketlog.Status
					if b.Block.Height == 0 {
						ticketInfo.IsGenesis = true
					} else {
						ticketInfo.IsGenesis = false
					}
					ticketInfo.CreateTime = b.Block.BlockTime
					ticketInfo.MinerAddress = ticketlog.Addr

					err = ex.localDB.Set(StatisticTicketKey(ticketInfo.TicketId), types.Encode(&ticketInfo))
					if err != nil {
						return &kvset, err
					}
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoKey(ticketInfo.TicketId), types.Encode(&ticketInfo)})
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoOrderKey(ticketInfo.MinerAddress, ticketInfo.CreateTime, ticketInfo.TicketId), types.Encode(&ticketInfo)})

				case types.TyLogMinerTicket:
					var ticketInfo types.TicketMinerInfo

					ticketBytes, err := ex.localDB.Get(StatisticTicketInfoKey(ticketlog.TicketId))
					if err == nil {
						err = types.Decode(ticketBytes, &ticketInfo)
						if err != nil {
							return &kvset, err
						}
					} else if err != types.ErrNotFound {
						return &kvset, err
					}

					ticketStatistic.CurrentOpenCount -= 1
					ticketStatistic.TotalMinerCount += 1

					ticketInfo.Status = ticketlog.Status
					ticketInfo.PrevStatus = ticketlog.PrevStatus
					ticketInfo.MinerTime = b.Block.BlockTime
					err = ex.localDB.Set(StatisticTicketKey(ticketInfo.TicketId), types.Encode(&ticketInfo))
					if err != nil {
						return &kvset, err
					}

					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoKey(ticketInfo.TicketId), types.Encode(&ticketInfo)})
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoOrderKey(ticketInfo.MinerAddress, ticketInfo.CreateTime, ticketInfo.TicketId), types.Encode(&ticketInfo)})
				case types.TyLogCloseTicket:
					var ticketInfo types.TicketMinerInfo

					ticketBytes, err := ex.localDB.Get(StatisticTicketInfoKey(ticketlog.TicketId))
					if err == nil {
						err = types.Decode(ticketBytes, &ticketInfo)
						if err != nil {
							return &kvset, err
						}
					} else if err != types.ErrNotFound {
						return &kvset, err
					}

					//未挖到矿，取消
					if ticketlog.PrevStatus == 1 {
						ticketStatistic.CurrentOpenCount -= 1
						ticketStatistic.TotalCancleCount += 1
					}

					ticketInfo.Status = ticketlog.Status
					ticketInfo.PrevStatus = ticketlog.PrevStatus
					ticketInfo.CloseTime = b.Block.BlockTime
					err = ex.localDB.Set(StatisticTicketKey(ticketInfo.TicketId), types.Encode(&ticketInfo))
					if err != nil {
						return &kvset, err
					}
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoKey(ticketInfo.TicketId), types.Encode(&ticketInfo)})
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoOrderKey(ticketInfo.MinerAddress, ticketInfo.CreateTime, ticketInfo.TicketId), types.Encode(&ticketInfo)})
				default:
					elog.Error("countTicket switch err", "item.Ty", item.Ty)
				}

				err = ex.localDB.Set(StatisticTicketKey(ticketlog.Addr), types.Encode(&ticketStatistic))
				if err != nil {
					return &kvset, err
				}
				kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketKey(ticketlog.Addr), types.Encode(&ticketStatistic)})
			}
		}
	}

	return &kvset, nil
}

func delCountTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	var kvset types.LocalDBSet

	for i := 0; i < len(b.Block.Txs); i++ {
		if string(b.Block.Txs[i].Execer) == "ticket" {
			if b.Receipts[i].GetTy() != types.ExecOk {
				continue
			}
			for j := 0; j < len(b.Receipts[i].Logs); j++ {
				var ticketStatistic types.TicketStatistic
				item := b.Receipts[i].Logs[j]

				if item.Ty == types.TyLogNewTicket ||
					item.Ty == types.TyLogNewTicket ||
					item.Ty == types.TyLogMinerTicket ||
					item.Ty == types.TyLogCloseTicket {
					//item.Ty == types.TyLogTicketBind {
				} else {
					continue
				}

				var ticketlog types.ReceiptTicket
				err := types.Decode(item.Log, &ticketlog)
				if err != nil {
					panic(err) //数据错误了，已经被修改了
				}

				ticketBytes, err := ex.localDB.Get(StatisticTicketKey(ticketlog.Addr))
				if err == nil {
					err = types.Decode(ticketBytes, &ticketStatistic)
					if err != nil {
						return &kvset, err
					}
				} else if err != types.ErrNotFound {
					return &kvset, err
				}

				switch item.Ty {
				case types.TyLogNewTicket:
					var ticketInfo types.TicketMinerInfo

					ticketInfo.TicketId = ticketlog.TicketId
					ticketInfo.Status = ticketlog.Status
					if b.Block.Height == 0 {
						ticketInfo.IsGenesis = true
					} else {
						ticketInfo.IsGenesis = false
					}
					ticketInfo.CreateTime = b.Block.BlockTime
					ticketInfo.MinerAddress = ticketlog.Addr

					err = ex.localDB.Set(StatisticTicketKey(ticketInfo.TicketId), types.Encode(&types.TicketMinerInfo{}))
					if err != nil {
						return &kvset, err
					}

					ticketStatistic.CurrentOpenCount -= 1

					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoKey(ticketInfo.TicketId), types.Encode(&types.TicketMinerInfo{})})
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoOrderKey(ticketInfo.MinerAddress, ticketInfo.CreateTime, ticketInfo.TicketId), types.Encode(&types.TicketMinerInfo{})})

				case types.TyLogMinerTicket:
					var ticketInfo types.TicketMinerInfo

					ticketBytes, err := ex.localDB.Get(StatisticTicketInfoKey(ticketlog.TicketId))
					if err == nil {
						err = types.Decode(ticketBytes, &ticketInfo)
						if err != nil {
							return &kvset, err
						}
					} else if err != types.ErrNotFound {
						return &kvset, err
					}

					ticketStatistic.CurrentOpenCount += 1
					ticketStatistic.TotalMinerCount -= 1

					ticketInfo.Status = ticketlog.PrevStatus
					ticketInfo.PrevStatus = 0
					ticketInfo.MinerTime = 0
					err = ex.localDB.Set(StatisticTicketKey(ticketInfo.TicketId), types.Encode(&ticketInfo))
					if err != nil {
						return &kvset, err
					}

					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoKey(ticketInfo.TicketId), types.Encode(&ticketInfo)})
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoOrderKey(ticketInfo.MinerAddress, ticketInfo.CreateTime, ticketInfo.TicketId), types.Encode(&ticketInfo)})
				case types.TyLogCloseTicket:
					var ticketInfo types.TicketMinerInfo

					ticketBytes, err := ex.localDB.Get(StatisticTicketInfoKey(ticketlog.TicketId))
					if err == nil {
						err = types.Decode(ticketBytes, &ticketInfo)
						if err != nil {
							return &kvset, err
						}
					} else if err != types.ErrNotFound {
						return &kvset, err
					}

					//未挖到矿，取消
					if ticketlog.PrevStatus == 1 {
						ticketStatistic.CurrentOpenCount += 1
						ticketStatistic.TotalCancleCount -= 1
					}

					ticketInfo.Status = ticketlog.PrevStatus
					if ticketlog.PrevStatus == 1 {
						ticketInfo.PrevStatus = 0
					} else {
						ticketInfo.PrevStatus = 1
					}
					ticketInfo.CloseTime = 0
					err = ex.localDB.Set(StatisticTicketKey(ticketInfo.TicketId), types.Encode(&ticketInfo))
					if err != nil {
						return &kvset, err
					}
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoKey(ticketInfo.TicketId), types.Encode(&ticketInfo)})
					kvset.KV = append(kvset.KV, &types.KeyValue{StatisticTicketInfoOrderKey(ticketInfo.MinerAddress, ticketInfo.CreateTime, ticketInfo.TicketId), types.Encode(&ticketInfo)})
				default:
					elog.Error("countTicket switch err", "item.Ty", item.Ty)
				}
				err = ex.localDB.Set(StatisticTicketKey(ticketlog.Addr), types.Encode(&ticketStatistic))
				if err != nil {
					return &kvset, err
				}
			}
		}
	}

	return &kvset, nil
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

