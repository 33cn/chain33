package executor

import (
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (p *privacy) Exec_Public2Privacy(payload *ty.Public2Privacy, tx *types.Transaction, index int) (*types.Receipt, error) {
	if payload.Tokenname != types.BTY {
		return nil, types.ErrNotSupport
	}
	txhashstr := common.Bytes2Hex(tx.Hash())
	coinsAccount := p.GetCoinsAccount()
	from := tx.From()
	receipt, err := coinsAccount.ExecWithdraw(address.ExecAddress(string(tx.Execer)), from, payload.Amount)
	if err != nil {
		privacylog.Error("PrivacyTrading Exec", "txhash", txhashstr, "ExecWithdraw error ", err)
		return nil, err
	}

	txhash := common.ToHex(tx.Hash())
	output := payload.GetOutput().GetKeyoutput()
	//因为只有包含当前交易的block被执行完成之后，同步到相应的钱包之后，
	//才能将相应的utxo作为input，进行支付，所以此处不需要进行将KV设置到
	//executor中的临时数据库中，只需要将kv返回给blockchain就行
	//即：一个块中产生的UTXO是不能够在同一个高度进行支付的
	for index, keyOutput := range output {
		key := CalcPrivacyOutputKey(payload.Tokenname, keyOutput.Amount, txhash, index)
		value := types.Encode(keyOutput)
		receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
	}

	receiptPrivacyOutput := &ty.ReceiptPrivacyOutput{
		Token:     payload.Tokenname,
		Keyoutput: payload.GetOutput().Keyoutput,
	}
	execlog := &types.ReceiptLog{ty.TyLogPrivacyOutput, types.Encode(receiptPrivacyOutput)}
	receipt.Logs = append(receipt.Logs, execlog)

	//////////////////debug code begin///////////////
	privacylog.Debug("PrivacyTrading Exec", "ActionPublic2Privacy txhash", txhashstr, "receipt is", receipt)
	//////////////////debug code end///////////////

	return receipt, nil
}

func (p *privacy) Exec_Privacy2Privacy(payload *ty.Privacy2Privacy, tx *types.Transaction, index int) (*types.Receipt, error) {
	if payload.Tokenname != types.BTY {
		return nil, types.ErrNotSupport
	}
	txhashstr := common.Bytes2Hex(tx.Hash())
	receipt := &types.Receipt{KV: make([]*types.KeyValue, 0)}
	privacyInput := payload.Input
	for _, keyInput := range privacyInput.Keyinput {
		value := []byte{KeyImageSpentAlready}
		key := calcPrivacyKeyImageKey(payload.Tokenname, keyInput.KeyImage)
		stateDB := p.GetStateDB()
		stateDB.Set(key, value)
		receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
	}

	execlog := &types.ReceiptLog{ty.TyLogPrivacyInput, types.Encode(payload.GetInput())}
	receipt.Logs = append(receipt.Logs, execlog)

	txhash := common.ToHex(tx.Hash())
	output := payload.GetOutput().GetKeyoutput()
	for index, keyOutput := range output {
		key := CalcPrivacyOutputKey(payload.Tokenname, keyOutput.Amount, txhash, index)
		value := types.Encode(keyOutput)
		receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
	}

	receiptPrivacyOutput := &ty.ReceiptPrivacyOutput{
		Token:     payload.Tokenname,
		Keyoutput: payload.GetOutput().Keyoutput,
	}
	execlog = &types.ReceiptLog{ty.TyLogPrivacyOutput, types.Encode(receiptPrivacyOutput)}
	receipt.Logs = append(receipt.Logs, execlog)

	receipt.Ty = types.ExecOk

	//////////////////debug code begin///////////////
	privacylog.Debug("PrivacyTrading Exec", "ActionPrivacy2Privacy txhash", txhashstr, "receipt is", receipt)
	//////////////////debug code end///////////////
	return receipt, nil
}

func (p *privacy) Exec_Privacy2Public(payload *ty.Privacy2Public, tx *types.Transaction, index int) (*types.Receipt, error) {
	if payload.Tokenname != types.BTY {
		return nil, types.ErrNotSupport
	}
	txhashstr := common.Bytes2Hex(tx.Hash())
	coinsAccount := p.GetCoinsAccount()
	receipt, err := coinsAccount.ExecDeposit(tx.To, address.ExecAddress(string(tx.Execer)), payload.Amount)
	if err != nil {
		privacylog.Error("PrivacyTrading Exec", "ActionPrivacy2Public txhash", txhashstr, "ExecDeposit error ", err)
		return nil, err
	}
	privacyInput := payload.Input
	for _, keyInput := range privacyInput.Keyinput {
		value := []byte{KeyImageSpentAlready}
		key := calcPrivacyKeyImageKey(payload.Tokenname, keyInput.KeyImage)
		stateDB := p.GetStateDB()
		stateDB.Set(key, value)
		receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
	}

	execlog := &types.ReceiptLog{ty.TyLogPrivacyInput, types.Encode(payload.GetInput())}
	receipt.Logs = append(receipt.Logs, execlog)

	txhash := common.ToHex(tx.Hash())
	output := payload.GetOutput().GetKeyoutput()
	for index, keyOutput := range output {
		key := CalcPrivacyOutputKey(payload.Tokenname, keyOutput.Amount, txhash, index)
		value := types.Encode(keyOutput)
		receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
	}

	receiptPrivacyOutput := &ty.ReceiptPrivacyOutput{
		Token:     payload.Tokenname,
		Keyoutput: payload.GetOutput().Keyoutput,
	}
	execlog = &types.ReceiptLog{ty.TyLogPrivacyOutput, types.Encode(receiptPrivacyOutput)}
	receipt.Logs = append(receipt.Logs, execlog)

	receipt.Ty = types.ExecOk

	//////////////////debug code begin///////////////
	privacylog.Debug("PrivacyTrading Exec", "ActionPrivacy2Privacy txhash", txhashstr, "receipt is", receipt)
	//////////////////debug code end///////////////
	return receipt, nil
}
