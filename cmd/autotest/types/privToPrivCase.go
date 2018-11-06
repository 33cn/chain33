package types

import (
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

func (testCase *PrivToPrivCase) SendCommand(packID string) (PackFunc, error) {

	return DefaultSend(testCase, &PrivToPrivPack{}, packID)
}

func (pack *PrivToPrivPack) GetCheckHandlerMap() interface{} {

	funcMap := make(CheckHandlerMapDiscard, 2)
	funcMap["utxo"] = pack.checkUtxo
	return funcMap
}

func (pack *PrivToPrivPack) checkUtxo(txInfo map[string]interface{}) bool {

	interCase := pack.TCase.(*PrivToPrivCase)
	logArr := txInfo["receipt"].(map[string]interface{})["logs"].([]interface{})
	inputLog := logArr[1].(map[string]interface{})["log"].(map[string]interface{})
	outputLog := logArr[2].(map[string]interface{})["log"].(map[string]interface{})
	amount, _ := strconv.ParseFloat(interCase.Amount, 64)
	fee, _ := strconv.ParseFloat(txInfo["tx"].(map[string]interface{})["fee"].(string), 64)

	utxoInput := CalcTxUtxoAmount(inputLog, "keyinput")
	utxoOutput := CalcTxUtxoAmount(outputLog, "keyoutput")

	fromAvail, err1 := CalcUtxoAvailAmount(interCase.From, pack.TxHash)
	fromSpend, err2 := CalcUtxoSpendAmount(interCase.From, pack.TxHash)
	toAvail, err3 := CalcUtxoAvailAmount(interCase.To, pack.TxHash)

	utxoCheck := IsBalanceEqualFloat(fromAvail, utxoInput-amount-fee) &&
		IsBalanceEqualFloat(toAvail, amount) &&
		IsBalanceEqualFloat(fromSpend, utxoInput)

	pack.FLog.Info("Private2PrivateUtxoDetail", "TestID", pack.PackID,
		"FromAddr", interCase.From, "ToAddr", interCase.To, "Fee", fee,
		"TransferAmount", interCase.Amount, "UtxoInput", utxoInput, "UtxoOutput", utxoOutput,
		"FromAvailable", fromAvail, "FromSpend", fromSpend, "ToAvailable", toAvail,
		"CalcFromAvailErr", err1, "CalcFromSpendErr", err2, "CalcToAvailErr", err3)

	return IsBalanceEqualFloat(fee, utxoInput-utxoOutput) && utxoCheck
}
