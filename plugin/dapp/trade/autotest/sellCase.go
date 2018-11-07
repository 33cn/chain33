package autotest

import (
	"strconv"

	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
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

func (testCase *SellCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &SellPack{}, packID)
}

func (pack *SellPack) GetCheckHandlerMap() interface{} {

	funcMap := make(CheckHandlerMapDiscard, 2)
	funcMap["frozen"] = pack.checkFrozen
	funcMap["balance"] = pack.checkBalance

	return funcMap
}

func (pack *SellPack) GetDependData() interface{} {

	return pack.orderInfo
}

func (pack *SellPack) checkBalance(txInfo map[string]interface{}) bool {

	/*fromAddr := txInfo["tx"].(map[string]interface{})["from"].(string)
	toAddr := txInfo["tx"].(map[string]interface{})["to"].(string)*/
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	interCase := pack.TCase.(*SellCase)
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	fee, _ := strconv.ParseFloat(feeStr, 64)
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.FLog.Info("SellBalanceDetails", "TestID", pack.PackID,
		"Fee", feeStr, "SellAmount", interCase.Amount,
		"SellerBalancePrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"SellerBalanceCurr", logSend["current"].(map[string]interface{})["balance"].(string))

	//save sell order info
	sellOrderInfo := logArr[2].(map[string]interface{})["log"].(map[string]interface{})["base"].(map[string]interface{})
	pack.orderInfo = &SellOrderInfo{}
	pack.orderInfo.sellID = sellOrderInfo["sellID"].(string)

	return CheckBalanceDeltaWithAddr(logFee, interCase.From, -fee) &&
		CheckBalanceDeltaWithAddr(logSend, interCase.From, -amount)

}

func (pack *SellPack) checkFrozen(txInfo map[string]interface{}) bool {

	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	interCase := pack.TCase.(*SellCase)
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.FLog.Info("SellFrozenDetails", "TestID", pack.PackID,
		"SellAmount", interCase.Amount,
		"SellerFrozenPrev", logSend["prev"].(map[string]interface{})["frozen"].(string),
		"SellerFrozenCurr", logSend["current"].(map[string]interface{})["frozen"].(string))

	return CheckFrozenDeltaWithAddr(logSend, interCase.From, amount)

}
