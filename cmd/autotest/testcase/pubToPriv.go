package testcase

import (
	"errors"
	"strconv"
)

//pub2priv case
type PubToPrivCase struct {
	BaseCase
	From   string `toml:"from"`
	To     string `toml:"to"`
	Amount string `toml:"amount"`
}

type PubToPrivPack struct {
	BaseCasePack
}

func (testCase *PubToPrivCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendPrivacyTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := PubToPrivPack{}
	pack.txHash = txHash
	pack.tCase = testCase
	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (pack *PubToPrivPack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	funcMap["balance"] = pack.checkBalance
	funcMap["utxo"] = pack.checkUtxo
	return funcMap
}

func (pack *PubToPrivPack) checkBalance(txInfo map[string]interface{}) bool {

	interCase := pack.tCase.(*PubToPrivCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})

	fee, _ := strconv.ParseFloat(feeStr, 64)
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.fLog.Info("Pub2PrivateDetails", "TestID", pack.packID,
		"Fee", feeStr, "Amount", interCase.Amount, "FromAddr", interCase.From,
		"FromPrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"FromCurr", logSend["current"].(map[string]interface{})["balance"].(string))

	return checkBalanceDeltaWithAddr(logFee, interCase.From, -fee) &&
		checkBalanceDeltaWithAddr(logSend, interCase.From, -amount)
}

func (pack *PubToPrivPack) checkUtxo(txInfo map[string]interface{}) bool {

	interCase := pack.tCase.(*PubToPrivCase)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	outputLog := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	//get available utxo with addr
	availUtxo, err := calcUtxoAvailAmount(interCase.To, pack.txHash)
	totalOutput := calcTxUtxoAmount(outputLog, "keyoutput")
	availCheck := isBalanceEqualFloat(availUtxo, amount)

	pack.fLog.Info("Pub2PrivateUtxoDetail", "TestID", pack.packID,
		"TransferAmount", interCase.Amount, "UtxoOutput", totalOutput,
		"ToAddr", interCase.To, "UtxoAvailable", availUtxo, "CalcAvailErr", err)

	return availCheck && isBalanceEqualFloat(totalOutput, amount)

}
