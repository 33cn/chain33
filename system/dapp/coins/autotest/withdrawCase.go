// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autotest

import (
	"strconv"

	"github.com/33cn/chain33/cmd/autotest/types"
)

// WithdrawCase defines the withdraw case
type WithdrawCase struct {
	types.BaseCase
	Addr   string `toml:"addr"`
	Amount string `toml:"amount"`
}

// WithdrawPack defines the withdraw pack
type WithdrawPack struct {
	types.BaseCasePack
}

// SendCommand send command of withdrawcase
func (testCase *WithdrawCase) SendCommand(packID string) (types.PackFunc, error) {

	return types.DefaultSend(testCase, &WithdrawPack{}, packID)
}

// GetCheckHandlerMap get check handler for map
func (pack *WithdrawPack) GetCheckHandlerMap() interface{} {

	funcMap := make(types.CheckHandlerMapDiscard, 1)
	funcMap["balance"] = pack.checkBalance

	return funcMap
}

func (pack *WithdrawPack) checkBalance(txInfo map[string]interface{}) bool {

	/*fromAddr := txInfo["tx"].(map[string]interface{})["from"].(string)
	toAddr := txInfo["tx"].(map[string]interface{})["to"].(string)*/
	interCase := pack.TCase.(*WithdrawCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	withdrawFrom := txInfo["fromaddr"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})

	logWithdraw := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	logRecv := logArr[3].(map[string]interface{})["log"].(map[string]interface{})
	fee, err := strconv.ParseFloat(feeStr, 64)
	if err != nil {
		return false
	}
	Amount, err := strconv.ParseFloat(interCase.Amount, 64)
	if err != nil {
		return false
	}

	pack.FLog.Info("WithdrawBalanceDetails", "TestID", pack.PackID,
		"Fee", feeStr, "Amount", Amount, "Addr", interCase.Addr, "ExecAddr", withdrawFrom,
		"WithdrawPrev", logWithdraw["prev"].(map[string]interface{})["balance"].(string),
		"WithdrawCurr", logWithdraw["current"].(map[string]interface{})["balance"].(string),
		"FromPrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"FromCurr", logSend["current"].(map[string]interface{})["balance"].(string),
		"ToPrev", logRecv["prev"].(map[string]interface{})["balance"].(string),
		"ToCurr", logRecv["current"].(map[string]interface{})["balance"].(string))

	return types.CheckBalanceDeltaWithAddr(logFee, interCase.Addr, -fee) &&
		types.CheckBalanceDeltaWithAddr(logWithdraw, interCase.Addr, -Amount) &&
		types.CheckBalanceDeltaWithAddr(logSend, withdrawFrom, -Amount) &&
		types.CheckBalanceDeltaWithAddr(logRecv, interCase.Addr, Amount)
}
