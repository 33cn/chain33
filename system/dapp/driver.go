package dapp

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"bytes"
	"reflect"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var blog = log.New("module", "execs.base")

const (
	TxIndexFrom = 1
	TxIndexTo   = 2
)

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
	//设置执行器的真实名称
	SetName(string)
	SetCurrentExecName(string)
	Allow(tx *types.Transaction, index int) error
	IsFriend(myexec []byte, writekey []byte, othertx *types.Transaction) bool
	GetActionName(tx *types.Transaction) string
	SetEnv(height, blocktime int64, difficulty uint64)
	CheckTx(tx *types.Transaction, index int) error
	Exec(tx *types.Transaction, index int) (*types.Receipt, error)
	ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error)
	Query(funcName string, params []byte) (types.Message, error)
	IsFree() bool
	SetApi(client.QueueProtocolAPI)
	SetTxs(txs []*types.Transaction)
	SetReceipt(receipts []*types.ReceiptData)

	//GetTxs and TxGroup
	GetTxs() []*types.Transaction
	GetTxGroup(index int) ([]*types.Transaction, error)
	GetPayloadValue() types.Message
	GetFuncMap() map[string]reflect.Method
	GetExecutorType() types.ExecutorType
}

type DriverBase struct {
	statedb      dbm.KV
	localdb      dbm.KVDB
	coinsaccount *account.DB
	height       int64
	blocktime    int64
	name         string
	curname      string
	child        Driver
	childValue   reflect.Value
	isFree       bool
	difficulty   uint64
	api          client.QueueProtocolAPI
	txs          []*types.Transaction
	receipts     []*types.ReceiptData
	ety          types.ExecutorType
}

func (d *DriverBase) GetPayloadValue() types.Message {
	if d.ety == nil {
		return nil
	}
	return d.ety.GetPayload()
}

func (d *DriverBase) GetExecutorType() types.ExecutorType {
	return d.ety
}

func (d *DriverBase) GetFuncMap() map[string]reflect.Method {
	if d.ety == nil {
		return nil
	}
	return d.ety.GetExecFuncMap()
}

func (d *DriverBase) SetApi(api client.QueueProtocolAPI) {
	d.api = api
}

func (d *DriverBase) GetApi() client.QueueProtocolAPI {
	return d.api
}

func (d *DriverBase) SetEnv(height, blocktime int64, difficulty uint64) {
	d.height = height
	d.blocktime = blocktime
	d.difficulty = difficulty
}

func (d *DriverBase) SetIsFree(isFree bool) {
	d.isFree = isFree
}

func (d *DriverBase) IsFree() bool {
	return d.isFree
}

func (d *DriverBase) SetExecutorType(e types.ExecutorType) {
	d.ety = e
}

func (d *DriverBase) SetChild(e Driver) {
	d.child = e
	d.childValue = reflect.ValueOf(e)
}

func (d *DriverBase) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	lset, err := d.callLocal("ExecLocal_", tx, receipt, index)
	if err != nil {
		blog.Debug("call ExecLocal", "tx.Execer", string(tx.Execer), "err", err)
		return &set, nil
	}
	//merge
	if lset != nil && lset.KV != nil {
		set.KV = append(set.KV, lset.KV...)
	}
	return &set, nil
}

func (d *DriverBase) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	var set types.LocalDBSet
	lset, err := d.callLocal("ExecDelLocal_", tx, receipt, index)
	if err != nil {
		blog.Error("call ExecDelLocal", "execer", string(tx.Execer), "err", err)
		return &set, nil
	}
	//merge
	if lset != nil && lset.KV != nil {
		set.KV = append(set.KV, lset.KV...)
	}
	return &set, nil
}

func (d *DriverBase) callLocal(prefix string, tx *types.Transaction, receipt *types.ReceiptData, index int) (set *types.LocalDBSet, err error) {
	if d.ety == nil {
		return nil, types.ErrActionNotSupport
	}
	defer func() {
		if r := recover(); r != nil {
			blog.Error("call localexec error", "prefix", prefix, "tx.exec", tx.Execer, "info", r)
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

func CheckAddress(addr string, height int64) error {
	if IsDriverAddress(addr, height) {
		return nil
	}
	return address.CheckAddress(addr)
}

//调用子类的CheckTx, 也可以不调用，实现自己的CheckTx
func (d *DriverBase) Exec(tx *types.Transaction, index int) (receipt *types.Receipt, err error) {
	if d.ety == nil {
		return nil, nil
	}
	defer func() {
		if r := recover(); r != nil {
			blog.Error("call exec error", "tx.exec", tx.Execer, "info", r)
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
	//参数1
	r1 := valueret[0].Interface()
	if r1 != nil {
		if r, ok := r1.(*types.Receipt); ok {
			receipt = r
		} else {
			return nil, types.ErrMethodReturnType
		}
	}
	//参数2
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

//默认情况下，tx.To 地址指向合约地址
func (d *DriverBase) CheckTx(tx *types.Transaction, index int) error {
	execer := string(tx.Execer)
	if ExecAddress(execer) != tx.To {
		return types.ErrToAddrNotSameToExecAddr
	}
	return nil
}

func (d *DriverBase) SetStateDB(db dbm.KV) {
	if d.coinsaccount == nil {
		//log.Error("new CoinsAccount")
		d.coinsaccount = account.NewCoinsAccount()
	}
	d.statedb = db
	d.coinsaccount.SetDB(db)
}

func (d *DriverBase) GetTxGroup(index int) ([]*types.Transaction, error) {
	if len(d.txs) <= index {
		return nil, types.ErrTxGroupIndex
	}
	tx := d.txs[index]
	c := int(tx.GroupCount)
	if c <= 0 || c > int(types.MaxTxGroupSize) {
		return nil, types.ErrTxGroupCount
	}
	for i := index; i >= 0 && i >= index-c; i-- {
		if bytes.Equal(d.txs[i].Header, d.txs[i].Hash()) { //find header
			txgroup := types.Transactions{Txs: d.txs[i : i+c]}
			err := txgroup.Check(d.GetHeight(), types.GInt("MinFee"))
			if err != nil {
				return nil, err
			}
			return txgroup.Txs, nil
		}
	}
	return nil, types.ErrTxGroupFormat
}

func (d *DriverBase) GetReceipt() []*types.ReceiptData {
	return d.receipts
}

func (d *DriverBase) SetReceipt(receipts []*types.ReceiptData) {
	d.receipts = receipts
}

func (d *DriverBase) GetStateDB() dbm.KV {
	return d.statedb
}

func (d *DriverBase) SetLocalDB(db dbm.KVDB) {
	d.localdb = db
}

func (d *DriverBase) GetLocalDB() dbm.KVDB {
	return d.localdb
}

func (d *DriverBase) GetHeight() int64 {
	return d.height
}

func (d *DriverBase) GetBlockTime() int64 {
	return d.blocktime
}

func (d *DriverBase) GetDifficulty() uint64 {
	return d.difficulty
}

func (d *DriverBase) GetName() string {
	if d.name == "" {
		return d.child.GetDriverName()
	}
	return d.name
}

func (d *DriverBase) GetCurrentExecName() string {
	if d.curname == "" {
		return d.child.GetDriverName()
	}
	return d.curname
}

func (d *DriverBase) SetName(name string) {
	d.name = name
}

func (d *DriverBase) SetCurrentExecName(name string) {
	d.curname = name
}

func (d *DriverBase) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

func (d *DriverBase) CheckSignatureData(tx *types.Transaction, index int) bool {
	return true
}

func (d *DriverBase) GetCoinsAccount() *account.DB {
	if d.coinsaccount == nil {
		d.coinsaccount = account.NewCoinsAccount()
		d.coinsaccount.SetDB(d.statedb)
	}
	return d.coinsaccount
}

func (d *DriverBase) GetTxs() []*types.Transaction {
	return d.txs
}

func (d *DriverBase) SetTxs(txs []*types.Transaction) {
	d.txs = txs
}
