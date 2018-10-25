package executor


	//del：tx
	kvdel := d.GetTx(tx, receipt, index)
	for k := range kvdel {
		kvdel[k].Value = nil
	}
	//del: addr index
	txindex := d.getTxIndex(tx, receipt, index)
	if len(txindex.from) != 0 {
		fromkey1 := types.CalcTxAddrDirHashKey(txindex.from, TxIndexFrom, txindex.heightstr)
		fromkey2 := types.CalcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{Key: fromkey1, Value: nil})
		set.KV = append(set.KV, &types.KeyValue{Key: fromkey2, Value: nil})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.from, 1, false)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}
	if len(txindex.to) != 0 {
		tokey1 := types.CalcTxAddrDirHashKey(txindex.to, TxIndexTo, txindex.heightstr)
		tokey2 := types.CalcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{Key: tokey1, Value: nil})
		set.KV = append(set.KV, &types.KeyValue{Key: tokey2, Value: nil})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.to, 1, false)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}
	set.KV = append(set.KV, kvdel...)



	//获取公共的信息
func (d *DriverBase) GetTx(tx *types.Transaction, receipt *types.ReceiptData, index int) []*types.KeyValue {
	txhash := tx.Hash()
	//构造txresult 信息保存到db中
	var txresult types.TxResult
	txresult.Height = d.GetHeight()
	txresult.Index = int32(index)
	txresult.Tx = tx
	txresult.Receiptdate = receipt
	txresult.Blocktime = d.GetBlockTime()
	txresult.ActionName = d.child.GetActionName(tx)
	var kvlist []*types.KeyValue
	kvlist = append(kvlist, &types.KeyValue{Key: types.CalcTxKey(txhash), Value: types.Encode(&txresult)})
	if types.IsEnable("quickIndex") {
		kvlist = append(kvlist, &types.KeyValue{Key: types.CalcTxShortKey(txhash), Value: []byte("1")})
	}
	return kvlist
}

type txIndex struct {
	from      string
	to        string
	heightstr string
	index     *types.ReplyTxInfo
}

//交易中 from/to 的索引
func (d *DriverBase) getTxIndex(tx *types.Transaction, receipt *types.ReceiptData, index int) *txIndex {
	var txIndexInfo txIndex
	var txinf types.ReplyTxInfo
	txinf.Hash = tx.Hash()
	txinf.Height = d.GetHeight()
	txinf.Index = int64(index)

	txIndexInfo.index = &txinf
	heightstr := fmt.Sprintf("%018d", d.GetHeight()*types.MaxTxsPerBlock+int64(index))
	txIndexInfo.heightstr = heightstr

	txIndexInfo.from = address.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	txIndexInfo.to = tx.GetRealToAddr()
	return &txIndexInfo
}


	//保存：tx
	kv := d.GetTx(tx, receipt, index)
	set.KV = append(set.KV, kv...)
	//保存: from/to
	txindex := d.getTxIndex(tx, receipt, index)
	txinfobyte := types.Encode(txindex.index)
	if len(txindex.from) != 0 {
		fromkey1 := types.CalcTxAddrDirHashKey(txindex.from, TxIndexFrom, txindex.heightstr)
		fromkey2 := types.CalcTxAddrHashKey(txindex.from, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{fromkey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{fromkey2, txinfobyte})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.from, 1, true)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}
	if len(txindex.to) != 0 {
		tokey1 := types.CalcTxAddrDirHashKey(txindex.to, TxIndexTo, txindex.heightstr)
		tokey2 := types.CalcTxAddrHashKey(txindex.to, txindex.heightstr)
		set.KV = append(set.KV, &types.KeyValue{tokey1, txinfobyte})
		set.KV = append(set.KV, &types.KeyValue{tokey2, txinfobyte})
		kv, err := updateAddrTxsCount(d.GetLocalDB(), txindex.to, 1, true)
		if err == nil && kv != nil {
			set.KV = append(set.KV, kv)
		}
	}