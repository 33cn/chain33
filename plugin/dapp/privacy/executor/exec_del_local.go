package executor

import (
	"gitlab.33.cn/chain33/chain33/common"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (p *privacy) execDelLocal(tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	txhashstr := common.Bytes2Hex(tx.Hash())
	dbSet := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		privacylog.Error("execDelLocal", "txhash", txhashstr, "receipt.GetTy() = ", receiptData.GetTy())
		return dbSet, nil
	}
	localDB := p.GetLocalDB()
	for i, item := range receiptData.Logs {
		if item.Ty != ty.TyLogPrivacyOutput {
			continue
		}
		var receiptPrivacyOutput ty.ReceiptPrivacyOutput
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
					if count, ok := amountTypes.AmountMap[keyOutput.Amount]; ok {
						count--
						if 0 == count {
							delete(amountTypes.AmountMap, keyOutput.Amount)
						} else {
							amountTypes.AmountMap[keyOutput.Amount] = count
						}

						value2 := types.Encode(&amountTypes)
						kv := &types.KeyValue{key2, value2}
						dbSet.KV = append(dbSet.KV, kv)
						//在本地的query数据库进行设置，这样可以防止相同的新增amout不会被重复生成kv,而进行重复的设置
						localDB.Set(key2, nil)
					}
				}
			}

			//kv3,添加存在隐私交易token的类型
			var tokenNames ty.TokenNamesOfUTXO
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
							dbSet.KV = append(dbSet.KV, kv)
							localDB.Set(key3, nil)
						}
					}
				}
			}
		}
	}
	return dbSet, nil
}

func (p *privacy) ExecDelLocal_Public2Privacy(payload *ty.Public2Privacy, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return p.execDelLocal(tx, receiptData, index)
}

func (p *privacy) ExecDelLocal_Privacy2Privacy(payload *ty.Privacy2Privacy, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return p.execDelLocal(tx, receiptData, index)
}

func (p *privacy) ExecDelLocal_Privacy2Public(payload *ty.Privacy2Public, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return p.execDelLocal(tx, receiptData, index)
}
