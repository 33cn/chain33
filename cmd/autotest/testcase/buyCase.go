// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testcase

import (
	"errors"
	"fmt"
	"strconv"
)

type BuyCase struct {
	BaseCase
	From        string `toml:"from"`
	To          string `toml:"to"`
	TokenAmount string `toml:"tokenAmount"`
	BtyAmount   string `toml:"btyAmount"`
}

type BuyPack struct {
	BaseCasePack
}

type DependBuyCase struct {
	BuyCase
}

func (testCase *BuyCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := BuyPack{}
	pack.txHash = txHash
	pack.tCase = testCase

	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (testCase *DependBuyCase) setDependData(depData interface{}) {

	if depData != nil {
		orderInfo := depData.(*SellOrderInfo)
		testCase.Command = fmt.Sprintf("%s -s %s", testCase.Command, orderInfo.sellID)
	}
}

func (pack *BuyPack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	funcMap["frozen"] = pack.checkFrozen
	funcMap["balance"] = pack.checkBalance

	return funcMap
}

func (pack *BuyPack) checkBalance(txInfo map[string]interface{}) bool {

	/*fromAddr := txInfo["tx"].(map[string]interface{})["from"].(string)
	toAddr := txInfo["tx"].(map[string]interface{})["to"].(string)*/
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	interCase := pack.tCase.(*BuyCase)

	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logBuyBty := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	logSellBty := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	logBuyToken := logArr[4].(map[string]interface{})["log"].(map[string]interface{})

	fee, _ := strconv.ParseFloat(feeStr, 64)
	tokenAmount, _ := strconv.ParseFloat(interCase.TokenAmount, 64)
	btyAmount, _ := strconv.ParseFloat(interCase.BtyAmount, 64)

	pack.fLog.Info("BuyBalanceDetails", "ID", pack.packID,
		"Fee", feeStr, "TokenAmount", interCase.TokenAmount, "BtyAmount", interCase.BtyAmount,
		"SellerBtyPrev", logSellBty["prev"].(map[string]interface{})["balance"].(string),
		"SellerBtyCurr", logSellBty["current"].(map[string]interface{})["balance"].(string),
		"BuyerBtyPrev", logBuyBty["prev"].(map[string]interface{})["balance"].(string),
		"BuyerBtyCurr", logBuyBty["current"].(map[string]interface{})["balance"].(string),
		"BuyerTokenPrev", logBuyToken["prev"].(map[string]interface{})["balance"].(string),
		"BuyerTokenCurr", logBuyToken["current"].(map[string]interface{})["balance"].(string))

	return checkBalanceDeltaWithAddr(logFee, interCase.From, -fee) &&
		checkBalanceDeltaWithAddr(logBuyBty, interCase.From, -btyAmount) &&
		checkBalanceDeltaWithAddr(logSellBty, interCase.To, btyAmount) &&
		checkBalanceDeltaWithAddr(logBuyToken, interCase.From, tokenAmount)

}

func (pack *BuyPack) checkFrozen(txInfo map[string]interface{}) bool {

	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	interCase := pack.tCase.(*BuyCase)
	logSellToken := logArr[3].(map[string]interface{})["log"].(map[string]interface{})
	tokenAmount, _ := strconv.ParseFloat(interCase.TokenAmount, 64)

	pack.fLog.Info("BuyFrozenDetails", "ID", pack.packID,
		"BuyTokenAmount", interCase.TokenAmount,
		"SellerTokenPrev", logSellToken["prev"].(map[string]interface{})["frozen"].(string),
		"SellerTokenCurr", logSellToken["current"].(map[string]interface{})["frozen"].(string))

	return checkFrozenDeltaWithAddr(logSellToken, interCase.To, -tokenAmount)

}
