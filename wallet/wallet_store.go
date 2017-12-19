package wallet

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	"github.com/golang/protobuf/proto"
)

var WalletPassKey = []byte("WalletPassWord")
var WalletFeeAmount = []byte("WalletFeeAmount")

var storelog = walletlog.New("submodule", "store")

type WalletStore struct {
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

func NewWalletStore(db dbm.DB) *WalletStore {
	return &WalletStore{
		db: db,
	}
}

func (ws *WalletStore) NewBatch(sync bool) dbm.Batch {
	storeBatch := ws.db.NewBatch(sync)
	return storeBatch
}

func (ws *WalletStore) SetWalletPassword(newpass string) {
	ws.db.SetSync(WalletPassKey, []byte(newpass))
}

func (ws *WalletStore) GetWalletPassword() string {
	bytes := ws.db.Get(WalletPassKey)
	if bytes == nil {
		return ""
	}
	return string(bytes)
}

func (ws *WalletStore) SetFeeAmount(FeeAmount int64) error {
	bytes, err := json.Marshal(FeeAmount)
	if err != nil {
		walletlog.Error("SetFeeAmount marshal FeeAmount", "err", err)
		return err
	}

	ws.db.SetSync(WalletFeeAmount, bytes)
	return nil
}

func (ws *WalletStore) GetFeeAmount() int64 {
	var FeeAmount int64
	bytes := ws.db.Get(WalletFeeAmount)
	if bytes == nil {
		return  1000000
	}
	err := json.Unmarshal(bytes, &FeeAmount)
	if err != nil {
		walletlog.Error("GetFeeAmount unmarshal", "err", err)
		return 1000000
	}
	return FeeAmount
}

func (ws *WalletStore) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	if len(addr) == 0 {
		err := errors.New("input addr is null")
		return err
	}
	if account == nil {
		err := errors.New("input account is null")
		return err
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
		return err
	}

	//需要同时修改三个表，Account，Addr，Label，批量处理
	newbatch := ws.db.NewBatch(true)
	ws.db.Set(calcAccountKey(timestamp, addr), accountbyte)
	ws.db.Set(calcAddrKey(addr), accountbyte)
	ws.db.Set(calcLabelKey(account.GetLabel()), accountbyte)
	newbatch.Write()
	return nil
}

func (ws *WalletStore) GetAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(addr) == 0 {
		err := errors.New("input addr is null")
		return nil, err
	}
	bytes := ws.db.Get(calcAddrKey(addr))
	if bytes == nil {
		err := errors.New("does not exist in wallet!")
		return nil, err
	}
	err := proto.Unmarshal(bytes, &account)
	if err != nil {
		walletlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, err
	}
	return &account, nil
}

func (ws *WalletStore) GetAccountByLabel(label string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(label) == 0 {
		err := errors.New("input label is null")
		return nil, err
	}
	bytes := ws.db.Get(calcLabelKey(label))
	if bytes == nil {
		err := errors.New("does not exist in wallet!")
		return nil, err
	}
	err := proto.Unmarshal(bytes, &account)
	if err != nil {
		walletlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
		return nil, err
	}
	return &account, nil
}

func (ws *WalletStore) GetAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {

	if len(addr) == 0 {
		err := errors.New("input addr is null")
		return nil, err
	}
	accbytes := ws.db.PrefixScan([]byte(addr))
	if len(accbytes) == 0 {
		err := errors.New("does not exist Account!")
		return nil, err
	}
	WalletAccountStores := make([]*types.WalletAccountStore, len(accbytes))
	for index, accbyte := range accbytes {
		var walletaccount types.WalletAccountStore
		err := proto.Unmarshal(accbyte, &walletaccount)
		if err != nil {
			walletlog.Error("GetAccountByAddr", "proto.Unmarshal err:", err)
			return nil, err
		}
		WalletAccountStores[index] = &walletaccount
	}
	return WalletAccountStores, nil
}

//迭代获取从指定key：height*100000+index 开始向前或者向后查找指定count的交易
func (ws *WalletStore) GetTxDetailByIter(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	var txDetails types.WalletTxDetails
	if TxList == nil {
		err := errors.New("GetTxDetailByIter TxList is null")
		return nil, err
	}

	var txbytes [][]byte
	//FromTx是空字符串时。默认从最新的交易开始取count个
	if len(TxList.FromTx) == 0 {
		txbytes = ws.db.IteratorScanFromLast([]byte(calcTxKey(string(TxList.FromTx))), TxList.Count, TxList.Direction)
		if len(txbytes) == 0 {
			err := errors.New("does not exist tx!")
			return nil, err
		}
	} else {
		txbytes = ws.db.IteratorScan([]byte(calcTxKey(string(TxList.FromTx))), TxList.Count, TxList.Direction)
		if len(txbytes) == 0 {
			err := errors.New("does not exist tx!")
			return nil, err
		}
	}

	txDetails.TxDetails = make([]*types.WalletTxDetail, len(txbytes))
	for index, txdetailbyte := range txbytes {
		var txdetail types.WalletTxDetail
		err := proto.Unmarshal(txdetailbyte, &txdetail)
		if err != nil {
			walletlog.Error("GetTxDetailByIter", "proto.Unmarshal err:", err)
			return nil, err
		}
		txDetails.TxDetails[index] = &txdetail
	}
	return &txDetails, nil
}
