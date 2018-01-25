		var txinf types.ReplyTxInfo
		txinf.Hash = txhash
		txinf.Height = blockdetail.Block.Height
		txinf.Index = int64(index)
		txinfobyte, err := proto.Marshal(&txinf)
		if err != nil {
			storelog.Error("indexTxs Encode txinf err", "Height", blockdetail.Block.Height, "index", index)
			return err
		}



		blockheight := blockdetail.Block.Height*MaxTxsPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)

		//对block请求localKV

		//from addr
		pubkey := blockdetail.Block.Txs[index].Signature.GetPubkey()
		addr := account.PubKeyToAddress(pubkey)
		fromaddress := addr.String()
		if len(fromaddress) != 0 {
			fromkey := calcTxAddrDirHashKey(fromaddress, 1, heightstr)
			//fmt.Sprintf("%s:0:%s", fromaddress, heightstr)
			storeBatch.Set(fromkey, txinfobyte)
			storeBatch.Set(calcTxAddrHashKey(fromaddress, heightstr), txinfobyte)
			//storelog.Debug("indexTxs address ", "fromkey", fromkey, "value", txhash)
		}
		//toaddr
		toaddr := blockdetail.Block.Txs[index].GetTo()
		if len(toaddr) != 0 {
			tokey := calcTxAddrDirHashKey(toaddr, 2, heightstr)
			//fmt.Sprintf("%s:1:%s", toaddr, heightstr)
			storeBatch.Set([]byte(tokey), txinfobyte)
			storeBatch.Set(calcTxAddrHashKey(toaddr, heightstr), txinfobyte)

			//更新地址收到的amount
			var action types.CoinsAction
			err := types.Decode(blockdetail.Block.Txs[index].GetPayload(), &action)
			if err == nil {
				if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
					transfer := action.GetTransfer()
					bs.UpdateAddrReciver(cacheDB, toaddr, transfer.Amount, true)
				} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
					gen := action.GetGenesis()
					bs.UpdateAddrReciver(cacheDB, toaddr, gen.Amount, true)
				}
			}
		}
		//storelog.Debug("indexTxs Set txresult", "Height", blockdetail.Block.Height, "index", index, "txhashbyte", txhash)
	}