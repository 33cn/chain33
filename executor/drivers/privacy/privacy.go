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
	"math/rand"
	"sort"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var privacylog = log.New("module", "execs.privacy")

func Init() {
	// drivers.Register(newPrivacy().GetName(), newPrivacy, types.ForkV21Privacy)
	// 如果需要在开发环境下使用隐私交易，则需要使用下面这行代码，否则用上面的代码
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
	return types.ExecName("privacy")
}

func (p *privacy) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	txhashstr := common.Bytes2Hex(tx.Hash())
	privacylog.Debug("PrivacyTrading Exec", "Enter Exec txhash", txhashstr)
	defer privacylog.Debug("PrivacyTrading Exec", "Leave Exec txhash", txhashstr)
	_, err := p.DriverBase.Exec(tx, index)
	if err != nil {
		privacylog.Error("PrivacyTrading Exec", " txhash", txhashstr, "DriverBase.Exec error ", err)
		return nil, err
	}
	var action types.PrivacyAction
	err = types.Decode(tx.Payload, &action)
	if err != nil {
		privacylog.Error("PrivacyTrading Exec", "txhash", txhashstr, "Decode error ", err)
		return nil, err
	}
	privacylog.Info("PrivacyTrading Exec", "txhash", txhashstr, "action type ", action.Ty)
	if action.Ty == types.ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		public2Privacy := action.GetPublic2Privacy()
		if types.BTY == public2Privacy.Tokenname {
			coinsAccount := p.GetCoinsAccount()
			from := tx.From()
			receipt, err := coinsAccount.ExecWithdraw(p.GetAddr(), from, public2Privacy.Amount)
			if err != nil {
				privacylog.Error("PrivacyTrading Exec", "txhash", txhashstr, "ExecWithdraw error ", err)
				return nil, err
			}

			txhash := common.ToHex(tx.Hash())
			output := public2Privacy.GetOutput().GetKeyoutput()
			//因为只有包含当前交易的block被执行完成之后，同步到相应的钱包之后，
			//才能将相应的utxo作为input，进行支付，所以此处不需要进行将KV设置到
			//executor中的临时数据库中，只需要将kv返回给blockchain就行
			//即：一个块中产生的UTXO是不能够在同一个高度进行支付的
			for index, keyOutput := range output {
				key := CalcPrivacyOutputKey(public2Privacy.Tokenname, keyOutput.Amount, txhash, index)
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
			privacylog.Debug("PrivacyTrading Exec", "ActionPublic2Privacy txhash", txhashstr, "receipt is", receipt)
			//////////////////debug code end///////////////

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
				key := CalcPrivacyOutputKey(privacy2Privacy.Tokenname, keyOutput.Amount, txhash, index)
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
			privacylog.Debug("PrivacyTrading Exec", "ActionPrivacy2Privacy txhash", txhashstr, "receipt is", receipt)
			//////////////////debug code end///////////////
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
				privacylog.Error("PrivacyTrading Exec", "ActionPrivacy2Public txhash", txhashstr, "ExecDeposit error ", err)
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
				key := CalcPrivacyOutputKey(privacy2public.Tokenname, keyOutput.Amount, txhash, index)
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
			privacylog.Debug("PrivacyTrading Exec", "ActionPrivacy2Public txhash", txhashstr, "receipt is", receipt)
			//////////////////debug code end///////////////
			return receipt, nil
		} else {
			//token 转账操作
			privacylog.Error("PrivacyTrading Exec", "Do not support operator txhash", txhashstr)
		}

	}
	return nil, types.ErrActionNotSupport
}

func (p *privacy) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	txhashstr := common.Bytes2Hex(tx.Hash())
	privacylog.Debug("PrivacyTrading ExecLocal", "Enter ExecLocal txhash", txhashstr)
	defer privacylog.Debug("PrivacyTrading ExecLocal", "Leave ExecLocal txhash", txhashstr)

	set, err := p.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		privacylog.Error("PrivacyTrading ExecLocal", "DriverBase.ExecLocal txhash", txhashstr, "ExecLocal error ", err)
		return nil, err
	}

	if receipt.GetTy() != types.ExecOk {
		privacylog.Error("PrivacyTrading ExecLocal", "txhash", txhashstr, "receipt.GetTy() = ", receipt.GetTy())
		return set, nil
	}

	localDB := p.GetLocalDB()
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogPrivacyOutput {
			var receiptPrivacyOutput types.ReceiptPrivacyOutput
			err := types.Decode(item.Log, &receiptPrivacyOutput)
			if err != nil {
				privacylog.Error("PrivacyTrading ExecLocal", "txhash", txhashstr, "Decode item.Log error ", err)
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
						privacylog.Error("PrivacyTrading ExecLocal", "txhash", txhashstr, "value2 Decode error ", err)
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
				if err == nil && len(value3) != 0 {
					err := types.Decode(value3, &tokenNames)
					if err == nil {
						if _, ok := tokenNames.TokensMap[token]; !ok {
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
	txhashstr := common.Bytes2Hex(tx.Hash())
	privacylog.Debug("PrivacyTrading ExecDelLocal", "Enter ExecDelLocal txhash", txhashstr)
	defer privacylog.Debug("PrivacyTrading ExecDelLocal", "Leave ExecDelLocal txhash", txhashstr)

	set, err := p.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		privacylog.Error("PrivacyTrading ExecDelLocal", "txhash", txhashstr, "DriverBase.ExecDelLocal error ", err)
		return nil, err
	}

	if receipt.GetTy() != types.ExecOk {
		privacylog.Error("PrivacyTrading ExecDelLocal", "txhash", txhashstr, "receipt.GetTy() = ", receipt.GetTy())
		return set, nil
	}
	localDB := p.GetLocalDB()

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogPrivacyOutput {
			var receiptPrivacyOutput types.ReceiptPrivacyOutput
			err := types.Decode(item.Log, &receiptPrivacyOutput)
			if err != nil {
				privacylog.Error("PrivacyTrading ExecDelLocal", "txhash", txhashstr, "Decode item.Log error ", err)
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

	case "GetUTXOGlobalIndex":
		var getUtxoIndexReq types.ReqUTXOGlobalIndex
		err := types.Decode(params, &getUtxoIndexReq)
		if err != nil {
			return nil, err
		}
		privacylog.Info("GetUTXOGlobalIndex", "get utxo global index", err)

		return p.getGlobalUtxoIndex(&getUtxoIndexReq)
	case "GetTxsByAddr": // 通过查询获取交易hash这里主要通过获取隐私合约执行器地址的交易
		var in types.ReqAddr
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return p.GetTxsByAddr(&in)
	}
	return nil, types.ErrActionNotSupport
}

func (p *privacy) getUtxosByTokenAndAmount(tokenName string, amount int64, count int32) ([]*types.LocalUTXOItem, error) {
	localDB := p.GetLocalDB()
	var utxos []*types.LocalUTXOItem
	prefix := CalcPrivacyUTXOkeyHeightPrefix(tokenName, amount)
	values, err := localDB.List(prefix, nil, count, 0)
	if err != nil {
		return utxos, err
	}

	for _, value := range values {
		var utxo types.LocalUTXOItem
		err := types.Decode(value, &utxo)
		if err != nil {
			privacylog.Info("getUtxosByTokenAndAmount", "decode to LocalUTXOItem failed because of", err)
			return utxos, err
		}
		utxos = append(utxos, &utxo)
	}

	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Height <= utxos[j].Height
	})
	return utxos, nil
}

func (p *privacy) getGlobalUtxoIndex(getUtxoIndexReq *types.ReqUTXOGlobalIndex) (types.Message, error) {
	debugBeginTime := time.Now()
	utxoGlobalIndexResp := &types.ResUTXOGlobalIndex{}
	tokenName := getUtxoIndexReq.Tokenname
	currentHeight := p.GetHeight()
	for _, amount := range getUtxoIndexReq.Amount {
		utxos, err := p.getUtxosByTokenAndAmount(tokenName, amount, types.UTXOCacheCount)
		if err != nil {
			return utxoGlobalIndexResp, err
		}

		index := len(utxos) - 1
		for ; index >= 0; index-- {
			if utxos[index].GetHeight()+types.ConfirmedHeight <= currentHeight {
				break
			}
		}

		mixCount := getUtxoIndexReq.MixCount
		totalCnt := int32(index + 1)
		if mixCount > totalCnt {
			mixCount = totalCnt
		}

		utxoIndex4Amount := &types.UTXOIndex4Amount{
			Amount: amount,
		}

		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		positions := random.Perm(int(totalCnt))
		for i := int(mixCount - 1); i >= 0; i-- {
			position := positions[i]
			item := utxos[position]
			utxoGlobalIndex := &types.UTXOGlobalIndex{
				Outindex: item.GetOutindex(),
				Txhash:   item.GetTxhash(),
			}
			utxo := &types.UTXOBasic{
				UtxoGlobalIndex: utxoGlobalIndex,
				OnetimePubkey:   item.GetOnetimepubkey(),
			}
			utxoIndex4Amount.Utxos = append(utxoIndex4Amount.Utxos, utxo)
		}
		utxoGlobalIndexResp.UtxoIndex4Amount = append(utxoGlobalIndexResp.UtxoIndex4Amount, utxoIndex4Amount)
	}

	duration := time.Since(debugBeginTime)
	privacylog.Debug("getGlobalUtxoIndex cost", duration)
	return utxoGlobalIndexResp, nil
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
	txhashstr := common.Bytes2Hex(tx.Hash())
	privacylog.Debug("PrivacyTrading CheckTx", "Enter CheckTx txhash", txhashstr)
	defer privacylog.Debug("PrivacyTrading CheckTx", "Leave CheckTx txhash", txhashstr)

	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "Decode tx.Payload error", err)
		return err
	}
	privacylog.Info("PrivacyTrading CheckTx", "txhash", txhashstr, "action type ", action.Ty)
	if types.ActionPublic2Privacy == action.Ty {
		return nil
	}
	input := action.GetInput()
	output := action.GetOutput()
	if input == nil || output == nil {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "input", input, "output", output)
		return nil
	}
	//如果是私到私 或者私到公，交易费扣除则需要utxo实现,交易费并不生成真正的UTXO,也是即时燃烧掉而已
	var amount int64
	keyinput := input.Keyinput
	keyOutput := output.Keyoutput
	token := action.GetTokenName()
	if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		amount = action.GetPrivacy2Public().Amount
	}

	if tx.Fee < types.PrivacyTxFee {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "fee set:", tx.Fee, "required:", types.PrivacyTxFee, " error ErrPrivacyTxFeeNotEnough")
		return types.ErrPrivacyTxFeeNotEnough
	}

	var ringSignature types.RingSignature
	if err := types.Decode(tx.Signature.Signature, &ringSignature); err != nil {
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "Decode tx.Signature.Signature error ", err)
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
			keys = append(keys, CalcPrivacyOutputKey(token, input.Amount, common.ToHex(globalIndex.Txhash), int(globalIndex.Outindex)))
			pubkeys = append(pubkeys, ringSignature.Items[i].Pubkey[j])
		}
	}
	res, errIndex := p.checkUTXOValid(keyImages)
	if !res {
		if errIndex >= 0 && errIndex < int32(len(keyinput)) {
			input := keyinput[errIndex]
			privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "UTXO spent already errindex", errIndex, "utxo amout", input.Amount, "utxo keyimage", common.ToHex(input.KeyImage))
		}
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "checkUTXOValid failed ")
		return types.ErrDoubleSpendOccur
	}

	res, errIndex = p.checkPubKeyValid(keys, pubkeys)
	if !res {
		if errIndex >= 0 && errIndex < int32(len(pubkeys)) {
			privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "Wrong pubkey errIndex ", errIndex)
		}
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "checkPubKeyValid ", false)
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
		privacylog.Error("PrivacyTrading CheckTx", "txhash", txhashstr, "fee available:", feeAmount, "required:", types.PrivacyTxFee)
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
