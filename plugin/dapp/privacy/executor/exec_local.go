package executor

import (
	"gitlab.33.cn/chain33/chain33/common"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (p *privacy) execLocal(receiptData *types.ReceiptData, tx *types.Transaction, index int) (*types.LocalDBSet, error) {
	dbSet := &types.LocalDBSet{}
	txhashstr := common.Bytes2Hex(tx.Hash())
	localDB := p.GetLocalDB()
	for _, item := range receiptData.Logs {
		if item.Ty != ty.TyLogPrivacyOutput {
			continue
		}
		var receiptPrivacyOutput ty.ReceiptPrivacyOutput
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
			localUTXOItem := &ty.LocalUTXOItem{
				Height:        p.GetHeight(),
				Txindex:       int32(index),
				Outindex:      int32(outputIndex),
				Txhash:        txhashInByte,
				Onetimepubkey: keyOutput.Onetimepubkey,
			}
			value := types.Encode(localUTXOItem)
			kv := &types.KeyValue{key, value}
			dbSet.KV = append(dbSet.KV, kv)

			//kv2，添加各种不同额度的kv记录，能让我们很方便的获知本系统存在的所有不同的额度的UTXO
			var amountTypes ty.AmountsOfUTXO
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
					dbSet.KV = append(dbSet.KV, kv)
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
				dbSet.KV = append(dbSet.KV, kv)
				localDB.Set(key2, types.Encode(&amountTypes))
			}

			//kv3,添加存在隐私交易token的类型
			var tokenNames ty.TokenNamesOfUTXO
			key3 := CalcprivacyKeyTokenTypes()
			value3, err := localDB.Get(key3)
			if err == nil && len(value3) != 0 {
				err := types.Decode(value3, &tokenNames)
				if err == nil {
					if _, ok := tokenNames.TokensMap[token]; !ok {
						tokenNames.TokensMap[token] = txhash
						kv := &types.KeyValue{key3, types.Encode(&tokenNames)}
						dbSet.KV = append(dbSet.KV, kv)
						localDB.Set(key3, types.Encode(&tokenNames))
					}
				}
			} else {
				tokenNames.TokensMap = make(map[string]string)
				tokenNames.TokensMap[token] = txhash
				kv := &types.KeyValue{key3, types.Encode(&tokenNames)}
				dbSet.KV = append(dbSet.KV, kv)
				localDB.Set(key3, types.Encode(&tokenNames))
			}
		}
	}
	return dbSet, nil
}

func (g *privacy) ExecLocal_Public2Privacy(payload *ty.Public2Privacy, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return g.execLocal(receiptData, tx, index)
}

func (g *privacy) ExecLocal_Privacy2Privacy(payload *ty.Privacy2Privacy, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return g.execLocal(receiptData, tx, index)
}

func (g *privacy) ExecLocal_Privacy2Public(payload *ty.Privacy2Public, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return g.execLocal(receiptData, tx, index)
}
