// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"encoding/json"
	"strconv"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"

	_ "github.com/33cn/chain33/system/dapp/coins/types" //load system plugin
)

// DecodeLog decode log
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
		rTy = "Unknown"
	}
	rd := &ReceiptDataResult{Ty: rlog.Ty, TyName: rTy}
	for _, l := range rlog.Logs {
		var lTy string
		var logIns json.RawMessage
		lLog, err := common.FromHex(l.Log)
		if err != nil {
			return nil, err
		}
		logType := types.LoadLog(execer, int64(l.Ty))
		if logType == nil {
			lTy = "unkownType"
			logIns = nil
		} else {
			logIns, _ = logType.JSON(lLog)
			lTy = logType.Name()
		}
		rd.Logs = append(rd.Logs, &ReceiptLogResult{Ty: l.Ty, TyName: lTy, Log: logIns, RawLog: l.Log})
	}
	return rd, nil
}

// ConvertWalletTxDetailToJSON conver the wallet tx detail to json
func ConvertWalletTxDetailToJSON(in *types.WalletTxDetails, out *WalletTxDetails) error {
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

// DecodeTx docode transaction
func DecodeTx(tx *types.Transaction) (*Transaction, error) {
	if tx == nil {
		return nil, types.ErrEmpty
	}
	execStr := string(tx.Execer)
	var pl types.Message
	plType := types.LoadExecutorType(execStr)
	var err error
	if plType != nil {
		pl, err = plType.DecodePayload(tx)
		if err != nil {
			pl = nil
		}
	}
	if string(tx.Execer) == "user.write" {
		pl = decodeUserWrite(tx.GetPayload())
	}
	var pljson json.RawMessage
	if pl != nil {
		pljson, _ = types.PBToJSONUTF8(pl)
	}
	result := &Transaction{
		Execer:     string(tx.Execer),
		Payload:    pljson,
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
		Hash:       common.ToHex(tx.Hash()),
	}
	if result.Amount != 0 {
		result.AmountFmt = strconv.FormatFloat(float64(result.Amount)/float64(types.Coin), 'f', 4, 64)
	}
	feeResult := strconv.FormatFloat(float64(tx.Fee)/float64(types.Coin), 'f', 4, 64)
	result.FeeFmt = feeResult
	return result, nil
}

func decodeUserWrite(payload []byte) *types.UserWrite {
	var article types.UserWrite
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
