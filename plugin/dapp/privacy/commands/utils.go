package commands

import (
	"strconv"

	"gitlab.33.cn/chain33/chain33/common"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func decodeTransaction(tx *rpctypes.Transaction) *TxResult {
	feeResult := strconv.FormatFloat(float64(tx.Fee)/float64(types.Coin), 'f', 4, 64)
	result := &TxResult{
		Execer:     tx.Execer,
		Payload:    tx.Payload,
		RawPayload: tx.RawPayload,
		Signature:  tx.Signature,
		Fee:        feeResult,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.To,
		From:       tx.From,
		GroupCount: tx.GroupCount,
		Header:     tx.Header,
		Next:       tx.Next,
	}

	if tx.Amount != 0 {
		result.Amount = strconv.FormatFloat(float64(tx.Amount)/float64(types.Coin), 'f', 4, 64)
	}
	if tx.From != "" {
		result.From = tx.From
	}
	payload, err := common.FromHex(tx.RawPayload)
	if err != nil {
		return result
	}
	exec := types.LoadExecutorType(tx.Execer)
	if exec != nil {
		tx.Payload, _ = exec.DecodePayload(&types.Transaction{Payload: payload})
	}
	return result
}

func decodeLog(execer []byte, rlog rpctypes.ReceiptDataResult) *ReceiptData {
	rd := &ReceiptData{Ty: rlog.Ty, TyName: rlog.TyName}
	for _, l := range rlog.Logs {
		rl := &ReceiptLog{Ty: l.Ty, TyName: l.TyName, RawLog: l.RawLog}
		logty := types.LoadLog(execer, int64(l.Ty))
		if logty == nil {
			rl.Log = nil
		}
		data, err := common.FromHex(l.RawLog)
		if err != nil {
			rl.Log = nil
		}
		rl.Log, err = logty.Json(data)
		if err != nil {
			rl.Log = nil
		}
		rd.Logs = append(rd.Logs, rl)
	}
	return rd
}
