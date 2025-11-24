// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/types"
)

var (
	storelog = log15.New("wallet", "store")
)

// AddrInfo 通过seed指定index创建的账户信息，目前主要用于空投地址
type AddrInfo struct {
	Index  uint32 `json:"index,omitempty"`
	Addr   string `json:"addr,omitempty"`
	Pubkey string `json:"pubkey,omitempty"`
}

// NewStore 新建存储对象
func NewStore(db db.DB) *Store {
	return &Store{db: db, blkBatch: db.NewBatch(true)}
}

// Store 钱包通用数据库存储类，实现对钱包账户数据库操作的基本实现
type Store struct {
	db        db.DB
	blkBatch  db.Batch
	batchLock sync.Mutex
}

// Close 关闭数据库
func (store *Store) Close() {
	store.db.Close()
}

// GetDB 获取数据库操作接口
func (store *Store) GetDB() db.DB {
	return store.db
}

// NewBatch 新建批处理操作对象接口
func (store *Store) NewBatch(sync bool) db.Batch {
	return store.db.NewBatch(sync)
}

// GetBlockBatch 新建批处理操作对象接口
func (store *Store) GetBlockBatch(sync bool) db.Batch {
	store.batchLock.Lock()
	store.blkBatch.Reset()
	store.blkBatch.UpdateWriteSync(sync)
	return store.blkBatch
}

// FreeBlockBatch free
func (store *Store) FreeBlockBatch() {
	store.batchLock.Unlock()
}

// Get 取值
func (store *Store) Get(key []byte) ([]byte, error) {
	return store.db.Get(key)
}

// Set 设置值
func (store *Store) Set(key []byte, value []byte) (err error) {
	return store.db.Set(key, value)
}

// NewListHelper 新建列表复制操作对象
func (store *Store) NewListHelper() *db.ListHelper {
	return db.NewListHelper(store.db)
}

// GetAccountByte 获取账号byte类型
func (store *Store) GetAccountByte(update bool, addr string, account *types.WalletAccountStore) ([]byte, error) {
	if len(addr) == 0 {
		storelog.Error("GetAccountByte addr is nil")
		return nil, types.ErrInvalidParam
	}
	if account == nil {
		storelog.Error("GetAccountByte account is nil")
		return nil, types.ErrInvalidParam
	}

	timestamp := fmt.Sprintf("%018d", types.Now().Unix())
	//更新时需要使用原来的Accountkey
	if update {
		timestamp = account.TimeStamp
	}
	account.TimeStamp = timestamp

	accountbyte := types.Encode(account)
	return accountbyte, nil
}

// SetWalletAccount 保存钱包账户信息
func (store *Store) SetWalletAccount(update bool, addr string, account *types.WalletAccountStore) error {
	accountbyte, err := store.GetAccountByte(update, addr, account)
	if err != nil {
		storelog.Error("SetWalletAccount", "GetAccountByte error", err)
		return err
	}
	//需要同时修改三个表，Account，Addr，Label，批量处理
	newbatch := store.NewBatch(true)
	newbatch.Set(CalcAccountKey(account.TimeStamp, addr), accountbyte)
	newbatch.Set(CalcAddrKey(addr), accountbyte)
	newbatch.Set(CalcLabelKey(account.GetLabel()), accountbyte)
	return newbatch.Write()
}

// SetWalletAccountInBatch 保存钱包账号信息
func (store *Store) SetWalletAccountInBatch(update bool, addr string, account *types.WalletAccountStore, newbatch db.Batch) error {
	accountbyte, err := store.GetAccountByte(update, addr, account)
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

// GetAccountByAddr 根据地址获取账号信息
func (store *Store) GetAccountByAddr(addr string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(addr) == 0 {
		storelog.Error("GetAccountByAddr addr is empty")
		return nil, types.ErrInvalidParam
	}
	data, err := store.Get(CalcAddrKey(addr))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Debug("GetAccountByAddr addr", "err", err)
		}
		return nil, types.ErrAddrNotExist
	}
	err = types.Decode(data, &account)
	if err != nil {
		storelog.Error("GetAccountByAddr", "types.Decode err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

// GetAccountByLabel 根据标签获取账号信息
func (store *Store) GetAccountByLabel(label string) (*types.WalletAccountStore, error) {
	var account types.WalletAccountStore
	if len(label) == 0 {
		storelog.Error("GetAccountByLabel label is empty")
		return nil, types.ErrInvalidParam
	}
	data, err := store.Get(CalcLabelKey(label))
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Error("GetAccountByLabel label", "err", err)
		}
		return nil, types.ErrLabelNotExist
	}
	err = types.Decode(data, &account)
	if err != nil {
		storelog.Error("GetAccountByAddr", "types.Decode err:", err)
		return nil, types.ErrUnmarshal
	}
	return &account, nil
}

// GetAccountByPrefix 根据前缀获取账号信息列表
func (store *Store) GetAccountByPrefix(addr string) ([]*types.WalletAccountStore, error) {
	if len(addr) == 0 {
		storelog.Error("GetAccountByPrefix addr is nil")
		return nil, types.ErrInvalidParam
	}
	list := store.NewListHelper()
	accbytes := list.PrefixScan([]byte(addr))
	if len(accbytes) == 0 {
		storelog.Debug("GetAccountByPrefix addr not exist")
		return nil, types.ErrAccountNotExist
	}
	WalletAccountStores := make([]*types.WalletAccountStore, len(accbytes))
	for index, accbyte := range accbytes {
		var walletaccount types.WalletAccountStore
		err := types.Decode(accbyte, &walletaccount)
		if err != nil {
			storelog.Error("GetAccountByAddr", "types.Decode err:", err)
			return nil, types.ErrUnmarshal
		}
		WalletAccountStores[index] = &walletaccount
	}
	return WalletAccountStores, nil
}

// GetTxDetailByIter 迭代获取从指定key：height*100000+index 开始向前或者向后查找指定count的交易
func (store *Store) GetTxDetailByIter(TxList *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	var txDetails types.WalletTxDetails
	if TxList == nil {
		storelog.Error("GetTxDetailByIter TxList is nil")
		return nil, types.ErrInvalidParam
	}

	var txbytes [][]byte
	//FromTx是空字符串时，
	//Direction == 0从最新的交易开始倒序取count个
	//Direction == 1从最早的交易开始正序取count个
	if len(TxList.FromTx) == 0 {
		list := store.NewListHelper()
		if TxList.Direction == 0 {
			txbytes = list.IteratorScanFromLast(CalcTxKey(""), TxList.Count, db.ListDESC)
		} else {
			txbytes = list.IteratorScanFromFirst(CalcTxKey(""), TxList.Count, db.ListASC)
		}
		if len(txbytes) == 0 {
			storelog.Error("GetTxDetailByIter IteratorScanFromLast does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	} else {
		list := store.NewListHelper()
		txbytes = list.IteratorScan(CalcTxKey(""), CalcTxKey(string(TxList.FromTx)), TxList.Count, TxList.Direction)
		if len(txbytes) == 0 {
			storelog.Error("GetTxDetailByIter IteratorScan does not exist tx!")
			return nil, types.ErrTxNotExist
		}
	}

	txDetails.TxDetails = make([]*types.WalletTxDetail, len(txbytes))
	for index, txdetailbyte := range txbytes {
		var txdetail types.WalletTxDetail
		err := types.Decode(txdetailbyte, &txdetail)
		if err != nil {
			storelog.Error("GetTxDetailByIter", "types.Decode err:", err)
			return nil, types.ErrUnmarshal
		}
		txdetail.Txhash = txdetail.GetTx().Hash()
		txDetails.TxDetails[index] = &txdetail
	}
	return &txDetails, nil
}

// SetEncryptionFlag 设置加密方式标志
func (store *Store) SetEncryptionFlag(batch db.Batch) error {
	var flag int64 = 1
	data, err := json.Marshal(flag)
	if err != nil {
		storelog.Error("SetEncryptionFlag marshal flag", "err", err)
		return types.ErrMarshal
	}

	batch.Set(CalcEncryptionFlag(), data)
	return nil
}

// GetEncryptionFlag 获取加密方式
func (store *Store) GetEncryptionFlag() int64 {
	var flag int64
	data, err := store.Get(CalcEncryptionFlag())
	if data == nil || err != nil {
		data, err = store.Get(CalckeyEncryptionCompFlag())
		if data == nil || err != nil {
			return 0
		}
	}
	err = json.Unmarshal(data, &flag)
	if err != nil {
		storelog.Error("GetEncryptionFlag unmarshal", "err", err)
		return 0
	}
	return flag
}

// SetPasswordHash 保存密码哈希
func (store *Store) SetPasswordHash(password string, batch db.Batch) error {
	WalletPwHash := &types.WalletPwHash{}
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

// VerifyPasswordHash 检查密码有效性
func (store *Store) VerifyPasswordHash(password string) bool {
	var WalletPwHash types.WalletPwHash
	pwhashbytes, err := store.Get(CalcPasswordHash())
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

// DelAccountByLabel 根据标签名称，删除对应的账号信息
func (store *Store) DelAccountByLabel(label string) {
	err := store.GetDB().DeleteSync(CalcLabelKey(label))
	if err != nil {
		storelog.Error("DelAccountByLabel", "err", err)
	}
}

// SetWalletVersion 升级数据库的版本号
func (store *Store) SetWalletVersion(ver int64) error {
	data, err := json.Marshal(ver)
	if err != nil {
		storelog.Error("SetWalletVerKey marshal version", "err", err)
		return types.ErrMarshal
	}
	return store.GetDB().SetSync(version.WalletVerKey, data)
}

// GetWalletVersion 获取wallet数据库的版本号
func (store *Store) GetWalletVersion() int64 {
	var ver int64
	data, err := store.Get(version.WalletVerKey)
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

// HasSeed 判断钱包是否已经保存seed
func (store *Store) HasSeed() (bool, error) {
	seed, err := store.Get(CalcWalletSeed())
	if len(seed) == 0 || err != nil {
		return false, types.ErrSeedExist
	}
	return true, nil
}

// GetAirDropIndex 获取指定index的空投地址
func (store *Store) GetAirDropIndex() (string, error) {
	var airDrop AddrInfo
	data, err := store.Get(CalcAirDropIndex())
	if data == nil || err != nil {
		if err != db.ErrNotFoundInDb {
			storelog.Debug("GetAirDropIndex", "err", err)
		}
		return "", types.ErrAddrNotExist
	}
	err = json.Unmarshal(data, &airDrop)
	if err != nil {
		storelog.Error("GetWalletVersion unmarshal", "err", err)
		return "", err
	}
	storelog.Debug("GetAirDropIndex ", "airDrop", airDrop)
	return airDrop.Addr, nil
}

// SetAirDropIndex 存储指定index的空投地址信息
func (store *Store) SetAirDropIndex(airDropIndex *AddrInfo) error {
	data, err := json.Marshal(airDropIndex)
	if err != nil {
		storelog.Error("SetAirDropIndex marshal", "err", err)
		return err
	}

	storelog.Debug("SetAirDropIndex ", "airDropIndex", airDropIndex)
	return store.GetDB().SetSync(CalcAirDropIndex(), data)
}
