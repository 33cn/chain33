package autotest

import (
	"strconv"

	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
)

//pub2priv case
type CreateUtxosCase struct {
	BaseCase
	From   string `toml:"from"`
	To     string `toml:"to"`
	Amount string `toml:"amount"`
}

type CreateUtxosPack struct {
	BaseCasePack
}

func (testCase *CreateUtxosCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &CreateUtxosPack{}, packID)
}

func (pack *CreateUtxosPack) GetCheckHandlerMap() interface{} {

	funcMap := make(CheckHandlerMapDiscard, 2)
	funcMap["balance"] = pack.checkBalance
	funcMap["utxo"] = pack.checkUtxo
	return funcMap
}

func (pack *CreateUtxosPack) checkBalance(txInfo map[string]interface{}) bool {

	interCase := pack.TCase.(*CreateUtxosCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logSend := logArr[1].(map[string]interface{})["log"].(map[string]interface{})

	fee, _ := strconv.ParseFloat(feeStr, 64)
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.FLog.Info("PrivCreateutxosDetail", "TestID", pack.PackID,
		"Fee", feeStr, "Amount", interCase.Amount, "FromAddr", interCase.From,
		"FromPrev", logSend["prev"].(map[string]interface{})["balance"].(string),
		"FromCurr", logSend["current"].(map[string]interface{})["balance"].(string))

	return CheckBalanceDeltaWithAddr(logFee, interCase.From, -fee) &&
		CheckBalanceDeltaWithAddr(logSend, interCase.From, -amount)
}

func (pack *CreateUtxosPack) checkUtxo(txInfo map[string]interface{}) bool {

	interCase := pack.TCase.(*CreateUtxosCase)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	outputLog := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	//get available utxo with addr
	availUtxo, err := CalcUtxoAvailAmount(interCase.To, pack.TxHash)
	totalOutput := CalcTxUtxoAmount(outputLog, "keyoutput")
	availCheck := IsBalanceEqualFloat(availUtxo, amount)

	pack.FLog.Info("PrivCreateutxosDetail", "TestID", pack.PackID,
		"TransferAmount", interCase.Amount, "UtxoOutput", totalOutput,
		"ToAddr", interCase.To, "UtxoAvailable", availUtxo, "CalcAvailErr", err)

	return availCheck && IsBalanceEqualFloat(totalOutput, amount)

}
