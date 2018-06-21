package privacy

/*
privacy执行器支持隐私交易的执行，

主要提供操作有以下几种：
1）公开地址转账到一次性地址中，即：public address -> one-time addrss
2）隐私转账，隐私余额会被继续转账到一次性地址中 one-time address -> one-time address；
3）隐私余额转账到公开地址中， 即：one-time address -> public address

操作流程：
1）如果需要进行coin或token的隐私转账，则需要首先将其balance转账至privacy合约账户中；
2）此时用户可以发起隐私交易，在交易信息中指定接收的公钥对(A,B),执行成功之后，balance会被存入到一次性地址中；
3）发起交易，

*/

import (
	"bytes"

	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var privacylog = log.New("module", "execs.privacy")

func init() {
	drivers.Register(newPrivacy().GetName(), newPrivacy, 0)
}

type privacy struct {
	drivers.DriverBase
}

func newPrivacy() drivers.Driver {
	t := &privacy{}
	t.SetChild(t)
	return t
}

func (p *privacy) GetName() string {
	return "privacy"
}

func (p *privacy) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := p.DriverBase.Exec(tx, index)
	if err != nil {
		return nil, err
	}
	var action types.PrivacyAction
	err = types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	height := p.GetHeight()
	privacylog.Info("Privacy exec", "action type", action.Ty)
	if action.Ty == types.ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		public2Privacy := action.GetPublic2Privacy()
		if types.BTY == public2Privacy.Tokenname {
			coinsAccount := p.GetCoinsAccount()
			from := account.From(tx).String()
			receipt, err := coinsAccount.ExecWithdraw(p.GetAddr(), from, public2Privacy.Amount)
			if err != nil {
				privacylog.Error("Privacy exec ActionPublic2Privacy", "ExecWithdraw failed due to ", err)
				return nil, err
			}

			txhash := common.ToHex(tx.Hash())
			output := public2Privacy.GetOutput().GetKeyoutput()
			//因为只有包含当前交易的block被执行完成之后，同步到相应的钱包之后，
			//才能将相应的utxo作为input，进行支付，所以此处不需要进行将KV设置到
			//executor中的临时数据库中，只需要将kv返回给blockchain就行
			//即：一个块中产生的UTXO是不能够在同一个高度进行支付的
			for index, keyOutput := range output {
				key := CalcPrivacyOutputKey(public2Privacy.Tokenname, keyOutput.Amount, height, txhash, index)
				value := types.Encode(keyOutput)
				receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
			}

			receiptPrivacyOutput := &types.ReceiptPrivacyOutput{
				Token:     public2Privacy.Tokenname,
				Keyoutput: public2Privacy.GetOutput().Keyoutput,
			}
			execlog := &types.ReceiptLog{types.TyLogPrivacyOutput, types.Encode(receiptPrivacyOutput)}
			receipt.Logs = append(receipt.Logs, execlog)

			//////////////////debug code begin///////////////
			privacylog.Debug("Privacy exec ActionPublic2Privacy", "receipt is", receipt)
			//////////////////debug code end///////////////

			privacylog.Info(fmt.Sprintf("Finish Exec Privacy Pub->Priv Tx. HASH %s", txhash))
			return receipt, nil
		} else {
			//token 转账操作

		}

	} else if action.Ty == types.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		privacy2Privacy := action.GetPrivacy2Privacy()
		if types.BTY == privacy2Privacy.Tokenname {
			receipt := &types.Receipt{KV: make([]*types.KeyValue, 0)}
			privacyInput := privacy2Privacy.Input
			for _, keyInput := range privacyInput.Keyinput {
				value := []byte{KeyImageSpentAlready}
				key := calcPrivacyKeyImageKey(privacy2Privacy.Tokenname, keyInput.KeyImage)
				stateDB := p.GetStateDB()
				stateDB.Set(key, value)
				receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
			}

			execlog := &types.ReceiptLog{types.TyLogPrivacyInput, types.Encode(privacy2Privacy.GetInput())}
			receipt.Logs = append(receipt.Logs, execlog)

			txhash := common.ToHex(tx.Hash())
			output := privacy2Privacy.GetOutput().GetKeyoutput()
			for index, keyOutput := range output {
				key := CalcPrivacyOutputKey(privacy2Privacy.Tokenname, keyOutput.Amount, height, txhash, index)
				value := types.Encode(keyOutput)
				receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
			}

			receiptPrivacyOutput := &types.ReceiptPrivacyOutput{
				Token:     privacy2Privacy.Tokenname,
				Keyoutput: privacy2Privacy.GetOutput().Keyoutput,
			}
			execlog = &types.ReceiptLog{types.TyLogPrivacyOutput, types.Encode(receiptPrivacyOutput)}
			receipt.Logs = append(receipt.Logs, execlog)

			receipt.Ty = types.ExecOk

			//////////////////debug code begin///////////////
			privacylog.Debug("Privacy exec ActionPrivacy2Privacy", "receipt is", receipt)
			//////////////////debug code end///////////////
			privacylog.Info(fmt.Sprintf("Finish Exec Privacy Priv->Priv Tx. HASH %s", txhash))
			return receipt, nil

		} else {
			//token 转账操作

		}

	} else if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		privacy2public := action.GetPrivacy2Public()
		if types.BTY == privacy2public.Tokenname {
			coinsAccount := p.GetCoinsAccount()
			receipt, err := coinsAccount.ExecDeposit(tx.To, p.GetAddr(), privacy2public.Amount)
			if err != nil {
				privacylog.Error("Privacy exec ActionPrivacy2Public", "ExecDeposit failed due to ", err)
				return nil, err
			}

			privacyInput := privacy2public.Input
			for _, keyInput := range privacyInput.Keyinput {
				value := []byte{KeyImageSpentAlready}
				// TODO: 隐私交易，需要按照规则写key
				key := calcPrivacyKeyImageKey(privacy2public.Tokenname, keyInput.KeyImage)
				stateDB := p.GetStateDB()
				//stateDB.Set(keyInput.KeyImage, value)
				stateDB.Set(key, value)
				receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
			}

			execlog := &types.ReceiptLog{types.TyLogPrivacyInput, types.Encode(privacy2public.GetInput())}
			receipt.Logs = append(receipt.Logs, execlog)

			txhash := common.ToHex(tx.Hash())
			output := privacy2public.GetOutput().GetKeyoutput()
			for index, keyOutput := range output {
				key := CalcPrivacyOutputKey(privacy2public.Tokenname, keyOutput.Amount, height, txhash, index)
				value := types.Encode(keyOutput)
				receipt.KV = append(receipt.KV, &types.KeyValue{key, value})
			}
			receiptPrivacyOutput := &types.ReceiptPrivacyOutput{
				Token:     privacy2public.Tokenname,
				Keyoutput: privacy2public.GetOutput().Keyoutput,
			}
			execlog = &types.ReceiptLog{types.TyLogPrivacyOutput, types.Encode(receiptPrivacyOutput)}
			receipt.Logs = append(receipt.Logs, execlog)

			//////////////////debug code begin///////////////
			privacylog.Debug("Privacy exec ActionPrivacy2Public", "receipt is", receipt)
			//////////////////debug code end///////////////
			privacylog.Info(fmt.Sprintf("Finish Exec Privacy Priv->Pub Tx. HASH %s", txhash))
			return receipt, nil
		} else {
			//token 转账操作

		}

	}
	return nil, types.ErrActionNotSupport
}

func (p *privacy) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := p.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}

	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	localDB := p.GetLocalDB()

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogPrivacyOutput {
			var receiptPrivacyOutput types.ReceiptPrivacyOutput
			err := types.Decode(item.Log, &receiptPrivacyOutput)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}

			token := receiptPrivacyOutput.Token
			txhashInByte := tx.Hash()
			txhash := common.ToHex(txhashInByte)
			for outputIndex, keyOutput := range receiptPrivacyOutput.Keyoutput {
				//kv1，添加一个具体的UTXO，方便我们可以查询相应token下特定额度下，不同高度时，不同txhash的UTXO
				key := CalcPrivacyUTXOkeyHeight(token, keyOutput.Amount, p.GetHeight(), txhash, index, outputIndex)
				localUTXOItem := &types.LocalUTXOItem{
					Height:        p.GetHeight(),
					Txindex:       int32(index),
					Outindex:      int32(outputIndex),
					Txhash:        txhashInByte,
					Onetimepubkey: keyOutput.Onetimepubkey,
				}
				value := types.Encode(localUTXOItem)
				kv := &types.KeyValue{key, value}
				set.KV = append(set.KV, kv)

				//kv2，添加各种不同额度的kv记录，能让我们很方便的获知本系统存在的所有不同的额度的UTXO
				var amountTypes types.AmountsOfUTXO
				key2 := CalcprivacyKeyTokenAmountType(token)
				value2, err := localDB.Get(key2)
				//如果该种token不是第一次进行隐私操作
				if err == nil && value2 != nil {
					err := types.Decode(value2, &amountTypes)
					if err == nil {
						//当本地数据库不存在这个额度时，则进行添加
						amount, ok := amountTypes.AmountMap[keyOutput.Amount]
						if !ok {
							amountTypes.AmountMap[keyOutput.Amount] = 1
						} else {
							//todo:考虑后续溢出的情况
							amountTypes.AmountMap[keyOutput.Amount] = amount + 1
						}
						kv := &types.KeyValue{key2, types.Encode(&amountTypes)}
						set.KV = append(set.KV, kv)
						//在本地的query数据库进行设置，这样可以防止相同的新增amout不会被重复生成kv,而进行重复的设置
						localDB.Set(key2, types.Encode(&amountTypes))
					} else {
						panic(err)
					}
				} else {
					//如果该种token第一次进行隐私操作
					amountTypes.AmountMap = make(map[int64]int64)
					amountTypes.AmountMap[keyOutput.Amount] = 1
					kv := &types.KeyValue{key2, types.Encode(&amountTypes)}
					set.KV = append(set.KV, kv)
					localDB.Set(key2, types.Encode(&amountTypes))
				}

				//kv3,添加存在隐私交易token的类型
				var tokenNames types.TokenNamesOfUTXO
				key3 := CalcprivacyKeyTokenTypes()
				value3, err := localDB.Get(key3)
				if err == nil && value3 != nil {
					err := types.Decode(value3, &tokenNames)
					if err == nil {
						if _, ok := tokenNames.TokensMap[token]; !ok {
							if nil == tokenNames.TokensMap {
								tokenNames.TokensMap = make(map[string]string)
							}
							tokenNames.TokensMap[token] = txhash
							kv := &types.KeyValue{key3, types.Encode(&tokenNames)}
							set.KV = append(set.KV, kv)
							localDB.Set(key3, types.Encode(&tokenNames))
						}
					}
				} else {
					tokenNames.TokensMap = make(map[string]string)
					tokenNames.TokensMap[token] = txhash
					kv := &types.KeyValue{key3, types.Encode(&tokenNames)}
					set.KV = append(set.KV, kv)
					localDB.Set(key3, types.Encode(&tokenNames))
				}
			}
		}
	}

	return set, nil
}

func (p *privacy) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := p.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}

	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	localDB := p.GetLocalDB()

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogPrivacyOutput {
			var receiptPrivacyOutput types.ReceiptPrivacyOutput
			err := types.Decode(item.Log, &receiptPrivacyOutput)
			if err != nil {
				panic(err)
			}

			token := receiptPrivacyOutput.Token
			txhashInByte := tx.Hash()
			txhash := common.ToHex(txhashInByte)
			for m, keyOutput := range receiptPrivacyOutput.Keyoutput {
				//kv1，添加一个具体的UTXO，方便我们可以查询相应token下特定额度下，不同高度时，不同txhash的UTXO
				key := CalcPrivacyUTXOkeyHeight(token, keyOutput.Amount, p.GetHeight(), txhash, i, m)
				kv := &types.KeyValue{key, nil}
				set.KV = append(set.KV, kv)

				//kv2，添加各种不同额度的kv记录，能让我们很方便的获知本系统存在的所有不同的额度的UTXO
				var amountTypes types.AmountsOfUTXO
				key2 := CalcprivacyKeyTokenAmountType(token)
				value2, err := localDB.Get(key2)
				//如果该种token不是第一次进行隐私操作
				if err == nil && value2 != nil {
					err := types.Decode(value2, &amountTypes)
					if err == nil {
						//当本地数据库不存在这个额度时，则进行添加
						if count, ok := amountTypes.AmountMap[keyOutput.Amount]; ok {
							count--
							if 0 == count {
								delete(amountTypes.AmountMap, keyOutput.Amount)
							} else {
								amountTypes.AmountMap[keyOutput.Amount] = count
							}

							value2 := types.Encode(&amountTypes)
							kv := &types.KeyValue{key2, value2}
							set.KV = append(set.KV, kv)
							//在本地的query数据库进行设置，这样可以防止相同的新增amout不会被重复生成kv,而进行重复的设置
							localDB.Set(key2, nil)
						}
					}
				}

				//kv3,添加存在隐私交易token的类型
				var tokenNames types.TokenNamesOfUTXO
				key3 := CalcprivacyKeyTokenTypes()
				value3, err := localDB.Get(key3)
				if err == nil && value3 != nil {
					err := types.Decode(value3, &tokenNames)
					if err == nil {
						if settxhash, ok := tokenNames.TokensMap[token]; ok {
							if settxhash == txhash {
								delete(tokenNames.TokensMap, token)
								value3 := types.Encode(&tokenNames)
								kv := &types.KeyValue{key3, value3}
								set.KV = append(set.KV, kv)
								localDB.Set(key3, nil)
							}
						}
					}
				}
			}
		}
	}

	return set, nil
}

func (p *privacy) Query(funcName string, params []byte) (types.Message, error) {
	privacylog.Info("privacy Query", "name", funcName)
	switch funcName {
	case "ShowAmountsOfUTXO":
		var reqtoken types.ReqPrivacyToken
		err := types.Decode(params, &reqtoken)
		if err != nil {
			return nil, err
		}
		privacylog.Info("ShowAmountsOfUTXO", "query tokens", reqtoken)

		return p.ShowAmountsOfUTXO(&reqtoken)
	case "ShowUTXOs4SpecifiedAmount":
		var reqtoken types.ReqPrivacyToken
		err := types.Decode(params, &reqtoken)
		if err != nil {
			return nil, err
		}
		privacylog.Info("ShowUTXOs4SpecifiedAmount", "query tokens", reqtoken)

		return p.ShowUTXOs4SpecifiedAmount(&reqtoken)

	}
	return nil, types.ErrActionNotSupport
}

//获取指定amount下的所有utxo，这样就可以查询当前系统不同amout下存在的UTXO,可以帮助查询用于混淆用的资源
//也可以确认币种的碎片化问题
//显示存在的各种不同的额度的UTXO,如1,3,5,10,20,30,100...
func (p *privacy) ShowAmountsOfUTXO(reqtoken *types.ReqPrivacyToken) (types.Message, error) {
	querydb := p.GetLocalDB()

	key := CalcprivacyKeyTokenAmountType(reqtoken.Token)
	replyAmounts := &types.ReplyPrivacyAmounts{}
	value, err := querydb.Get(key)
	if err != nil {
		return replyAmounts, err
	}
	if value != nil {
		var amountTypes types.AmountsOfUTXO
		err := types.Decode(value, &amountTypes)
		if err == nil {
			for amount, count := range amountTypes.AmountMap {
				amountDetail := &types.AmountDetail{
					Amount: amount,
					Count:  count,
				}
				replyAmounts.AmountDetail = append(replyAmounts.AmountDetail, amountDetail)
			}
		}

	}
	return replyAmounts, nil
}

//显示在指定额度下的UTXO的具体信息，如区块高度，交易hash，输出索引等具体信息
func (p *privacy) ShowUTXOs4SpecifiedAmount(reqtoken *types.ReqPrivacyToken) (types.Message, error) {
	querydb := p.GetLocalDB()

	var replyUTXOsOfAmount types.ReplyUTXOsOfAmount
	values, err := querydb.List(CalcPrivacyUTXOkeyHeightPrefix(reqtoken.Token, reqtoken.Amount), nil, 0, 0)
	if err != nil {
		return &replyUTXOsOfAmount, err
	}
	if len(values) != 0 {
		for _, value := range values {
			var localUTXOItem types.LocalUTXOItem
			err := types.Decode(value, &localUTXOItem)
			if err == nil {
				replyUTXOsOfAmount.LocalUTXOItems = append(replyUTXOsOfAmount.LocalUTXOItems, &localUTXOItem)
			}
		}
	}

	return &replyUTXOsOfAmount, nil
}

func (p *privacy) CheckTx(tx *types.Transaction, index int) error {
	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return err
	}
	//如果是私到私 或者私到公，交易费扣除则需要utxo实现,交易费并不生成真正的UTXO,也是即时燃烧掉而已
	var keyinput []*types.KeyInput
	var keyOutput []*types.KeyOutput
	var token string
	var amount int64
	if action.Ty == types.ActionPublic2Privacy {
		return nil
	} else if action.Ty == types.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		keyinput = action.GetPrivacy2Privacy().Input.Keyinput
		keyOutput = action.GetPrivacy2Privacy().Output.Keyoutput
		token = action.GetPrivacy2Privacy().Tokenname
	} else if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		keyinput = action.GetPrivacy2Public().Input.Keyinput
		keyOutput = action.GetPrivacy2Public().Output.Keyoutput
		token = action.GetPrivacy2Public().Tokenname
		amount = action.GetPrivacy2Public().Amount
	}

	if tx.Fee < types.PrivacyTxFee {
		privacylog.Error("Privacy CheckTx failed due to ErrPrivacyTxFeeNotEnough", "fee set:", tx.Fee,
			"required:", types.PrivacyTxFee)
		return types.ErrPrivacyTxFeeNotEnough
	}

	var ringSignature types.RingSignature
	if err := types.Decode(tx.Signature.Signature, &ringSignature); err != nil {
		privacylog.Error("Privacy exec failed to decode ring signature")
		return err
	}

	totalInput := int64(0)
	totalOutput := int64(0)
	inputCnt := len(keyinput)
	keyImages := make([][]byte, inputCnt)
	keys := make([][]byte, 0)
	pubkeys := make([][]byte, 0)
	for i, input := range keyinput {
		totalInput += input.Amount
		keyImages[i] = calcPrivacyKeyImageKey(token, input.KeyImage)
		for j, globalIndex := range input.UtxoGlobalIndex {
			keys = append(keys, CalcPrivacyOutputKey(token, input.Amount, globalIndex.Height, common.ToHex(globalIndex.Txhash), int(globalIndex.Outindex)))
			pubkeys = append(pubkeys, ringSignature.Items[i].Pubkey[j])
		}
	}
	res, errIndex := p.checkUTXOValid(keyImages)
	if !res {
		if errIndex >= 0 && errIndex < int32(len(keyinput)) {
			input := keyinput[errIndex]
			privacylog.Error("UTXO spent already", "privacy tx hash", common.ToHex(tx.Hash()),
				"errindex", errIndex, "utxo amout", input.Amount, "utxo keyimage", common.ToHex(input.KeyImage))
		}

		privacylog.Error("exec UTXO double spend check failed")
		return types.ErrDoubeSpendOccur
	}

	res, errIndex = p.checkPubKeyValid(keys, pubkeys)
	if !res {
		if errIndex >= 0 && errIndex < int32(len(pubkeys)) {
			privacylog.Error("Wrong pubkey", "errIndex", errIndex)
		}
		privacylog.Error("exec UTXO double spend check failed")
		return types.ErrPubkeysOfUTXO
	}

	for _, output := range keyOutput {
		totalOutput += output.Amount
	}

	feeAmount := int64(0)
	if action.Ty == types.ActionPrivacy2Privacy {
		feeAmount = totalInput - totalOutput
	} else {
		feeAmount = totalInput - totalOutput - amount
	}

	if feeAmount < types.PrivacyTxFee {
		privacylog.Error("Privacy CheckTx failed due to ErrPrivacyTxFeeNotEnough", "fee available:", feeAmount,
			"required:", types.PrivacyTxFee)
		return types.ErrPrivacyTxFeeNotEnough
	}

	return nil
}

//通过keyImage确认是否存在双花，有效即不存在双花，返回true，反之则返回false
func (p *privacy) checkUTXOValid(keyImages [][]byte) (bool, int32) {
	stateDB := p.GetStateDB()
	values, err := stateDB.BatchGet(keyImages)
	if err != nil {
		privacylog.Error("exec module", "checkUTXOValid failed to get value from statDB")
		return false, Invalid_index
	}
	if len(values) != len(keyImages) {
		privacylog.Error("exec module", "checkUTXOValid return different count value with keys")
		return false, Invalid_index
	}
	for i, value := range values {
		if value != nil {
			privacylog.Error("exec module", "checkUTXOValid i=", i, " value=", value)
			return false, int32(i)
		}
	}

	return true, Invalid_index
}

func (p *privacy) checkPubKeyValid(keys [][]byte, pubkeys [][]byte) (bool, int32) {
	values, err := p.GetStateDB().BatchGet(keys)
	if err != nil {
		privacylog.Error("exec module", "checkPubKeyValid failed to get value from statDB with err", err)
		return false, Invalid_index
	}

	if len(values) != len(pubkeys) {
		privacylog.Error("exec module", "checkPubKeyValid return different count value with keys")
		return false, Invalid_index
	}

	for i, value := range values {
		var keyoutput types.KeyOutput
		types.Decode(value, &keyoutput)
		if !bytes.Equal(keyoutput.Onetimepubkey, pubkeys[i]) {
			privacylog.Error("exec module", "Invalid pubkey for tx with hash", string(keys[i]))
			return false, int32(i)
		}
	}

	return true, Invalid_index
}
