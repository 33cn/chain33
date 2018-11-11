// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testcase

import (
	"errors"
	"strconv"
)

//pub2priv case
type PrivToPrivCase struct {
	BaseCase
	From   string `toml:"from"`
	To     string `toml:"to"`
	Amount string `toml:"amount"`
}

type PrivToPrivPack struct {
	BaseCasePack
}

func (testCase *PrivToPrivCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendPrivacyTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := PrivToPrivPack{}
	pack.txHash = txHash
	pack.tCase = testCase
	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (pack *PrivToPrivPack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	funcMap["utxo"] = pack.checkUtxo
	return funcMap
}

func (pack *PrivToPrivPack) checkUtxo(txInfo map[string]interface{}) bool {

	interCase := pack.tCase.(*PrivToPrivCase)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	inputLog := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	outputLog := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)
	fee, _ := strconv.ParseFloat(txInfo["tx"].(map[string]interface{})["fee"].(string), 64)

	utxoInput := calcTxUtxoAmount(inputLog, "keyinput")
	utxoOutput := calcTxUtxoAmount(outputLog, "keyoutput")

	fromAvail, err1 := calcUtxoAvailAmount(interCase.From, pack.txHash)
	fromSpend, err2 := calcUtxoSpendAmount(interCase.From, pack.txHash)
	toAvail, err3 := calcUtxoAvailAmount(interCase.To, pack.txHash)

	utxoCheck := isBalanceEqualFloat(fromAvail, utxoInput-amount-fee) &&
		isBalanceEqualFloat(toAvail, amount) &&
		isBalanceEqualFloat(fromSpend, utxoInput)

	pack.fLog.Info("Private2PrivateUtxoDetail", "TestID", pack.packID,
		"FromAddr", interCase.From, "ToAddr", interCase.To, "Fee", fee,
		"TransferAmount", interCase.Amount, "UtxoInput", utxoInput, "UtxoOutput", utxoOutput,
		"FromAvailable", fromAvail, "FromSpend", fromSpend, "ToAvailable", toAvail,
		"CalcFromAvailErr", err1, "CalcFromSpendErr", err2, "CalcToAvailErr", err3)

	return isBalanceEqualFloat(fee, utxoInput-utxoOutput) && utxoCheck
}
