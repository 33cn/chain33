// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autotest

import (
	"strconv"

	"github.com/33cn/chain33/cmd/autotest/types"
)

// TransferCase transfer case
type TransferCase struct {
	types.BaseCase
	From   string `toml:"from"`
	To     string `toml:"to"`
	Amount string `toml:"amount"`
}

// TransferPack transfer pack
type TransferPack struct {
	types.BaseCasePack
}

// SendCommand sed command
func (testCase *TransferCase) SendCommand(packID string) (types.PackFunc, error) {

	return types.DefaultSend(testCase, &TransferPack{}, packID)
}

// GetCheckHandlerMap get check handle for map
func (pack *TransferPack) GetCheckHandlerMap() interface{} {

	funcMap := make(types.CheckHandlerMapDiscard, 2)
	funcMap["balance"] = pack.checkBalance

	return funcMap
}

func (pack *TransferPack) checkBalance(txInfo map[string]interface{}) bool {
	/*fromAddr := txInfo["tx"].(map[string]interface{})["from"].(string)
	toAddr := txInfo["tx"].(map[string]interface{})["to"].(string)*/
	interCase := pack.TCase.(*TransferCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	logRecv := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	fee, err := strconv.ParseFloat(feeStr, 64)
	if err != nil {
		return false
	}
	Amount, err := strconv.ParseFloat(interCase.Amount, 64)
	if err != nil {
		return false
	}

	pack.FLog.Info("TransferBalanceDetails", "TestID", pack.PackID,
		"Fee", feeStr, "Amount", interCase.Amount,
		"FromPrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"FromCurr", logSend["current"].(map[string]interface{})["balance"].(string),
		"ToPrev", logRecv["prev"].(map[string]interface{})["balance"].(string),
		"ToCurr", logRecv["current"].(map[string]interface{})["balance"].(string))

	depositCheck := true
	//transfer to contract, deposit
	if len(logArr) == 4 {
		logDeposit := logArr[3].(map[string]interface{})["log"].(map[string]interface{})
		depositCheck = types.CheckBalanceDeltaWithAddr(logDeposit, interCase.From, Amount)
	}

	return types.CheckBalanceDeltaWithAddr(logFee, interCase.From, -fee) &&
		types.CheckBalanceDeltaWithAddr(logSend, interCase.From, -Amount) &&
		types.CheckBalanceDeltaWithAddr(logRecv, interCase.To, Amount) && depositCheck
}
