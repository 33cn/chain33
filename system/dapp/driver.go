// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dapp 系统基础dapp包
package dapp

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"bytes"
	"reflect"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/client/api"
	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var blog = log.New("module", "execs.base")

const (
	// TxIndexFrom transaction index from
	TxIndexFrom = 1
	// TxIndexTo transaction index to
	TxIndexTo = 2

	//ExecLocalSameTime Exec 的时候 同时执行 ExecLocal
	ExecLocalSameTime = int64(1)
)

// Driver defines some interface
type Driver interface {
	SetStateDB(dbm.KV)
	GetCoinsAccount() *account.DB
	SetLocalDB(dbm.KVDB)
	//当前交易执行器名称
	GetCurrentExecName() string
	//驱动的名字，这个名称是固定的
	GetDriverName() string
	//执行器的别名(一个驱动(code),允许创建多个执行器，类似evm一份代码可以创建多个合约）
	GetName() string
	GetExecutorAPI() api.ExecutorAPI
	//设置执行器的真实名称
	SetName(string)
	SetCurrentExecName(string)
	Allow(tx *types.Transaction, index int) error
	IsFriend(myexec []byte, writekey []byte, othertx *types.Transaction) bool
	GetActionName(tx *types.Transaction) string
	SetEnv(height, blocktime int64, difficulty uint64)
	SetBlockInfo([]byte, []byte, int64)
	CheckTx(tx *types.Transaction, index int) error
	Exec(tx *types.Transaction, index int) (*types.Receipt, error)
	ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	Query(funcName string, params []byte) (types.Message, error)
	IsFree() bool
	SetAPI(client.QueueProtocolAPI)
	SetExecutorAPI(queueapi client.QueueProtocolAPI, chain33api types.Chain33Client)
	SetTxs(txs []*types.Transaction)
	SetReceipt(receipts []*types.ReceiptData)

	//GetTxs and TxGroup
	GetTxs() []*types.Transaction
	GetTxGroup(index int) ([]*types.Transaction, error)
	GetPayloadValue() types.Message
	GetFuncMap() map[string]reflect.Method
	GetExecutorType() types.ExecutorType
	CheckReceiptExecOk() bool
	ExecutorOrder() int64
	Upgrade() (*types.LocalDBSet, error)
}

// DriverBase defines driverbase type
type DriverBase struct {
	statedb              dbm.KV
	localdb              dbm.KVDB
	coinsaccount         *account.DB
	height               int64
	blocktime            int64
	parentHash, mainHash []byte
	mainHeight           int64
	name                 string
	curname              string
	child                Driver
	childValue           reflect.Value
	isFree               bool
	difficulty           uint64
	api                  client.QueueProtocolAPI
	execapi              api.ExecutorAPI
	txs                  []*types.Transaction
	receipts             []*types.ReceiptData
	ety                  types.ExecutorType
}

//Upgrade default upgrade only print a message
func (d *DriverBase) Upgrade() (*types.LocalDBSet, error) {
	blog.Info("upgrade ", "dapp", d.GetName())
	return nil, nil
}

// GetPayloadValue define get payload func
func (d *DriverBase) GetPayloadValue() types.Message {
	if d.ety == nil {
		return nil
	}
	return d.ety.GetPayload()
}

// GetExecutorType defines get executortype func
func (d *DriverBase) GetExecutorType() types.ExecutorType {
	return d.ety
}

//ExecutorOrder 执行顺序, 如果要使用 ExecLocalSameTime
//那么会同时执行 ExecLocal
func (d *DriverBase) ExecutorOrder() int64 {
	return 0
}

//GetLastHash 获取最后区块的hash，主链和平行链不同
func (d *DriverBase) GetLastHash() []byte {
	types.AssertConfig(d.api)
	cfg := d.api.GetConfig()
	if cfg.IsPara() {
		return d.mainHash
	}
	return d.parentHash
}

//GetParentHash 获取上一个区块的hash
func (d *DriverBase) GetParentHash() []byte {
	return d.parentHash
}

// GetFuncMap defines get execfuncmap func
func (d *DriverBase) GetFuncMap() map[string]reflect.Method {
	if d.ety == nil {
		return nil
	}
	return d.ety.GetExecFuncMap()
}

// SetAPI set queue protocol api
func (d *DriverBase) SetAPI(queueapi client.QueueProtocolAPI) {
	d.api = queueapi
}

// SetExecutorAPI set queue protocol api
func (d *DriverBase) SetExecutorAPI(queueapi client.QueueProtocolAPI, chain33api types.Chain33Client) {
	d.execapi = api.New(queueapi, chain33api)
}

// GetAPI return queue protocol api
func (d *DriverBase) GetAPI() client.QueueProtocolAPI {
	return d.api
}

// GetExecutorAPI return executor api
func (d *DriverBase) GetExecutorAPI() api.ExecutorAPI {
	return d.execapi
}

// SetEnv set env
func (d *DriverBase) SetEnv(height, blocktime int64, difficulty uint64) {
	d.height = height
	d.blocktime = blocktime
	d.difficulty = difficulty
}

//SetBlockInfo 设置区块的信息
func (d *DriverBase) SetBlockInfo(parentHash, mainHash []byte, mainHeight int64) {
	d.parentHash = parentHash
	d.mainHash = mainHash
	d.mainHeight = mainHeight
}

// SetIsFree set isfree
func (d *DriverBase) SetIsFree(isFree bool) {
	d.isFree = isFree
}

// IsFree return isfree
func (d *DriverBase) IsFree() bool {
	return d.isFree
}

// SetExecutorType set exectortype
func (d *DriverBase) SetExecutorType(e types.ExecutorType) {
	d.ety = e
}

// SetChild set childvalue
func (d *DriverBase) SetChild(e Driver) {
	d.child = e
	d.childValue = reflect.ValueOf(e)
}

// ExecLocal local exec
func (d *DriverBase) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	lset, err := d.callLocal("ExecLocal_", tx, receipt, index)
	if err != nil || lset == nil { // 不能向上层返回LocalDBSet为nil, 以及error
		blog.Debug("call ExecLocal", "tx.Execer", string(tx.Execer), "err", err)
		return &types.LocalDBSet{}, nil
	}
	return lset, nil
}

// ExecDelLocal local execdel
func (d *DriverBase) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	lset, err := d.callLocal("ExecDelLocal_", tx, receipt, index)
	if err != nil || lset == nil { // 不能向上层返回LocalDBSet为nil, 以及error
		blog.Error("call ExecDelLocal", "execer", string(tx.Execer), "err", err)
		return &types.LocalDBSet{}, nil
	}
	return lset, nil
}

func (d *DriverBase) callLocal(prefix string, tx *types.Transaction, receipt *types.ReceiptData, index int) (set *types.LocalDBSet, err error) {
	if d.ety == nil {
		return nil, types.ErrActionNotSupport
	}

	if d.child.CheckReceiptExecOk() {
		if receipt.GetTy() != types.ExecOk {
			return &types.LocalDBSet{}, nil
		}
	}

	defer func() {
		if r := recover(); r != nil {
			blog.Error("call localexec error", "prefix", prefix, "tx.exec", string(tx.Execer), "info", r)
			err = types.ErrActionNotSupport
			set = nil
		}
	}()
	name, value, err := d.ety.DecodePayloadValue(tx)
	if err != nil {
		return nil, err
	}
	//call action
	funcname := prefix + name
	funcmap := d.child.GetFuncMap()
	if _, ok := funcmap[funcname]; !ok {
		return nil, types.ErrActionNotSupport
	}
	valueret := funcmap[funcname].Func.Call([]reflect.Value{d.childValue, value, reflect.ValueOf(tx), reflect.ValueOf(receipt), reflect.ValueOf(index)})
	if !types.IsOK(valueret, 2) {
		return nil, types.ErrMethodReturnType
	}
	r1 := valueret[0].Interface()
	if r1 != nil {
		if r, ok := r1.(*types.LocalDBSet); ok {
			set = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	r2 := valueret[1].Interface()
	err = nil
	if r2 != nil {
		if r, ok := r2.(error); ok {
			err = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	return set, err
}

// CheckAddress check address
func CheckAddress(cfg *types.Chain33Config, addr string, height int64) error {
	if IsDriverAddress(addr, height) {
		return nil
	}
	err := address.CheckAddress(addr)
	if !cfg.IsFork(height, "ForkMultiSignAddress") && err == address.ErrCheckVersion {
		return nil
	}
	if !cfg.IsFork(height, "ForkBase58AddressCheck") && err == address.ErrAddressChecksum {
		return nil
	}

	return err
}

// Exec call the check exectx subclass, you can also do it without calling , implement your own checktx
func (d *DriverBase) Exec(tx *types.Transaction, index int) (receipt *types.Receipt, err error) {
	if d.ety == nil {
		return nil, nil
	}
	defer func() {
		if r := recover(); r != nil {
			blog.Error("call exec error", "tx.exec", string(tx.Execer), "info", r)
			err = types.ErrActionNotSupport
			receipt = nil
		}
	}()
	//为了兼容原来的系统,多加了一个判断
	if d.child.GetPayloadValue() == nil {
		return nil, nil
	}
	if d.ety == nil {
		return nil, nil
	}
	name, value, err := d.ety.DecodePayloadValue(tx)
	if err != nil {
		return nil, err
	}
	funcmap := d.child.GetFuncMap()
	funcname := "Exec_" + name
	if _, ok := funcmap[funcname]; !ok {
		return nil, types.ErrActionNotSupport
	}
	valueret := funcmap[funcname].Func.Call([]reflect.Value{d.childValue, value, reflect.ValueOf(tx), reflect.ValueOf(index)})
	if !types.IsOK(valueret, 2) {
		return nil, types.ErrMethodReturnType
	}
	//parameter 1
	r1 := valueret[0].Interface()
	if r1 != nil {
		if r, ok := r1.(*types.Receipt); ok {
			receipt = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	//parameter 2
	r2 := valueret[1].Interface()
	err = nil
	if r2 != nil {
		if r, ok := r2.(error); ok {
			err = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	return receipt, err
}

// CheckTx  default:，tx.To address points to the contract address
func (d *DriverBase) CheckTx(tx *types.Transaction, index int) error {
	execer := string(tx.Execer)
	if ExecAddress(execer) != tx.To {
		return types.ErrToAddrNotSameToExecAddr
	}
	return nil
}

// SetStateDB set db state
func (d *DriverBase) SetStateDB(db dbm.KV) {
	if d.coinsaccount == nil {
		//log.Error("new CoinsAccount")
		types.AssertConfig(d.api)
		d.coinsaccount = account.NewCoinsAccount(d.api.GetConfig())
	}
	d.statedb = db
	d.coinsaccount.SetDB(db)
}

// GetTxGroup get txgroup
func (d *DriverBase) GetTxGroup(index int) ([]*types.Transaction, error) {
	if len(d.txs) <= index {
		return nil, types.ErrTxGroupIndex
	}
	tx := d.txs[index]
	c := int(tx.GroupCount)
	if c <= 0 || c > int(types.MaxTxGroupSize) {
		return nil, types.ErrTxGroupCount
	}
	types.AssertConfig(d.api)
	cfg := d.api.GetConfig()
	for i := index; i >= 0 && i >= index-c; i-- {
		if bytes.Equal(d.txs[i].Header, d.txs[i].Hash()) { //find header
			txgroup := types.Transactions{Txs: d.txs[i : i+c]}
			err := txgroup.Check(cfg, d.GetHeight(), cfg.GetMinTxFeeRate(), cfg.GetMaxTxFee())
			if err != nil {
				return nil, err
			}
			return txgroup.Txs, nil
		}
	}
	return nil, types.ErrTxGroupFormat
}

// GetReceipt return receipts
func (d *DriverBase) GetReceipt() []*types.ReceiptData {
	return d.receipts
}

// SetReceipt set receipt
func (d *DriverBase) SetReceipt(receipts []*types.ReceiptData) {
	d.receipts = receipts
}

// GetStateDB set statedb
func (d *DriverBase) GetStateDB() dbm.KV {
	return d.statedb
}

// SetLocalDB set localdb
func (d *DriverBase) SetLocalDB(db dbm.KVDB) {
	d.localdb = db
}

// GetLocalDB return localdb
func (d *DriverBase) GetLocalDB() dbm.KVDB {
	return d.localdb
}

// GetHeight return height
func (d *DriverBase) GetHeight() int64 {
	return d.height
}

// GetMainHeight return height
func (d *DriverBase) GetMainHeight() int64 {
	types.AssertConfig(d.api)
	if d.api.GetConfig().IsPara() {
		return d.mainHeight
	}
	return d.height
}

// GetBlockTime return block time
func (d *DriverBase) GetBlockTime() int64 {
	return d.blocktime
}

// GetDifficulty return difficulty
func (d *DriverBase) GetDifficulty() uint64 {
	return d.difficulty
}

// GetName defines return name func
func (d *DriverBase) GetName() string {
	if d.name == "" {
		return d.child.GetDriverName()
	}
	return d.name
}

// GetCurrentExecName defines get current execname
func (d *DriverBase) GetCurrentExecName() string {
	if d.curname == "" {
		return d.child.GetDriverName()
	}
	return d.curname
}

// SetName set name
func (d *DriverBase) SetName(name string) {
	d.name = name
}

// SetCurrentExecName set current execname
func (d *DriverBase) SetCurrentExecName(name string) {
	d.curname = name
}

// GetActionName get action name
func (d *DriverBase) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

// CheckSignatureData check signature data
func (d *DriverBase) CheckSignatureData(tx *types.Transaction, index int) bool {
	return true
}

// GetCoinsAccount get coins account
func (d *DriverBase) GetCoinsAccount() *account.DB {
	if d.coinsaccount == nil {
		types.AssertConfig(d.api)
		d.coinsaccount = account.NewCoinsAccount(d.api.GetConfig())
		d.coinsaccount.SetDB(d.statedb)
	}
	return d.coinsaccount
}

// GetTxs get transactions
func (d *DriverBase) GetTxs() []*types.Transaction {
	return d.txs
}

// SetTxs set transactions
func (d *DriverBase) SetTxs(txs []*types.Transaction) {
	d.txs = txs
}

// CheckReceiptExecOk default return true to check if receipt ty is ok, for specific plugin can overwrite it self
func (d *DriverBase) CheckReceiptExecOk() bool {
	return false
}

//AddRollbackKV add rollback kv
func (d *DriverBase) AddRollbackKV(tx *types.Transaction, execer []byte, kvs []*types.KeyValue) []*types.KeyValue {
	k := types.CalcRollbackKey(types.GetRealExecName(execer), tx.Hash())
	kvc := NewKVCreator(d.GetLocalDB(), types.CalcLocalPrefix(execer), k)
	kvc.AddListNoPrefix(kvs)
	kvc.AddRollbackKV()
	return kvc.KVList()
}

//DelRollbackKV del rollback kv when exec_del_local
func (d *DriverBase) DelRollbackKV(tx *types.Transaction, execer []byte) ([]*types.KeyValue, error) {
	krollback := types.CalcRollbackKey(types.GetRealExecName(execer), tx.Hash())
	kvc := NewKVCreator(d.GetLocalDB(), types.CalcLocalPrefix(execer), krollback)
	kvs, err := kvc.GetRollbackKVList()
	if err != nil {
		return nil, err
	}
	for _, kv := range kvs {
		kvc.AddNoPrefix(kv.Key, kv.Value)
	}
	kvc.DelRollbackKV()
	return kvc.KVList(), nil
}
