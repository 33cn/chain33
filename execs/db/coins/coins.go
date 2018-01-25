func AddTx() {
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
	
func DelTx() {
		blockheight := blockdetail.Block.Height*MaxTxsPerBlock + int64(index)
		heightstr := fmt.Sprintf("%018d", blockheight)

		//from addr
		pubkey := blockdetail.Block.Txs[index].Signature.GetPubkey()
		addr := account.PubKeyToAddress(pubkey)
		fromaddress := addr.String()
		if len(fromaddress) != 0 {
			fromkey := calcTxAddrDirHashKey(fromaddress, 1, heightstr)
			storeBatch.Delete(fromkey)
			storeBatch.Delete(calcTxAddrHashKey(fromaddress, heightstr))
			storelog.Error("DelTxs address ", "fromaddress", fromaddress, "value", txhash)
		}
		//toaddr
		toaddr := blockdetail.Block.Txs[index].GetTo()
		if len(toaddr) != 0 {
			tokey := calcTxAddrDirHashKey(toaddr, 2, heightstr)
			storeBatch.Delete([]byte(tokey))
			storeBatch.Delete(calcTxAddrHashKey(toaddr, heightstr))
			storelog.Error("DelTxs address ", "toaddr", toaddr, "value", txhash)

			//更新地址收到的amount,从原来的基础上减去Amount
			var action types.CoinsAction
			err := types.Decode(blockdetail.Block.Txs[index].GetPayload(), &action)
			if err == nil {
				if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
					transfer := action.GetTransfer()
					bs.UpdateAddrReciver(cacheDB, toaddr, transfer.Amount, false)
				} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
					gen := action.GetGenesis()
					bs.UpdateAddrReciver(cacheDB, toaddr, gen.Amount, false)
				}
			}
		}
		storelog.Error("DelTxs txresult", "Height", blockdetail.Block.Height, "index", index, "txhashbyte", txhash)
	}
}

func NewCacheDB() *CacheDB {
	return &CacheDB{make(map[string]*AddrRecv)}
}

func (db *CacheDB) Get(bs *BlockStore, addr string) (int64, error) {
	if value, ok := db.cache[addr]; ok {
		return value.amount, nil
	}
	var Reciveramount int64 = 0
	AddrReciver := bs.db.Get(calcAddrKey(addr))
	if len(AddrReciver) != 0 {
		err := json.Unmarshal(AddrReciver, &Reciveramount)
		if err != nil {
			storelog.Error("CacheDB Get unmarshal", "error", err)
			return 0, err
		}
	}
	var addrRecv AddrRecv
	addrRecv.amount = Reciveramount
	addrRecv.addr = addr
	db.cache[addr] = &addrRecv
	return Reciveramount, nil
}

func (db *CacheDB) Set(addr string, amount int64) {
	var addrRecv AddrRecv
	addrRecv.amount = amount
	addrRecv.addr = addr
	db.cache[addr] = &addrRecv
}

func (db *CacheDB) SetBatch(storeBatch dbm.Batch) {
	for _, v := range db.cache {
		amountbytes, err := json.Marshal(v.amount)
		if err != nil {
			storelog.Error("UpdateAddrReciver marshal", "error", err)
			continue
		}
		//storelog.Error("SetBatch Set", "key", string(k), "value", v.amount)
		storeBatch.Set(calcAddrKey(v.addr), amountbytes)
	}

	for k, _ := range db.cache {
		//storelog.Error("SetBatch delete", "key", string(k), "value", v.amount)
		delete(db.cache, k)
	}
}

type CacheDB struct {
	cache map[string]*AddrRecv
}
type AddrRecv struct {
	addr   string
	amount int64
}


//存储地址上收币的信息
func calcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("Addr:%s", addr))
}

//获取地址收到的amount
func (bs *BlockStore) GetAddrReciver(addr string) (int64, error) {
	if len(addr) == 0 {
		err := errors.New("input addr is null")
		return 0, err
	}
	var Reciveramount int64
	AddrReciver := bs.db.Get(calcAddrKey(addr))
	if len(AddrReciver) == 0 {
		err := errors.New("does not exist AddrReciver!")
		return 0, err
	}
	err := json.Unmarshal(AddrReciver, &Reciveramount)
	if err != nil {
		storelog.Error("GetAddrReciver unmarshal", "error", err)
		return 0, nil
	}
	return Reciveramount, nil
}