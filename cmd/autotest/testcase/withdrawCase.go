// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testcase

import (
	"errors"
	"strconv"
)

type WithdrawCase struct {
	BaseCase
	Addr   string `toml:"addr"`
	Amount string `toml:"amount"`
}

type WithdrawPack struct {
	BaseCasePack
}

func (testCase *WithdrawCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := WithdrawPack{}
	pack.txHash = txHash
	pack.tCase = testCase
	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (pack *WithdrawPack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 1)
	funcMap["balance"] = pack.checkBalance

	return funcMap
}

func (pack *WithdrawPack) checkBalance(txInfo map[string]interface{}) bool {

	/*fromAddr := txInfo["tx"].(map[string]interface{})["from"].(string)
	toAddr := txInfo["tx"].(map[string]interface{})["to"].(string)*/
	interCase := pack.tCase.(*WithdrawCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	withdrawFrom := txInfo["fromaddr"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})

	logWithdraw := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	logRecv := logArr[3].(map[string]interface{})["log"].(map[string]interface{})
	fee, _ := strconv.ParseFloat(feeStr, 64)
	Amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.fLog.Info("WithdrawBalanceDetails", "TestID", pack.packID,
		"Fee", feeStr, "Amount", Amount, "Addr", interCase.Addr, "ExecAddr", withdrawFrom,
		"WithdrawPrev", logWithdraw["prev"].(map[string]interface{})["balance"].(string),
		"WithdrawCurr", logWithdraw["current"].(map[string]interface{})["balance"].(string),
		"FromPrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"FromCurr", logSend["current"].(map[string]interface{})["balance"].(string),
		"ToPrev", logRecv["prev"].(map[string]interface{})["balance"].(string),
		"ToCurr", logRecv["current"].(map[string]interface{})["balance"].(string))

	return checkBalanceDeltaWithAddr(logFee, interCase.Addr, -fee) &&
		checkBalanceDeltaWithAddr(logWithdraw, interCase.Addr, -Amount) &&
		checkBalanceDeltaWithAddr(logSend, withdrawFrom, -Amount) &&
		checkBalanceDeltaWithAddr(logRecv, interCase.Addr, Amount)
}
