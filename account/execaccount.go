// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/types"
)

// LoadExecAccount Load exec account from address and exec
func (acc *DB) LoadExecAccount(addr, execaddr string) *types.Account {
	value, err := acc.db.Get(acc.execAccountKey(addr, execaddr))
	if err != nil {
		return &types.Account{Addr: addr}
	}
	var acc1 types.Account
	err = types.Decode(value, &acc1)
	if err != nil {
		panic(err) //数据库已经损坏
	}
	return &acc1
}

// LoadExecAccountQueue load exec account from statedb
func (acc *DB) LoadExecAccountQueue(api client.QueueProtocolAPI, addr, execaddr string) (*types.Account, error) {
	header, err := api.GetLastHeader()
	if err != nil {
		return nil, err
	}
	return acc.LoadExecAccountHistoryQueue(api, addr, execaddr, header.GetStateHash())
}

// SaveExecAccount save exec account data to db
func (acc *DB) SaveExecAccount(execaddr string, acc1 *types.Account) {
	set := acc.GetExecKVSet(execaddr, acc1)
	for i := 0; i < len(set); i++ {
		err := acc.db.Set(set[i].GetKey(), set[i].Value)
		if err != nil {
			panic(err)
		}
	}
}

// GetExecKVSet 将执行账户数据转为数据库存储kv
func (acc *DB) GetExecKVSet(execaddr string, acc1 *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc1)
	kvset = append(kvset, &types.KeyValue{
		Key:   acc.execAccountKey(acc1.Addr, execaddr),
		Value: value,
	})
	return kvset
}

func (acc *DB) execAccountKey(address, execaddr string) (key []byte) {
	key = make([]byte, 0, len(acc.execAccountKeyPerfix)+len(execaddr)+len(address)+1)
	key = append(key, acc.execAccountKeyPerfix...)
	key = append(key, []byte(execaddr)...)
	key = append(key, []byte(":")...)
	key = append(key, []byte(address)...)
	return key
}

// TransferToExec transfer coins from address to exec address
func (acc *DB) TransferToExec(from, to string, amount int64) (*types.Receipt, error) {
	receipt, err := acc.Transfer(from, to, amount)
	if err != nil {
		return nil, err
	}
	receipt2, err := acc.ExecDeposit(from, to, amount)
	if err != nil {
		//存款不应该出任何问题
		panic(err)
	}
	return acc.mergeReceipt(receipt, receipt2), nil
}

// TransferWithdraw 撤回转帐
func (acc *DB) TransferWithdraw(from, to string, amount int64) (*types.Receipt, error) {
	//先判断可以取款
	if err := acc.CheckTransfer(to, from, amount); err != nil {
		return nil, err
	}
	receipt, err := acc.ExecWithdraw(to, from, amount)
	if err != nil {
		return nil, err
	}
	//然后执行transfer
	receipt2, err := acc.Transfer(to, from, amount)
	if err != nil {
		panic(err) //在withdraw
	}
	return acc.mergeReceipt(receipt, receipt2), nil
}

//ExecFrozen 执行冻结资金，四个操作中 Deposit 自动完成，不需要模块外的函数来调用
func (acc *DB) ExecFrozen(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	if acc1.Balance-amount < 0 {
		alog.Error("ExecFrozen", "balance", acc1.Balance, "amount", amount)
		return nil, types.ErrNoBalance
	}
	copyacc := *acc1
	acc1.Balance -= amount
	acc1.Frozen += amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecFrozen)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecActive 执行激活资金
func (acc *DB) ExecActive(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	if acc1.Frozen-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyacc := *acc1
	acc1.Balance += amount
	acc1.Frozen -= amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecActive)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecTransfer 执行转帐
func (acc *DB) ExecTransfer(from, to, execaddr string, amount int64) (*types.Receipt, error) {
	if from == to {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := acc.LoadExecAccount(from, execaddr)
	accTo := acc.LoadExecAccount(to, execaddr)

	if accFrom.GetBalance()-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyaccFrom := *accFrom
	copyaccTo := *accTo

	accFrom.Balance -= amount
	accTo.Balance += amount

	receiptBalanceFrom := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccFrom,
		Current:  accFrom,
	}
	receiptBalanceTo := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccTo,
		Current:  accTo,
	}

	acc.SaveExecAccount(execaddr, accFrom)
	acc.SaveExecAccount(execaddr, accTo)
	return acc.execReceipt2(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
}

// ExecTransferFrozen 从自己冻结的钱里面扣除，转移到别人的活动钱包里面去
func (acc *DB) ExecTransferFrozen(from, to, execaddr string, amount int64) (*types.Receipt, error) {
	if from == to {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := acc.LoadExecAccount(from, execaddr)
	accTo := acc.LoadExecAccount(to, execaddr)
	b := accFrom.GetFrozen() - amount
	if b < 0 {
		return nil, types.ErrNoBalance
	}
	copyaccFrom := *accFrom
	copyaccTo := *accTo

	accFrom.Frozen -= amount
	accTo.Balance += amount

	receiptBalanceFrom := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccFrom,
		Current:  accFrom,
	}
	receiptBalanceTo := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyaccTo,
		Current:  accTo,
	}

	acc.SaveExecAccount(execaddr, accFrom)
	acc.SaveExecAccount(execaddr, accTo)
	return acc.execReceipt2(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
}

// ExecAddress 根据执行器名称获取执行器地址
func (acc *DB) ExecAddress(name string) string {
	return address.ExecAddress(name)
}

// ExecDepositFrozen 执行增发coins
func (acc *DB) ExecDepositFrozen(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	//这个函数只有挖矿的合约才能调用
	allow := false
	for _, exec := range types.GetMinerExecs() {
		if acc.ExecAddress(exec) == execaddr {
			allow = true
			break
		}
	}
	if !allow {
		return nil, types.ErrNotAllowDeposit
	}
	receipt1, err := acc.depositBalance(execaddr, amount)
	if err != nil {
		return nil, err
	}
	receipt2, err := acc.execDepositFrozen(addr, execaddr, amount)
	if err != nil {
		return nil, err
	}
	return acc.mergeReceipt(receipt1, receipt2), nil
}

func (acc *DB) execDepositFrozen(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	copyacc := *acc1
	acc1.Frozen += amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecDeposit)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecDeposit  在当前addr的execaddr地址中存款
func (acc *DB) ExecDeposit(addr, execaddr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	copyacc := *acc1
	acc1.Balance += amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	//alog.Debug("execDeposit", "addr", addr, "execaddr", execaddr, "account", acc)
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecDeposit)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

// ExecWithdraw 执行撤回转帐
func (acc *DB) ExecWithdraw(execaddr, addr string, amount int64) (*types.Receipt, error) {
	if addr == execaddr {
		return nil, types.ErrSendSameToRecv
	}
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadExecAccount(addr, execaddr)
	if acc1.Balance-amount < 0 {
		return nil, types.ErrNoBalance
	}
	copyacc := *acc1
	acc1.Balance -= amount
	receiptBalance := &types.ReceiptExecAccountTransfer{
		ExecAddr: execaddr,
		Prev:     &copyacc,
		Current:  acc1,
	}
	acc.SaveExecAccount(execaddr, acc1)
	ty := int32(types.TyLogExecWithdraw)
	return acc.execReceipt(ty, acc1, receiptBalance), nil
}

func (acc *DB) execReceipt(ty int32, acc1 *types.Account, r *types.ReceiptExecAccountTransfer) *types.Receipt {
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(r),
	}
	kv := acc.GetExecKVSet(r.ExecAddr, acc1)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}
}

func (acc *DB) execReceipt2(acc1, acc2 *types.Account, r1, r2 *types.ReceiptExecAccountTransfer) *types.Receipt {
	ty := int32(types.TyLogExecTransfer)
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(r1),
	}
	log2 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(r2),
	}
	kv := acc.GetExecKVSet(r1.ExecAddr, acc1)
	kv = append(kv, acc.GetExecKVSet(r2.ExecAddr, acc2)...)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1, log2},
	}
}

func (acc *DB) mergeReceipt(receipt, receipt2 *types.Receipt) *types.Receipt {
	receipt.Logs = append(receipt.Logs, receipt2.Logs...)
	receipt.KV = append(receipt.KV, receipt2.KV...)
	return receipt
}

// LoadExecAccountHistoryQueue  载入当前statehash,载入执行账户数据
func (acc *DB) LoadExecAccountHistoryQueue(api client.QueueProtocolAPI, addr, execaddr string, stateHash []byte) (*types.Account, error) {
	get := types.StoreGet{StateHash: stateHash}
	get.Keys = append(get.Keys, acc.execAccountKey(addr, execaddr))
	values, err := api.StoreGet(&get)
	if err != nil {
		return nil, err
	}
	if len(values.Values) <= 0 {
		return nil, types.ErrNotFound
	}
	value := values.Values[0]
	if value == nil {
		return &types.Account{Addr: addr}, nil
	}

	var acc1 types.Account
	err = types.Decode(value, &acc1)
	if err != nil {
		return nil, err
	}

	return &acc1, nil
}
