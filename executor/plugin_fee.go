package executor

totalFee.Fee += tx.Fee
totalFee.TxCount++
//保存手续费
feekv, err := saveFee(execute, &totalFee, b.ParentHash, b.Hash())
if err != nil {
	msg.Reply(exec.client.NewMessage("", types.EventAddBlock, err))
	return
}

func totalFeeKey(hash []byte) []byte {
	key := []byte("TotalFeeKey:")
	return append(key, hash...)
}

func saveFee(ex *executor, fee *types.TotalFee, parentHash, hash []byte) (*types.KeyValue, error) {
	totalFee := &types.TotalFee{}
	totalFeeBytes, err := ex.localDB.Get(totalFeeKey(parentHash))
	if err == nil {
		err = types.Decode(totalFeeBytes, totalFee)
		if err != nil {
			return nil, err
		}
	} else if err != types.ErrNotFound {
		return nil, err
	}

	totalFee.Fee += fee.Fee
	totalFee.TxCount += fee.TxCount
	return &types.KeyValue{totalFeeKey(hash), types.Encode(totalFee)}, nil
}

func delFee(ex *executor, hash []byte) (*types.KeyValue, error) {
	return &types.KeyValue{totalFeeKey(hash), types.Encode(&types.TotalFee{})}, nil
}


	//删除手续费
	feekv, err := delFee(execute, b.Hash())
	if err != nil {
		msg.Reply(exec.client.NewMessage("", types.EventDelBlock, err))
		return
	}