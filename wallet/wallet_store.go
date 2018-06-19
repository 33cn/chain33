package wallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	WalletPassKey   = []byte("WalletPassWord")
	WalletFeeAmount = []byte("WalletFeeAmount")
	EncryptionFlag  = []byte("Encryption")
	PasswordHash    = []byte("PasswordHash")
	storelog        = walletlog.New("submodule", "store")
)

const (
	Privacy4Addr = "Privacy4Addr-"
	//PrivacyTokenAddrUTXO = "token-addr-UTXO-"
	PrivacyUTXO     = "UTXO-"
	PrivacyFTXO     = "FTXO-" //Frozen TXO
	PrivacySTXO     = "STXO-"
	PrivacyTokenMap = "PrivacyTokenMap"
	FTXOTimeout     = types.ConfirmedHeight * 16 //Ftxo超时时间
	FTXOs4Tx        = "FTXOs4Tx"
	RecvPrivacyTx   = "RecvPrivacyTx"
	SendPrivacyTx   = "SendPrivacyTx"
)

type Store struct {
	db dbm.DB
}

//用于所有Account账户的输出list，需要安装时间排序
func calcAccountKey(timestamp string, addr string) []byte {
	//timestamp := fmt.Sprintf("%018d", time.Now().Unix())
	return []byte(fmt.Sprintf("Account:%s:%s", timestamp, addr))
}

//通过addr地址查询Account账户信息
func calcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("Addr:%s", addr))
}

//通过label查询Account账户信息
func calcLabelKey(label string) []byte {
	return []byte(fmt.Sprintf("Label:%s", label))
}

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
	return []byte(fmt.Sprintf(PrivacyUTXO+"%s-%d", txhash, index))
}

func calcPrivacyAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf(Privacy4Addr+"%s", addr))
}

func calcUTXOKey4TokenAddr(token, addr, txhash string, index int) []byte {
	return []byte(fmt.Sprintf(PrivacyUTXO+"%s-%s-%s-%d", token, addr, txhash, index))
}

func calcPrivacyUTXOPrefix(token string) []byte {
	return []byte(fmt.Sprintf(PrivacyUTXO+"%s-", token))
}

func calcPrivacyUTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf(PrivacyUTXO+"%s-%s-", token, addr))
}

func calcPrivacy4TokenMap() []byte {
	return []byte(PrivacyTokenMap)
}

//TODO:整理一下，通过类型确定不同类型的key
//func calcSTXOKey(token, addr, txhash string, index int) []byte {
//	return []byte(fmt.Sprintf(PrivacySTXO+"%s-%s-%s-%d", token, addr, txhash, index))
//}
//
//func calcSTXOPrefix(token string) []byte {
//	return []byte(fmt.Sprintf(PrivacySTXO+"%s", token))
//}

func calcSTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf(PrivacySTXO+"%s-%s", token, addr))
}

func calcSTXOTokenAddrTxKey(token, addr, txhash string) []byte {
	return []byte(fmt.Sprintf(PrivacySTXO+"%s-%s-%s", token, addr, txhash))
}

//func calcFTXOKey(token, addr, txhash string, index int) []byte {
//	return []byte(fmt.Sprintf(PrivacyFTXO+"%s-%s-%s-%d", token, addr, txhash, index))
//}
//
//func calcFTXOPrefix(token string) []byte {
//	return []byte(fmt.Sprintf(PrivacyFTXO+"%s", token))
//}
//
//func calcFTXOPrefix4Addr(token, addr string) []byte {
//	return []byte(fmt.Sprintf(PrivacyFTXO+"%s-%s", token, addr))
//}

//通过height*100000+index 查询Tx交易信息
//key:Tx:height*100000+index
func calcTxKey(key string) []byte {
	return []byte(fmt.Sprintf("Tx:%s", key))
}

func calcRecvPrivacyTxKey(addr, key string) []byte {
	return []byte(fmt.Sprintf(RecvPrivacyTx + ":%s-%s", addr, key))
}

func calcSendPrivacyTxKey(addr, key string) []byte {
	return []byte(fmt.Sprintf(SendPrivacyTx + ":%s-%s", addr, key))
}

func calcKey4FTXOsInTx(key string) []byte {
	return []byte(fmt.Sprintf("FTXOs4Tx:%s", key))
}

func calcKey4STXOsInTx(key string) []byte {
	return []byte(fmt.Sprintf("STXOs4Tx:%s", key))
}

func calcKey4UTXOsSpentInTx(key string) []byte {
	return []byte(fmt.Sprintf("UTXOsSpentInTx:%s", key))
}

func NewStore(db dbm.DB) *Store {
	return &Store{
		db: db,
	}
}

func (ws *Store) NewBatch(sync bool) dbm.Batch {
	storeBatch := ws.db.NewBatch(sync)
	return storeBatch
}

func (ws *Store) SetWalletPassword(newpass string) {
	ws.db.SetSync(WalletPassKey, []byte(newpass))
}

func (ws *Store) GetWalletPassword() string {
	Passwordbytes, err := ws.db.Get(WalletPassKey)
	if Passwordbytes == nil || err != nil {
		return ""
	}
	return string(Passwordbytes)
}

func (ws *Store) SetFeeAmount(FeeAmount int64) error {
	FeeAmountbytes, err := json.Marshal(FeeAmount)
	if err != nil {
		walletlog.Error("SetFeeAmount marshal FeeAmount", "err", err)
		return types.ErrMarshal
	}

	ws.db.SetSync(WalletFeeAmount, FeeAmountbytes)
	return nil
}

func (ws *Store) GetFeeAmount() int64 {
	var FeeAmount int64
	FeeAmountbytes, err := ws.db.Get(WalletFeeAmount)
	if FeeAmountbytes == nil || err != nil {
		return minFee
	}
	err = json.Unmarshal(FeeAmountbytes, &FeeAmount)
	if err != nil {
		walletlog.Error("GetFeeAmount unmarshal", "err", err)
		return minFee
	}
	return FeeAmount
}

func (ws *Store) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	if len(addr) == 0 {
		walletlog.Error("SetWalletAccount addr is nil")
		return types.ErrInputPara
	}
	if account == nil {
		walletlog.Error("SetWalletAccount account is nil")
		return types.ErrInputPara
	}

	timestamp := fmt.Sprintf("%018d", time.Now().Unix())
	//更新时需要使用原来的Accountkey
	if update {
		timestamp = account.TimeStamp
	}
	account.TimeStamp = timestamp

	accountbyte, err := proto.Marshal(account)
	if err != nil {
		walletlog.Error("SetWalletAccount proto.Marshal err!", "err", err)
		return types.ErrMarshal
	}

	//需要同时修改三个表，Account，Addr，Label，批量处理
	newbatch := ws.db.NewBatch(true)
	ws.db.Set(calcAccountKey(timestamp, addr), accountbyte)
	ws.db.Set(calcAddrKey(addr), accountbyte)
	ws.db.Set(calcLabelKey(account.GetLabel()), accountbyte)
	newbatch.Write()
	return nil
}

func (ws *Store) GetAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(addr) == 0 {
		walletlog.Error("GetAccountByAddr addr is nil")
		return nil, types.ErrInputPara
	}
	data, err := ws.db.Get(calcAddrKey(addr))
	if data == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			walletlog.Debug("GetAccountByAddr addr", "err", err)
		}
		return nil, types.ErrAddrNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		walletlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

func (ws *Store) GetAccountByLabel(label string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(label) == 0 {
		walletlog.Error("SetWalletAccount label is nil")
		return nil, types.ErrInputPara
	}
	data, err := ws.db.Get(calcLabelKey(label))
	if data == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			walletlog.Error("GetAccountByLabel label", "err", err)
		}
		return nil, types.ErrLabelNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		walletlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

func (ws *Store) GetAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {

	if len(addr) == 0 {
		walletlog.Error("GetAccountByPrefix addr is nil")
		return nil, types.ErrInputPara
	}
	list := dbm.NewListHelper(ws.db)
	accbytes := list.PrefixScan([]byte(addr))
	if len(accbytes) == 0 {
		walletlog.Error("GetAccountByPrefix addr not exist")
		return nil, types.ErrAccountNotExist
	}
	WalletAccountStores := make([]*types.WalletAccountStore, len(accbytes))
	for index, accbyte := range accbytes {
		var walletaccount types.WalletAccountStore
		err := proto.Unmarshal(accbyte, &walletaccount)
		if err != nil {
			walletlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		WalletAccountStores[index] = &walletaccount
	}
	return WalletAccountStores, nil
}

//迭代获取从指定key：height*100000+index 开始向前或者向后查找指定count的交易
func (ws *Store) GetTxDetailByIter(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	var txDetails types.WalletTxDetails
	if TxList == nil {
		walletlog.Error("GetTxDetailByIter TxList is nil")
		return nil, types.ErrInputPara
	}
	if TxList.SendRecvPrivacy != sendTx && TxList.SendRecvPrivacy != recvTx {
		walletlog.Error("GetTxDetailByIter only suport type is sendTx and recvTx")
		return nil, types.ErrInputPara
	}

	var txbytes [][]byte
	//FromTx是空字符串时。默认从最新的交易开始取count个
	if len(TxList.FromTx) == 0 {
		list := dbm.NewListHelper(ws.db)
		if !TxList.Isprivacy {
			txbytes = list.IteratorScanFromLast([]byte(calcTxKey("")), TxList.Count)
		} else {
			var keyPrefix []byte
			if sendTx == TxList.SendRecvPrivacy {
				keyPrefix = calcSendPrivacyTxKey(TxList.Address, "")
			} else {
				keyPrefix = calcRecvPrivacyTxKey(TxList.Address,"")
			}

			txkeybytes := list.IteratorScanFromLast(keyPrefix, TxList.Count)
			for _, keybyte := range txkeybytes {
				value, err := ws.db.Get(keybyte)
				if err != nil {
					continue
				}
				if nil == value {
					continue
				}
				txbytes = append(txbytes, value)
			}
		}

		if len(txbytes) == 0 {
			walletlog.Error("GetTxDetailByIter IteratorScanFromLast does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	} else {
		list := dbm.NewListHelper(ws.db)
		if !TxList.Isprivacy {
			txbytes = list.IteratorScan([]byte("Tx:"), []byte(calcTxKey(string(TxList.FromTx))), TxList.Count, TxList.Direction)
		} else {
			var txkeybytes [][]byte
			if sendTx == TxList.SendRecvPrivacy {
				txkeybytes = list.IteratorScan([]byte(SendPrivacyTx), []byte(calcSendPrivacyTxKey(TxList.Address, string(TxList.FromTx))), TxList.Count, TxList.Direction)
			} else {
				txkeybytes = list.IteratorScan([]byte(RecvPrivacyTx), []byte(calcRecvPrivacyTxKey(TxList.Address, string(TxList.FromTx))), TxList.Count, TxList.Direction)
			}

			for _, keybyte := range txkeybytes {
				value, err := ws.db.Get(keybyte)
				if err != nil {
					continue
				}
				if nil == value {
					continue
				}
				txbytes = append(txbytes, value)
			}
		}

		if len(txbytes) == 0 {
			walletlog.Error("GetTxDetailByIter IteratorScan does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	}

	txDetails.TxDetails = make([]*types.WalletTxDetail, len(txbytes))
	for index, txdetailbyte := range txbytes {
		var txdetail types.WalletTxDetail
		err := proto.Unmarshal(txdetailbyte, &txdetail)
		if err != nil {
			walletlog.Error("GetTxDetailByIter", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		txhash := txdetail.GetTx().Hash()
		txdetail.Txhash = txhash
		if string(txdetail.Tx.GetExecer()) == "coins" && txdetail.Tx.ActionName() == "withdraw" {
			//swap from and to
			txdetail.SenderRecver, txdetail.Tx.To = txdetail.Tx.To, txdetail.SenderRecver
		}

		txDetails.TxDetails[index] = &txdetail
	}

	return &txDetails, nil
}

func (ws *Store) SetEncryptionFlag() error {
	var flag int64 = 1
	data, err := json.Marshal(flag)
	if err != nil {
		walletlog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}

	ws.db.SetSync(EncryptionFlag, data)
	return nil
}

func (ws *Store) GetEncryptionFlag() int64 {
	var flag int64
	data, err := ws.db.Get(EncryptionFlag)
	if data == nil || err != nil {
		return 0
	}
	err = json.Unmarshal(data, &flag)
	if err != nil {
		walletlog.Error("GetEncryptionFlag unmarshal", "err", err)
		return 0
	}
	return flag
}

func (ws *Store) SetPasswordHash(password string) error {
	var WalletPwHash types.WalletPwHash
	//获取一个随机字符串
	randstr := fmt.Sprintf("fuzamei:$@%s", crypto.CRandHex(16))
	WalletPwHash.Randstr = randstr

	//通过password和随机字符串生成一个hash值
	pwhashstr := fmt.Sprintf("%s:%s", password, WalletPwHash.Randstr)
	pwhash := sha256.Sum256([]byte(pwhashstr))
	WalletPwHash.PwHash = pwhash[:]

	pwhashbytes, err := json.Marshal(WalletPwHash)
	if err != nil {
		walletlog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}

	ws.db.SetSync(PasswordHash, pwhashbytes)
	return nil
}

func (ws *Store) VerifyPasswordHash(password string) bool {
	var WalletPwHash types.WalletPwHash
	pwhashbytes, err := ws.db.Get(PasswordHash)
	if pwhashbytes == nil || err != nil {
		return false
	}
	err = json.Unmarshal(pwhashbytes, &WalletPwHash)
	if err != nil {
		walletlog.Error("GetEncryptionFlag unmarshal", "err", err)
		return false
	}
	pwhashstr := fmt.Sprintf("%s:%s", password, WalletPwHash.Randstr)
	pwhash := sha256.Sum256([]byte(pwhashstr))
	Pwhash := pwhash[:]
	//通过新的密码计算pwhash最对比
	return bytes.Equal(WalletPwHash.GetPwHash(), Pwhash)
}

func (ws *Store) DelAccountByLabel(label string) {
	ws.db.DeleteSync(calcLabelKey(label))
}

func (ws *Store) GetWalletFTXO() ([]*types.FTXOsSTXOsInOneTx, []string, error) {
	prefix := FTXOs4Tx
	list := dbm.NewListHelper(ws.db)
	values := list.List([]byte(prefix), nil, 0, 0)
	var Ftxoes []*types.FTXOsSTXOsInOneTx
	var key []string
	for _, value := range values {
		value1, err := ws.db.Get(value)
		if err != nil {
			continue
		}
		FTXOsInOneTx := &types.FTXOsSTXOsInOneTx{}
		err = types.Decode(value1, FTXOsInOneTx)
		if nil != err {
			walletlog.Error("DecodeString Error", "Error", err.Error())
			return nil, nil, err
		}

		Ftxoes = append(Ftxoes, FTXOsInOneTx)
		key = append(key, string(value))
	}
	return Ftxoes, key, nil
}

func (ws *Store) getWalletPrivacyTokenMap() *types.TokenNamesOfUTXO {
	var tokenNamesOfUTXO types.TokenNamesOfUTXO
	value, err := ws.db.Get(calcPrivacy4TokenMap())
	if err != nil {
		walletlog.Error("getWalletPrivacyTokenMap get from db err!", "err", err)
		return &tokenNamesOfUTXO
	}
	if value == nil {
		return &tokenNamesOfUTXO
	}
	err = proto.Unmarshal(value, &tokenNamesOfUTXO)
	if err != nil {
		walletlog.Error("getWalletPrivacyTokenMap proto.Unmarshal err!", "err", err)
		return &tokenNamesOfUTXO
	}

	return &tokenNamesOfUTXO
}

//func (ws *Store) updateWalletPrivacyTokenMap(tokenNames *types.TokenNamesOfUTXO, newbatch dbm.Batch, token, txhash string, addDelType int32) error {
//	if AddTx == addDelType {
//		privacyTokenNames, err := proto.Marshal(tokenNames)
//		if err != nil {
//			walletlog.Error("updateWalletPrivacyTokenMap proto.Marshal err!", "err", err)
//			return types.ErrMarshal
//		}
//		newbatch.Set(calcPrivacy4TokenMap(), privacyTokenNames)
//	} else {
//		value, err := ws.db.Get(calcPrivacy4TokenMap())
//		if err != nil {
//			walletlog.Error("updateWalletPrivacyTokenMap get from db err!", "err", err)
//			return err
//		}
//		var tokenNamesMap types.TokenNamesOfUTXO
//		if err := proto.Unmarshal(value, &tokenNamesMap); err != nil {
//			walletlog.Error("updateWalletPrivacyTokenMap proto.Unmarshal err!", "err", err)
//			return types.ErrUnmarshal
//		}
//
//		if txhashSaved, _ := tokenNamesMap.TokensMap[token]; txhashSaved == txhash {
//			delete(tokenNamesMap.TokensMap, token)
//		}
//
//		privacyTokenNames, err := proto.Marshal(&tokenNamesMap)
//		if err != nil {
//			walletlog.Error("updateWalletPrivacyTokenMap proto.Marshal err!", "err", err)
//			return types.ErrMarshal
//		}
//		newbatch.Set(calcPrivacy4TokenMap(), privacyTokenNames)
//	}
//
//	return nil
//}

//UTXO---->moveUTXO2FTXO---->FTXO---->moveFTXO2STXO---->STXO
//1.calcUTXOKey------------>types.PrivacyDBStore 该kv值在db中的存储一旦写入就不再改变，除非产生该UTXO的交易被撤销
//2.calcUTXOKey4TokenAddr-->calcUTXOKey，创建kv，方便查询现在某个地址下某种token的可用utxo
func (ws *Store) setUTXO(addr, txhash *string, outindex int, dbStore *types.PrivacyDBStore, newbatch dbm.Batch) error {
	if 0 == len(*addr) || 0 == len(*txhash) {
		walletlog.Error("setUTXO addr or txhash is nil")
		return types.ErrInputPara
	}
	if dbStore == nil {
		walletlog.Error("setUTXO privacy is nil")
		return types.ErrInputPara
	}

	privacyStorebyte, err := proto.Marshal(dbStore)
	if err != nil {
		walletlog.Error("setUTXO proto.Marshal err!", "err", err)
		return types.ErrMarshal
	}

	utxoKey := calcUTXOKey(*txhash, outindex)
	walletlog.Debug("setUTXO", "addr", *addr, "tx with hash", *txhash,
		"PrivacyDBStore", *dbStore)
	newbatch.Set(calcUTXOKey4TokenAddr(dbStore.Tokenname, *addr, *txhash, outindex), utxoKey)
	newbatch.Set(utxoKey, privacyStorebyte)
	return nil
}

func (ws *Store) unsetUTXO(addr, txhash *string, outindex int, token string, newbatch dbm.Batch) error {
	if 0 == len(*addr) || 0 == len(*txhash) {
		walletlog.Error("unsetUTXO addr or txhash is nil")
		return types.ErrInputPara
	}

	k1 := calcUTXOKey(*txhash, outindex)
	k2 := calcUTXOKey4TokenAddr(token, *addr, *txhash, outindex)
	if val, err := ws.db.Get(k1); err != nil || val == nil {
		walletlog.Error("unsetUTXO get value for keys are nil",
			"calcUTXOKey", string(k1), "calcUTXOKey4TokenAddr", string(k2))
		return types.ErrNotFound
	}
	newbatch.Delete(k1)
	if val, err := ws.db.Get(k2); err == nil && val != nil {
		newbatch.Delete(k2)
	}

	return nil
}

//calcUTXOKey4TokenAddr---X--->calcUTXOKey 被删除,该地址下某种token的这个utxo变为不可用
//calcKey4UTXOsSpentInTx------>types.FTXOsSTXOsInOneTx,将当前交易的所有花费的utxo进行打包，设置为ftxo，同时通过支付交易hash索引
//calcKey4FTXOsInTx----------->calcKey4UTXOsSpentInTx,创建该交易冻结的所有的utxo的信息
//状态转移，将utxo转移至ftxo，同时记录该生成tx的花费的utxo，这样在确认执行成功之后就可以快速将相应的FTXO转换成STXO
func (ws *Store) moveUTXO2FTXO(token, sender, txhash string, selectedUtxos []*txOutputInfo) {
	FTXOsInOneTx := &types.FTXOsSTXOsInOneTx{}
	newbatch := ws.NewBatch(true)
	for _, txOutputInfo := range selectedUtxos {
		key := calcUTXOKey4TokenAddr(token, sender, common.Bytes2Hex(txOutputInfo.utxoGlobalIndex.Txhash), int(txOutputInfo.utxoGlobalIndex.Outindex))
		newbatch.Delete(key)
		utxo := &types.UTXO{
			Amount:txOutputInfo.amount,
			UtxoBasic: &types.UTXOBasic{
				UtxoGlobalIndex:txOutputInfo.utxoGlobalIndex,
				OnetimePubkey:txOutputInfo.onetimePublicKey,
			},
		}
		FTXOsInOneTx.Utxos = append(FTXOsInOneTx.Utxos, utxo)
	}
	FTXOsInOneTx.Tokenname = token
	FTXOsInOneTx.Sender = sender
	FTXOsInOneTx.TimeoutSec = FTXOTimeout
	FTXOsInOneTx.Txhash = txhash
	//设置在该交易中花费的UTXO
	key1 := calcKey4UTXOsSpentInTx(txhash)
	value1 := types.Encode(FTXOsInOneTx)
	newbatch.Set(key1, value1)

	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key2 := calcKey4FTXOsInTx(txhash)
	value2 := key1
	newbatch.Set(key2, value2)
	newbatch.Write()
}

//将FTXO重置为UTXO
func (ws *Store) unmoveUTXO2FTXO(token, sender, txhash string, newbatch dbm.Batch) {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key1 := calcKey4FTXOsInTx(txhash)
	value1, err := ws.db.Get(key1)
	if err != nil {
		walletlog.Error("unmoveUTXO2FTXO", "db Get(key1) error ", err)
		return
	}
	if nil == value1 {
		walletlog.Error("unmoveUTXO2FTXO", "Get nil value for key", string(key1))
		return
	}
	key2 := value1
	value2, err := ws.db.Get(key2)
	if err != nil {
		walletlog.Error("unmoveUTXO2FTXO", "db Get(key2) error ", err)
		return
	}
	if nil == value2 {
		walletlog.Error("unmoveUTXO2FTXO", "Get nil value for key", string(key2))
		return
	}
	var ftxosInOneTx types.FTXOsSTXOsInOneTx
	err = types.Decode(value2, &ftxosInOneTx)
	if nil != err {
		walletlog.Error("unmoveUTXO2FTXO", "Failed to decode FTXOsSTXOsInOneTx for value", value2)
		return
	}
	if err == nil {
		for _, ftxo := range ftxosInOneTx.Utxos {
			utxohash := common.Bytes2Hex(ftxo.UtxoBasic.UtxoGlobalIndex.Txhash)
			newbatch.Set(calcUTXOKey4TokenAddr(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, utxohash, int(ftxo.UtxoBasic.UtxoGlobalIndex.Outindex)), calcUTXOKey(utxohash, int(ftxo.UtxoBasic.UtxoGlobalIndex.Outindex)))
		}
	}
	// 需要将FTXO的所有相关信息删除掉
	newbatch.Delete(key1)
	newbatch.Delete(key2)
}

func (ws *Store) updateFTXOTimeoutCount(timecount int, txhash string, newbatch dbm.Batch) {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key1 := calcKey4FTXOsInTx(txhash)
	value1, err := ws.db.Get(key1)
	if nil != err {
		walletlog.Error("unmoveUTXO2FTXO", "Get nil value for key", string(key1))
		return
	}
	key2 := value1
	value2, err := ws.db.Get(key2)
	if nil != err {
		walletlog.Error("unmoveUTXO2FTXO", "Get nil value for key", string(key2))
		return
	}
	var ftxosInOneTx types.FTXOsSTXOsInOneTx
	err = types.Decode(value2, &ftxosInOneTx)
	if nil != err {
		walletlog.Error("unmoveUTXO2FTXO", "Failed to decode FTXOsSTXOsInOneTx for value", value2)
		return
	}
	ftxosInOneTx.TimeoutSec -= int32(timecount)
	if ftxosInOneTx.TimeoutSec < 0 {
		ftxosInOneTx.TimeoutSec = 0
	}
	newValue := types.Encode(&ftxosInOneTx)
	newbatch.Set(key2, newValue)

	//newbatch.Write()
}

//calcKey4FTXOsInTx-----x------>calcKey4UTXOsSpentInTx,被删除，
//calcKey4STXOsInTx------------>calcKey4UTXOsSpentInTx
//切换types.FTXOsSTXOsInOneTx的状态
func (ws *Store) moveFTXO2STXO(txhash string, newbatch dbm.Batch) error {
	//设置在该交易中花费的UTXO
	key1 := calcKey4FTXOsInTx(txhash)
	value1, err := ws.db.Get(key1)
	if err != nil {
		walletlog.Error("moveFTXO2STXO", "Get(key1) error ", err)
		return err
	}
	if value1 == nil {
		walletlog.Error("moveFTXO2STXO", "Get nil value for txhash", txhash)
		return types.ErrNotFound
	}
	newbatch.Delete(key1)

	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key2 := calcKey4STXOsInTx(txhash)
	value2 := value1
	newbatch.Set(key2, value2)

	// 在此处加入stxo-token-addr-txhash 作为key，便于查找花费出去交易
	key := value2
	value, err := ws.db.Get(key)
	if err != nil {
		walletlog.Error("moveFTXO2STXO", "db Get(key) error ", err)
	}
	var ftxosInOneTx types.FTXOsSTXOsInOneTx
	err = types.Decode(value, &ftxosInOneTx)
	if nil != err {
		walletlog.Error("moveFTXO2STXO", "Failed to decode FTXOsSTXOsInOneTx for value", value)
	}
	key3 := calcSTXOTokenAddrTxKey(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, ftxosInOneTx.Txhash)
	newbatch.Set(key3, value2)

	newbatch.Write()

	return nil
}

func (ws *Store) unmoveFTXO2STXO(txhash string, newbatch dbm.Batch) error {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key2 := calcKey4STXOsInTx(txhash)
	value2, err := ws.db.Get(key2)
	if err != nil {
		walletlog.Error("unmoveFTXO2STXO", "Get(key2) error ", err)
		return err
	}
	if value2 == nil {
		walletlog.Debug("unmoveFTXO2STXO", "Get nil value for txhash", txhash)
		return types.ErrNotFound
	}
	newbatch.Delete(key2)

	//删除stxo-token-addr-txhash key
	key := value2
	value, err := ws.db.Get(key)
	if err != nil {
		walletlog.Error("unmoveFTXO2STXO", "db Get(key) error ", err)
	}

	var ftxosInOneTx types.FTXOsSTXOsInOneTx
	err = types.Decode(value, &ftxosInOneTx)
	if nil != err {
		walletlog.Error("unmoveFTXO2STXO", "Failed to decode FTXOsSTXOsInOneTx for value", value)
	}

	key3 := calcSTXOTokenAddrTxKey(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, ftxosInOneTx.Txhash)
	newbatch.Delete(key3)

	//设置在该交易中花费的UTXO
	key1 := calcKey4FTXOsInTx(txhash)
	value1 := value2
	newbatch.Set(key1, value1)

	newbatch.Write()

	return nil
}

func (ws *Store) getPrivacyTokenUTXOs(token, addr string) *walletUTXOs {
	prefix := calcPrivacyUTXOPrefix4Addr(token, addr)
	list := dbm.NewListHelper(ws.db)
	values := list.List(prefix, nil, 0, 0)
	if len(values) != 0 {
		outs4token := &walletUTXOs{}
		for _, value := range values {
			var privacyDBStore types.PrivacyDBStore
			accByte, err := ws.db.Get(value)
			if err != nil {
				panic(err)
			}
			err = types.Decode(accByte, &privacyDBStore)
			if err == nil {
				utxoGlobalIndex := &types.UTXOGlobalIndex{
					Height:   privacyDBStore.Height,
					Txindex:  privacyDBStore.Txindex,
					Outindex: privacyDBStore.OutIndex,
					Txhash:   privacyDBStore.Txhash,
				}
				txOutputInfo := &txOutputInfo{
					amount:           privacyDBStore.Amount,
					utxoGlobalIndex:  utxoGlobalIndex,
					txPublicKeyR:     privacyDBStore.TxPublicKeyR,
					onetimePublicKey: privacyDBStore.OnetimePublicKey,
				}

				outs4token.outs = append(outs4token.outs, txOutputInfo)
			} else {
				walletlog.Error("Failed to decode PrivacyDBStore for getWalletPrivacyTokenUTXOs")
			}
		}
		return outs4token
	}

	return nil
}

func (ws *Store) listAvailableUTXOs(token, addr string) ([]*types.PrivacyDBStore, error) {
	if 0 == len(addr) {
		walletlog.Error("listWalletPrivacyAccount addr is nil")
		return nil, types.ErrInputPara
	}

	list := dbm.NewListHelper(ws.db)
	onetimeAccbytes := list.PrefixScan(calcPrivacyUTXOPrefix4Addr(token, addr))
	if len(onetimeAccbytes) == 0 {
		walletlog.Error("listWalletPrivacyAccount ", "addr not exist", addr)
		return nil, nil
	}

	privacyDBStoreSlice := make([]*types.PrivacyDBStore, len(onetimeAccbytes))
	for index, acckeyByte := range onetimeAccbytes {
		var accPrivacy types.PrivacyDBStore
		accByte, err := ws.db.Get(acckeyByte)
		if err != nil {
			walletlog.Error("listWalletPrivacyAccount", "db Get err:", err)
			return nil, err
		}
		err = proto.Unmarshal(accByte, &accPrivacy)
		if err != nil {
			walletlog.Error("listWalletPrivacyAccount", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		privacyDBStoreSlice[index] = &accPrivacy
	}
	return privacyDBStoreSlice, nil
}

func (ws *Store) listSpendUTXOs(token, addr string) (*types.UTXOHaveTxHashs, error) {
	if 0 == len(addr) {
		walletlog.Error("listWalletPrivacyAccount addr is nil")
		return nil, types.ErrInputPara
	}
	prefix := calcSTXOPrefix4Addr(token, addr)
	list := dbm.NewListHelper(ws.db)
	Key4FTXOsInTxs := list.PrefixScan(prefix)
	if len(Key4FTXOsInTxs) == 0 {
		walletlog.Error("listWalletSpendUTXOsPrivacyAccount ", "addr not exist", addr)
		return nil, nil
	}

	var utxoHaveTxHashs types.UTXOHaveTxHashs
	for _, Key4FTXOsInTx := range Key4FTXOsInTxs {
		value, err := ws.db.Get(Key4FTXOsInTx)
		if err != nil {
			continue
		}
		var ftxosInOneTx types.FTXOsSTXOsInOneTx
		err = types.Decode(value, &ftxosInOneTx)
		if nil != err {
			walletlog.Error("listSpendUTXOs", "Failed to decode FTXOsSTXOsInOneTx for value", value)
			return nil, types.ErrInputPara
		}

		for _, ftxo := range ftxosInOneTx.Utxos {
			utxohash := common.Bytes2Hex(ftxo.UtxoBasic.UtxoGlobalIndex.Txhash)
			value1, err := ws.db.Get(calcUTXOKey(utxohash, int(ftxo.UtxoBasic.UtxoGlobalIndex.Outindex)))
			if err != nil {
				continue
			}
			var accPrivacy types.PrivacyDBStore
			err = proto.Unmarshal(value1, &accPrivacy)
			if err != nil {
				walletlog.Error("listWalletPrivacyAccount", "proto.Unmarshal err:", err)
				return nil, types.ErrUnmarshal
			}

			utxoBasic := &types.UTXOBasic{
				UtxoGlobalIndex: &types.UTXOGlobalIndex{
					Height:   accPrivacy.Height,
					Txindex:  accPrivacy.Txindex,
					Outindex: accPrivacy.OutIndex,
					Txhash:   accPrivacy.Txhash,
				},
				OnetimePubkey: accPrivacy.OnetimePublicKey,
			}

			var utxoHaveTxHash types.UTXOHaveTxHash
			utxoHaveTxHash.Amount = accPrivacy.Amount
			utxoHaveTxHash.TxHash = ftxosInOneTx.Txhash
			utxoHaveTxHash.UtxoBasic = utxoBasic

			utxoHaveTxHashs.UtxoHaveTxHashs = append(utxoHaveTxHashs.UtxoHaveTxHashs, &utxoHaveTxHash)
		}
	}
	return &utxoHaveTxHashs, nil
}
func (ws *Store) SetWalletAccountPrivacy(addr string, privacy *types.WalletAccountPrivacy) error {
	if len(addr) == 0 {
		walletlog.Error("SetWalletAccountPrivacy addr is nil")
		return types.ErrInputPara
	}
	if privacy == nil {
		walletlog.Error("SetWalletAccountPrivacy privacy is nil")
		return types.ErrInputPara
	}

	privacybyte, err := proto.Marshal(privacy)
	if err != nil {
		walletlog.Error("SetWalletAccountPrivacy proto.Marshal err!", "err", err)
		return types.ErrMarshal
	}

	newbatch := ws.db.NewBatch(true)
	ws.db.Set(calcPrivacyAddrKey(addr), privacybyte)
	newbatch.Write()

	return nil
}

func (ws *Store) GetWalletAccountPrivacy(addr string) (*types.WalletAccountPrivacy, error) {
	if len(addr) == 0 {
		walletlog.Error("GetWalletAccountPrivacy addr is nil")
		return nil, types.ErrInputPara
	}

	privacyByte, err := ws.db.Get(calcPrivacyAddrKey(addr))
	if err != nil {
		walletlog.Error("GetWalletAccountPrivacy", "db Get error ", err)
		return nil, err
	}
	if nil == privacyByte {
		return nil, types.ErrPrivacyNotExist
	}
	var accPrivacy types.WalletAccountPrivacy
	err = proto.Unmarshal(privacyByte, &accPrivacy)
	if err != nil {
		walletlog.Error("GetWalletAccountPrivacy", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &accPrivacy, nil
}
