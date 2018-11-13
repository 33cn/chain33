// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autotest

import (
	"strconv"

	. "github.com/33cn/chain33/cmd/autotest/types"
)

type WithdrawCase struct {
	BaseCase
	Addr   string `toml:"addr"`
	Amount string `toml:"amount"`
}

type WithdrawPack struct {
	BaseCasePack
}

func (testCase *WithdrawCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &WithdrawPack{}, packID)
}

func (pack *WithdrawPack) GetCheckHandlerMap() interface{} {

	funcMap := make(CheckHandlerMapDiscard, 1)
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
	fee, _ := strconv.ParseFloat(feeStr, 64)
	Amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.FLog.Info("WithdrawBalanceDetails", "TestID", pack.PackID,
		"Fee", feeStr, "Amount", Amount, "Addr", interCase.Addr, "ExecAddr", withdrawFrom,
		"WithdrawPrev", logWithdraw["prev"].(map[string]interface{})["balance"].(string),
		"WithdrawCurr", logWithdraw["current"].(map[string]interface{})["balance"].(string),
		"FromPrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"FromCurr", logSend["current"].(map[string]interface{})["balance"].(string),
		"ToPrev", logRecv["prev"].(map[string]interface{})["balance"].(string),
		"ToCurr", logRecv["current"].(map[string]interface{})["balance"].(string))

	return CheckBalanceDeltaWithAddr(logFee, interCase.Addr, -fee) &&
		CheckBalanceDeltaWithAddr(logWithdraw, interCase.Addr, -Amount) &&
		CheckBalanceDeltaWithAddr(logSend, withdrawFrom, -Amount) &&
		CheckBalanceDeltaWithAddr(logRecv, interCase.Addr, Amount)
}
