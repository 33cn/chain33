package wallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"time"
)

var (
	WalletPassKey   = []byte("WalletPassWord")
	WalletFeeAmount = []byte("WalletFeeAmount")
	EncryptionFlag  = []byte("Encryption")
	PasswordHash    = []byte("PasswordHash")
	WalletVerKey    = []byte("WalletVerKey")
	storelog        = walletlog.New("submodule", "store")
)

const (
	Privacy4Addr       = "Privacy4Addr-"
	PrivacyUTXO        = "UTXO-"
	PrivacySTXO        = "STXO-"
	PrivacyTokenMap    = "PrivacyTokenMap"
	FTXOTimeout        = types.ConfirmedHeight * types.BlockDurPerSecCnt //Ftxo超时时间
	FTXOTimeout4Revert = 256 * types.BlockDurPerSecCnt                   //revert Ftxo超时时间
	FTXOs4Tx           = "FTXOs4Tx"
	STXOs4Tx           = "STXOs4Tx"
	RevertSendtx       = "RevertSendtx"
	RecvPrivacyTx      = "RecvPrivacyTx"
	SendPrivacyTx      = "SendPrivacyTx"
	createTxPrefix     = "CreateTx"
)

type Store struct {
	db dbm.DB
}

//用于所有Account账户的输出list，需要安装时间排序
func calcAccountKey(timestamp string, addr string) []byte {
	//timestamp := fmt.Sprintf("%018d", types.Now().Unix())
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

func calcPrivacyUTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf(PrivacyUTXO+"%s-%s-", token, addr))
}

func calcPrivacy4TokenMap() []byte {
	return []byte(PrivacyTokenMap)
}

func calcSTXOPrefix4Addr(token, addr string) []byte {
	return []byte(fmt.Sprintf(PrivacySTXO+"%s-%s", token, addr))
}

func calcSTXOTokenAddrTxKey(token, addr, txhash string) []byte {
	return []byte(fmt.Sprintf(PrivacySTXO+"%s-%s-%s", token, addr, txhash))
}

//通过height*100000+index 查询Tx交易信息
//key:Tx:height*100000+index
func calcTxKey(key string) []byte {
	return []byte(fmt.Sprintf("Tx:%s", key))
}

// calcRecvPrivacyTxKey 计算以指定地址作为接收地址的交易信息索引
// addr为接收地址
// key为通过calcTxKey(heightstr)计算出来的值
func calcRecvPrivacyTxKey(tokenname, addr, key string) []byte {
	return []byte(fmt.Sprintf(RecvPrivacyTx+":%s-%s-%s", tokenname, addr, key))
}

// calcSendPrivacyTxKey 计算以指定地址作为发送地址的交易信息索引
// addr为发送地址
// key为通过calcTxKey(heightstr)计算出来的值
func calcSendPrivacyTxKey(tokenname, addr, key string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", SendPrivacyTx, tokenname, addr, key))
}

func calcKey4FTXOsInTx(token, addr, txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", FTXOs4Tx, token, addr, txhash))
}

func calcRevertSendTxKey(tokenname, addr, txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s-%s", RevertSendtx, tokenname, addr, txhash))
}

func calcFTXOsKeyPrefix(token, addr string) []byte {
	var prefix string
	if len(token) > 0 && len(addr) > 0 {
		prefix = fmt.Sprintf("%s:%s-%s-", FTXOs4Tx, token, addr)
	} else if len(token) > 0 {
		prefix = fmt.Sprintf("%s:%s-", FTXOs4Tx, token)
	} else {
		prefix = fmt.Sprintf("%s:", FTXOs4Tx)
	}
	return []byte(prefix)
}

func calcKey4STXOsInTx(txhash string) []byte {
	return []byte(fmt.Sprintf(STXOs4Tx+":%s", txhash))
}

func calcKey4UTXOsSpentInTx(key string) []byte {
	return []byte(fmt.Sprintf("UTXOsSpentInTx:%s", key))
}

// 保存协助前端创建交易的Key
// txhash：为common.ToHex()后的字符串
func calcCreateTxKey(token, txhash string) []byte {
	return []byte(fmt.Sprintf("%s:%s-%s", createTxPrefix, token, txhash))
}

func calcCreateTxKeyPrefix(token string) []byte {
	if len(token) > 0 {
		return []byte(fmt.Sprintf("%s:%s-", createTxPrefix, token))
	}
	return []byte(fmt.Sprintf("%s:", createTxPrefix))
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

func (ws *Store) GetAccountByte(update bool, addr string, account *types.WalletAccountStore) ([]byte, error) {
	if len(addr) == 0 {
		walletlog.Error("SetWalletAccount addr is nil")
		return nil, types.ErrInputPara
	}
	if account == nil {
		walletlog.Error("SetWalletAccount account is nil")
		return nil, types.ErrInputPara
	}

	timestamp := fmt.Sprintf("%018d", types.Now().Unix())
	//更新时需要使用原来的Accountkey
	if update {
		timestamp = account.TimeStamp
	}
	account.TimeStamp = timestamp

	accountbyte, err := proto.Marshal(account)
	if err != nil {
		walletlog.Error("SetWalletAccount proto.Marshal err!", "err", err)
		return nil, types.ErrMarshal
	}
	return accountbyte, nil
}

func (ws *Store) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	accountbyte, err := ws.GetAccountByte(update, addr, account)
	if err != nil {
		return err
	}
	//需要同时修改三个表，Account，Addr，Label，批量处理
	newbatch := ws.db.NewBatch(true)
	newbatch.Set(calcAccountKey(account.TimeStamp, addr), accountbyte)
	newbatch.Set(calcAddrKey(addr), accountbyte)
	newbatch.Set(calcLabelKey(account.GetLabel()), accountbyte)
	newbatch.Write()
	return nil
}

func (ws *Store) SetWalletAccountInBatch(update bool, addr string, account *types.WalletAccountStore, newbatch dbm.Batch) error {
	accountbyte, err := ws.GetAccountByte(update, addr, account)
	if err != nil {
		return err
	}
	//需要同时修改三个表，Account，Addr，Label，批量处理
	newbatch.Set(calcAccountKey(account.TimeStamp, addr), accountbyte)
	newbatch.Set(calcAddrKey(addr), accountbyte)
	newbatch.Set(calcLabelKey(account.GetLabel()), accountbyte)
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
func (ws *Store) getTxDetailByIter(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
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
		if TxList.GetMode() == walletQueryModeNormal {
			txbytes = list.IteratorScanFromLast([]byte(calcTxKey("")), TxList.Count)
		} else if TxList.GetMode() == walletQueryModePrivacy {
			var keyPrefix []byte
			if sendTx == TxList.SendRecvPrivacy {
				keyPrefix = calcSendPrivacyTxKey(TxList.Tokenname, TxList.Address, "")
			} else {
				keyPrefix = calcRecvPrivacyTxKey(TxList.Tokenname, TxList.Address, "")
			}

			txkeybytes := list.IteratorScanFromLast(keyPrefix, TxList.Count)
			for _, keybyte := range txkeybytes {
				value, err := ws.db.Get(keybyte)
				if err != nil {
					walletlog.Error(fmt.Sprintf("GetTxDetailByIter() db.Get() error. ", err))
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
		if TxList.GetMode() == walletQueryModeNormal {
			txbytes = list.IteratorScan([]byte("Tx:"), []byte(calcTxKey(string(TxList.FromTx))), TxList.Count, TxList.Direction)
		} else if TxList.GetMode() == walletQueryModePrivacy {
			var txkeybytes [][]byte
			if sendTx == TxList.SendRecvPrivacy {
				txkeybytes = list.IteratorScan([]byte(SendPrivacyTx), []byte(calcSendPrivacyTxKey(TxList.Tokenname, TxList.Address, string(TxList.FromTx))), TxList.Count, TxList.Direction)
			} else {
				txkeybytes = list.IteratorScan([]byte(RecvPrivacyTx), []byte(calcRecvPrivacyTxKey(TxList.Tokenname, TxList.Address, string(TxList.FromTx))), TxList.Count, TxList.Direction)
			}

			for _, keybyte := range txkeybytes {
				value, err := ws.db.Get(keybyte)
				if err != nil {
					walletlog.Error(fmt.Sprintf("GetTxDetailByIter() db.Get() error. ", err))
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
		if txdetail.GetTx().IsWithdraw() {
			//swap from and to
			txdetail.Fromaddr, txdetail.Tx.To = txdetail.Tx.To, txdetail.Fromaddr
		}

		txDetails.TxDetails[index] = &txdetail
	}

	return &txDetails, nil
}

func (ws *Store) SetEncryptionFlag(batch dbm.Batch) error {
	var flag int64 = 1
	data, err := json.Marshal(flag)
	if err != nil {
		walletlog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}

	batch.Set(EncryptionFlag, data)
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

func (ws *Store) SetPasswordHash(password string, batch dbm.Batch) error {
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
	batch.Set(PasswordHash, pwhashbytes)
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

func (ws *Store) GetWalletFtxoStxo(prefix string) ([]*types.FTXOsSTXOsInOneTx, []string, error) {
	//prefix := FTXOs4Tx
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

func (ws *Store) updateFTXOFreezeTime(freezetime int64, token, sender, txhash string, newbatch dbm.Batch) error {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key1 := calcKey4FTXOsInTx(token, sender, txhash)
	value1, err := ws.db.Get(key1)
	if nil != err {
		walletlog.Error("unmoveUTXO2FTXO", "Get nil value for key", string(key1))
		return err
	}
	key2 := value1
	value2, err := ws.db.Get(key2)
	if nil != err {
		walletlog.Error("unmoveUTXO2FTXO", "Get nil value for key", string(key2))
		return nil
	}
	var ftxosInOneTx types.FTXOsSTXOsInOneTx
	err = types.Decode(value2, &ftxosInOneTx)
	if nil != err {
		walletlog.Error("unmoveUTXO2FTXO", "Failed to decode FTXOsSTXOsInOneTx for value", value2)
		return err
	}
	ftxosInOneTx.Freezetime = freezetime
	newValue := types.Encode(&ftxosInOneTx)
	newbatch.Set(key2, newValue)
	return nil
}

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

	//如果该交易产生的UTXO是包含在之前被回退对外支付的交易，则不重新添加相应的UTXO
	if revertFtos, _, _ := ws.GetWalletFtxoStxo(RevertSendtx); nil != revertFtos {
		for _, ftxos4tx := range revertFtos {
			for _, ftxo := range ftxos4tx.Utxos {
				if common.Bytes2Hex(ftxo.UtxoBasic.UtxoGlobalIndex.Txhash) == *txhash {
					walletlog.Info("setUTXO for reverted tx", "txHash", *txhash, "outindex", outindex)
					return nil
				}
			}
		}
	}

	if Ftos, _, _ := ws.GetWalletFtxoStxo(FTXOs4Tx); nil != Ftos {
		for _, ftxos4tx := range Ftos {
			for _, ftxo := range ftxos4tx.Utxos {
				if common.Bytes2Hex(ftxo.UtxoBasic.UtxoGlobalIndex.Txhash) == *txhash {
					walletlog.Info("setUTXO for FTXOs4Tx tx", "txHash", *txhash, "outindex", outindex)
					return nil
				}
			}
		}
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

//当发生交易退回时回退产生的UTXO
//1.如果该utxo未被冻结，即未转移至FTXO，则直接删除就可以
//2.如果该utxo已经被转移至FTXO，则需要先将其所在交易的所有FTXO都回退至UTXO，然后再进行删除
//第2种情况下，因为只是交易被回退，但是交易还是会被重新打包执行，这时需要将该交易进行记录，
//等到回退的交易再次执行时，需要能够从FTXO转换至STXO
func (ws *Store) unsetUTXO(addr, txhash *string, outindex int, token string, newbatch dbm.Batch) error {
	if 0 == len(*addr) || 0 == len(*txhash) {
		walletlog.Error("unsetUTXO addr or txhash is nil")
		return types.ErrInputPara
	}

	found := false
	ftxos4Txs, _, _ := ws.GetWalletFtxoStxo(RevertSendtx)
	for _, ftxos4Singletx := range ftxos4Txs {
		for _, utxo := range ftxos4Singletx.Utxos {
			if common.Bytes2Hex(utxo.UtxoBasic.UtxoGlobalIndex.Txhash) == *txhash {
				found = true
				break
			}
		}
		if found {
			return nil
		}
	}

	k1 := calcUTXOKey(*txhash, outindex)
	if val, err := ws.db.Get(k1); err != nil || val == nil {
		walletlog.Error("unsetUTXO get value for keys are nil", "calcUTXOKey", string(k1))
		return types.ErrNotFound
	}
	newbatch.Delete(k1)

	k2 := calcUTXOKey4TokenAddr(token, *addr, *txhash, outindex)
	val, err := ws.db.Get(k2)
	if err != nil || val == nil {
		//当发生需要回退的UTXO不能从当前特定地址可用的UTXO找到时，
		//需要从当前的FTXO从查找并回退
		walletlog.Error("unsetUTXO get nil value", "calcUTXOKey4TokenAddr", string(k2))
		return types.ErrRecoverUTXO

	}
	newbatch.Delete(k2)

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
	FTXOsInOneTx.Freezetime = time.Now().UnixNano()
	FTXOsInOneTx.Txhash = txhash
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

//将FTXO重置为UTXO
func (ws *Store) moveFTXO2UTXO(key1 []byte, newbatch dbm.Batch) {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	value1, err := ws.db.Get(key1)
	if err != nil {
		walletlog.Error("moveFTXO2UTXO", "db Get(key1) error ", err)
		return
	}
	if nil == value1 {
		walletlog.Error("moveFTXO2UTXO", "Get nil value for key", string(key1))
		return

	}
	key2 := value1
	value2, err := ws.db.Get(key2)
	if err != nil {

		walletlog.Error("moveFTXO2UTXO", "db Get(key2) error ", err)
		return
	}
	if nil == value2 {
		walletlog.Error("moveFTXO2UTXO", "Get nil value for key", string(key2))
		return
	}
	var ftxosInOneTx types.FTXOsSTXOsInOneTx
	err = types.Decode(value2, &ftxosInOneTx)
	if nil != err {

		walletlog.Error("moveFTXO2UTXO", "Failed to decode FTXOsSTXOsInOneTx for value", value2)
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

//calcKey4FTXOsInTx-----x------>calcKey4UTXOsSpentInTx,被删除，
//calcKey4STXOsInTx------------>calcKey4UTXOsSpentInTx
//切换types.FTXOsSTXOsInOneTx的状态
func (ws *Store) moveFTXO2STXO(key1 []byte, txhash string, newbatch dbm.Batch) error {
	//设置在该交易中花费的UTXO
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

	walletlog.Info("moveFTXO2STXO", "tx hash", txhash)

	return nil
}

//由于块的回退的原因，导致其中的交易需要被回退，即将stxo回退到ftxo，
//正常情况下，被回退的交易会被重新加入到新的区块中并得到执行
func (ws *Store) moveSTXO2FTXO(txhash string, newbatch dbm.Batch) error {
	//设置ftxo的key，使其能够方便地获取到对应的交易花费的utxo
	key2 := calcKey4STXOsInTx(txhash)
	value2, err := ws.db.Get(key2)
	if err != nil {
		walletlog.Error("moveSTXO2FTXO", "Get(key2) error ", err)
		return err
	}
	if value2 == nil {
		walletlog.Debug("moveSTXO2FTXO", "Get nil value for txhash", txhash)
		return types.ErrNotFound
	}
	newbatch.Delete(key2)

	key := value2
	value, err := ws.db.Get(key)
	if err != nil {
		walletlog.Error("moveSTXO2FTXO", "db Get(key) error ", err)
	}

	var ftxosInOneTx types.FTXOsSTXOsInOneTx
	err = types.Decode(value, &ftxosInOneTx)
	if nil != err {
		walletlog.Error("moveSTXO2FTXO", "Failed to decode FTXOsSTXOsInOneTx for value", value)
	}

	//删除stxo-token-addr-txhash key
	key3 := calcSTXOTokenAddrTxKey(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, ftxosInOneTx.Txhash)
	newbatch.Delete(key3)

	//设置在该交易中花费的UTXO
	key1 := calcRevertSendTxKey(ftxosInOneTx.Tokenname, ftxosInOneTx.Sender, txhash)
	value1 := value2
	newbatch.Set(key1, value1)
	walletlog.Info("moveSTXO2FTXO", "txhash ", txhash)

	//更新数据库中的超时时间设置，回退处理的超时处理时间设置为256区块时间
	ftxosInOneTx.Freezetime = time.Now().UnixNano()
	value = types.Encode(&ftxosInOneTx)
	newbatch.Set(key, value)

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

func (ws *Store) listFrozenUTXOs(token, addr string) ([]*types.FTXOsSTXOsInOneTx, error) {
	if 0 == len(addr) {
		walletlog.Error("listFrozenUTXOs addr is nil")
		return nil, types.ErrInputPara
	}
	list := dbm.NewListHelper(ws.db)
	values := list.List(calcFTXOsKeyPrefix(token, addr), nil, 0, 0)
	if len(values) == 0 {
		walletlog.Error("listFrozenUTXOs ", "addr not exist", addr)
		return nil, nil
	}

	ftxoslice := make([]*types.FTXOsSTXOsInOneTx, 0)
	for _, acckeyByte := range values {
		var ftxotx types.FTXOsSTXOsInOneTx
		accByte, err := ws.db.Get(acckeyByte)
		if err != nil {
			walletlog.Error("listFrozenUTXOs", "db Get err:", err)
			return nil, err
		}

		err = proto.Unmarshal(accByte, &ftxotx)
		if err != nil {
			walletlog.Error("listFrozenUTXOs", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		ftxoslice = append(ftxoslice, &ftxotx)
	}
	return ftxoslice, nil

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
		return nil, types.ErrNotFound
	}

	var utxoHaveTxHashs types.UTXOHaveTxHashs
	utxoHaveTxHashs.UtxoHaveTxHashs = make([]*types.UTXOHaveTxHash, 0)
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

func (ws *Store) SetCreateTransactionCache(key []byte, cache *types.CreateTransactionCache) error {
	if len(key) <= 0 || cache == nil {
		return types.ErrInvalidParam
	}

	return ws.db.Set(key, types.Encode(cache))
}

func (ws *Store) GetCreateTransactionCache(key []byte) (*types.CreateTransactionCache, error) {
	if len(key) <= 0 {
		return nil, types.ErrInvalidParam
	}

	data, err := ws.db.Get(key)
	if err != nil {
		walletlog.Error("GetCreateTransactionCache", "db.Get err:", err)
		return nil, err
	}
	cache := types.CreateTransactionCache{}
	err = proto.Unmarshal(data, &cache)
	if err != nil {
		walletlog.Error("GetCreateTransactionCache", "proto.Unmarshal err:", err)
		return nil, err
	}
	return &cache, nil
}

func (ws *Store) DeleteCreateTransactionCache(key []byte) {
	if len(key) <= 0 {
		return
	}
	ws.db.Delete(key)
}

func (ws *Store) listCreateTransactionCache(token string) ([]*types.CreateTransactionCache, error) {
	prefix := calcCreateTxKeyPrefix(token)
	list := dbm.NewListHelper(ws.db)
	values := list.PrefixScan(prefix)
	caches := make([]*types.CreateTransactionCache, 0)
	if len(values) == 0 {
		return caches, nil
	}
	for _, value := range values {
		cache := types.CreateTransactionCache{}
		err := types.Decode(value, &cache)
		if err != nil {
			return caches, err
		}
		caches = append(caches, &cache)
	}
	return caches, nil
}
//升级数据库的版本号
func (ws *Store) SetWalletVersion(version int64) error {
	data, err := json.Marshal(version)
	if err != nil {
		walletlog.Error("SetWalletVerKey marshal version", "err", err)
		return types.ErrMarshal
	}

	ws.db.SetSync(WalletVerKey, data)
	return nil
}

// 获取wallet数据库的版本号
func (ws *Store) GetWalletVersion() int64 {
	var version int64
	data, err := ws.db.Get(WalletVerKey)
	if data == nil || err != nil {
		return 0
	}
	err = json.Unmarshal(data, &version)
	if err != nil {
		walletlog.Error("GetWalletVersion unmarshal", "err", err)
		return 0
	}
	return version
}
