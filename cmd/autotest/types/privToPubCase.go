package types

import (
	"strconv"
)

//pub2priv case
type PrivToPubCase struct {
	BaseCase
	From   string `toml:"from"`
	To     string `toml:"to"`
	Amount string `toml:"amount"`
}

type PrivToPubPack struct {
	BaseCasePack
}

func (testCase *PrivToPubCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &PrivToPubPack{}, packID)
}

func (pack *PrivToPubPack) GetCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	funcMap["balance"] = pack.checkBalance
	funcMap["utxo"] = pack.checkUtxo
	return funcMap
}

func (pack *PrivToPubPack) checkBalance(txInfo map[string]interface{}) bool {

	interCase := pack.TCase.(*PrivToPubCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	from := txInfo["fromaddr"].(string) //privacy contract addr
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logPub := logArr[1].(map[string]interface{})["log"].(map[string]interface{})

	fee, _ := strconv.ParseFloat(feeStr, 64)
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.FLog.Info("Private2PubDetails", "TestID", pack.PackID,
		"Fee", feeStr, "Amount", interCase.Amount,
		"FromAddr", interCase.From, "ToAddr", interCase.To,
		"ToPrev", logPub["prev"].(map[string]interface{})["balance"].(string),
		"ToCurr", logPub["current"].(map[string]interface{})["balance"].(string))

	return CheckBalanceDeltaWithAddr(logFee, from, -fee) &&
		CheckBalanceDeltaWithAddr(logPub, interCase.To, amount)
}

func (pack *PrivToPubPack) checkUtxo(txInfo map[string]interface{}) bool {

	interCase := pack.TCase.(*PrivToPubCase)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	inputLog := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	outputLog := logArr[3].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)
	fee, _ := strconv.ParseFloat(txInfo["tx"].(map[string]interface{})["fee"].(string), 64)

	utxoInput := CalcTxUtxoAmount(inputLog, "keyinput")
	utxoOutput := CalcTxUtxoAmount(outputLog, "keyoutput")
	//get available utxo with addr
	availUtxo, err1 := CalcUtxoAvailAmount(interCase.From, pack.TxHash)
	//get spend utxo with addr
	spendUtxo, err2 := CalcUtxoSpendAmount(interCase.From, pack.TxHash)

	utxoCheck := IsBalanceEqualFloat(availUtxo, utxoOutput) && IsBalanceEqualFloat(spendUtxo, utxoInput)

	pack.FLog.Info("Private2PubUtxoDetail", "TestID", pack.PackID, "Fee", fee,
		"TransferAmount", interCase.Amount, "UtxoInput", utxoInput, "UtxoOutput", utxoOutput,
		"FromAddr", interCase.From, "UtxoAvailable", availUtxo, "UtxoSpend", spendUtxo,
		"CalcAvailErr", err1, "CalcSpendErr", err2)

	return IsBalanceEqualFloat(amount, utxoInput-utxoOutput-fee) && utxoCheck
}
