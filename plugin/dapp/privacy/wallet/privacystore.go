package wallet

import (
	"bytes"
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/db"
	privacy "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/crypto"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"

	"github.com/golang/protobuf/proto"
)

const (
	// 隐私交易数据库版本号
	PRIVACYDBVERSION int64 = 1
)

func NewStore(db db.DB) *privacyStore {
	return &privacyStore{Store: wcom.NewStore(db)}
}

// privacyStore 隐私交易数据库存储操作类
type privacyStore struct {
	*wcom.Store
}

func (store *privacyStore) getVersion() int64 {
	var version int64
	data, err := store.Get(calcPrivacyDBVersion())
	if err != nil || data == nil {
		bizlog.Error("getVersion", "db.Get error", err)
		return 0
	}
	err = json.Unmarshal(data, &version)
	if err != nil {
		bizlog.Error("getVersion", "json.Unmarshal error", err)
		return 0
	}
	return version
}

func (store *privacyStore) setVersion() error {
	version := PRIVACYDBVERSION
	data, err := json.Marshal(&version)
	if err != nil || data == nil {
		bizlog.Error("setVersion", "json.Marshal error", err)
		return err
	}
	err = store.GetDB().SetSync(calcPrivacyDBVersion(), data)
	if err != nil {
		bizlog.Error("setVersion", "db.SetSync error", err)
	}
	return err
}

func (store *privacyStore) getAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {
	if len(addr) == 0 {
		bizlog.Error("getAccountByPrefix addr is nil")
		return nil, types.ErrInvalidParam
	}
	list := store.NewListHelper()
	accbytes := list.PrefixScan([]byte(addr))
	if len(accbytes) == 0 {
		bizlog.Error("getAccountByPrefix addr not exist")
		return nil, types.ErrAccountNotExist
	}
	WalletAccountStores := make([]*types.WalletAccountStore, len(accbytes))
	for index, accbyte := range accbytes {
		var walletaccount types.WalletAccountStore
		err := proto.Unmarshal(accbyte, &walletaccount)
		if err != nil {
			bizlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		WalletAccountStores[index] = &walletaccount
	}
	return WalletAccountStores, nil
}

func (store *privacyStore) getWalletAccountPrivacy(addr string) (*privacytypes.WalletAccountPrivacy, error) {
	if len(addr) == 0 {
		bizlog.Error("GetWalletAccountPrivacy addr is nil")
		return nil, types.ErrInvalidParam
	}

	privacyByte, err := store.Get(calcPrivacyAddrKey(addr))
	if err != nil {
		bizlog.Error("GetWalletAccountPrivacy", "db Get error ", err)
		return nil, err
	}
	if nil == privacyByte {
		return nil, privacytypes.ErrPrivacyNotEnabled
	}
	var accPrivacy privacytypes.WalletAccountPrivacy
	err = proto.Unmarshal(privacyByte, &accPrivacy)
	if err != nil {
		bizlog.Error("GetWalletAccountPrivacy", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &accPrivacy, nil
}

func (store *privacyStore) getAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(addr) == 0 {
		bizlog.Error("GetAccountByAddr addr is nil")
		return nil, types.ErrInvalidParam
	}
	data, err := store.Get(calcAddrKey(addr))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			bizlog.Debug("GetAccountByAddr addr", "err", err)
		}
		return nil, types.ErrAddrNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		bizlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

func (store *privacyStore) setWalletAccountPrivacy(addr string, privacy *privacytypes.WalletAccountPrivacy) error {
	if len(addr) == 0 {
		bizlog.Error("SetWalletAccountPrivacy addr is nil")
		return types.ErrInvalidParam
	}
	if privacy == nil {
		bizlog.Error("SetWalletAccountPrivacy privacy is nil")
		return types.ErrInvalidParam
	}

	privacybyte, err := proto.Marshal(privacy)
	if err != nil {
		bizlog.Error("SetWalletAccountPrivacy proto.Marshal err!", "err", err)
		return types.ErrMarshal
	}

	newbatch := store.NewBatch(true)
	newbatch.Set(calcPrivacyAddrKey(addr), privacybyte)
	newbatch.Write()

	return nil
}
func (store *privacyStore) listAvailableUTXOs(token, addr string) ([]*privacytypes.PrivacyDBStore, error) {
	if 0 == len(addr) {
		bizlog.Error("listWalletPrivacyAccount addr is nil")
		return nil, types.ErrInvalidParam
	}

	list := store.NewListHelper()
	onetimeAccbytes := list.PrefixScan(calcPrivacyUTXOPrefix4Addr(token, addr))
	if len(onetimeAccbytes) == 0 {
		bizlog.Error("listWalletPrivacyAccount ", "addr not exist", addr)
		return nil, nil
	}

	privacyDBStoreSlice := make([]*privacytypes.PrivacyDBStore, len(onetimeAccbytes))
	for index, acckeyByte := range onetimeAccbytes {
		var accPrivacy privacytypes.PrivacyDBStore
		accByte, err := store.Get(acckeyByte)
		if err != nil {
			bizlog.Error("listWalletPrivacyAccount", "db Get err:", err)
			return nil, err
		}
		err = proto.Unmarshal(accByte, &accPrivacy)
		if err != nil {
			bizlog.Error("listWalletPrivacyAccount", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		privacyDBStoreSlice[index] = &accPrivacy
	}
	return privacyDBStoreSlice, nil
}

func (store *privacyStore) listFrozenUTXOs(token, addr string) ([]*privacytypes.FTXOsSTXOsInOneTx, error) {
	if 0 == len(addr) {
		bizlog.Error("listFrozenUTXOs addr is nil")
		return nil, types.ErrInvalidParam
	}
	list := store.NewListHelper()
	values := list.List(calcFTXOsKeyPrefix(token, addr), nil, 0, 0)
	if len(values) == 0 {
		bizlog.Error("listFrozenUTXOs ", "addr not exist", addr)
		return nil, nil
	}

	ftxoslice := make([]*privacytypes.FTXOsSTXOsInOneTx, 0)
	for _, acckeyByte := range values {
		var ftxotx privacytypes.FTXOsSTXOsInOneTx
		accByte, err := store.Get(acckeyByte)
		if err != nil {
			bizlog.Error("listFrozenUTXOs", "db Get err:", err)
			return nil, err
		}

		err = proto.Unmarshal(accByte, &ftxotx)
		if err != nil {
			bizlog.Error("listFrozenUTXOs", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		ftxoslice = append(ftxoslice, &ftxotx)
	}
	return ftxoslice, nil
}

func (store *privacyStore) getWalletPrivacyTxDetails(param *privacytypes.ReqPrivacyTransactionList) (*types.WalletTxDetails, error) {
	if param == nil {
		bizlog.Error("getWalletPrivacyTxDetails param is nil")
		return nil, types.ErrInvalidParam
	}
	if param.SendRecvFlag != sendTx && param.SendRecvFlag != recvTx {
		bizlog.Error("procPrivacyTransactionList", "invalid sendrecvflag ", param.SendRecvFlag)
		return nil, types.ErrInvalidParam
	}
	var txbytes [][]byte
	list := store.NewListHelper()
	if len(param.Seedtxhash) == 0 {
		var keyPrefix []byte
		if param.SendRecvFlag == sendTx {
			keyPrefix = calcSendPrivacyTxKey(param.Tokenname, param.Address, "")
		} else {
			keyPrefix = calcRecvPrivacyTxKey(param.Tokenname, param.Address, "")
		}
		txkeybytes := list.IteratorScanFromLast(keyPrefix, param.Count)
		for _, keybyte := range txkeybytes {
			value, err := store.Get(keybyte)
			if err != nil {
				bizlog.Error("getWalletPrivacyTxDetails", "db Get error", err)
				continue
			}
			if nil == value {
				continue
			}
			txbytes = append(txbytes, value)
		}
		if len(txbytes) == 0 {
			bizlog.Error("getWalletPrivacyTxDetails does not exist tx!")
			return nil, types.ErrTxNotExist
		}

	} else {
		list := store.NewListHelper()
		var txkeybytes [][]byte
		if param.SendRecvFlag == sendTx {
			txkeybytes = list.IteratorScan([]byte(SendPrivacyTx), calcSendPrivacyTxKey(param.Tokenname, param.Address, string(param.Seedtxhash)), param.Count, param.Direction)
		} else {
			txkeybytes = list.IteratorScan([]byte(RecvPrivacyTx), calcRecvPrivacyTxKey(param.Tokenname, param.Address, string(param.Seedtxhash)), param.Count, param.Direction)
		}
		for _, keybyte := range txkeybytes {
			value, err := store.Get(keybyte)
			if err != nil {
				bizlog.Error("getWalletPrivacyTxDetails", "db Get error", err)
				continue
			}
			if nil == value {
				continue
			}
			txbytes = append(txbytes, value)
		}

		if len(txbytes) == 0 {
			bizlog.Error("getWalletPrivacyTxDetails does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	}

	txDetails := new(types.WalletTxDetails)
	txDetails.TxDetails = make([]*types.WalletTxDetail, len(txbytes))
	for index, txdetailbyte := range txbytes {
		var txdetail types.WalletTxDetail
		err := proto.Unmarshal(txdetailbyte, &txdetail)
		if err != nil {
			bizlog.Error("getWalletPrivacyTxDetails", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		txhash := txdetail.GetTx().Hash()
		txdetail.Txhash = txhash
		if txdetail.GetTx().IsWithdraw() {
			//swap from and to
			txdetail.Fromaddr, txdetail.Tx.To = txdetail.Tx.To, txdetail.Fromaddr
		}

		txDetails.TxDetails[index] = &txdetail
	}

	return txDetails, nil
}

func (store *privacyStore) getPrivacyTokenUTXOs(token, addr string) (*walletUTXOs, error) {
	list := store.NewListHelper()
	prefix := calcPrivacyUTXOPrefix4Addr(token, addr)
	values := list.List(prefix, nil, 0, 0)
	wutxos := new(walletUTXOs)
	if len(values) == 0 {
		return wutxos, nil
	}
	for _, value := range values {
		if len(value) == 0 {
			continue
		}
		accByte, err := store.Get(value)
		if err != nil {
			return nil, types.ErrDataBaseDamage
		}
		privacyDBStore := new(privacytypes.PrivacyDBStore)
		err = types.Decode(accByte, privacyDBStore)
		if err != nil {
			bizlog.Error("getPrivacyTokenUTXOs", "decode PrivacyDBStore error. ", err)
			return nil, types.ErrDataBaseDamage
		}
		wutxo := &walletUTXO{
			height: privacyDBStore.Height,
			outinfo: &txOutputInfo{
				amount:           privacyDBStore.Amount,
				txPublicKeyR:     privacyDBStore.TxPublicKeyR,
				onetimePublicKey: privacyDBStore.OnetimePublicKey,
				utxoGlobalIndex: &privacytypes.UTXOGlobalIndex{
					Outindex: privacyDBStore.OutIndex,
					Txhash:   privacyDBStore.Txhash,
				},
			},
		}
		wutxos.utxos = append(wutxos.utxos, wutxo)
	}
	return wutxos, nil
}

//calcUTXOKey4TokenAddr---X--->calcUTXOKey 被删除,该地址下某种token的这个utxo变为不可用
//calcKey4UTXOsSpentInTx------>types.FTXOsSTXOsInOneTx,将当前交易的所有花费的utxo进行打包，设置为ftxo，同时通过支付交易hash索引
//calcKey4FTXOsInTx----------->calcKey4UTXOsSpentInTx,创建该交易冻结的所有的utxo的信息
//状态转移，将utxo转移至ftxo，同时记录该生成tx的花费的utxo，这样在确认执行成功之后就可以快速将相应的FTXO转换成STXO
func (store *privacyStore) moveUTXO2FTXO(tx *types.Transaction, token, sender, txhash string, selectedUtxos []*txOutputInfo) {
	FTXOsInOneTx := &privacytypes.FTXOsSTXOsInOneTx{}
	newbatch := store.NewBatch(true)
	for _, txOutputInfo := range selectedUtxos {
		key := calcUTXOKey4TokenAddr(token, sender, common.Bytes2Hex(txOutputInfo.utxoGlobalIndex.Txhash), int(txOutputInfo.utxoGlobalIndex.Outindex))
		newbatch.Delete(key)
		utxo := &privacytypes.UTXO{
			Amount: txOutputInfo.amount,
			UtxoBasic: &privacytypes.UTXOBasic{
				UtxoGlobalIndex: txOutputInfo.utxoGlobalIndex,
				OnetimePubkey:   txOutputInfo.onetimePublicKey,
			},
		}
		FTXOsInOneTx.Utxos = append(FTXOsInOneTx.Utxos, utxo)
	}
	FTXOsInOneTx.Tokenname = token
	FTXOsInOneTx.Sender = sender
	FTXOsInOneTx.Txhash = txhash
	FTXOsInOneTx.SetExpire(tx)
	//设置在该交易中花费的UTXO
	key1 := calcKey4UTXOsSpentInTx(txhash)
	value1 := types.Encode(FTXOsInOneTx)
	newbatch.Set(key1, value1)

	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key2 := calcKey4FTXOsInTx(token, sender, txhash)
	value2 := key1
	newbatch.Set(key2, value2)

	newbatch.Write()
}

func (store *privacyStore) getRescanUtxosFlag4Addr(req *privacytypes.ReqRescanUtxos) (*privacytypes.RepRescanUtxos, error) {
	var storeAddrs []string
	if len(req.Addrs) == 0 {
		WalletAccStores, err := store.getAccountByPrefix("Account")
		if err != nil || len(WalletAccStores) == 0 {
			bizlog.Info("getRescanUtxosFlag4Addr", "GetAccountByPrefix:err", err)
			return nil, types.ErrNotFound
		}
		for _, WalletAccStore := range WalletAccStores {
			storeAddrs = append(storeAddrs, WalletAccStore.Addr)
		}
	} else {
		storeAddrs = append(storeAddrs, req.Addrs...)
	}

	var repRescanUtxos privacytypes.RepRescanUtxos
	for _, addr := range storeAddrs {
		value, err := store.Get(calcRescanUtxosFlagKey(addr))
		if err != nil {
			continue
			bizlog.Error("getRescanUtxosFlag4Addr", "Failed to get calcRescanUtxosFlagKey(addr) for value", addr)
		}

		var data types.Int64
		err = types.Decode(value, &data)
		if nil != err {
			continue
			bizlog.Error("getRescanUtxosFlag4Addr", "Failed to decode types.Int64 for value", value)
		}
		result := &privacytypes.RepRescanResult{
			Addr: addr,
			Flag: int32(data.Data),
		}
		repRescanUtxos.RepRescanResults = append(repRescanUtxos.RepRescanResults, result)
	}

	if len(repRescanUtxos.RepRescanResults) == 0 {
		return nil, types.ErrNotFound
	}

	repRescanUtxos.Flag = req.Flag

	return &repRescanUtxos, nil
}

func (store *privacyStore) saveREscanUTXOsAddresses(addrs []string) {
	newbatch := store.NewBatch(true)
	for _, addr := range addrs {
		data := &types.Int64{
			Data: int64(privacytypes.UtxoFlagNoScan),
		}
		value := types.Encode(data)
		newbatch.Set(calcRescanUtxosFlagKey(addr), value)
	}
	newbatch.Write()
}

func (store *privacyStore) setScanPrivacyInputUTXO(count int32) []*privacytypes.UTXOGlobalIndex {
	prefix := []byte(ScanPrivacyInput)
	list := store.NewListHelper()
	values := list.List(prefix, nil, count, 0)
	var utxoGlobalIndexs []*privacytypes.UTXOGlobalIndex
	if len(values) != 0 {
		var utxoGlobalIndex privacytypes.UTXOGlobalIndex
		for _, value := range values {
			err := types.Decode(value, &utxoGlobalIndex)
			if err == nil {
				utxoGlobalIndexs = append(utxoGlobalIndexs, &utxoGlobalIndex)
			}
		}
	}
	return utxoGlobalIndexs
}

func (store *privacyStore) isUTXOExist(txhash string, outindex int) (*privacytypes.PrivacyDBStore, error) {
	value1, err := store.Get(calcUTXOKey(txhash, outindex))
	if err != nil {
		bizlog.Error("IsUTXOExist", "Get calcUTXOKey error:", err)
		return nil, err
	}
	var accPrivacy privacytypes.PrivacyDBStore
	err = proto.Unmarshal(value1, &accPrivacy)
	if err != nil {
		bizlog.Error("IsUTXOExist", "proto.Unmarshal err:", err)
		return nil, err
	}
	return &accPrivacy, nil
}

func (store *privacyStore) updateScanInputUTXOs(utxoGlobalIndexs []*privacytypes.UTXOGlobalIndex) {
	if len(utxoGlobalIndexs) <= 0 {
		return
	}
	newbatch := store.NewBatch(true)
	var utxos []*privacytypes.UTXO
	var owner string
	var token string
	var txhash string
	for _, utxoGlobal := range utxoGlobalIndexs {
		accPrivacy, err := store.isUTXOExist(common.Bytes2Hex(utxoGlobal.Txhash), int(utxoGlobal.Outindex))
		if err == nil && accPrivacy != nil {
			utxo := &privacytypes.UTXO{
				Amount: accPrivacy.Amount,
				UtxoBasic: &privacytypes.UTXOBasic{
					UtxoGlobalIndex: utxoGlobal,
					OnetimePubkey:   accPrivacy.OnetimePublicKey,
				},
			}
			utxos = append(utxos, utxo)
			owner = accPrivacy.Owner
			token = accPrivacy.Tokenname
			txhash = common.Bytes2Hex(accPrivacy.Txhash)
		}
		key := calcScanPrivacyInputUTXOKey(common.Bytes2Hex(utxoGlobal.Txhash), int(utxoGlobal.Outindex))
		newbatch.Delete(key)
	}
	if len(utxos) > 0 {
		store.moveUTXO2STXO(owner, token, txhash, utxos, newbatch)
	}
	newbatch.Write()
}

func (store *privacyStore) moveUTXO2STXO(owner, token, txhash string, utxos []*privacytypes.UTXO, newbatch db.Batch) {
	if len(utxos) == 0 {
		return
	}

	FTXOsInOneTx := &privacytypes.FTXOsSTXOsInOneTx{}

	FTXOsInOneTx.Utxos = utxos
	FTXOsInOneTx.Sender = owner
	FTXOsInOneTx.Tokenname = token
	FTXOsInOneTx.Txhash = txhash

	for _, utxo := range utxos {
		Txhash := utxo.UtxoBasic.UtxoGlobalIndex.Txhash
		Outindex := utxo.UtxoBasic.UtxoGlobalIndex.Outindex
		//删除存在的UTXO索引
		key := calcUTXOKey4TokenAddr(token, owner, common.Bytes2Hex(Txhash), int(Outindex))
		newbatch.Delete(key)
	}

	//设置在该交易中花费的UTXO
	key1 := calcKey4UTXOsSpentInTx(txhash)
	value1 := types.Encode(FTXOsInOneTx)
	newbatch.Set(key1, value1)

	//设置花费stxo的key
	key2 := calcKey4STXOsInTx(txhash)
	value2 := key1
	newbatch.Set(key2, value2)

	// 在此处加入stxo-token-addr-txhash 作为key，便于查找花费出去交易
	key3 := calcSTXOTokenAddrTxKey(FTXOsInOneTx.Tokenname, FTXOsInOneTx.Sender, FTXOsInOneTx.Txhash)
	value3 := key1
	newbatch.Set(key3, value3)

	bizlog.Info("moveUTXO2STXO", "tx hash", txhash)
}

func (store *privacyStore) selectPrivacyTransactionToWallet(txDetals *types.TransactionDetails, privacyInfo []addrAndprivacy) {
	newbatch := store.NewBatch(true)
	for _, txdetal := range txDetals.Txs {
		if !bytes.Equal([]byte(privacytypes.PrivacyX), txdetal.Tx.Execer) {
			continue
		}
		store.selectCurrentWalletPrivacyTx(txdetal, int32(txdetal.Index), privacyInfo, newbatch)
	}
	newbatch.Write()
}

func (store *privacyStore) selectCurrentWalletPrivacyTx(txDetal *types.TransactionDetail, index int32, privacyInfo []addrAndprivacy, newbatch db.Batch) {
	tx := txDetal.Tx
	amount, err := tx.Amount()
	if err != nil {
		bizlog.Error("selectCurrentWalletPrivacyTx failed to tx.Amount()")
		return
	}

	txExecRes := txDetal.Receipt.Ty
	height := txDetal.Height

	txhashInbytes := tx.Hash()
	txhash := common.Bytes2Hex(txhashInbytes)
	var privateAction privacytypes.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		bizlog.Error("selectCurrentWalletPrivacyTx failed to decode payload")
		return
	}
	bizlog.Info("selectCurrentWalletPrivacyTx", "tx hash", txhash)
	var RpubKey []byte
	var privacyOutput *privacytypes.PrivacyOutput
	var privacyInput *privacytypes.PrivacyInput
	var tokenname string
	if privacytypes.ActionPublic2Privacy == privateAction.Ty {
		RpubKey = privateAction.GetPublic2Privacy().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPublic2Privacy().GetOutput()
		tokenname = privateAction.GetPublic2Privacy().GetTokenname()
	} else if privacytypes.ActionPrivacy2Privacy == privateAction.Ty {
		RpubKey = privateAction.GetPrivacy2Privacy().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPrivacy2Privacy().GetOutput()
		tokenname = privateAction.GetPrivacy2Privacy().GetTokenname()
		privacyInput = privateAction.GetPrivacy2Privacy().GetInput()
	} else if privacytypes.ActionPrivacy2Public == privateAction.Ty {
		RpubKey = privateAction.GetPrivacy2Public().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPrivacy2Public().GetOutput()
		tokenname = privateAction.GetPrivacy2Public().GetTokenname()
		privacyInput = privateAction.GetPrivacy2Public().GetInput()
	}

	//处理output
	if nil != privacyOutput && len(privacyOutput.Keyoutput) > 0 {
		utxoProcessed := make([]bool, len(privacyOutput.Keyoutput))
		for _, info := range privacyInfo {
			bizlog.Debug("SelectCurrentWalletPrivacyTx", "individual privacyInfo's addr", *info.Addr)
			privacykeyParirs := info.PrivacyKeyPair
			bizlog.Debug("SelectCurrentWalletPrivacyTx", "individual ViewPubkey", common.Bytes2Hex(privacykeyParirs.ViewPubkey.Bytes()),
				"individual SpendPubkey", common.Bytes2Hex(privacykeyParirs.SpendPubkey.Bytes()))

			var utxos []*privacytypes.UTXO
			for indexoutput, output := range privacyOutput.Keyoutput {
				if utxoProcessed[indexoutput] {
					continue
				}
				priv, err := privacy.RecoverOnetimePriKey(RpubKey, privacykeyParirs.ViewPrivKey, privacykeyParirs.SpendPrivKey, int64(indexoutput))
				if err == nil {
					recoverPub := priv.PubKey().Bytes()[:]
					if bytes.Equal(recoverPub, output.Onetimepubkey) {
						//为了避免匹配成功之后不必要的验证计算，需要统计匹配次数
						//因为目前只会往一个隐私账户转账，
						//1.一般情况下，只会匹配一次，如果是往其他钱包账户转账，
						//2.但是如果是往本钱包的其他地址转账，因为可能存在的change，会匹配2次
						utxoProcessed[indexoutput] = true
						bizlog.Debug("SelectCurrentWalletPrivacyTx got privacy tx belong to current wallet",
							"Address", *info.Addr, "tx with hash", txhash, "Amount", amount)
						//只有当该交易执行成功才进行相应的UTXO的处理
						if types.ExecOk == txExecRes {

							// 先判断该UTXO的hash是否存在，不存在则写入
							accPrivacy, err := store.isUTXOExist(common.Bytes2Hex(txhashInbytes), indexoutput)
							if err == nil && accPrivacy != nil {
								continue
							}

							info2store := &privacytypes.PrivacyDBStore{
								Txhash:           txhashInbytes,
								Tokenname:        tokenname,
								Amount:           output.Amount,
								OutIndex:         int32(indexoutput),
								TxPublicKeyR:     RpubKey,
								OnetimePublicKey: output.Onetimepubkey,
								Owner:            *info.Addr,
								Height:           height,
								Txindex:          index,
								//Blockhash:        block.Block.Hash(),
							}

							utxoGlobalIndex := &privacytypes.UTXOGlobalIndex{
								Outindex: int32(indexoutput),
								Txhash:   txhashInbytes,
							}

							utxoCreated := &privacytypes.UTXO{
								Amount: output.Amount,
								UtxoBasic: &privacytypes.UTXOBasic{
									UtxoGlobalIndex: utxoGlobalIndex,
									OnetimePubkey:   output.Onetimepubkey,
								},
							}

							utxos = append(utxos, utxoCreated)
							store.setUTXO(info.Addr, &txhash, indexoutput, info2store, newbatch)
						}
					}
				}
			}
		}
	}

	//处理input
	if nil != privacyInput && len(privacyInput.Keyinput) > 0 {
		var utxoGlobalIndexs []*privacytypes.UTXOGlobalIndex
		for _, input := range privacyInput.Keyinput {
			utxoGlobalIndexs = append(utxoGlobalIndexs, input.UtxoGlobalIndex...)
		}

		if len(utxoGlobalIndexs) > 0 {
			store.storeScanPrivacyInputUTXO(utxoGlobalIndexs, newbatch)
		}
	}
}

// setUTXO 添加UTXO信息
// addr 接收该UTXO的账户地址,表示该地址拥有了UTXO的使用权
// txhash 该UTXO的来源交易输出的交易哈希,没有0x
// outindex 该UTXO的来源交易输出的索引位置
// dbStore 构建的钱包UTXO详细数据信息
//UTXO---->moveUTXO2FTXO---->FTXO---->moveFTXO2STXO---->STXO
//1.calcUTXOKey------------>types.PrivacyDBStore 该kv值在db中的存储一旦写入就不再改变，除非产生该UTXO的交易被撤销
//2.calcUTXOKey4TokenAddr-->calcUTXOKey，创建kv，方便查询现在某个地址下某种token的可用utxo
func (store *privacyStore) setUTXO(addr, txhash *string, outindex int, dbStore *privacytypes.PrivacyDBStore, newbatch db.Batch) error {
	if 0 == len(*addr) || 0 == len(*txhash) {
		bizlog.Error("setUTXO addr or txhash is nil")
		return types.ErrInvalidParam
	}
	if dbStore == nil {
		bizlog.Error("setUTXO privacy is nil")
		return types.ErrInvalidParam
	}

	privacyStorebyte, err := proto.Marshal(dbStore)
	if err != nil {
		bizlog.Error("setUTXO proto.Marshal err!", "err", err)
		return types.ErrMarshal
	}

	utxoKey := calcUTXOKey(*txhash, outindex)
	bizlog.Debug("setUTXO", "addr", *addr, "tx with hash", *txhash, "amount:", dbStore.Amount/types.Coin)
	newbatch.Set(calcUTXOKey4TokenAddr(dbStore.Tokenname, *addr, *txhash, outindex), utxoKey)
	newbatch.Set(utxoKey, privacyStorebyte)
	return nil
}

func (store *privacyStore) storeScanPrivacyInputUTXO(utxoGlobalIndexs []*privacytypes.UTXOGlobalIndex, newbatch db.Batch) {
	for _, utxoGlobalIndex := range utxoGlobalIndexs {
		key1 := calcScanPrivacyInputUTXOKey(common.Bytes2Hex(utxoGlobalIndex.Txhash), int(utxoGlobalIndex.Outindex))
		utxoIndex := &privacytypes.UTXOGlobalIndex{
			Txhash:   utxoGlobalIndex.Txhash,
			Outindex: utxoGlobalIndex.Outindex,
		}
		value1 := types.Encode(utxoIndex)
		newbatch.Set(key1, value1)
	}
}

func (store *privacyStore) listSpendUTXOs(token, addr string) (*privacytypes.UTXOHaveTxHashs, error) {
	if 0 == len(addr) {
		bizlog.Error("listSpendUTXOs addr is nil")
		return nil, types.ErrInvalidParam
	}
	prefix := calcSTXOPrefix4Addr(token, addr)
	list := store.NewListHelper()
	Key4FTXOsInTxs := list.PrefixScan(prefix)
	if len(Key4FTXOsInTxs) == 0 {
		bizlog.Error("listSpendUTXOs ", "addr not exist", addr)
		return nil, types.ErrNotFound
	}

	var utxoHaveTxHashs privacytypes.UTXOHaveTxHashs
	utxoHaveTxHashs.UtxoHaveTxHashs = make([]*privacytypes.UTXOHaveTxHash, 0)
	for _, Key4FTXOsInTx := range Key4FTXOsInTxs {
		value, err := store.Get(Key4FTXOsInTx)
		if err != nil {
			continue
		}
		var ftxosInOneTx privacytypes.FTXOsSTXOsInOneTx
		err = types.Decode(value, &ftxosInOneTx)
		if nil != err {
			bizlog.Error("listSpendUTXOs", "Failed to decode FTXOsSTXOsInOneTx for value", value)
			return nil, types.ErrInvalidParam
		}

		for _, ftxo := range ftxosInOneTx.Utxos {
			utxohash := common.Bytes2Hex(ftxo.UtxoBasic.UtxoGlobalIndex.Txhash)
			value1, err := store.Get(calcUTXOKey(utxohash, int(ftxo.UtxoBasic.UtxoGlobalIndex.Outindex)))
			if err != nil {
				continue
			}
			var accPrivacy privacytypes.PrivacyDBStore
			err = proto.Unmarshal(value1, &accPrivacy)
			if err != nil {
				bizlog.Error("listWalletPrivacyAccount", "proto.Unmarshal err:", err)
				return nil, types.ErrUnmarshal
			}

			utxoBasic := &privacytypes.UTXOBasic{
				UtxoGlobalIndex: &privacytypes.UTXOGlobalIndex{
					Outindex: accPrivacy.OutIndex,
					Txhash:   accPrivacy.Txhash,
				},
				OnetimePubkey: accPrivacy.OnetimePublicKey,
			}

			var utxoHaveTxHash privacytypes.UTXOHaveTxHash
			utxoHaveTxHash.Amount = accPrivacy.Amount
			utxoHaveTxHash.TxHash = ftxosInOneTx.Txhash
			utxoHaveTxHash.UtxoBasic = utxoBasic

			utxoHaveTxHashs.UtxoHaveTxHashs = append(utxoHaveTxHashs.UtxoHaveTxHashs, &utxoHaveTxHash)
		}
	}
	return &utxoHaveTxHashs, nil
}

func (store *privacyStore) getWalletFtxoStxo(prefix string) ([]*privacytypes.FTXOsSTXOsInOneTx, []string, error) {
	list := store.NewListHelper()
	values := list.List([]byte(prefix), nil, 0, 0)
	var Ftxoes []*privacytypes.FTXOsSTXOsInOneTx
	var key []string
	for _, value := range values {
		value1, err := store.Get(value)
		if err != nil {
			continue
		}

		FTXOsInOneTx := &privacytypes.FTXOsSTXOsInOneTx{}
		err = types.Decode(value1, FTXOsInOneTx)
		if nil != err {
			bizlog.Error("DecodeString Error", "Error", err.Error())
			return nil, nil, err
		}

		Ftxoes = append(Ftxoes, FTXOsInOneTx)
		key = append(key, string(value))
	}
	return Ftxoes, key, nil
}

func (store *privacyStore) getFTXOlist() ([]*privacytypes.FTXOsSTXOsInOneTx, [][]byte) {
	curFTXOTxs, _, _ := store.getWalletFtxoStxo(FrozenUTXOs)
	revertFTXOTxs, _, _ := store.getWalletFtxoStxo(RevertSendtx)
	var keys [][]byte
	for _, ftxo := range curFTXOTxs {
		keys = append(keys, calcKey4FTXOsInTx(ftxo.Tokenname, ftxo.Sender, ftxo.Txhash))
	}
	for _, ftxo := range revertFTXOTxs {
		keys = append(keys, calcRevertSendTxKey(ftxo.Tokenname, ftxo.Sender, ftxo.Txhash))
	}
	curFTXOTxs = append(curFTXOTxs, revertFTXOTxs...)
	return curFTXOTxs, keys
}

//calcKey4FTXOsInTx-----x------>calcKey4UTXOsSpentInTx,被删除，
//calcKey4STXOsInTx------------>calcKey4UTXOsSpentInTx
//切换types.FTXOsSTXOsInOneTx的状态
func (store *privacyStore) moveFTXO2STXO(key1 []byte, txhash string, newbatch db.Batch) error {
	//设置在该交易中花费的UTXO
	value1, err := store.Get(key1)
	if err != nil {
		bizlog.Error("moveFTXO2STXO", "Get(key1) error ", err, "key1", string(key1))
		return err
	}
	if value1 == nil {
		bizlog.Error("moveFTXO2STXO", "Get nil value for txhash", txhash)
		return types.ErrNotFound
	}
	newbatch.Delete(key1)

	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key2 := calcKey4STXOsInTx(txhash)
	value2 := value1
	newbatch.Set(key2, value2)

	// 在此处加入stxo-token-addr-txhash 作为key，便于查找花费出去交易
	key := value2
	value, err := store.Get(key)
	if err != nil {
		bizlog.Error("moveFTXO2STXO", "db Get(key) error ", err, "key", key)
	}
	var ftxosInOneTx privacytypes.FTXOsSTXOsInOneTx
	err = types.Decode(value, &ftxosInOneTx)
	if nil != err {
		bizlog.Error("moveFTXO2STXO", "Failed to decode FTXOsSTXOsInOneTx for value", value)
	}
	key3 := calcSTXOTokenAddrTxKey(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, ftxosInOneTx.Txhash)
	newbatch.Set(key3, value2)
	newbatch.Write()

	bizlog.Info("moveFTXO2STXO", "tx hash", txhash)
	return nil
}

//将FTXO重置为UTXO
// moveFTXO2UTXO 当交易因为区块被回退而进行回滚时,需要将交易对应的冻结UTXO移动到可用UTXO队列中
// 由于交易回退可能会因为UTXO对应的交易过期,未被打入区块等情况,导致UTXO不可用,所以需要检查UTXO对应的交易是否有效
func (store *privacyStore) moveFTXO2UTXO(key1 []byte, newbatch db.Batch) {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	value1, err := store.Get(key1)
	if err != nil {
		bizlog.Error("moveFTXO2UTXO", "db Get(key1) error ", err)
		return
	}
	if nil == value1 {
		bizlog.Error("moveFTXO2UTXO", "Get nil value for key", string(key1))
		return

	}
	newbatch.Delete(key1)

	key2 := value1
	value2, err := store.Get(key2)
	if err != nil {
		bizlog.Error("moveFTXO2UTXO", "db Get(key2) error ", err)
		return
	}
	if nil == value2 {
		bizlog.Error("moveFTXO2UTXO", "Get nil value for key", string(key2))
		return
	}
	newbatch.Delete(key2)

	var ftxosInOneTx privacytypes.FTXOsSTXOsInOneTx
	err = types.Decode(value2, &ftxosInOneTx)
	if nil != err {
		bizlog.Error("moveFTXO2UTXO", "Failed to decode FTXOsSTXOsInOneTx for value", value2)
		return
	}
	for _, ftxo := range ftxosInOneTx.Utxos {
		utxohash := common.Bytes2Hex(ftxo.UtxoBasic.UtxoGlobalIndex.Txhash)
		outindex := int(ftxo.UtxoBasic.UtxoGlobalIndex.Outindex)
		key := calcUTXOKey4TokenAddr(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, utxohash, outindex)
		value := calcUTXOKey(utxohash, int(ftxo.UtxoBasic.UtxoGlobalIndex.Outindex))
		bizlog.Debug("moveFTXO2UTXO", "addr", ftxosInOneTx.Sender, "tx with hash", utxohash, "amount", ftxo.Amount/types.Coin)
		newbatch.Set(key, value)
	}
	bizlog.Debug("moveFTXO2UTXO", "addr", ftxosInOneTx.Sender, "tx with hash", ftxosInOneTx.Txhash)
}

// unsetUTXO 当区块发生回退时,交易也需要回退,从而需要更新钱包中的UTXO相关信息,其具体步骤如下
// 1.清除可用UTXO列表中的UTXO索引信息
// 2.清除冻结UTXO列表中的UTXO索引信息
// 3.清除因为回退而将花费UTXO移入到冻结UTXO列表的UTXO索引信息
// 4.清除UTXO信息
// addr 使用UTXO的地址
// txhash 使用UTXO的交易哈希,没有0x
func (store *privacyStore) unsetUTXO(addr, txhash *string, outindex int, token string, newbatch db.Batch) error {
	if 0 == len(*addr) || 0 == len(*txhash) || outindex < 0 || len(token) <= 0 {
		bizlog.Error("unsetUTXO", "InvalidParam addr", *addr, "txhash", *txhash, "outindex", outindex, "token", token)
		return types.ErrInvalidParam
	}
	// 1.删除可用UTXO列表的索引关系
	ftxokey := calcUTXOKey(*txhash, outindex)
	newbatch.Delete(ftxokey)
	// 2.删除冻结UTXO列表的索引关系
	ftxokey = calcKey4FTXOsInTx(token, *addr, *txhash)
	newbatch.Delete(ftxokey)
	// 3.删除回退冻结UTXO列表中的索引关系
	ftxokey = calcRevertSendTxKey(token, *addr, *txhash)
	newbatch.Delete(ftxokey)
	// 4.清除可用UTXO索引信息
	utxokey := calcUTXOKey4TokenAddr(token, *addr, *txhash, outindex)
	newbatch.Delete(utxokey)

	bizlog.Debug("PrivacyTrading unsetUTXO", "addr", *addr, "tx with hash", *txhash, "outindex", outindex)
	return nil
}

//由于块的回退的原因，导致其中的交易需要被回退，即将stxo回退到ftxo，
//正常情况下，被回退的交易会被重新加入到新的区块中并得到执行
func (store *privacyStore) moveSTXO2FTXO(tx *types.Transaction, txhash string, newbatch db.Batch) error {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key2 := calcKey4STXOsInTx(txhash)
	value2, err := store.Get(key2)
	if err != nil {
		bizlog.Error("moveSTXO2FTXO", "Get(key2) error ", err)
		return err
	}
	if value2 == nil {
		bizlog.Debug("moveSTXO2FTXO", "Get nil value for txhash", txhash)
		return types.ErrNotFound
	}
	newbatch.Delete(key2)

	key := value2
	value, err := store.Get(key)
	if err != nil {
		bizlog.Error("moveSTXO2FTXO", "db Get(key) error ", err)
	}

	var ftxosInOneTx privacytypes.FTXOsSTXOsInOneTx
	err = types.Decode(value, &ftxosInOneTx)
	if nil != err {
		bizlog.Error("moveSTXO2FTXO", "Failed to decode FTXOsSTXOsInOneTx for value", value)
	}

	//删除stxo-token-addr-txhash key
	key3 := calcSTXOTokenAddrTxKey(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, ftxosInOneTx.Txhash)
	newbatch.Delete(key3)

	//设置在该交易中花费的UTXO
	key1 := calcRevertSendTxKey(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, txhash)
	value1 := value2
	newbatch.Set(key1, value1)
	bizlog.Info("moveSTXO2FTXO", "txhash ", txhash)

	ftxosInOneTx.SetExpire(tx)
	value = types.Encode(&ftxosInOneTx)
	newbatch.Set(key, value)

	newbatch.Write()
	return nil
}

func (store *privacyStore) moveFTXO2UTXOWhenFTXOExpire(blockheight, blocktime int64) {
	dbbatch := store.NewBatch(true)
	curFTXOTxs, keys := store.getFTXOlist()
	for i, ftxo := range curFTXOTxs {
		if !ftxo.IsExpire(blockheight, blocktime) {
			continue
		}
		store.moveFTXO2UTXO(keys[i], dbbatch)
		bizlog.Debug("moveFTXO2UTXOWhenFTXOExpire", "moveFTXO2UTXO key", string(keys[i]), "ftxo.IsExpire", ftxo.IsExpire(blockheight, blocktime))
	}
	dbbatch.Write()
}
