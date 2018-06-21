package wallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	WalletPassKey   = []byte("WalletPassWord")
	WalletFeeAmount = []byte("WalletFeeAmount")
	EncryptionFlag  = []byte("Encryption")
	PasswordHash    = []byte("PasswordHash")
	WalletVerKey    = []byte("WalletVerKey")
	storelog        = walletlog.New("submodule", "store")
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

//通过height*100000+index 查询Tx交易信息
//key:Tx:height*100000+index
func calcTxKey(key string) []byte {
	return []byte(fmt.Sprintf("Tx:%s", key))
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

	timestamp := fmt.Sprintf("%018d", time.Now().Unix())
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
func (ws *Store) GetTxDetailByIter(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	var txDetails types.WalletTxDetails
	if TxList == nil {
		walletlog.Error("GetTxDetailByIter TxList is nil")
		return nil, types.ErrInputPara
	}

	var txbytes [][]byte
	//FromTx是空字符串时。默认从最新的交易开始取count个
	if len(TxList.FromTx) == 0 {
		list := dbm.NewListHelper(ws.db)
		txbytes = list.IteratorScanFromLast(calcTxKey(""), TxList.Count)
		if len(txbytes) == 0 {
			walletlog.Error("GetTxDetailByIter IteratorScanFromLast does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	} else {
		list := dbm.NewListHelper(ws.db)
		txbytes = list.IteratorScan([]byte("Tx:"), calcTxKey(string(TxList.FromTx)), TxList.Count, TxList.Direction)
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
		//print
		//walletlog.Debug("GetTxDetailByIter", "txdetail:", txdetail.String())
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
