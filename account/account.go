// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package account 实现chain33 区块链资产操作
package account

//package for account manger
//1. load from db
//2. save to db
//3. KVSet
//4. Transfer
//5. Add
//6. Sub
//7. Account balance query
//8. gen a private key -> private key to address (bitcoin likes)

import (
	"strings"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

var alog = log.New("module", "account")

// DB for account
type DB struct {
	db                   dbm.KV
	accountKeyPerfix     []byte
	execAccountKeyPerfix []byte
	execer               string
	symbol               string
	accountKeyBuffer     []byte
}

// NewCoinsAccount 新建账户
func NewCoinsAccount() *DB {
	prefix := "mavl-coins-" + types.GetCoinSymbol() + "-"
	return newAccountDB(prefix)
}

// NewAccountDB 新建DB账户
func NewAccountDB(execer string, symbol string, db dbm.KV) (*DB, error) {
	//如果execer 和  symbol 中存在 "-", 那么创建失败
	if strings.ContainsRune(execer, '-') {
		return nil, types.ErrExecNameNotAllow
	}
	if strings.ContainsRune(symbol, '-') {
		return nil, types.ErrSymbolNameNotAllow
	}
	accDB := newAccountDB(symbolPrefix(execer, symbol))
	accDB.execer = execer
	accDB.symbol = symbol
	accDB.SetDB(db)
	return accDB, nil
}

func newAccountDB(prefix string) *DB {
	acc := &DB{}
	acc.accountKeyPerfix = []byte(prefix)
	acc.accountKeyBuffer = make([]byte, 0, len(acc.accountKeyPerfix)+64)
	acc.accountKeyBuffer = append(acc.accountKeyBuffer, acc.accountKeyPerfix...)
	acc.execAccountKeyPerfix = append([]byte(prefix), []byte("exec-")...)
	//alog.Warn("NewAccountDB", "prefix", prefix, "key1", string(acc.accountKeyPerfix), "key2", string(acc.execAccountKeyPerfix))
	return acc
}

// SetDB set db
func (acc *DB) SetDB(db dbm.KV) *DB {
	acc.db = db
	return acc
}

func (acc *DB) accountReadKey(addr string) []byte {
	acc.accountKeyBuffer = acc.accountKeyBuffer[0:len(acc.accountKeyPerfix)]
	acc.accountKeyBuffer = append(acc.accountKeyBuffer, []byte(addr)...)
	return acc.accountKeyBuffer
}

// LoadAccount 根据地址载入账户
func (acc *DB) LoadAccount(addr string) *types.Account {
	value, err := acc.db.Get(acc.accountReadKey(addr))
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

// CheckTransfer 检查交易
func (acc *DB) CheckTransfer(from, to string, amount int64) error {
	if !types.CheckAmount(amount) {
		return types.ErrAmount
	}
	accFrom := acc.LoadAccount(from)
	b := accFrom.GetBalance() - amount
	if b < 0 {
		return types.ErrNoBalance
	}
	return nil
}

// Transfer 执行交易
func (acc *DB) Transfer(from, to string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	accFrom := acc.LoadAccount(from)
	accTo := acc.LoadAccount(to)
	if accFrom.Addr == accTo.Addr {
		return nil, types.ErrSendSameToRecv
	}
	if accFrom.GetBalance()-amount >= 0 {
		copyfrom := *accFrom
		copyto := *accTo

		accFrom.Balance = accFrom.GetBalance() - amount
		accTo.Balance = accTo.GetBalance() + amount

		receiptBalanceFrom := &types.ReceiptAccountTransfer{
			Prev:    &copyfrom,
			Current: accFrom,
		}
		receiptBalanceTo := &types.ReceiptAccountTransfer{
			Prev:    &copyto,
			Current: accTo,
		}
		fromkv := acc.GetKVSet(accFrom)
		tokv := acc.GetKVSet(accTo)
		acc.SaveKVSet(fromkv)
		acc.SaveKVSet(tokv)
		return acc.transferReceipt(fromkv, tokv, receiptBalanceFrom, receiptBalanceTo), nil
	}

	return nil, types.ErrNoBalance
}

func (acc *DB) depositBalance(execaddr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}
	acc1 := acc.LoadAccount(execaddr)
	copyacc := *acc1
	acc1.Balance += amount
	receiptBalance := &types.ReceiptAccountTransfer{
		Prev:    &copyacc,
		Current: acc1,
	}
	kv := acc.GetKVSet(acc1)
	acc.SaveKVSet(kv)
	log1 := &types.ReceiptLog{
		Ty:  int32(types.TyLogDeposit),
		Log: types.Encode(receiptBalance),
	}
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}, nil
}

func (acc *DB) transferReceipt(fromkv, tokv []*types.KeyValue, receiptFrom, receiptTo proto.Message) *types.Receipt {
	ty := int32(types.TyLogTransfer)
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptFrom),
	}
	log2 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptTo),
	}
	kv := make([]*types.KeyValue, 0, len(fromkv)+len(tokv))
	kv = append(kv, fromkv...)
	kv = append(kv, tokv...)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1, log2},
	}
}

// SaveAccount 保存账户到数据库
func (acc *DB) SaveAccount(acc1 *types.Account) {
	set := acc.GetKVSet(acc1)
	for i := 0; i < len(set); i++ {
		err := acc.db.Set(set[i].GetKey(), set[i].Value)
		if err != nil {
			panic(err)
		}
	}
}

//SaveKVSet 保存Key Value set
func (acc *DB) SaveKVSet(set []*types.KeyValue) {
	for i := 0; i < len(set); i++ {
		err := acc.db.Set(set[i].GetKey(), set[i].Value)
		if err != nil {
			panic(err)
		}
	}
}

// GetKVSet 将账户数据转为数据库存储kv
func (acc *DB) GetKVSet(acc1 *types.Account) (kvset []*types.KeyValue) {
	value := types.Encode(acc1)
	kvset = make([]*types.KeyValue, 1)
	kvset[0] = &types.KeyValue{
		Key:   acc.AccountKey(acc1.Addr),
		Value: value,
	}
	return kvset
}

// LoadAccounts 从stateDB中载入若干账户
// TODO:使用API的方式访问,暂时与LoadAccounts()共存,后续将删除LoadAccounts()
func (acc *DB) LoadAccounts(api client.QueueProtocolAPI, addrs []string) (accs []*types.Account, err error) {
	header, err := api.GetLastHeader()
	if err != nil {
		return nil, err
	}
	return acc.loadAccountsHistory(api, addrs, header.GetStateHash())
}

// LoadAccountsDB 载入账户
func (acc *DB) LoadAccountsDB(addrs []string) (accs []*types.Account, err error) {
	for i := 0; i < len(addrs); i++ {
		acc1 := acc.LoadAccount(addrs[i])
		accs = append(accs, acc1)
	}
	return accs, nil
}

// AccountKey return the key of address in DB
func (acc *DB) AccountKey(address string) (key []byte) {
	key = make([]byte, 0, len(acc.accountKeyPerfix)+len(address))
	key = append(key, acc.accountKeyPerfix...)
	key = append(key, []byte(address)...)
	return key
}

func symbolPrefix(execer string, symbol string) string {
	return "mavl-" + execer + "-" + symbol + "-"
}

func symbolExecPrefix(execer string, symbol string) string {
	return "mavl-" + execer + "-" + symbol + "-exec"
}

// GetTotalCoins 获取代币总量
func (acc *DB) GetTotalCoins(api client.QueueProtocolAPI, in *types.ReqGetTotalCoins) (reply *types.ReplyGetTotalCoins, err error) {
	req := types.IterateRangeByStateHash{}
	req.StateHash = in.StateHash
	req.Count = in.Count
	start := symbolPrefix(in.Execer, in.Symbol)
	end := symbolExecPrefix(in.Execer, in.Symbol)
	if in.StartKey == nil {
		req.Start = []byte(start)
	} else {
		req.Start = in.StartKey
	}
	req.End = []byte(end)
	return api.StoreGetTotalCoins(&req)
}

func (acc *DB) loadAccountsHistory(api client.QueueProtocolAPI, addrs []string, stateHash []byte) (accs []*types.Account, err error) {
	get := types.StoreGet{StateHash: stateHash}
	for i := 0; i < len(addrs); i++ {
		get.Keys = append(get.Keys, acc.AccountKey(addrs[i]))
	}

	values, err := api.StoreGet(&get)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(values.Values); i++ {
		value := values.Values[i]
		if value == nil {
			accs = append(accs, &types.Account{Addr: addrs[i]})
		} else {
			var acc types.Account
			err := types.Decode(value, &acc)
			if err != nil {
				return nil, err
			}
			accs = append(accs, &acc)
		}
	}
	return accs, nil
}

// GetBalance 获取某个状态下账户余额
func (acc *DB) GetBalance(api client.QueueProtocolAPI, in *types.ReqBalance) ([]*types.Account, error) {
	// load account
	if in.AssetExec == string(types.GetParaExec([]byte(in.Execer))) || "" == in.Execer {
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := address.CheckAddress(addr); err != nil {
				addr = address.ExecAddress(addr)
			}
			exaddrs = append(exaddrs, addr)
		}
		var accounts []*types.Account
		var err error
		if len(in.StateHash) == 0 {
			accounts, err = acc.LoadAccounts(api, exaddrs)
			if err != nil {
				log.Error("GetBalance", "err", err.Error())
				return nil, err
			}
		} else {
			hash, err := common.FromHex(in.StateHash)
			if err != nil {
				return nil, err
			}
			accounts, err = acc.loadAccountsHistory(api, exaddrs, hash)
			if err != nil {
				log.Error("GetBalance", "err", err.Error())
				return nil, err
			}
		}
		return accounts, nil
	}

	// load exec account
	execaddress := address.ExecAddress(in.GetExecer())
	addrs := in.GetAddresses()
	var accounts []*types.Account
	for _, addr := range addrs {
		var account *types.Account
		var err error
		if len(in.StateHash) == 0 {
			account, err = acc.LoadExecAccountQueue(api, addr, execaddress)
			if err != nil {
				log.Error("GetBalance", "err", err.Error())
				continue
			}
		} else {
			hash, err := common.FromHex(in.StateHash)
			if err != nil {
				return nil, err
			}
			account, err = acc.LoadExecAccountHistoryQueue(api, addr, execaddress, hash)
			if err != nil {
				log.Error("GetBalance", "err", err.Error())
				continue
			}
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

// GetExecBalance 通过account模块获取地址账户在合约中的余额
func (acc *DB) GetExecBalance(api client.QueueProtocolAPI, in *types.ReqGetExecBalance) (reply *types.ReplyGetExecBalance, err error) {
	req := types.StoreList{}
	req.StateHash = in.StateHash

	prefix := symbolExecPrefix(in.Execer, in.Symbol)
	if len(in.ExecAddr) > 0 {
		prefix = prefix + "-" + string(in.ExecAddr) + ":"
	} else {
		prefix = prefix + "-"
	}

	req.Start = []byte(prefix)
	req.End = genPrefixEdge(req.Start)
	req.Suffix = in.Addr
	req.Mode = 2 //1：为[start,end）模式，按前缀或者范围进行查找。2：为prefix + suffix遍历模式，先按前缀查找，再判断后缀是否满足条件。
	req.Count = in.Count

	if len(in.NextKey) > 0 {
		req.Start = in.NextKey
	}

	reply = &types.ReplyGetExecBalance{}
	//log.Info("DB.GetExecBalance", "hash", common.ToHex(req.StateHash), "Prefix", string(req.Start), "End", string(req.End), "Addr", string(req.Suffix))

	res, err := api.StoreList(&req)
	if err != nil {
		err = types.ErrTypeAsset
		return nil, err
	}

	for i := 0; i < len(res.Keys); i++ {
		strKey := string(res.Keys[i])
		log.Info("DB.GetExecBalance process one record", "key", strKey)
		if !strings.HasPrefix(strKey, prefix) {
			log.Error("accountDB.GetExecBalance key does not match prefix", "key", strKey, "prefix", prefix)
			return nil, types.ErrTypeAsset
		}
		//如果prefix形如：mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp:  ,则是查找addr在一个合约地址上的余额，找到一个值即可结束。
		if strings.HasSuffix(prefix, ":") {
			addr := strKey[len(prefix):]
			execAddr := []byte(prefix[(len(prefix) - len(addr) - 1):(len(prefix) - 1)])
			log.Info("DB.GetExecBalance record for specific exec addr", "execAddr", string(execAddr), "addr", addr)
			reply.AddItem(execAddr, res.Values[i])
		} else {
			combinAddr := strKey[len(prefix):]
			addrs := strings.Split(combinAddr, ":")
			if 2 != len(addrs) {
				log.Error("accountDB.GetExecBalance key does not contain exec-addr & addr", "key", strKey, "combinAddr", combinAddr)
				return nil, types.ErrTypeAsset
			}
			//log.Info("DB.GetExecBalance", "execAddr", addrs[0], "addr", addrs[1])
			reply.AddItem([]byte(addrs[0]), res.Values[i])
		}
	}

	reply.NextKey = res.NextKey

	return reply, nil
}

func genPrefixEdge(prefix []byte) (r []byte) {
	for j := 0; j < len(prefix); j++ {
		r = append(r, prefix[j])
	}

	i := len(prefix) - 1
	for i >= 0 {
		if r[i] < 0xff {
			r[i]++
			break
		} else {
			i--
		}
	}

	return r
}

// Mint 铸币
func (acc *DB) Mint(addr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}

	accTo := acc.LoadAccount(addr)
	balance, err := safeAdd(accTo.Balance, amount)
	if err != nil {
		return nil, err
	}

	copyAcc := *accTo
	accTo.Balance = balance

	receipt := &types.ReceiptAccountMint{
		Prev:    &copyAcc,
		Current: accTo,
	}
	kv := acc.GetKVSet(accTo)
	acc.SaveKVSet(kv)
	return acc.mintReceipt(kv, receipt), nil
}

func (acc *DB) mintReceipt(kv []*types.KeyValue, receipt proto.Message) *types.Receipt {
	ty := int32(types.TyLogMint)
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receipt),
	}

	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}
}

// Burn 然收
func (acc *DB) Burn(addr string, amount int64) (*types.Receipt, error) {
	if !types.CheckAmount(amount) {
		return nil, types.ErrAmount
	}

	accTo := acc.LoadAccount(addr)
	if accTo.Balance < amount {
		return nil, types.ErrNoBalance
	}

	copyAcc := *accTo
	accTo.Balance = accTo.Balance - amount

	receipt := &types.ReceiptAccountBurn{
		Prev:    &copyAcc,
		Current: accTo,
	}
	kv := acc.GetKVSet(accTo)
	acc.SaveKVSet(kv)
	return acc.burnReceipt(kv, receipt), nil
}

func (acc *DB) burnReceipt(kv []*types.KeyValue, receipt proto.Message) *types.Receipt {
	ty := int32(types.TyLogBurn)
	log1 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receipt),
	}

	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log1},
	}
}
