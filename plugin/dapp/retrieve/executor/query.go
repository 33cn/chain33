package executor

import (
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (r *Retrieve) Query_GetRetrieveInfo(in *rt.ReqRetrieveInfo) (types.Message, error) {
	rlog.Debug("Retrieve Query", "backupaddr", in.BackupAddress, "defaddr", in.DefaultAddress)
	info, err := getRetrieveInfo(r.GetLocalDB(), in.BackupAddress, in.DefaultAddress)
	if info == nil {
		return nil, err
	}
	if info.Status == retrievePrepare {
		info.RemainTime = info.DelayPeriod - (r.GetBlockTime() - info.PrepareTime)
		if info.RemainTime < 0 {
			info.RemainTime = 0
		}
	}
	return info, nil
}
