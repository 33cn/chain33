package testcase

import (
	"errors"
	"strconv"
)

type SellCase struct {
	BaseCase
	From   string `toml:"from"`
	Amount string `toml:"amount"`
}

type SellPack struct {
	BaseCasePack
	orderInfo *SellOrderInfo
}

type SellOrderInfo struct {
	sellID string
}

func (testCase *SellCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := SellPack{}
	pack.txHash = txHash
	pack.tCase = testCase

	pack.packID = packID
	pack.checkTimes = 0
	pack.orderInfo = &SellOrderInfo{}
	return &pack, nil
}

func (pack *SellPack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	funcMap["frozen"] = pack.checkFrozen
	funcMap["balance"] = pack.checkBalance

	return funcMap
}

func (pack *SellPack) getDependData() interface{} {

	return pack.orderInfo
}

func (pack *SellPack) checkBalance(txInfo map[string]interface{}) bool {

	/*fromAddr := txInfo["tx"].(map[string]interface{})["from"].(string)
	toAddr := txInfo["tx"].(map[string]interface{})["to"].(string)*/
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	interCase := pack.tCase.(*SellCase)
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	fee, _ := strconv.ParseFloat(feeStr, 64)
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.fLog.Info("SellBalanceDetails", "TestID", pack.packID,
		"Fee", feeStr, "SellAmount", interCase.Amount,
		"SellerBalancePrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"SellerBalanceCurr", logSend["current"].(map[string]interface{})["balance"].(string))

	//save sell order info
	sellOrderInfo := logArr[2].(map[string]interface{})["log"].(map[string]interface{})["base"].(map[string]interface{})
	pack.orderInfo.sellID = sellOrderInfo["sellID"].(string)

	return checkBalanceDeltaWithAddr(logFee, interCase.From, -fee) &&
		checkBalanceDeltaWithAddr(logSend, interCase.From, -amount)

}

func (pack *SellPack) checkFrozen(txInfo map[string]interface{}) bool {

	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	interCase := pack.tCase.(*SellCase)
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.fLog.Info("SellFrozenDetails", "TestID", pack.packID,
		"SellAmount", interCase.Amount,
		"SellerFrozenPrev", logSend["prev"].(map[string]interface{})["frozen"].(string),
		"SellerFrozenCurr", logSend["current"].(map[string]interface{})["frozen"].(string))

	return checkFrozenDeltaWithAddr(logSend, interCase.From, amount)

}
