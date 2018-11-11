// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testcase

import (
	"errors"
	"strconv"
)

type TransferCase struct {
	BaseCase
	From   string `toml:"from"`
	To     string `toml:"to"`
	Amount string `toml:"amount"`
}

type TransferPack struct {
	BaseCasePack
}

func (testCase *TransferCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := TransferPack{}
	pack.txHash = txHash
	pack.tCase = testCase
	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (pack *TransferPack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	funcMap["balance"] = pack.checkBalance

	return funcMap
}

func (pack *TransferPack) checkBalance(txInfo map[string]interface{}) bool {

	/*fromAddr := txInfo["tx"].(map[string]interface{})["from"].(string)
	toAddr := txInfo["tx"].(map[string]interface{})["to"].(string)*/
	interCase := pack.tCase.(*TransferCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	logRecv := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	fee, _ := strconv.ParseFloat(feeStr, 64)
	Amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.fLog.Info("TransferBalanceDetails", "TestID", pack.packID,
		"Fee", feeStr, "Amount", interCase.Amount,
		"FromPrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"FromCurr", logSend["current"].(map[string]interface{})["balance"].(string),
		"ToPrev", logRecv["prev"].(map[string]interface{})["balance"].(string),
		"ToCurr", logRecv["current"].(map[string]interface{})["balance"].(string))

	depositCheck := true
	//transfer to contract, deposit
	if len(logArr) == 4 {
		logDeposit := logArr[3].(map[string]interface{})["log"].(map[string]interface{})
		depositCheck = checkBalanceDeltaWithAddr(logDeposit, interCase.From, Amount)
	}

	return checkBalanceDeltaWithAddr(logFee, interCase.From, -fee) &&
		checkBalanceDeltaWithAddr(logSend, interCase.From, -Amount) &&
		checkBalanceDeltaWithAddr(logRecv, interCase.To, Amount) && depositCheck
}
