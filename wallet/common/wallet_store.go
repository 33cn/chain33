package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	storelog = log15.New("wallet", "store")
)

func NewStore(db db.DB) *WalletStore {
	return &WalletStore{db: db}
}

// 钱包通用数据库存储类，实现对钱包账户数据库操作的基本实现
type WalletStore struct {
	db db.DB
}

func (ws *WalletStore) Close() {
	ws.db.Close()
}

func (ws *WalletStore) GetDB() db.DB {
	return ws.db
}

func (ws *WalletStore) NewBatch(sync bool) db.Batch {
	return ws.db.NewBatch(sync)
}

func (ws *WalletStore) Get(key []byte) ([]byte, error) {
	return ws.db.Get(key)
}

func (ws *WalletStore) Set(key []byte, value []byte) (err error) {
	return ws.db.Set(key, value)
}

func (ws *WalletStore) NewListHelper() *db.ListHelper {
	return db.NewListHelper(ws.db)
}

func (ws *WalletStore) SetWalletPassword(newpass string) {
	ws.GetDB().SetSync(CalcWalletPassKey(), []byte(newpass))
}

func (ws *WalletStore) GetWalletPassword() string {
	passwordbytes, err := ws.Get(CalcWalletPassKey())
	if passwordbytes == nil || err != nil {
		storelog.Error("GetWalletPassword", "Get from db error", err)
		return ""
	}
	return string(passwordbytes)
}

func (ws *WalletStore) SetFeeAmount(FeeAmount int64) error {
	FeeAmountbytes, err := json.Marshal(FeeAmount)
	if err != nil {
		storelog.Error("SetFeeAmount", "marshal FeeAmount error", err)
		return types.ErrMarshal
	}

	ws.GetDB().SetSync(CalcWalletPassKey(), FeeAmountbytes)
	return nil
}

func (ws *WalletStore) GetFeeAmount(minFee int64) int64 {
	FeeAmountbytes, err := ws.Get(CalcWalletPassKey())
	if FeeAmountbytes == nil || err != nil {
		storelog.Error("GetFeeAmount", "Get from db error", err)
		return minFee
	}
	var FeeAmount int64
	err = json.Unmarshal(FeeAmountbytes, &FeeAmount)
	if err != nil {
		storelog.Error("GetFeeAmount", "json unmarshal error", err)
		return minFee
	}
	return FeeAmount
}

func (ws *WalletStore) GetAccountByte(update bool, addr string, account *types.WalletAccountStore) ([]byte, error) {
	if len(addr) == 0 {
		storelog.Error("GetAccountByte addr is nil")
		return nil, types.ErrInputPara
	}
	if account == nil {
		storelog.Error("GetAccountByte account is nil")
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
		storelog.Error("GetAccountByte", " proto.Marshal error", err)
		return nil, types.ErrMarshal
	}
	return accountbyte, nil
}

func (ws *WalletStore) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	accountbyte, err := ws.GetAccountByte(update, addr, account)
	if err != nil {
		storelog.Error("SetWalletAccount", "GetAccountByte error", err)
		return err
	}
	//需要同时修改三个表，Account，Addr，Label，批量处理
	newbatch := ws.NewBatch(true)
	newbatch.Set(CalcAccountKey(account.TimeStamp, addr), accountbyte)
	newbatch.Set(CalcAddrKey(addr), accountbyte)
	newbatch.Set(CalcLabelKey(account.GetLabel()), accountbyte)
	newbatch.Write()
	return nil
}

func (ws *WalletStore) SetWalletAccountInBatch(update bool, addr string, account *types.WalletAccountStore, newbatch db.Batch) error {
	accountbyte, err := ws.GetAccountByte(update, addr, account)
	if err != nil {
		storelog.Error("SetWalletAccount", "GetAccountByte error", err)
		return err
	}
	//需要同时修改三个表，Account，Addr，Label，批量处理
	newbatch.Set(CalcAccountKey(account.TimeStamp, addr), accountbyte)
	newbatch.Set(CalcAddrKey(addr), accountbyte)
	newbatch.Set(CalcLabelKey(account.GetLabel()), accountbyte)
	return nil
}

func (ws *WalletStore) GetAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(addr) == 0 {
		storelog.Error("GetAccountByAddr addr is empty")
		return nil, types.ErrInputPara
	}
	data, err := ws.Get(CalcAddrKey(addr))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Debug("GetAccountByAddr addr", "err", err)
		}
		return nil, types.ErrAddrNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		storelog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

func (ws *WalletStore) GetAccountByLabel(label string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(label) == 0 {
		storelog.Error("GetAccountByLabel label is empty")
		return nil, types.ErrInputPara
	}
	data, err := ws.Get(CalcLabelKey(label))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Error("GetAccountByLabel label", "err", err)
		}
		return nil, types.ErrLabelNotExist
	}
	err = proto.Unmarshal(data, &account)
	if err != nil {
		storelog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

func (ws *WalletStore) GetAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {
	if len(addr) == 0 {
		storelog.Error("GetAccountByPrefix addr is nil")
		return nil, types.ErrInputPara
	}
	list := ws.NewListHelper()
	accbytes := list.PrefixScan([]byte(addr))
	if len(accbytes) == 0 {
		storelog.Error("GetAccountByPrefix addr not exist")
		return nil, types.ErrAccountNotExist
	}
	WalletAccountStores := make([]*types.WalletAccountStore, len(accbytes))
	for index, accbyte := range accbytes {
		var walletaccount types.WalletAccountStore
		err := proto.Unmarshal(accbyte, &walletaccount)
		if err != nil {
			storelog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		WalletAccountStores[index] = &walletaccount
	}
	return WalletAccountStores, nil
}

//迭代获取从指定key：height*100000+index 开始向前或者向后查找指定count的交易
func (ws *WalletStore) GetTxDetailByIter(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	var txDetails types.WalletTxDetails
	if TxList == nil {
		storelog.Error("GetTxDetailByIter TxList is nil")
		return nil, types.ErrInputPara
	}

	var txbytes [][]byte
	//FromTx是空字符串时。默认从最新的交易开始取count个
	if len(TxList.FromTx) == 0 {
		list := ws.NewListHelper()
		txbytes = list.IteratorScanFromLast(CalcTxKey(""), TxList.Count)
		if len(txbytes) == 0 {
			storelog.Error("GetTxDetailByIter IteratorScanFromLast does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	} else {
		list := ws.NewListHelper()
		txbytes = list.IteratorScan([]byte("Tx:"), CalcTxKey(string(TxList.FromTx)), TxList.Count, TxList.Direction)
		if len(txbytes) == 0 {
			storelog.Error("GetTxDetailByIter IteratorScan does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	}

	txDetails.TxDetails = make([]*types.WalletTxDetail, len(txbytes))
	for index, txdetailbyte := range txbytes {
		var txdetail types.WalletTxDetail
		err := proto.Unmarshal(txdetailbyte, &txdetail)
		if err != nil {
			storelog.Error("GetTxDetailByIter", "proto.Unmarshal err:", err)
			return nil, types.ErrUnmarshal
		}
		if string(txdetail.Tx.GetExecer()) == "coins" && txdetail.Tx.ActionName() == "withdraw" {
			//swap from and to
			txdetail.Fromaddr, txdetail.Tx.To = txdetail.Tx.To, txdetail.Fromaddr
		}
		txhash := txdetail.GetTx().Hash()
		txdetail.Txhash = txhash
		txDetails.TxDetails[index] = &txdetail
	}
	return &txDetails, nil
}

func (ws *WalletStore) SetEncryptionFlag(batch db.Batch) error {
	var flag int64 = 1
	data, err := json.Marshal(flag)
	if err != nil {
		storelog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}

	batch.Set(CalcEncryptionFlag(), data)
	return nil
}

func (ws *WalletStore) GetEncryptionFlag() int64 {
	var flag int64
	data, err := ws.Get(CalcEncryptionFlag())
	if data == nil || err != nil {
		return 0
	}
	err = json.Unmarshal(data, &flag)
	if err != nil {
		storelog.Error("GetEncryptionFlag unmarshal", "err", err)
		return 0
	}
	return flag
}

func (ws *WalletStore) SetPasswordHash(password string, batch db.Batch) error {
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
		storelog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}
	batch.Set(CalcPasswordHash(), pwhashbytes)
	return nil
}

func (ws *WalletStore) VerifyPasswordHash(password string) bool {
	var WalletPwHash types.WalletPwHash
	pwhashbytes, err := ws.Get(CalcPasswordHash())
	if pwhashbytes == nil || err != nil {
		return false
	}
	err = json.Unmarshal(pwhashbytes, &WalletPwHash)
	if err != nil {
		storelog.Error("VerifyPasswordHash unmarshal", "err", err)
		return false
	}
	pwhashstr := fmt.Sprintf("%s:%s", password, WalletPwHash.Randstr)
	pwhash := sha256.Sum256([]byte(pwhashstr))
	Pwhash := pwhash[:]
	//通过新的密码计算pwhash最对比
	return bytes.Equal(WalletPwHash.GetPwHash(), Pwhash)
}

func (ws *WalletStore) DelAccountByLabel(label string) {
	ws.GetDB().DeleteSync(CalcLabelKey(label))
}

//升级数据库的版本号
func (ws *WalletStore) SetWalletVersion(ver int64) error {
	data, err := json.Marshal(ver)
	if err != nil {
		storelog.Error("SetWalletVerKey marshal version", "err", err)
		return types.ErrMarshal
	}

	ws.GetDB().SetSync(version.WalletVerKey, data)
	return nil
}

// 获取wallet数据库的版本号
func (ws *WalletStore) GetWalletVersion() int64 {
	var ver int64
	data, err := ws.Get(version.WalletVerKey)
	if data == nil || err != nil {
		return 0
	}
	err = json.Unmarshal(data, &ver)
	if err != nil {
		storelog.Error("GetWalletVersion unmarshal", "err", err)
		return 0
	}
	return ver
}

func (ws *WalletStore) GetAutoMinerFlag() int32 {
	flag := int32(0)
	value, err := ws.Get(CalcWalletAutoMiner())
	if err != nil {
		storelog.Error("GetAutoMinerFlag", "Get error", err)
		return flag
	}
	if value != nil && string(value) == "1" {
		flag = 1
	}
	return flag
}

func (ws *WalletStore) SetAutoMinerFlag(flag int32) {
	if flag == 1 {
		ws.Set(CalcWalletAutoMiner(), []byte("1"))
	} else {
		ws.Set(CalcWalletAutoMiner(), []byte("0"))
	}
}

//判断钱包是否已经保存seed
func (ws *WalletStore) HasSeed() (bool, error) {
	seed, err := ws.Get(CalcWalletSeed())
	if len(seed) == 0 || err != nil {
		return false, types.ErrSeedExist
	}
	return true, nil
}
