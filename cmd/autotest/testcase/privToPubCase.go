package testcase

import (
	"errors"
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

func (testCase *PrivToPubCase) doSendCommand(packID string) (PackFunc, error) {

	txHash, bSuccess := sendPrivacyTxCommand(testCase.Command)
	if !bSuccess {
		return nil, errors.New(txHash)
	}
	pack := PrivToPubPack{}
	pack.txHash = txHash
	pack.tCase = testCase
	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

func (pack *PrivToPubPack) getCheckHandlerMap() CheckHandlerMap {

	funcMap := make(map[string]CheckHandlerFunc, 2)
	funcMap["balance"] = pack.checkBalance
	funcMap["utxo"] = pack.checkUtxo
	return funcMap
}

func (pack *PrivToPubPack) checkBalance(txInfo map[string]interface{}) bool {

	interCase := pack.tCase.(*PrivToPubCase)
	feeStr := txInfo["tx"].(map[string]interface{})["fee"].(string)
	from := txInfo["fromaddr"].(string) //privacy contract addr
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	logFee := logArr[0].(map[string]interface{})["log"].(map[string]interface{})
	logPub := logArr[1].(map[string]interface{})["log"].(map[string]interface{})

	fee, _ := strconv.ParseFloat(feeStr, 64)
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)

	pack.fLog.Info("Private2PubDetails", "TestID", pack.packID,
		"Fee", feeStr, "Amount", interCase.Amount,
		"FromAddr", interCase.From, "ToAddr", interCase.To,
		"ToPrev", logPub["prev"].(map[string]interface{})["balance"].(string),
		"ToCurr", logPub["current"].(map[string]interface{})["balance"].(string))

	return checkBalanceDeltaWithAddr(logFee, from, -fee) &&
		checkBalanceDeltaWithAddr(logPub, interCase.To, amount)
}

func (pack *PrivToPubPack) checkUtxo(txInfo map[string]interface{}) bool {

	interCase := pack.tCase.(*PrivToPubCase)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	inputLog := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	outputLog := logArr[3].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)
	fee, _ := strconv.ParseFloat(txInfo["tx"].(map[string]interface{})["fee"].(string), 64)

	utxoInput := calcTxUtxoAmount(inputLog, "keyinput")
	utxoOutput := calcTxUtxoAmount(outputLog, "keyoutput")
	//get available utxo with addr
	availUtxo, err1 := calcUtxoAvailAmount(interCase.From, pack.txHash)
	//get spend utxo with addr
	spendUtxo, err2 := calcUtxoSpendAmount(interCase.From, pack.txHash)

	utxoCheck := isBalanceEqualFloat(availUtxo, utxoOutput) && isBalanceEqualFloat(spendUtxo, utxoInput)

	pack.fLog.Info("Private2PubUtxoDetail", "TestID", pack.packID, "Fee", fee,
		"TransferAmount", interCase.Amount, "UtxoInput", utxoInput, "UtxoOutput", utxoOutput,
		"FromAddr", interCase.From, "UtxoAvailable", availUtxo, "UtxoSpend", spendUtxo,
		"CalcAvailErr", err1, "CalcSpendErr", err2)

	return isBalanceEqualFloat(amount, utxoInput-utxoOutput-fee) && utxoCheck
}
