package privacybiz

import (
	"bytes"
	"fmt"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"

	"github.com/golang/protobuf/proto"
)

const (
	Privacy4Addr     = "Privacy4Addr"
	AvailUTXOs       = "UTXO"
	FrozenUTXOs      = "FTXOs4Tx"
	PrivacySTXO      = "STXO"
	PrivacyTokenMap  = "PrivacyTokenMap"
	STXOs4Tx         = "STXOs4Tx"
	RevertSendtx     = "RevertSendtx"
	RecvPrivacyTx    = "RecvPrivacyTx"
	SendPrivacyTx    = "SendPrivacyTx"
	ScanPrivacyInput = "ScanPrivacyInput"
	ReScanUtxosFlag  = "ReScanUtxosFlag"
)

// calcUTXOKey 计算可用UTXO的健值,为输出交易哈希+输出索引位置
//key and prefix for privacy
//types.PrivacyDBStore的数据存储由calcUTXOKey生成key，
//1.当该utxo的目的地址是钱包管理的其中一个隐私地址时，该key作为value，保存在calcUTXOKey4TokenAddr由生成的key对应的kv中；
//2.当进行支付时，calcUTXOKey4TokenAddr对应的kv被删除，进而由calcPrivacyFUTXOKey生成的key对应kv中，其中平移的只是key，
// 本身的具体数据并不进行重新存储，即将utxo变化为futxo；
//3.当包含该交易的块得到确认时，如果发现输入包含在futxo中，则通过类似的方法，将其key设置到stxo中，
//4.当发生区块链分叉回退时，即发生del block的情况时，同时
// 4.a 当确认其中的输入存在于stxo时，则将其从stxo中转移至ftxo中，
// 4.b 当确认其中的输出存在于utxo或ftxo中时，则将其从utxo或ftxo中同时进行删除，同时删除types.PrivacyDBStore在数据库中的值
// 4.c 当确认其中的输出存在于stxo中时，则发生了异常，正常情况下，花费该笔utxo的交易需要被先回退，进而回退该笔交易，观察此种情况的发生
func calcUTXOKey(txhash string, index int) []byte {
	return []byte(fmt.Sprintf("%s-%s-%d", AvailUTXOs, txhash, index))
}

func calcKey4UTXOsSpentInTx(key string) []byte {
	return []byte(fmt.Sprintf("UTXOsSpentInTx:%s", key))
}

// calcPrivacyAddrKey 获取隐私账户私钥对保存在钱包中的索引串
func calcPrivacyAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s", Privacy4Addr, addr))
}

//calcAddrKey 通过addr地址查询Account账户信息
func calcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("Addr:%s", addr))
}

// calcPrivacyUTXOPrefix4Addr 获取指定地址下可用UTXO信息索引的KEY值前缀
func calcPrivacyUTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s-", AvailUTXOs, token, addr))
}

// calcFTXOsKeyPrefix 获取指定地址下由于交易未被确认而让交易使用到的UTXO处于冻结状态信息的KEY值前缀
func calcFTXOsKeyPrefix(token, addr string) []byte {
	var prefix string
	if len(token) > 0 && len(addr) > 0 {
		prefix = fmt.Sprintf("%s:%s-%s-", FrozenUTXOs, token, addr)
	} else if len(token) > 0 {
		prefix = fmt.Sprintf("%s:%s-", FrozenUTXOs, token)
	} else {
		prefix = fmt.Sprintf("%s:", FrozenUTXOs)
	}
	return []byte(prefix)
}

// calcSendPrivacyTxKey 计算以指定地址作为发送地址的交易信息索引
// addr为发送地址
// key为通过calcTxKey(heightstr)计算出来的值
func calcSendPrivacyTxKey(tokenname, addr, key string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", SendPrivacyTx, tokenname, addr, key))
}

// calcRecvPrivacyTxKey 计算以指定地址作为接收地址的交易信息索引
// addr为接收地址
// key为通过calcTxKey(heightstr)计算出来的值
func calcRecvPrivacyTxKey(tokenname, addr, key string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", RecvPrivacyTx, tokenname, addr, key))
}

// calcUTXOKey4TokenAddr 计算当前地址可用UTXO的Key健值
func calcUTXOKey4TokenAddr(token, addr, txhash string, index int) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s-%s-%d", AvailUTXOs, token, addr, txhash, index))
}

// calcKey4FTXOsInTx 交易构建以后,将可用UTXO冻结的健值
func calcKey4FTXOsInTx(token, addr, txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", FrozenUTXOs, token, addr, txhash))
}

// calcRescanUtxosFlagKey 新账户导入时扫描区块上该地址相关的UTXO信息
func calcRescanUtxosFlagKey(addr string) []byte {
	return []byte(fmt.Sprintf("%s-%s", ReScanUtxosFlag, addr))
}

func calcScanPrivacyInputUTXOKey(txhash string, index int) []byte {
	return []byte(fmt.Sprintf("%s-%s-%d", ScanPrivacyInput, txhash, index))
}

func calcKey4STXOsInTx(txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s", STXOs4Tx, txhash))
}

// calcSTXOTokenAddrTxKey 计算当前地址已花费的UTXO
func calcSTXOTokenAddrTxKey(token, addr, txhash string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s-%s", PrivacySTXO, token, addr, txhash))
}

// privacyStore 隐私交易数据库存储操作类
type privacyStore struct {
	db db.DB
}

func (store *privacyStore) getAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {
	if len(addr) == 0 {
		bizlog.Error("getAccountByPrefix addr is nil")
		return nil, types.ErrInputPara
	}
	list := db.NewListHelper(store.db)
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

func (store *privacyStore) getWalletAccountPrivacy(addr string) (*types.WalletAccountPrivacy, error) {
	if len(addr) == 0 {
		bizlog.Error("GetWalletAccountPrivacy addr is nil")
		return nil, types.ErrInputPara
	}

	privacyByte, err := store.db.Get(calcPrivacyAddrKey(addr))
	if err != nil {
		bizlog.Error("GetWalletAccountPrivacy", "db Get error ", err)
		return nil, err
	}
	if nil == privacyByte {
		return nil, types.ErrPrivacyNotEnabled
	}
	var accPrivacy types.WalletAccountPrivacy
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
		return nil, types.ErrInputPara
	}
	data, err := store.db.Get(calcAddrKey(addr))
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

func (store *privacyStore) setWalletAccountPrivacy(addr string, privacy *types.WalletAccountPrivacy) error {
	if len(addr) == 0 {
		bizlog.Error("SetWalletAccountPrivacy addr is nil")
		return types.ErrInputPara
	}
	if privacy == nil {
		bizlog.Error("SetWalletAccountPrivacy privacy is nil")
		return types.ErrInputPara
	}

	privacybyte, err := proto.Marshal(privacy)
	if err != nil {
		bizlog.Error("SetWalletAccountPrivacy proto.Marshal err!", "err", err)
		return types.ErrMarshal
	}

	newbatch := store.db.NewBatch(true)
	store.db.Set(calcPrivacyAddrKey(addr), privacybyte)
	newbatch.Write()

	return nil
}
func (store *privacyStore) listAvailableUTXOs(token, addr string) ([]*types.PrivacyDBStore, error) {
	if 0 == len(addr) {
		bizlog.Error("listWalletPrivacyAccount addr is nil")
		return nil, types.ErrInputPara
	}

	list := db.NewListHelper(store.db)
	onetimeAccbytes := list.PrefixScan(calcPrivacyUTXOPrefix4Addr(token, addr))
	if len(onetimeAccbytes) == 0 {
		bizlog.Error("listWalletPrivacyAccount ", "addr not exist", addr)
		return nil, nil
	}

	privacyDBStoreSlice := make([]*types.PrivacyDBStore, len(onetimeAccbytes))
	for index, acckeyByte := range onetimeAccbytes {
		var accPrivacy types.PrivacyDBStore
		accByte, err := store.db.Get(acckeyByte)
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

func (store *privacyStore) listFrozenUTXOs(token, addr string) ([]*types.FTXOsSTXOsInOneTx, error) {
	if 0 == len(addr) {
		bizlog.Error("listFrozenUTXOs addr is nil")
		return nil, types.ErrInputPara
	}
	list := db.NewListHelper(store.db)
	values := list.List(calcFTXOsKeyPrefix(token, addr), nil, 0, 0)
	if len(values) == 0 {
		bizlog.Error("listFrozenUTXOs ", "addr not exist", addr)
		return nil, nil
	}

	ftxoslice := make([]*types.FTXOsSTXOsInOneTx, 0)
	for _, acckeyByte := range values {
		var ftxotx types.FTXOsSTXOsInOneTx
		accByte, err := store.db.Get(acckeyByte)
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

func (store *privacyStore) getWalletPrivacyTxDetails(param *types.ReqPrivacyTransactionList) (*types.WalletTxDetails, error) {
	if param == nil {
		bizlog.Error("getWalletPrivacyTxDetails param is nil")
		return nil, types.ErrInvalidParams
	}
	if param.SendRecvFlag != sendTx && param.SendRecvFlag != recvTx {
		bizlog.Error("procPrivacyTransactionList", "invalid sendrecvflag ", param.SendRecvFlag)
		return nil, types.ErrInvalidParams
	}
	var txbytes [][]byte
	list := db.NewListHelper(store.db)
	if len(param.Seedtxhash) == 0 {
		var keyPrefix []byte
		if param.SendRecvFlag == sendTx {
			keyPrefix = calcSendPrivacyTxKey(param.Tokenname, param.Address, "")
		} else {
			keyPrefix = calcRecvPrivacyTxKey(param.Tokenname, param.Address, "")
		}
		txkeybytes := list.IteratorScanFromLast(keyPrefix, param.Count)
		for _, keybyte := range txkeybytes {
			value, err := store.db.Get(keybyte)
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
		list := db.NewListHelper(store.db)
		var txkeybytes [][]byte
		if param.SendRecvFlag == sendTx {
			txkeybytes = list.IteratorScan([]byte(SendPrivacyTx), calcSendPrivacyTxKey(param.Tokenname, param.Address, string(param.Seedtxhash)), param.Count, param.Direction)
		} else {
			txkeybytes = list.IteratorScan([]byte(RecvPrivacyTx), calcRecvPrivacyTxKey(param.Tokenname, param.Address, string(param.Seedtxhash)), param.Count, param.Direction)
		}
		for _, keybyte := range txkeybytes {
			value, err := store.db.Get(keybyte)
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
	list := db.NewListHelper(store.db)
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
		accByte, err := store.db.Get(value)
		if err != nil {
			return nil, types.ErrDataBaseDamage
		}
		privacyDBStore := new(types.PrivacyDBStore)
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
				utxoGlobalIndex: &types.UTXOGlobalIndex{
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
	FTXOsInOneTx := &types.FTXOsSTXOsInOneTx{}
	newbatch := store.db.NewBatch(true)
	for _, txOutputInfo := range selectedUtxos {
		key := calcUTXOKey4TokenAddr(token, sender, common.Bytes2Hex(txOutputInfo.utxoGlobalIndex.Txhash), int(txOutputInfo.utxoGlobalIndex.Outindex))
		newbatch.Delete(key)
		utxo := &types.UTXO{
			Amount: txOutputInfo.amount,
			UtxoBasic: &types.UTXOBasic{
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

func (store *privacyStore) getRescanUtxosFlag4Addr(req *types.ReqRescanUtxos) (*types.RepRescanUtxos, error) {
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

	var repRescanUtxos types.RepRescanUtxos
	for _, addr := range storeAddrs {
		value, err := store.db.Get(calcRescanUtxosFlagKey(addr))
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
		result := &types.RepRescanResult{
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
	newbatch := store.db.NewBatch(true)
	for _, addr := range addrs {
		data := &types.Int64{
			Data: int64(types.UtxoFlagNoScan),
		}
		value := types.Encode(data)
		newbatch.Set(calcRescanUtxosFlagKey(addr), value)
	}
	newbatch.Write()
}

func (store *privacyStore) setScanPrivacyInputUTXO(count int32) []*types.UTXOGlobalIndex {
	prefix := []byte(ScanPrivacyInput)
	list := db.NewListHelper(store.db)
	values := list.List(prefix, nil, count, 0)
	var utxoGlobalIndexs []*types.UTXOGlobalIndex
	if len(values) != 0 {
		for _, value := range values {
			var utxoGlobalIndex types.UTXOGlobalIndex
			err := types.Decode(value, &utxoGlobalIndex)
			if err == nil {
				utxoGlobalIndexs = append(utxoGlobalIndexs, &utxoGlobalIndex)
			}
		}
	}
	return utxoGlobalIndexs
}

func (store *privacyStore) isUTXOExist(txhash string, outindex int) (*types.PrivacyDBStore, error) {
	value1, err := store.db.Get(calcUTXOKey(txhash, outindex))
	if err != nil {
		bizlog.Error("IsUTXOExist", "Get calcUTXOKey error:", err)
		return nil, err
	}
	var accPrivacy types.PrivacyDBStore
	err = proto.Unmarshal(value1, &accPrivacy)
	if err != nil {
		bizlog.Error("IsUTXOExist", "proto.Unmarshal err:", err)
		return nil, err
	}
	return &accPrivacy, nil
}

func (store *privacyStore) updateScanInputUTXOs(utxoGlobalIndexs []*types.UTXOGlobalIndex) {
	if len(utxoGlobalIndexs) <= 0 {
		return
	}
	newbatch := store.db.NewBatch(true)
	var utxos []*types.UTXO
	var owner string
	var token string
	var txhash string
	for _, utxoGlobal := range utxoGlobalIndexs {
		accPrivacy, err := store.isUTXOExist(common.Bytes2Hex(utxoGlobal.Txhash), int(utxoGlobal.Outindex))
		if err == nil && accPrivacy != nil {
			utxo := &types.UTXO{
				Amount: accPrivacy.Amount,
				UtxoBasic: &types.UTXOBasic{
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

func (store *privacyStore) moveUTXO2STXO(owner, token, txhash string, utxos []*types.UTXO, newbatch db.Batch) {
	if len(utxos) == 0 {
		return
	}

	FTXOsInOneTx := &types.FTXOsSTXOsInOneTx{}

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
	newbatch := store.db.NewBatch(true)
	for _, txdetal := range txDetals.Txs {
		if !bytes.Equal(types.ExecerPrivacy, txdetal.Tx.Execer) {
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
	var privateAction types.PrivacyAction
	if err := types.Decode(tx.GetPayload(), &privateAction); err != nil {
		bizlog.Error("selectCurrentWalletPrivacyTx failed to decode payload")
		return
	}
	bizlog.Info("selectCurrentWalletPrivacyTx", "tx hash", txhash)
	var RpubKey []byte
	var privacyOutput *types.PrivacyOutput
	var privacyInput *types.PrivacyInput
	var tokenname string
	if types.ActionPublic2Privacy == privateAction.Ty {
		RpubKey = privateAction.GetPublic2Privacy().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPublic2Privacy().GetOutput()
		tokenname = privateAction.GetPublic2Privacy().GetTokenname()
	} else if types.ActionPrivacy2Privacy == privateAction.Ty {
		RpubKey = privateAction.GetPrivacy2Privacy().GetOutput().GetRpubKeytx()
		privacyOutput = privateAction.GetPrivacy2Privacy().GetOutput()
		tokenname = privateAction.GetPrivacy2Privacy().GetTokenname()
		privacyInput = privateAction.GetPrivacy2Privacy().GetInput()
	} else if types.ActionPrivacy2Public == privateAction.Ty {
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

			var utxos []*types.UTXO
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

							info2store := &types.PrivacyDBStore{
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

							utxoGlobalIndex := &types.UTXOGlobalIndex{
								Outindex: int32(indexoutput),
								Txhash:   txhashInbytes,
							}

							utxoCreated := &types.UTXO{
								Amount: output.Amount,
								UtxoBasic: &types.UTXOBasic{
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
		var utxoGlobalIndexs []*types.UTXOGlobalIndex
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
func (store *privacyStore) setUTXO(addr, txhash *string, outindex int, dbStore *types.PrivacyDBStore, newbatch db.Batch) error {
	if 0 == len(*addr) || 0 == len(*txhash) {
		bizlog.Error("setUTXO addr or txhash is nil")
		return types.ErrInputPara
	}
	if dbStore == nil {
		bizlog.Error("setUTXO privacy is nil")
		return types.ErrInputPara
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

func (store *privacyStore) storeScanPrivacyInputUTXO(utxoGlobalIndexs []*types.UTXOGlobalIndex, newbatch db.Batch) {
	for _, utxoGlobalIndex := range utxoGlobalIndexs {
		key1 := calcScanPrivacyInputUTXOKey(common.Bytes2Hex(utxoGlobalIndex.Txhash), int(utxoGlobalIndex.Outindex))
		utxoIndex := &types.UTXOGlobalIndex{
			Txhash:   utxoGlobalIndex.Txhash,
			Outindex: utxoGlobalIndex.Outindex,
		}
		value1 := types.Encode(utxoIndex)
		newbatch.Set(key1, value1)
	}
}
