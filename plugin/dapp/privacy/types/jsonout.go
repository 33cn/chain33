package types

/*
func buildPrivacyInputResult(l *rpctypes.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	srcInput := &pty.PrivacyInput{}
	dstInput := &PrivacyInput{}
	if proto.Unmarshal(data, srcInput) != nil {
		return dstInput
	}
	for _, srcKeyInput := range srcInput.Keyinput {
		var dstUtxoGlobalIndex []*UTXOGlobalIndex
		for _, srcUTXOGlobalIndex := range srcKeyInput.UtxoGlobalIndex {
			dstUtxoGlobalIndex = append(dstUtxoGlobalIndex, &UTXOGlobalIndex{
				Outindex: srcUTXOGlobalIndex.GetOutindex(),
				Txhash:   common.ToHex(srcUTXOGlobalIndex.GetTxhash()),
			})
		}
		dstKeyInput := &KeyInput{
			Amount:          strconv.FormatFloat(float64(srcKeyInput.GetAmount())/float64(types.Coin), 'f', 4, 64),
			UtxoGlobalIndex: dstUtxoGlobalIndex,
			KeyImage:        common.ToHex(srcKeyInput.GetKeyImage()),
		}
		dstInput.Keyinput = append(dstInput.Keyinput, dstKeyInput)
	}
	return dstInput
}

func buildPrivacyOutputResult(l *rpctypes.ReceiptLogResult) interface{} {
	data, _ := common.FromHex(l.RawLog)
	srcOutput := &pty.ReceiptPrivacyOutput{}
	dstOutput := &ReceiptPrivacyOutput{}
	if proto.Unmarshal(data, srcOutput) != nil {
		return dstOutput
	}
	dstOutput.Token = srcOutput.Token
	for _, srcKeyoutput := range srcOutput.Keyoutput {
		dstKeyoutput := &KeyOutput{
			Amount:        strconv.FormatFloat(float64(srcKeyoutput.GetAmount())/float64(types.Coin), 'f', 4, 64),
			Onetimepubkey: common.ToHex(srcKeyoutput.Onetimepubkey),
		}
		dstOutput.Keyoutput = append(dstOutput.Keyoutput, dstKeyoutput)
	}
	return dstOutput
}

func decodePrivInput(input *pty.PrivacyInput) *PrivacyInput {
	inputResult := &PrivacyInput{}
	for _, value := range input.Keyinput {
		amt := float64(value.Amount) / float64(types.Coin)
		amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
		var ugis []*UTXOGlobalIndex
		for _, u := range value.UtxoGlobalIndex {
			ugis = append(ugis, &UTXOGlobalIndex{Outindex: u.Outindex, Txhash: common.ToHex(u.Txhash)})
		}
		kin := &KeyInput{
			Amount:          amtResult,
			KeyImage:        common.ToHex(value.KeyImage),
			UtxoGlobalIndex: ugis,
		}
		inputResult.Keyinput = append(inputResult.Keyinput, kin)
	}
	return inputResult
}

func decodePrivOutput(output *pty.PrivacyOutput) *PrivacyOutput {
	outputResult := &PrivacyOutput{RpubKeytx: common.ToHex(output.RpubKeytx)}
	for _, value := range output.Keyoutput {
		amt := float64(value.Amount) / float64(types.Coin)
		amtResult := strconv.FormatFloat(amt, 'f', 4, 64)
		kout := &KeyOutput{
			Amount:        amtResult,
			Onetimepubkey: common.ToHex(value.Onetimepubkey),
		}
		outputResult.Keyoutput = append(outputResult.Keyoutput, kout)
	}
	return outputResult
}
*/
