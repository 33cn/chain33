package types

import (
	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/types"
)

func DecodeLog(execer []byte, rlog *ReceiptData) (*ReceiptDataResult, error) {
	var rTy string
	switch rlog.Ty {
	case 0:
		rTy = "ExecErr"
	case 1:
		rTy = "ExecPack"
	case 2:
		rTy = "ExecOk"
	default:
		rTy = "Unkown"
	}
	rd := &ReceiptDataResult{Ty: rlog.Ty, TyName: rTy}

	for _, l := range rlog.Logs {
		var lTy string
		var logIns interface{}

		lLog, err := hex.DecodeString(l.Log[2:])
		if err != nil {
			return nil, err
		}

		logType := types.LoadLog(execer, int64(l.Ty))
		if logType == nil {
			lTy = "unkownType"
			logIns = nil
		} else {
			logIns, err = logType.Decode(lLog)
			lTy = logType.Name()
		}
		rd.Logs = append(rd.Logs, &ReceiptLogResult{Ty: l.Ty, TyName: lTy, Log: logIns, RawLog: l.Log})
	}
	return rd, nil
}
