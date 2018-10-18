package types

import (
	"bytes"
	"encoding/hex"

	"gitlab.33.cn/chain33/chain33/common"
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

func ConvertWalletTxDetailToJson(in *types.WalletTxDetails, out *WalletTxDetails) error {
	if in == nil || out == nil {
		return types.ErrInvalidParam
	}
	for _, tx := range in.TxDetails {
		var recp ReceiptData
		logs := tx.GetReceipt().GetLogs()
		recp.Ty = tx.GetReceipt().GetTy()
		for _, lg := range logs {
			recp.Logs = append(recp.Logs,
				&ReceiptLog{Ty: lg.Ty, Log: common.ToHex(lg.GetLog())})
		}
		rd, err := DecodeLog(tx.Tx.Execer, &recp)
		if err != nil {
			continue
		}
		tran, err := DecodeTx(tx.GetTx())
		if err != nil {
			continue
		}
		out.TxDetails = append(out.TxDetails, &WalletTxDetail{
			Tx:         tran,
			Receipt:    rd,
			Height:     tx.GetHeight(),
			Index:      tx.GetIndex(),
			BlockTime:  tx.GetBlocktime(),
			Amount:     tx.GetAmount(),
			FromAddr:   tx.GetFromaddr(),
			TxHash:     common.ToHex(tx.GetTxhash()),
			ActionName: tx.GetActionName(),
		})
	}
	return nil
}

func DecodeTx(tx *types.Transaction) (*Transaction, error) {
	if tx == nil {
		return nil, types.ErrEmpty
	}
	execStr := string(tx.Execer)
	var pl interface{}
	plType := types.LoadExecutorType(execStr)
	if plType != nil {
		var err error
		pl, err = plType.DecodePayload(tx)
		if err != nil {
			pl = map[string]interface{}{"unkownpayload": string(tx.Payload)}
		}
	}
	if string(tx.Execer) == "user.write" {
		pl = decodeUserWrite(tx.GetPayload())
	}
	if pl == nil {
		pl = map[string]interface{}{"rawlog": common.ToHex(tx.GetPayload())}
	}
	result := &Transaction{
		Execer:     string(tx.Execer),
		Payload:    pl,
		RawPayload: common.ToHex(tx.GetPayload()),
		Signature: &Signature{
			Ty:        tx.GetSignature().GetTy(),
			Pubkey:    common.ToHex(tx.GetSignature().GetPubkey()),
			Signature: common.ToHex(tx.GetSignature().GetSignature()),
		},
		Fee:        tx.Fee,
		Expire:     tx.Expire,
		Nonce:      tx.Nonce,
		To:         tx.GetRealToAddr(),
		From:       tx.From(),
		GroupCount: tx.GroupCount,
		Header:     common.ToHex(tx.Header),
		Next:       common.ToHex(tx.Next),
	}
	return result, nil
}

func decodeUserWrite(payload []byte) *UserWrite {
	var article UserWrite
	if len(payload) != 0 {
		if payload[0] == '#' {
			data := bytes.SplitN(payload[1:], []byte("#"), 2)
			if len(data) == 2 {
				article.Topic = string(data[0])
				article.Content = string(data[1])
				return &article
			}
		}
	}
	article.Topic = ""
	article.Content = string(payload)
	return &article
}
